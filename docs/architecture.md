# Architecture

How gotron is put together, for people working on the library itself. To learn how to _use_
it, read the [README](../README.md) or the [agent skill](../skills/gotron/SKILL.md).

## Table of contents

- [Layers](#layers)
- [Client wiring](#client-wiring)
- [Error handling internals](#error-handling-internals)
- [Design rules](#design-rules)
- [Where to go next](#where-to-go-next)

## Layers

Top to bottom:

1. **`tron.go` (package `gotron`)** — thin wrapper. `Tron` embeds `*client.Client`; `New` just
   constructs a client. It also re-exports constants (`Energy`, `Bandwidth`, `Mainnet`, …).
2. **`pkg/client/`** — the API surface. Methods are ergonomic: they take base58 address strings
   and typed amounts (`SUN`, `TokenAmount`), validate them, convert to protobuf, call the
   transport and unwrap the answer. One file per domain (`account.go`, `transfer.go`, `trc20.go`,
   `resources.go`, `staking.go`, `witness.go`, `block.go`, `transactions.go`, `contract.go`,
   `estimate_*.go`, `account_permissions.go`, …).
3. **`Transport` interface (`pkg/client/transport.go`)** — every low-level RPC, on raw protobuf
   (`[]byte` addresses, `*core.*` / `*api.*` messages). Five implementations wrap each other; see
   [transport.md](transport.md).
4. **Utility packages** — `pkg/address` (BIP39/BIP44 derivation, decred-based), `pkg/tronutils`
   (base58check, hex, keccak), `pkg/crypto`, `pkg/units` (`SUN`, `TokenAmount`, resource units),
   `pkg/client/abi` (ABI encoding, `LoadContractABI`).
5. **`schema/pb/`** — generated protobuf: `core/` protocol types, `api/` wallet RPC. Never edited by
   hand; regenerate with `make genproto`.

## Client wiring

`client.New(cfg)` (`pkg/client/client.go`) builds the transport stack:

Default (health checking enabled):

```
Client.transport
  -> MetricsTransport            (only when cfg.Metrics != nil)
    -> HealthAwareTransport      (tier-based fallback + per-node health)
      -> GRPCTransport | HTTPTransport   (one per NodeConfig)
```

Legacy (`cfg.Health.Disabled = true`):

```
Client.transport
  -> MetricsTransport            (optional)
    -> RoundRobinTransport       (atomic counter, no health)
      -> GRPCTransport | HTTPTransport
```

There is **no automatic retry**. A failed call returns its error to the caller; a network-level
failure only counts toward the node's health threshold, so the _next_ request avoids that node.
When every node of every tier is unhealthy, calls return `ErrNoHealthyNodes`.

`HealthAwareTransport`, `RoundRobinTransport` and `MetricsTransport` internals — selection rule,
probe cadence, error classification, shutdown — are in [transport.md](transport.md).

## Error handling internals

- Sentinel errors live in `pkg/client/errors.go`. Wrap them with `fmt.Errorf("%w: …", ErrX, …)` so
  callers can `errors.Is`; never return a bare string for a condition a caller may want to match.
- `TransportError{Host, Protocol, Method, Err}` wraps every RPC failure. gRPC errors get it from
  `transportErrorInterceptor` (`transport_grpc.go`), HTTP ones from `HTTPTransport.wrapErr`.
- `HTTPStatusError` is a non-2xx HTTP answer, wrapped inside a `TransportError`.
- `ContractValidateError` is a node refusing to _build_ a transaction. Both transports produce it
  (gRPC through `Result.code` via `checkTransaction`, HTTP through the `Error` field in
  `parseTxResponse`) and it unwraps to `ErrInvalidTransaction`.
- `BroadcastError` is a rejected broadcast, built by `Client.BroadcastTransaction` from
  `Return.code` / `Return.message`.
- `ErrNodeRefusedRequest` is an HTTP read the node refused with `{"Error": …}` (see `apiError`).
- Health classification (`health_classify.go`, `isNetworkError`): gRPC `Unavailable` /
  `DeadlineExceeded` / `Aborted` / `ResourceExhausted` / `Internal` / `Unknown`, HTTP 5xx / 408 /
  429, `context.DeadlineExceeded`, `net.Error` timeouts, `io.EOF`, `net.ErrClosed` count toward
  the failure threshold. Logical errors — `ContractValidateError`, `ErrNodeRefusedRequest`, other
  4xx — never do, and `TestValidateErrorsDoNotMarkANodeUnhealthy` pins that.

## Design rules

- **Amounts are typed, never bare numbers.** Every TRX-denominated value is a `SUN`, every TRC20
  amount a `TokenAmount` in the token's own minimal units — the two scales are unrelated and mixing
  them was the most common bug. Conversion happens only at the edges (`FromTRX`, `SUN.TRX()`,
  `FromTokenDecimal`, `FromTokenUnits`, `TokenAmount.Decimal`), which are the single place that
  rejects unrepresentable values, so per-method overflow guards are neither needed nor wanted.
  Resource units (energy, bandwidth), percentages and vote counts (TRON POWER) stay plain — they are
  not money.
- **Addresses are base58 strings at the `Client` boundary and `[]byte` below it.** Convert with
  `tronutils.DecodeCheck` / `tronutils.EncodeCheck`.
- **Configuration is a struct** (`Config`, `NodeConfig`, `HealthConfig`), not functional options.
- **Write methods never broadcast.** They return an unsigned `*api.TransactionExtention`; signing and
  broadcasting are separate calls the caller makes. Anything that edits `raw_data` afterwards (fee
  limit, permission id) must refresh the txid with `UpdateHash` before signing.
- **All transports stay in sync.** A `Transport` method exists in all six transport files
  (interface, gRPC, HTTP, round-robin, health-aware, metrics). The compiler enforces the interface,
  but a missing wrapper only surfaces at runtime. See the checklist in [transport.md](transport.md).
- **gRPC and HTTP must answer the same.** gRPC is the reference — its wire format is protobuf. Every
  `Transport` method has a parity test against a local node (see [testing.md](testing.md)); a
  remaining difference is a bug unless it is listed as accepted in [transport.md](transport.md).
- **Documented APIs are compile-checked.** `docs_test.go` fails when `README.md`, `doc.go`, the
  skill or these docs name a `pkg/address` / `pkg/tronutils` symbol that is not in its allow-list.
  Rename a symbol and its docs together.

## Where to go next

- [transport.md](transport.md) — the `Transport` interface, each implementation, how the HTTP
  transport decodes java-tron's JSON, the accepted gRPC/HTTP differences, and the checklist for
  adding an RPC method.
- [testing.md](testing.md) — test layout, public-node integration tests, `synctest` unit tests, and
  the local private network that runs the write paths and the parity tests.
