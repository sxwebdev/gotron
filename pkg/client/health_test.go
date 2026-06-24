package client

import (
	"context"
	"errors"
	"io"
	"net"
	"runtime"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcErr returns a *TransportError wrapping a gRPC status error with code c.
// All real RPC errors flow through transportErrorInterceptor, so the
// classifier must handle the wrapped form.
func grpcErr(c codes.Code) error {
	return &TransportError{
		Host:     "test",
		Protocol: "grpc",
		Method:   "/test/Method",
		Err:      status.Error(c, "test"),
	}
}

func bgctx() context.Context { return context.Background() }

// (a) Все primary healthy → round-robin внутри tier 0.
func TestHealthAware_AllPrimaryHealthy_RoundRobin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 0, 0}, HealthConfig{})

		for range 6 {
			_, err := h.transport.GetAccount(bgctx(), &core.Account{})
			require.NoError(t, err)
		}

		assert.Equal(t, int64(2), h.nodes[0].liveCallCount.Load())
		assert.Equal(t, int64(2), h.nodes[1].liveCallCount.Load())
		assert.Equal(t, int64(2), h.nodes[2].liveCallCount.Load())
	})
}

// (b) Primary падает (M=2 подряд network errors) → switch на tier 1.
func TestHealthAware_PrimaryFails_FailoverToTier1(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 0, 1}, HealthConfig{
			FailureThreshold: 2,
			SuccessThreshold: 2,
			// Long intervals to keep probes out of the picture for this test.
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})

		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		h.nodes[1].setNextErr(grpcErr(codes.Unavailable))

		// Four live calls — round-robin will hit each primary twice → both unhealthy.
		for range 4 {
			_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		}
		assert.False(t, h.nodeHealthy(0))
		assert.False(t, h.nodeHealthy(1))
		assert.True(t, h.nodeHealthy(2))
		assert.Equal(t, int64(1), h.activeTier())

		// Fifth call should go to tier 1.
		h.nodes[2].setNextErr(nil)
		_, err := h.transport.GetAccount(bgctx(), &core.Account{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), h.nodes[2].liveCallCount.Load())
	})
}

// (c) Primary восстанавливается (N=2 подряд успешных probe) → возврат на tier 0.
func TestHealthAware_PrimaryRecovers_BackToTier0(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			SuccessThreshold:     2,
			HealthyInterval:      30 * time.Second,
			UnhealthyInterval:    5 * time.Second,
			InactiveTierInterval: time.Hour, // keep fallback quiet
		})

		// Knock primary out via a single live failure.
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		require.False(t, h.nodeHealthy(0))
		require.Equal(t, int64(1), h.activeTier())

		// Probes will now succeed; need 2 in a row to recover.
		h.nodes[0].setProbeErr(nil)
		h.nodes[0].setNextErr(nil)

		// First unhealthy probe (after 5s).
		time.Sleep(6 * time.Second)
		synctest.Wait()
		assert.False(t, h.nodeHealthy(0), "after 1 successful probe, still unhealthy (need 2)")

		// Second probe → recovery.
		time.Sleep(5 * time.Second)
		synctest.Wait()
		assert.True(t, h.nodeHealthy(0))
		assert.Equal(t, int64(0), h.activeTier())

		// Live call now hits primary.
		live0Before := h.nodes[0].liveCallCount.Load()
		_, err := h.transport.GetAccount(bgctx(), &core.Account{})
		require.NoError(t, err)
		assert.Equal(t, live0Before+1, h.nodes[0].liveCallCount.Load())
	})
}

// (d) Все ноды unhealthy → ErrNoHealthyNodes.
func TestHealthAware_AllUnhealthy_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		h.nodes[1].setNextErr(grpcErr(codes.Unavailable))

		// Burn through all nodes; both flip to unhealthy.
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		require.False(t, h.nodeHealthy(0))
		require.False(t, h.nodeHealthy(1))

		_, err := h.transport.GetAccount(bgctx(), &core.Account{})
		assert.ErrorIs(t, err, ErrNoHealthyNodes)
	})
}

// (e) Multi-tier: tier 0 down, tier 1 down → tier 2 active.
func TestHealthAware_MultiTier_TwoTiersDown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1, 2}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{}) // tier 0 → unhealthy

		h.nodes[1].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{}) // tier 1 → unhealthy

		require.False(t, h.nodeHealthy(0))
		require.False(t, h.nodeHealthy(1))
		require.True(t, h.nodeHealthy(2))
		assert.Equal(t, int64(2), h.activeTier())

		_, err := h.transport.GetAccount(bgctx(), &core.Account{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), h.nodes[2].liveCallCount.Load())
	})
}

