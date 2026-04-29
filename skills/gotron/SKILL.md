---
name: gotron
description: >
  Gotron is a Go SDK for the Tron blockchain (github.com/sxwebdev/gotron). Use this skill whenever
  working in the gotron codebase — editing pkg/client, pkg/address, pkg/tronutils, or tests/,
  adding new RPC methods, writing integration tests, working with the Transport interface (gRPC,
  HTTP, round-robin), TRC20 token operations, address generation (BIP39/BIP44), resource delegation
  (bandwidth/energy), Prometheus metrics (MetricsCollector), or any file importing gotron packages.
  Also triggers when the user mentions Tron blockchain, TRX, SUN, TRC20, USDT on Tron, or TronGrid.
user-invocable: true
---

# Gotron SDK

## Overview

Gotron is a comprehensive Go client for the Tron blockchain. It supports dual transport (gRPC and HTTP REST), round-robin load balancing across multiple nodes, optional Prometheus metrics, TRC20 token operations, address generation via BIP39/BIP44, and resource delegation (bandwidth/energy).

The SDK follows a layered architecture:

1. **Top layer** — `tron.go` (package `gotron`): thin wrapper exposing `Tron` struct that embeds `*client.Client`
2. **Client layer** — `pkg/client/`: main API surface with all blockchain operations
3. **Transport layer** — pluggable `Transport` interface with gRPC, HTTP, RoundRobin, and Metrics implementations
4. **Utility layers** — `pkg/address/`, `pkg/tronutils/`, `pkg/units/`
5. **Schema layer** — `schema/pb/` generated protobuf types

## Architecture

### Transport chain

```
Client.transport
  -> MetricsTransport (optional, wraps next)
    -> RoundRobinTransport (distributes via atomic counter)
      -> GRPCTransport | HTTPTransport (per node)
```

`client.New(cfg)` builds this chain automatically:

- Creates one GRPCTransport or HTTPTransport per `NodeConfig`
- Wraps all in `RoundRobinTransport`
- If `cfg.Metrics != nil`, wraps in `MetricsTransport`

### Transport interface

Defined in `pkg/client/transport.go`. Every new RPC method must be added to:

1. `Transport` interface
2. `GRPCTransport` (`transport_grpc.go`)
3. `HTTPTransport` (`transport_http.go`)
4. `RoundRobinTransport` (`transport_roundrobin.go`)
5. `MetricsTransport` (`transport_metrics.go`)

See `references/transport-guide.md` for the full interface and implementation patterns.

### Error handling

- Sentinel errors in `pkg/client/errors.go`: `ErrInvalidConfig`, `ErrNotConnected`, `ErrInvalidAddress`, `ErrTransactionNotFound`, etc.
- `TransportError` struct wraps RPC errors with `Host`, `Protocol`, `Method` fields. Extract with `errors.As(err, &transportErr)`.
- gRPC errors are wrapped via `transportErrorInterceptor`; HTTP errors via `HTTPTransport.wrapErr`.

### Key conventions

- All amounts use `decimal.Decimal` (shopspring) to avoid floating-point precision errors
- TRX to SUN: multiply by 1,000,000 (TrxDecimals = 6)
- All operations are stateless and context-driven
- Addresses are base58-encoded strings at the Client API level, decoded to `[]byte` before passing to Transport
- Config struct (not functional options) for client construction
- No built-in retry logic — fails fast, errors propagated as-is

## Instructions

### Adding a new client method

1. Add the low-level RPC to the `Transport` interface in `pkg/client/transport.go`
2. Implement in `GRPCTransport` — call the appropriate `walletClient` method
3. Implement in `HTTPTransport` — use `doRequest`, `doRequestTransformed`, or `doRequestRaw` depending on response format:
   - `doRequest` — standard protojson-compatible responses
   - `doRequestTransformed` — responses with Tron's non-standard Any types (needs hex->base64, field normalization)
   - `doBlockRequest` — block responses where transactions need wrapping into `TransactionExtention`
   - `doRequestRaw` — when you need custom parsing (e.g., `GetAccount`, `TriggerConstantContract`)
4. Implement in `RoundRobinTransport` — delegate to `t.next().MethodName(ctx, ...)`
5. Implement in `MetricsTransport` — wrap with timing: `start := time.Now()` ... `t.after("MethodName", start, err)`
6. Add the high-level client method in the appropriate file under `pkg/client/` (e.g., `account.go`, `block.go`, `trc20.go`)
7. Write integration tests in `tests/` with both `_GRPC` and `_HTTP` suffixed test functions

### Writing tests

- Tests live in `tests/` package (separate from `pkg/client`)
- Use helpers from `tests/common_test.go`: `newGRPCClient(t)`, `newHTTPClient(t)`, `newMultiNodeClient(t)`
- Use `testify/require` for assertions
- Always test both protocols: `TestMethodName_GRPC` and `TestMethodName_HTTP`
- Use 10-second context timeout: `ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)`
- Known test data: address `TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g` (Binance), USDT `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`, block `79831098`
- Public nodes: gRPC `tron-grpc.publicnode.com:443` (TLS), HTTP `https://tron-rpc.publicnode.com`

See `references/testing-patterns.md` for complete test patterns.

### Working with addresses

Use `pkg/address` for generation/validation:

