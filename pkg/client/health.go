package client

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// nodeState holds the runtime state of a single node managed by HealthAwareTransport.
// All mutable fields are guarded by mu; transport, address and tier are immutable
// after construction.
type nodeState struct {
	transport Transport
	address   string
	tier      int

	mu                 sync.Mutex
	healthy            bool
	consecutiveSuccess int
	consecutiveFailure int

	// notifyCh is a size-1 buffered channel used by the parent transport to
	// wake the per-node health-loop when the active tier changes (so the loop
	// can recompute its probe interval without waiting for the current timer).
	notifyCh chan struct{}
}

// HealthAwareTransport selects the next underlying Transport across multiple
// nodes grouped by NodeConfig.Tier. Lower-numbered tiers are preferred:
// requests are routed to the lowest tier that has at least one healthy node.
// A higher tier is only used when every node of every lower tier is currently
// marked unhealthy.
//
// A background health-checker probes every node periodically (one goroutine
// per node, started in the constructor and stopped by Close). Probe cadence
// depends on node state and tier: HealthyInterval for the active tier,
// InactiveTierInterval for healthy fallbacks, UnhealthyInterval for any
// unhealthy node. Live RPC outcomes also feed the per-node thresholds, so
// reality and probes converge quickly.
type HealthAwareTransport struct {
	cfg HealthConfig

	nodes    []*nodeState
	tiers    [][]*nodeState
	tierKeys []int
	counters []atomic.Uint64

	metrics    MetricsCollector
	blockchain string
	publishMu  sync.Mutex

	// activeTier is the lowest-numbered tier that currently has at least one
	// healthy node, or -1 when every node is unhealthy. Stored atomically so
	// healthCheckLoop can read it without acquiring publishMu.
	activeTier atomic.Int64

	stopCh    chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// Compile-time interface check.
var _ Transport = (*HealthAwareTransport)(nil)

// NewHealthAwareTransport builds a HealthAwareTransport from a list of node
// configs. factory is called once per node to construct the underlying
// Transport (typically createTransportFromNode). cfg defaults are filled in
// when fields are zero. If cfg.Disabled is true, callers should skip this
// constructor entirely and build a RoundRobinTransport instead — the field
// is honored here for symmetry only by stopping the background loops.
//
// On error, any transports already created are closed.
func NewHealthAwareTransport(
	nodes []NodeConfig,
	factory func(NodeConfig) (Transport, error),
	cfg HealthConfig,
	metrics MetricsCollector,
	blockchain string,
) (*HealthAwareTransport, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("%w: at least one node is required", ErrInvalidConfig)
	}
	cfg = cfg.withDefaults()
	if blockchain == "" {
		blockchain = "tron"
	}

	created := make([]Transport, 0, len(nodes))
	states := make([]*nodeState, 0, len(nodes))
	for i, nc := range nodes {
		tr, err := factory(nc)
		if err != nil {
			for _, t := range created {
				_ = t.Close()
			}
			return nil, fmt.Errorf("failed to create transport for node %d: %w", i, err)
		}
		created = append(created, tr)
		states = append(states, &nodeState{
			transport: tr,
			address:   nc.Address,
			tier:      nc.Tier,
			healthy:   true,
			notifyCh:  make(chan struct{}, 1),
		})
	}

	tierMap := make(map[int][]*nodeState)
	for _, s := range states {
		tierMap[s.tier] = append(tierMap[s.tier], s)
	}
	keys := make([]int, 0, len(tierMap))
	for k := range tierMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	tiers := make([][]*nodeState, len(keys))
	for i, k := range keys {
		tiers[i] = tierMap[k]
	}

	h := &HealthAwareTransport{
		cfg:        cfg,
		nodes:      states,
		tiers:      tiers,
		tierKeys:   keys,
		counters:   make([]atomic.Uint64, len(keys)),
		metrics:    metrics,
		blockchain: blockchain,
		stopCh:     make(chan struct{}),
	}
	h.activeTier.Store(int64(keys[0])) // all healthy at start → lowest tier is active
	h.publishPoolMetrics()

	if !cfg.Disabled {
		for _, s := range states {
			h.wg.Add(1)
			go h.healthCheckLoop(s)
		}
	}
	return h, nil
}

// withDefaults fills zero-valued fields with sensible defaults. Returns a copy.
func (c HealthConfig) withDefaults() HealthConfig {
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 2
	}
	if c.SuccessThreshold <= 0 {
		c.SuccessThreshold = 2
	}
	if c.HealthyInterval <= 0 {
		c.HealthyInterval = 30 * time.Second
	}
	if c.UnhealthyInterval <= 0 {
		c.UnhealthyInterval = 5 * time.Second
	}
	if c.InactiveTierInterval <= 0 {
		c.InactiveTierInterval = 5 * time.Minute
	}
	if c.ProbeTimeout <= 0 {
		c.ProbeTimeout = 5 * time.Second
	}
	if c.Probe == nil {
		c.Probe = func(ctx context.Context, t Transport) error {
			_, err := t.GetNowBlock(ctx)
			return err
		}
	}
	if c.ClassifyErr == nil {
		c.ClassifyErr = isNetworkError
	}
	if c.Logger == nil {
		c.Logger = noopLogger{}
	}
	return c
}