// (f) Тикеры на 30s — probeCount растёт по виртуальному времени.
func TestHealthAware_HealthyProbeInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0}, HealthConfig{
			HealthyInterval:      30 * time.Second,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		// Probe must succeed so node stays healthy and we keep using HealthyInterval.
		h.nodes[0].setProbeErr(nil)

		synctest.Wait()
		assert.Equal(t, int64(0), h.nodes[0].probeCount.Load())

		time.Sleep(31 * time.Second)
		synctest.Wait()
		assert.Equal(t, int64(1), h.nodes[0].probeCount.Load())

		time.Sleep(30 * time.Second)
		synctest.Wait()
		assert.Equal(t, int64(2), h.nodes[0].probeCount.Load())
	})
}

// (g) Concurrent live requests + background probes don't race (run with -race).
func TestHealthAware_ConcurrentRequestsRace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 0, 1}, HealthConfig{
			HealthyInterval:      100 * time.Millisecond,
			UnhealthyInterval:    100 * time.Millisecond,
			InactiveTierInterval: 100 * time.Millisecond,
		})
		h.nodes[0].setProbeErr(nil)
		h.nodes[1].setProbeErr(nil)
		h.nodes[2].setProbeErr(nil)

		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() {
				for range 4 {
					_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
				}
			})
		}
		// Let the probes tick a few times during the storm.
		time.Sleep(500 * time.Millisecond)
		wg.Wait()
		synctest.Wait()
	})
}

// (h) Close() корректно останавливает все горутины — без leak'а.
func TestHealthAware_CloseStopsGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()

	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 0, 1, 1, 2}, HealthConfig{
			HealthyInterval:      100 * time.Millisecond,
			UnhealthyInterval:    100 * time.Millisecond,
			InactiveTierInterval: 100 * time.Millisecond,
		})
		// Drive a few probe rounds before close.
		time.Sleep(500 * time.Millisecond)
		synctest.Wait()
		require.NoError(t, h.transport.Close())
	})

	// Give the runtime a beat — Close + wg.Wait happens inside the bubble,
	// but goroutine cleanup may lag behind by one scheduler tick.
	for range 50 {
		if runtime.NumGoroutine() <= before+2 {
			break
		}
		runtime.Gosched()
	}
	assert.LessOrEqual(t, runtime.NumGoroutine(), before+2,
		"goroutine leak: had %d, now %d", before, runtime.NumGoroutine())
}

// (i) RPC-логические ошибки (InvalidArgument) НЕ помечают ноду unhealthy.
func TestHealthAware_LogicalErrorDoesNotMarkUnhealthy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0}, HealthConfig{
			FailureThreshold:     2,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.InvalidArgument))

		for range 10 {
			_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		}
		assert.True(t, h.nodeHealthy(0))
		assert.Equal(t, int64(10), h.nodes[0].liveCallCount.Load())
	})
}

// (j) Live-failure → нода unhealthy сразу при достижении порога; следующий
// запрос идёт на tier 1.
func TestHealthAware_FailureThresholdSynchronous(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     2,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))

		_, _ = h.transport.GetAccount(bgctx(), &core.Account{}) // failure 1
		assert.True(t, h.nodeHealthy(0))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{}) // failure 2 → unhealthy
		assert.False(t, h.nodeHealthy(0))

		h.nodes[1].setNextErr(nil)
		_, err := h.transport.GetAccount(bgctx(), &core.Account{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), h.nodes[1].liveCallCount.Load())
	})
}

// (k) Recovery threshold N=2 — один успешный probe ещё не возвращает в пул.
func TestHealthAware_RecoveryThreshold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			SuccessThreshold:     2,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    5 * time.Second,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		require.False(t, h.nodeHealthy(0))

		h.nodes[0].setProbeErr(nil)

		time.Sleep(6 * time.Second)
		synctest.Wait()
		assert.False(t, h.nodeHealthy(0), "1 successful probe must not be enough")

		time.Sleep(5 * time.Second)
		synctest.Wait()
		assert.True(t, h.nodeHealthy(0), "2 successful probes recover the node")
	})
}

// (m) Метрики SetPoolHealth обновляются при transition'ах.
func TestHealthAware_PoolMetricsUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 0}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		// Initial metric publish: 2 nodes, all healthy.
		assert.Equal(t, recordedPool{"tron", 2, 2, 0}, h.lastPool())

		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})

		assert.Equal(t, recordedPool{"tron", 2, 1, 1}, h.lastPool())
	})
}

// (n) Кастомный ClassifyErr имеет приоритет.
func TestHealthAware_CustomClassifyErr(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
			// Treat any non-nil error as network failure.
			ClassifyErr: func(err error) bool { return err != nil },
		})
		// InvalidArgument would normally be ignored, but custom classifier counts it.
		h.nodes[0].setNextErr(grpcErr(codes.InvalidArgument))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		assert.False(t, h.nodeHealthy(0))
	})
}

