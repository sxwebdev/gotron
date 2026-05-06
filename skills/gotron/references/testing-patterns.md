# Testing Patterns

## Test Location and Package

Two test layers:

- **Integration tests** live in `tests/` as `package tests` and hit real public Tron nodes.
- **Unit tests** for transport-layer components (the health-checker, classifier) live alongside the source in `pkg/client/` as `package client` and use `testing/synctest` for deterministic virtual-time tests with no network.

## Test Helpers

**File:** `tests/common_test.go`

Three client constructors for different test scenarios:

```go
func newGRPCClient(t *testing.T) *client.Client   // single gRPC node
func newHTTPClient(t *testing.T) *client.Client    // single HTTP node
func newMultiNodeClient(t *testing.T) *client.Client // gRPC + HTTP round-robin
```

## Public Test Nodes

```go
grpcAddress = "tron-grpc.publicnode.com:443"       // requires UseTLS: true
httpAddress = "https://tron-rpc.publicnode.com"
```

## Known Test Data

```go
testAddress  = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"  // Binance hot wallet (always active, has balance)
usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"  // USDT TRC20 contract
testBlockNum = uint64(79831098)                         // known block with transactions
```

## Test Naming Convention

Every test should have both protocol variants:

```go
func TestMethodName_GRPC(t *testing.T) { ... }
func TestMethodName_HTTP(t *testing.T) { ... }
```

For multi-node (round-robin) tests:

```go
func TestMethodName_MultiNode(t *testing.T) { ... }
```

## Standard Test Pattern

```go
func TestGetAccount_GRPC(t *testing.T) {
    c := newGRPCClient(t)
    defer c.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    account, err := c.GetAccount(ctx, testAddress)
    require.NoError(t, err)
    require.NotNil(t, account)
    require.Greater(t, account.Balance, int64(0))
}

func TestGetAccount_HTTP(t *testing.T) {
    c := newHTTPClient(t)
    defer c.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    account, err := c.GetAccount(ctx, testAddress)
    require.NoError(t, err)
    require.NotNil(t, account)
    require.Greater(t, account.Balance, int64(0))
}
```

## Test File Organization

| File                   | What it tests                                                                      |
| ---------------------- | ---------------------------------------------------------------------------------- |
| `account_test.go`      | GetAccount, GetAccountBalance, GetAccountResource, IsAccountActivated, ChainParams |
| `block_test.go`        | GetLastBlock, GetBlockByHeight, GetBlockByHash, GetTransactionInfoByBlockNum       |
| `contract_test.go`     | TRC20GetName, TRC20GetSymbol, TRC20GetDecimals, TRC20ContractBalance               |
| `asset_test.go`        | GetAssetIssueById, GetAssetIssueListByName                                         |
| `multinode_test.go`    | Round-robin distribution across gRPC + HTTP                                        |
| `compare_test.go`      | Cross-protocol comparison (same query, both transports, compare results)           |
| `large_blocks_test.go` | Large block range retrieval (tests MaxCallRecvMsgSize)                             |
| `common_test.go`       | Shared helpers and test constants                                                  |

Unit tests in `pkg/client/`:

| File                       | What it tests                                                                  |
| -------------------------- | ------------------------------------------------------------------------------ |
| `metrics_test.go`          | `MetricsTransport`, built-in Prometheus metrics, mock helpers                  |
| `health_test.go`           | `HealthAwareTransport` behaviour with `synctest`: tier fallback, recovery, etc.|
| `health_helpers_test.go`   | `controllableTransport` mock + `newHarness` for health tests                   |

## Running Tests

```bash
# All integration tests
go test ./tests/ -v -timeout 60s

# Specific test
go test ./tests/ -v -run TestGetAccount_GRPC -timeout 30s

# Unit tests (pkg/client) — include health-checker synctest tests
go test -race ./pkg/client/ -v

# Health tests only, stress-run for goroutine leaks
go test -race -count=10 -run TestHealthAware_CloseStopsGoroutines ./pkg/client/
```

## Unit Tests with synctest (HealthAwareTransport)

The default transport stack relies on background goroutines + timers, so it's
tested with Go 1.26's `testing/synctest` for deterministic virtual-time tests.
No network, no real time, no `time.Sleep` flakiness.

**Helpers (`pkg/client/health_helpers_test.go`):**

- `controllableTransport` — programmable Transport mock. Live calls increment
  `liveCallCount` and read `nextErr`; probes (routed via a custom
  `HealthConfig.Probe`) increment `probeCount` and read `probeErr`. The split
  keeps live and probe paths from racing the same counter.
- `newHarness(t, tiers []int, cfg HealthConfig) *testHarness` — builds a
  `HealthAwareTransport` with one node per entry in `tiers` (the int is the
  node's `Tier`), wires up a mock `MetricsCollector`, and registers a
  `t.Cleanup` to call `Close()`.

**Pattern:**

```go
func TestHealthAware_PrimaryFails_FailoverToTier1(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        h := newHarness(t, []int{0, 0, 1}, HealthConfig{
            FailureThreshold:     2,
            HealthyInterval:      time.Hour, // suppress probes for clarity
            UnhealthyInterval:    time.Hour,
            InactiveTierInterval: time.Hour,
        })
        h.nodes[0].setNextErr(grpcErr(codes.Unavailable))
        h.nodes[1].setNextErr(grpcErr(codes.Unavailable))

        for i := 0; i < 4; i++ {
            _, _ = h.transport.GetAccount(context.Background(), &core.Account{})
        }
        require.False(t, h.nodeHealthy(0))
        require.False(t, h.nodeHealthy(1))
        require.Equal(t, int64(1), h.activeTier())

        h.nodes[2].setNextErr(nil)
        _, err := h.transport.GetAccount(context.Background(), &core.Account{})
        require.NoError(t, err)
        require.Equal(t, int64(1), h.nodes[2].liveCallCount.Load())
    })
}
```

**Key idioms:**

- After advancing virtual time with `time.Sleep(...)`, call `synctest.Wait()`
  to let all goroutines reach a durably-blocked state before assertions.
- Set unrelated intervals to `time.Hour` so the test focuses on one timing
  axis (probe vs live, primary vs fallback).
- Use `grpcErr(codes.Unavailable)` from `health_test.go` for network-level
  errors; use `&HTTPStatusError{Code: 503}` (wrapped in `&TransportError{...}`)
  for HTTP failures.
- `synctest.Test` requires Go 1.26+ (the project uses 1.26.1).

## Writing a New Test

1. Decide which file it belongs to based on the operation domain
2. Create both `_GRPC` and `_HTTP` variants
3. Use `require` (not `assert`) for critical checks — fail fast on errors
4. Always set a context timeout (10 seconds is the convention)
5. Always `defer c.Close()` after creating the client
6. Use known test data (`testAddress`, `usdtContract`, `testBlockNum`) for reproducible results
7. For TRC20 tests, use the USDT contract which is always available on mainnet