// next picks the next healthy node according to tier priority + round-robin
// within a tier. Returns ErrNoHealthyNodes when nothing is available.
func (h *HealthAwareTransport) next() (*nodeState, error) {
	for tierIdx, group := range h.tiers {
		startIdx := h.counters[tierIdx].Add(1) - 1
		groupLen := uint64(len(group))
		for i := range groupLen {
			n := group[(startIdx+i)%groupLen]
			n.mu.Lock()
			ok := n.healthy
			n.mu.Unlock()
			if ok {
				return n, nil
			}
		}
	}
	return nil, ErrNoHealthyNodes
}

// recordOutcome feeds the result of a live RPC call into the per-node counters.
// Successes always count toward consecutiveSuccess; only network-level errors
// (per cfg.ClassifyErr) count toward consecutiveFailure — logical errors
// leave node health untouched.
func (h *HealthAwareTransport) recordOutcome(n *nodeState, err error) {
	if err == nil {
		h.markSuccess(n)
		return
	}
	if h.cfg.ClassifyErr(err) {
		h.markFailure(n, err)
	}
}

// markSuccess registers a successful probe or live call. When an unhealthy
// node accumulates SuccessThreshold consecutive successes, it transitions
// back to healthy and the active tier is recomputed.
func (h *HealthAwareTransport) markSuccess(n *nodeState) {
	n.mu.Lock()
	n.consecutiveFailure = 0
	n.consecutiveSuccess++
	transition := !n.healthy && n.consecutiveSuccess >= h.cfg.SuccessThreshold
	if transition {
		n.healthy = true
	}
	n.mu.Unlock()
	if transition {
		h.logTransition(n, "healthy", nil)
		h.onStateChange()
	}
}

// markFailure registers a network-level failure. When a healthy node hits
// FailureThreshold consecutive failures, it transitions to unhealthy and
// the active tier is recomputed (which may cause traffic to fail over).
func (h *HealthAwareTransport) markFailure(n *nodeState, cause error) {
	n.mu.Lock()
	n.consecutiveSuccess = 0
	n.consecutiveFailure++
	transition := n.healthy && n.consecutiveFailure >= h.cfg.FailureThreshold
	if transition {
		n.healthy = false
	}
	n.mu.Unlock()
	if transition {
		h.logTransition(n, "unhealthy", cause)
		h.onStateChange()
	}
}