// (o) Probe в полёте + Close — клин не возникает (ProbeTimeout не блокирует Close).
func TestHealthAware_CloseDuringProbe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		blocker := make(chan struct{})
		probeStarted := make(chan struct{})
		var once sync.Once

		h := newHarness(t, []int{0}, HealthConfig{
			HealthyInterval:      100 * time.Millisecond,
			UnhealthyInterval:    100 * time.Millisecond,
			InactiveTierInterval: 100 * time.Millisecond,
			ProbeTimeout:         time.Hour,
			Probe: func(ctx context.Context, _ Transport) error {
				once.Do(func() { close(probeStarted) })
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-blocker:
					return nil
				}
			},
		})

		// Wait for the first probe to land in the blocking branch.
		time.Sleep(150 * time.Millisecond)
		synctest.Wait()

		select {
		case <-probeStarted:
		default:
			t.Fatal("probe never started")
		}

		// Close must unblock the probe via stopCh-driven ctx cancellation.
		closeDone := make(chan error, 1)
		go func() { closeDone <- h.transport.Close() }()

		select {
		case err := <-closeDone:
			require.NoError(t, err)
		case <-time.After(time.Minute):
			t.Fatal("Close hung — probe ctx was not cancelled by stopCh")
		}
		_ = blocker // unused after Close, but harmless
	})
}

// (p) HTTP 5xx считается network — нода помечается unhealthy.
func TestHealthAware_HTTPStatus5xxIsNetwork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(&TransportError{
			Host: "http://x", Protocol: "http", Method: "/x",
			Err: &HTTPStatusError{Code: 503, Body: "service unavailable"},
		})
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		assert.False(t, h.nodeHealthy(0))
	})
}

// (q) HTTP 4xx (logical) — нода остаётся healthy.
func TestHealthAware_HTTP4xxIsLogical(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(&TransportError{
			Host: "http://x", Protocol: "http", Method: "/x",
			Err: &HTTPStatusError{Code: 400, Body: "bad request"},
		})
		for range 5 {
			_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		}
		assert.True(t, h.nodeHealthy(0))
	})
}

// (r) cfg.Health.Disabled = true — клиент собирает плоский RoundRobinTransport,
// никаких background probe.
func TestHealthAware_DisabledFallsBackToRoundRobin(t *testing.T) {
	cfg := Config{
		Nodes: []NodeConfig{
			{Protocol: ProtocolGRPC, Address: "tron-grpc.publicnode.com:443", UseTLS: true},
		},
		Health: HealthConfig{Disabled: true},
	}
	c, err := New(cfg)
	require.NoError(t, err)
	defer func() { _ = c.Close() }()

	// The transport stack should be RoundRobinTransport (no Metrics → no wrapping).
	_, ok := c.transport.(*RoundRobinTransport)
	assert.True(t, ok, "expected RoundRobinTransport when Health.Disabled, got %T", c.transport)
}

// (s) Только tier 0, fallback нет: при падении всех primary → ErrNoHealthyNodes.
func TestHealthAware_NoFallbackTier_AllPrimaryDown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 0}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		h.nodes[1].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})

		_, err := h.transport.GetAccount(bgctx(), &core.Account{})
		assert.ErrorIs(t, err, ErrNoHealthyNodes)
		assert.Equal(t, int64(-1), h.activeTier())
	})
}

// (t) Все tier'ы down — fallback всё равно регулярно пробуется.
func TestHealthAware_AllTiersDown_FallbackProbedToo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      time.Hour,
			UnhealthyInterval:    5 * time.Second,
			InactiveTierInterval: time.Hour,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		h.nodes[1].setNextErr(grpcErr(codes.Unavailable))

		// Force everything unhealthy via live calls.
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{}) // hits tier 1 (only healthy left)
		require.False(t, h.nodeHealthy(0))
		require.False(t, h.nodeHealthy(1))

		// Both unhealthy → both probed at UnhealthyInterval.
		h.nodes[0].setProbeErr(grpcErr(codes.Unavailable))
		h.nodes[1].setProbeErr(grpcErr(codes.Unavailable))
		time.Sleep(11 * time.Second)
		synctest.Wait()

		assert.GreaterOrEqual(t, h.nodes[0].probeCount.Load(), int64(1), "primary must be probed")
		assert.GreaterOrEqual(t, h.nodes[1].probeCount.Load(), int64(1), "fallback must be probed too")
	})
}

