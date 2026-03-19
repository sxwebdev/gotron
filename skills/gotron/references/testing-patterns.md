# Testing Patterns

## Test Location and Package

Tests live in `tests/` as a separate package (`package tests`), not alongside the source code. This keeps integration tests isolated from the library code.

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

## Running Tests

```bash
# All integration tests
go test ./tests/ -v -timeout 60s

# Specific test
go test ./tests/ -v -run TestGetAccount_GRPC -timeout 30s

# Unit tests (pkg/client)
go test ./pkg/client/ -v
```

## Writing a New Test

1. Decide which file it belongs to based on the operation domain
2. Create both `_GRPC` and `_HTTP` variants
3. Use `require` (not `assert`) for critical checks — fail fast on errors
4. Always set a context timeout (10 seconds is the convention)
5. Always `defer c.Close()` after creating the client
6. Use known test data (`testAddress`, `usdtContract`, `testBlockNum`) for reproducible results
7. For TRC20 tests, use the USDT contract which is always available on mainnet