// onStateChange recomputes activeTier, publishes pool-health metrics, and
// pings every node's notifyCh when the active tier has shifted so the
// per-node loops can recompute their probe interval immediately.
func (h *HealthAwareTransport) onStateChange() {
	h.publishMu.Lock()
	defer h.publishMu.Unlock()

	newActive := int64(-1)
	for tierIdx, group := range h.tiers {
		found := false
		for _, n := range group {
			n.mu.Lock()
			ok := n.healthy
			n.mu.Unlock()
			if ok {
				newActive = int64(h.tierKeys[tierIdx])
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	old := h.activeTier.Swap(newActive)
	h.publishPoolMetricsLocked()

	if old != newActive {
		h.cfg.Logger.Infof("gotron: tier shift blockchain=%s from=%d to=%d",
			h.blockchain, old, newActive)
		for _, n := range h.nodes {
			select {
			case n.notifyCh <- struct{}{}:
			default:
			}
		}
	}
}

// publishPoolMetrics is the public entry-point — takes the lock.
func (h *HealthAwareTransport) publishPoolMetrics() {
	h.publishMu.Lock()
	defer h.publishMu.Unlock()
	h.publishPoolMetricsLocked()
}

// publishPoolMetricsLocked must be called with publishMu held.
func (h *HealthAwareTransport) publishPoolMetricsLocked() {
	if h.metrics == nil {
		return
	}
	total := len(h.nodes)
	healthy := 0
	for _, n := range h.nodes {
		n.mu.Lock()
		if n.healthy {
			healthy++
		}
		n.mu.Unlock()
	}
	h.metrics.SetPoolHealth(h.blockchain, total, healthy, total-healthy)
}

// healthCheckLoop is the per-node background probing goroutine. It exits when
// stopCh is closed. Three wakeup sources:
//   - stopCh: shutdown
//   - notifyCh: active tier shifted, recompute interval without probing now
//   - timer.C: time to probe
func (h *HealthAwareTransport) healthCheckLoop(n *nodeState) {
	defer h.wg.Done()
	timer := time.NewTimer(h.intervalFor(n))
	defer timer.Stop()
	for {
		select {
		case <-h.stopCh:
			return
		case <-n.notifyCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(h.intervalFor(n))
		case <-timer.C:
			h.probeOnce(n)
			timer.Reset(h.intervalFor(n))
		}
	}
}

// probeOnce runs a single health-probe under ProbeTimeout, with a context that
// is also cancelled when stopCh closes — so Close() never has to wait for an
// in-flight probe to finish naturally.
func (h *HealthAwareTransport) probeOnce(n *nodeState) {
	defer func() {
		if r := recover(); r != nil {
			h.cfg.Logger.Infof("gotron: health probe panic blockchain=%s address=%s panic=%v",
				h.blockchain, n.address, r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), h.cfg.ProbeTimeout)
	defer cancel()

	// Plumb stopCh into ctx without leaking a goroutine: the helper below
	// exits as soon as the probe returns (defer close(done)) or stopCh closes.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-h.stopCh:
			cancel()
		case <-done:
		}
	}()

	err := h.cfg.Probe(ctx, n.transport)
	if err == nil {
		h.markSuccess(n)
		return
	}
	if h.cfg.ClassifyErr(err) {
		h.markFailure(n, err)
	}
}

// intervalFor decides the next probe delay for a node:
//   - unhealthy → UnhealthyInterval (any tier)
//   - healthy + active tier → HealthyInterval
//   - healthy + inactive tier → InactiveTierInterval
func (h *HealthAwareTransport) intervalFor(n *nodeState) time.Duration {
	n.mu.Lock()
	healthy := n.healthy
	n.mu.Unlock()
	if !healthy {
		return h.cfg.UnhealthyInterval
	}
	if int64(n.tier) == h.activeTier.Load() {
		return h.cfg.HealthyInterval
	}
	return h.cfg.InactiveTierInterval
}

func (h *HealthAwareTransport) logTransition(n *nodeState, to string, cause error) {
	if cause != nil {
		h.cfg.Logger.Infof("gotron: node state transition blockchain=%s address=%s tier=%d to=%s cause=%v",
			h.blockchain, n.address, n.tier, to, cause)
		return
	}
	h.cfg.Logger.Infof("gotron: node state transition blockchain=%s address=%s tier=%d to=%s",
		h.blockchain, n.address, n.tier, to)
}

// Close stops the health-check loops, waits for them to exit, and closes every
// underlying transport. Safe to call multiple times.
func (h *HealthAwareTransport) Close() error {
	h.closeOnce.Do(func() {
		close(h.stopCh)
	})
	h.wg.Wait()
	var lastErr error
	for _, n := range h.nodes {
		if err := n.transport.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ===== Transport interface methods =====

// Account operations

func (h *HealthAwareTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetAccount(ctx, account)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetAccountResource(ctx, account)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.CreateAccount(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

// Block operations

func (h *HealthAwareTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetNowBlock(ctx)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetBlockByNum(ctx, num)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetBlockById(ctx, id)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetBlockByLimitNext(ctx, start, end)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetBlockByLatestNum(ctx, num)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetTransactionInfoByBlockNum(ctx, num)
	h.recordOutcome(n, callErr)
	return res, callErr
}

// Transaction operations

func (h *HealthAwareTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetTransactionById(ctx, id)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetTransactionInfoById(ctx, id)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.BroadcastTransaction(ctx, tx)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.CreateTransaction(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

// Contract operations

func (h *HealthAwareTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.TriggerContract(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.TriggerConstantContract(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.EstimateEnergy(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.DeployContract(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetContract(ctx, address)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.UpdateSetting(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.UpdateEnergyLimit(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

// Resource operations

func (h *HealthAwareTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetAccountResourceMessage(ctx, account)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetDelegatedResource(ctx, msg)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetDelegatedResourceV2(ctx, msg)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetDelegatedResourceAccountIndex(ctx, address)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetDelegatedResourceAccountIndexV2(ctx, address)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetCanDelegatedMaxSize(ctx, msg)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.DelegateResource(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.UnDelegateResource(ctx, contract)
	h.recordOutcome(n, callErr)
	return res, callErr
}

// Asset operations

func (h *HealthAwareTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetAssetIssueById(ctx, id)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetAssetIssueListByName(ctx, name)
	h.recordOutcome(n, callErr)
	return res, callErr
}

// Network operations

func (h *HealthAwareTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.ListNodes(ctx)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetChainParameters(ctx)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.GetNextMaintenanceTime(ctx)
	h.recordOutcome(n, callErr)
	return res, callErr
}

func (h *HealthAwareTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	n, err := h.next()
	if err != nil {
		return nil, err
	}
	res, callErr := n.transport.TotalTransaction(ctx)
	h.recordOutcome(n, callErr)
	return res, callErr
}