// (u) Healthy fallback в неактивном tier пробуется на InactiveTierInterval —
// в начале теста primary живёт, fallback не должен получать probe слишком часто.
func TestHealthAware_InactiveTier_RareProbing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			HealthyInterval:      30 * time.Second,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: 5 * time.Minute,
		})
		h.nodes[0].setProbeErr(nil)
		h.nodes[1].setProbeErr(nil)

		// 4 minutes: primary probed ~8 times, fallback 0 (< 5m).
		time.Sleep(4 * time.Minute)
		synctest.Wait()

		assert.GreaterOrEqual(t, h.nodes[0].probeCount.Load(), int64(7))
		assert.LessOrEqual(t, h.nodes[0].probeCount.Load(), int64(9))
		assert.Equal(t, int64(0), h.nodes[1].probeCount.Load(),
			"fallback must not be probed before InactiveTierInterval elapses")
	})
}

// (v) Tier-shift вверх (primary упал) → fallback'ы переключаются на HealthyInterval.
func TestHealthAware_TierShift_PromotesFallbackProbing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			HealthyInterval:      30 * time.Second,
			UnhealthyInterval:    time.Hour,
			InactiveTierInterval: 10 * time.Minute,
		})
		h.nodes[1].setProbeErr(nil)

		// Settle a moment without any probes to fallback (1 min < 10 min).
		time.Sleep(time.Minute)
		synctest.Wait()
		require.Equal(t, int64(0), h.nodes[1].probeCount.Load())

		// Knock primary out → activeTier shifts to 1 → fallback should now use HealthyInterval.
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		require.Equal(t, int64(1), h.activeTier())

		time.Sleep(31 * time.Second)
		synctest.Wait()
		assert.GreaterOrEqual(t, h.nodes[1].probeCount.Load(), int64(1),
			"fallback should probe at HealthyInterval after tier-shift")
	})
}

// (w) Tier-shift вниз (primary восстанавливается) → fallback переключается обратно
// на InactiveTierInterval.
func TestHealthAware_TierShift_DemotesFallbackProbing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, []int{0, 1}, HealthConfig{
			FailureThreshold:     1,
			SuccessThreshold:     1,
			HealthyInterval:      30 * time.Second,
			UnhealthyInterval:    5 * time.Second,
			InactiveTierInterval: 10 * time.Minute,
		})
		h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
		h.nodes[1].setProbeErr(nil)

		// Knock primary out.
		_, _ = h.transport.GetAccount(bgctx(), &core.Account{})
		require.Equal(t, int64(1), h.activeTier())

		// Recover primary via probe.
		h.nodes[0].setProbeErr(nil)
		h.nodes[0].setNextErr(nil)
		time.Sleep(6 * time.Second)
		synctest.Wait()
		require.True(t, h.nodeHealthy(0))
		require.Equal(t, int64(0), h.activeTier())

		fallbackProbesAtShift := h.nodes[1].probeCount.Load()

		// 2 minutes with primary healthy → fallback in inactive tier, no extra probes
		// (10m > 2m).
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		assert.Equal(t, fallbackProbesAtShift, h.nodes[1].probeCount.Load(),
			"fallback should NOT be probed while primary is the active tier")
	})
}

// ===== isNetworkError unit tests =====

func TestIsNetworkError_GRPCCodes(t *testing.T) {
	for _, c := range []codes.Code{
		codes.Unavailable, codes.DeadlineExceeded, codes.Aborted,
		codes.ResourceExhausted, codes.Internal, codes.Unknown,
	} {
		assert.True(t, isNetworkError(grpcErr(c)), "code=%s should be network", c)
	}
	for _, c := range []codes.Code{
		codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated,
		codes.FailedPrecondition, codes.OutOfRange, codes.Unimplemented,
	} {
		assert.False(t, isNetworkError(grpcErr(c)), "code=%s should be logical", c)
	}
}

func TestIsNetworkError_HTTPStatus(t *testing.T) {
	wrap := func(code int) error {
		return &TransportError{Host: "h", Protocol: "http", Method: "/x", Err: &HTTPStatusError{Code: code}}
	}
	assert.True(t, isNetworkError(wrap(503)))
	assert.True(t, isNetworkError(wrap(500)))
	assert.True(t, isNetworkError(wrap(408)))
	assert.True(t, isNetworkError(wrap(429)))
	assert.False(t, isNetworkError(wrap(400)))
	assert.False(t, isNetworkError(wrap(404)))
}

func TestIsNetworkError_Misc(t *testing.T) {
	assert.False(t, isNetworkError(nil))
	assert.True(t, isNetworkError(context.DeadlineExceeded))
	assert.False(t, isNetworkError(context.Canceled))
	assert.True(t, isNetworkError(io.EOF))
	assert.True(t, isNetworkError(io.ErrUnexpectedEOF))
	assert.True(t, isNetworkError(net.ErrClosed))
	assert.False(t, isNetworkError(errors.New("random")))
}