- `address.Generate()` — random address
- `address.FromMnemonic(mnemonic, passphrase, index)` — BIP44 derivation (path: `m/44'/195'/0'/0/index`)
- `address.FromPrivateKey(hex)` — import existing key
- `address.Validate(addr)` — validate base58 format

### Working with TRC20 tokens

All TRC20 methods are in `pkg/client/trc20.go`. The low-level `TRC20Call` method handles ABI encoding. Higher-level methods:

- `TRC20GetName`, `TRC20GetSymbol`, `TRC20GetDecimals` — read-only (constant=true)
- `TRC20ContractBalance` — balance query
- `TRC20Send` — transfer tokens (requires signing + broadcast)
- `TRC20Approve`, `TRC20TransferFrom` — approval flow

### Estimating fees

Cost estimators all return `*EstimateResult { Energy, Bandwidth, Trx }` (TRX in actual TRX, not SUN).

- **Per transaction:** `EstimateBandwidth(tx)` for bandwidth points, `EstimateEnergy(...)` for contract energy.
  Both live in `pkg/client/estimate_resources.go`.
- **Activation only:** `EstimateActivationFee(ctx, from, to)` (local fake tx, fast) or `EstimateSystemContractActivation(ctx, caller, receiver)` (real CreateAccount RPC, more accurate). Both return zeros for already-activated receivers and are in `pkg/client/activate.go`.
- **Full transfer (TRX or TRC20):** `EstimateTransferResources(ctx, from, to, contract, amount, decimals)` returns `EstimateTransferResourcesResult` with `Total / Transfer / Activation` breakdown. `Activation` is zero for activated recipients; `Total = Transfer + Activation` (conservative upper bound — Tron consumes the activation fee inside the transfer tx itself, so real cost may be slightly lower).
- **Unactivated recipients are valid.** Sending TRX or TRC20 to an unactivated address activates it; do not gate transfers on `IsAccountActivated`. The sentinel `ErrAccountNotActivated` exists for callers that explicitly require an activated address.

### Prometheus metrics

Enable by passing `cfg.Metrics = client.NewMetrics(prometheus.DefaultRegisterer)`. Recorded metrics:

- `gotron_rpc_requests_total` (counter: blockchain, method, status)
- `gotron_rpc_duration_seconds` (histogram: blockchain, method)
- `gotron_rpc_in_flight` (gauge)
- `gotron_rpc_retries_total`, `gotron_rpc_pool_total`, `gotron_rpc_pool_healthy`, `gotron_rpc_pool_disabled`

Implement custom collectors via the `MetricsCollector` interface (3 methods: `RecordRequest`, `RecordRetry`, `SetPoolHealth`).

## Example: Adding a new RPC method

**Input:** "Add support for `GetNodeInfo` RPC"

**Steps Claude follows:**

1. Add to `Transport` interface in `pkg/client/transport.go`:

```go
GetNodeInfo(ctx context.Context) (*core.NodeInfo, error)
```

2. Implement in `transport_grpc.go`:

```go
func (t *GRPCTransport) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
    return t.walletClient.GetNodeInfo(ctx, new(api.EmptyMessage))
}
```

3. Implement in `transport_http.go`:

```go
func (t *HTTPTransport) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
    result := &core.NodeInfo{}
    if err := t.doRequest(ctx, "/wallet/getnodeinfo", nil, result); err != nil {
        return nil, err
    }
    return result, nil
}
```

4. Implement in `transport_roundrobin.go`:

```go
func (t *RoundRobinTransport) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
    return t.next().GetNodeInfo(ctx)
}
```

5. Implement in `transport_metrics.go`:

```go
func (t *MetricsTransport) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
    start := time.Now()
    result, err := t.transport.GetNodeInfo(ctx)
    t.after("GetNodeInfo", start, err)
    return result, err
}
```

6. Add client method in `pkg/client/network.go`:

```go
func (c *Client) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
    return c.transport.GetNodeInfo(ctx)
}
```

7. Add tests in `tests/network_test.go`:

```go
func TestGetNodeInfo_GRPC(t *testing.T) {
    c := newGRPCClient(t)
    defer c.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    info, err := c.GetNodeInfo(ctx)
    require.NoError(t, err)
    require.NotNil(t, info)
}
```

## Domain references

- Full API surface: see `references/api-surface.md`
- Transport layer deep dive: see `references/transport-guide.md`
- Constants, errors, enums: see `references/constants.md`
- Testing conventions: see `references/testing-patterns.md`

## Key principles

- **Both transports must stay in sync.** Every Transport interface method must have implementations in all 4 transport files. The compiler enforces the interface, but forgetting RoundRobin or Metrics will cause runtime issues.
- **HTTP transport needs JSON transformation.** Tron's HTTP API returns non-standard JSON (hex strings instead of base64, `type_url`/`value` instead of `@type`). Use the appropriate `doRequest*` variant.
- **Addresses are strings at the Client boundary.** Convert to `[]byte` with `tronutils.DecodeCheck(addr)` before passing to transport. This keeps the public API ergonomic while the transport layer works with raw bytes.
- **Decimal precision matters.** Never use `float64` for token amounts. Use `decimal.Decimal` for TRX and `*big.Int` for TRC20 token amounts.
- **Tests hit real public nodes.** No mocks — integration tests use `tron-grpc.publicnode.com:443` and `https://tron-rpc.publicnode.com`. Always add both `_GRPC` and `_HTTP` variants.
