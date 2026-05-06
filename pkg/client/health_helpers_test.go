package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// controllableTransport is a programmable Transport mock used by health_test.go.
// It separates "live" calls (everything triggered through HealthAwareTransport's
// Transport methods) from "probe" calls (which the harness routes through a
// custom HealthConfig.Probe that bypasses GetNowBlock). That split keeps tests
// from racing the same counter for both live and probe paths.
type controllableTransport struct {
	name string

	mu       sync.Mutex
	nextErr  error
	probeErr error

	liveCallCount atomic.Int64
	probeCount    atomic.Int64
	closed        atomic.Bool
}

func newCT(name string) *controllableTransport {
	return &controllableTransport{name: name}
}

func (c *controllableTransport) setNextErr(err error) {
	c.mu.Lock()
	c.nextErr = err
	c.mu.Unlock()
}

func (c *controllableTransport) setProbeErr(err error) {
	c.mu.Lock()
	c.probeErr = err
	c.mu.Unlock()
}

func (c *controllableTransport) currentNextErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nextErr
}

func (c *controllableTransport) currentProbeErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.probeErr
}

// Live entry-points for the Transport interface. All increment liveCallCount
// and return the currently configured nextErr.

func (c *controllableTransport) live() error {
	c.liveCallCount.Add(1)
	return c.currentNextErr()
}

func (c *controllableTransport) GetAccount(ctx context.Context, account *core.Account) (*core.Account, error) {
	return &core.Account{}, c.live()
}

func (c *controllableTransport) GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return &api.AccountResourceMessage{}, c.live()
}

func (c *controllableTransport) CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	return &api.BlockExtention{}, c.live()
}

func (c *controllableTransport) GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error) {
	return &api.BlockExtention{}, c.live()
}

func (c *controllableTransport) GetBlockById(ctx context.Context, id []byte) (*core.Block, error) {
	return &core.Block{}, c.live()
}

func (c *controllableTransport) GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	return &api.BlockListExtention{}, c.live()
}

func (c *controllableTransport) GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	return &api.BlockListExtention{}, c.live()
}

func (c *controllableTransport) GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	return &api.TransactionInfoList{}, c.live()
}

func (c *controllableTransport) GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error) {
	return &core.Transaction{}, c.live()
}

func (c *controllableTransport) GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error) {
	return &core.TransactionInfo{}, c.live()
}

func (c *controllableTransport) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	return &api.Return{}, c.live()
}

func (c *controllableTransport) CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return &api.EstimateEnergyMessage{}, c.live()
}

func (c *controllableTransport) DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) GetContract(ctx context.Context, address []byte) (*core.SmartContract, error) {
	return &core.SmartContract{}, c.live()
}

func (c *controllableTransport) UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error) {
	return &api.AccountResourceMessage{}, c.live()
}

func (c *controllableTransport) GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return &api.DelegatedResourceList{}, c.live()
}

func (c *controllableTransport) GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	return &api.DelegatedResourceList{}, c.live()
}

func (c *controllableTransport) GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	return &core.DelegatedResourceAccountIndex{}, c.live()
}

func (c *controllableTransport) GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error) {
	return &core.DelegatedResourceAccountIndex{}, c.live()
}

func (c *controllableTransport) GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	return &api.CanDelegatedMaxSizeResponseMessage{}, c.live()
}

func (c *controllableTransport) DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, c.live()
}

func (c *controllableTransport) GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error) {
	return &core.AssetIssueContract{}, c.live()
}

func (c *controllableTransport) GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error) {
	return &api.AssetIssueList{}, c.live()
}

func (c *controllableTransport) ListNodes(ctx context.Context) (*api.NodeList, error) {
	return &api.NodeList{}, c.live()
}

func (c *controllableTransport) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	return &core.ChainParameters{}, c.live()
}

func (c *controllableTransport) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	return &api.NumberMessage{}, c.live()
}

func (c *controllableTransport) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	return &api.NumberMessage{}, c.live()
}

func (c *controllableTransport) Close() error {
	c.closed.Store(true)
	return nil
}

// testHarness wires up a HealthAwareTransport with N controllableTransport
// nodes, one per entry in tiers. nodes[i].Tier = tiers[i]. The custom
// HealthConfig.Probe routes probes to controllableTransport.probeErr/probeCount,
// keeping live and probe paths fully separated in tests.
type testHarness struct {
	t         *testing.T
	nodes     []*controllableTransport
	transport *HealthAwareTransport
	metrics   *mockMetricsCollector
}

func newHarness(t *testing.T, tiers []int, cfg HealthConfig) *testHarness {
	t.Helper()

	nodes := make([]*controllableTransport, len(tiers))
	cfgs := make([]NodeConfig, len(tiers))
	for i, tier := range tiers {
		nodes[i] = newCT(fmt.Sprintf("n%d-t%d", i, tier))
		cfgs[i] = NodeConfig{
			Protocol: ProtocolGRPC,
			Address:  nodes[i].name,
			Tier:     tier,
		}
	}

	factory := func(nc NodeConfig) (Transport, error) {
		for _, n := range nodes {
			if n.name == nc.Address {
				return n, nil
			}
		}
		return nil, errors.New("test: unknown node " + nc.Address)
	}

	if cfg.Probe == nil {
		cfg.Probe = func(ctx context.Context, tr Transport) error {
			ct := tr.(*controllableTransport)
			ct.probeCount.Add(1)
			return ct.currentProbeErr()
		}
	}

	metrics := &mockMetricsCollector{}
	ht, err := NewHealthAwareTransport(cfgs, factory, cfg, metrics, "tron")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ht.Close()
	})
	return &testHarness{t: t, nodes: nodes, transport: ht, metrics: metrics}
}

// nodeHealthy returns the current healthy flag of node i.
func (h *testHarness) nodeHealthy(i int) bool {
	h.transport.nodes[i].mu.Lock()
	defer h.transport.nodes[i].mu.Unlock()
	return h.transport.nodes[i].healthy
}

// activeTier reads the current activeTier atomically.
func (h *testHarness) activeTier() int64 {
	return h.transport.activeTier.Load()
}

// lastPool returns the most recent SetPoolHealth call, or recordedPool zero
// value if none happened yet.
func (h *testHarness) lastPool() recordedPool {
	if len(h.metrics.pools) == 0 {
		return recordedPool{}
	}
	return h.metrics.pools[len(h.metrics.pools)-1]
}
