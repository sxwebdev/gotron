# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`github.com/sxwebdev/gotron` — a Go SDK for the Tron blockchain. It talks to Tron nodes over
gRPC **or** HTTP REST, load-balances across multiple nodes, and exposes an ergonomic string/decimal
API for accounts, transfers, TRC20 tokens, resource delegation, Stake 2.0, and SR voting.

Go 1.26. Depends on the `decred` crypto stack (not `btcsuite`), `shopspring/decimal` for money math,
and Tron's protobuf definitions vendored under `schema/pb/`.

## Commands

```bash
go build ./...                 # build everything (CI gate)
go vet ./...                   # vet (CI gate)
make fmt                       # go fix ./... then gofumpt -l -w .  (formatter is gofumpt, not gofmt)
make lint                      # golangci-lint run

# Unit tests — deterministic, no network. Always run with -race.
go test -race ./pkg/... -v
go test -race ./pkg/client/ -v          # includes the synctest health-checker tests

# Integration tests — hit LIVE public Tron nodes (tron-grpc.publicnode.com,
# tron-rpc.publicnode.com). Non-deterministic; require network. NOT run in CI.
make test                               # == go test ./tests/... -v
go test ./tests/ -v -run TestGetAccount_GRPC -timeout 30s   # a single integration test

# Local private network — write paths and gRPC/HTTP parity of every Transport method. The node is
# defined in tests/localnet (config + docker compose). Local tests skip unless GOTRON_LOCAL_NODE=1.
make localnet-up     # start a single-witness java-tron, wait for blocks
make test-local      # GOTRON_LOCAL_NODE=1 go test -race ./tests/ -run Local -v
make localnet-down   # remove it (the chain starts from genesis next time)

# Regenerate protobuf / ABI (rarely needed; require buf + abigen installed via `make install-dev-tools`)
make genproto
make genabi
```

**CI** (`.github/workflows/ci.yml`) runs build, vet, and `go test -race` with coverage **only on the
deterministic packages** — it excludes `pkg/client` and `tests/` (live-node, non-deterministic) and
`schema/pb` (generated). Total coverage must stay **≥ 95%** or the job fails. When you add code to a
covered package, add the unit tests that keep it above the gate.

## Architecture

Layered, top to bottom. Understanding the transport layer is the key to being productive here.

1. **`tron.go` (package `gotron`)** — thin top-level wrapper. `Tron` embeds `*client.Client`; `New`
   just constructs a client. Also re-exports constants (`Energy`, `Bandwidth`, `Mainnet`, …). Most
   real code lives one layer down.
2. **`pkg/client/` (package `client`)** — the actual API surface. Methods here are **ergonomic**:
   they take string addresses and `decimal.Decimal` amounts, validate, convert to protobuf, call the
   transport, and unwrap the result. One file per domain (`account.go`, `transfer.go`, `trc20.go`,
   `resources.go`, `staking.go`, `witness.go`, `block.go`, `transactions.go`, …).
3. **`Transport` interface (`pkg/client/transport.go`)** — every low-level RPC. Operates on **raw
   protobuf** (`[]byte` addresses, `*core.*`/`*api.*` messages). Five implementations wrap each other:
   - `transport_grpc.go` — calls the generated `api.WalletClient`.
   - `transport_http.go` — POSTs to `/wallet/<method>`. Tron's HTTP JSON is **non-standard** and does
     not round-trip through `protojson`; see the HTTP gotchas below.
   - `transport_roundrobin.go` — plain round-robin over N transports, no health logic. Legacy path,
     used only when `Health.Disabled = true`.
   - `health.go` (`HealthAwareTransport`) — **the default**. Groups nodes by `NodeConfig.Tier`, routes
     to the lowest-numbered tier with a healthy node, runs one background probe goroutine per node.
   - `transport_metrics.go` — optional outermost wrapper that times every call for `MetricsCollector`.
4. **Utility packages** — `pkg/address/` (BIP39/BIP44 mnemonic + key derivation, decred-based),
   `pkg/tronutils/` (base58check address encode/decode, hex, keccak), `pkg/crypto/`, `pkg/units/`
   (TRX/SUN/Energy/Bandwidth conversion helpers using `decimal`).
5. **`schema/pb/`** — generated protobuf. `core/` = protocol types, `api/` = wallet RPC. Never edit by
   hand; regenerate via `make genproto`.

### Client wiring (`pkg/client/client.go`)

`New(cfg)` builds the transport stack: `HealthAwareTransport` by default (or `RoundRobinTransport`
when `cfg.Health.Disabled`), then wraps it in `MetricsTransport` **iff** `cfg.Metrics != nil`. There
is **no automatic retry** — a failed live call returns its error to the caller; failures only count
toward a node's health threshold so the *next* request avoids it. When every node of every tier is
unhealthy, calls return `client.ErrNoHealthyNodes` (check with `errors.Is`).

### Adding a new RPC method — touches SEVEN files

Because the `Transport` interface is implemented by five types plus the client wrapper, a new method
must be added consistently to all of them or the build breaks. Order:

1. `transport.go` — add the signature under the right section comment.
2. `transport_grpc.go` — delegate to `t.walletClient.<Method>`.
3. `transport_http.go` — implement against the REST endpoint (`fetchTron` for reads, `doTxRequest`
   for transaction-creating endpoints).
4. `transport_roundrobin.go` — `return t.next().<Method>(...)`.
5. `health.go` — `next()` → call → `recordOutcome(n, err)`.
6. `transport_metrics.go` — time it via the `after(...)` helper.
7. `pkg/client/<domain>.go` — the ergonomic `Client` method (string args → proto → transport).

`docs/transport.md` has the full checklist with code templates (including the tests a new method
needs). Consult it before adding transport methods.

## Conventions and gotchas

- **Amounts are typed; never pass a bare number.** Tron has two unrelated amount scales and mixing
  them was the #1 source of bugs, so the unit lives in the type (`pkg/units`), not in a doc comment.
  - `SUN` — every TRX-denominated value: transfers, balances, stake, delegation, fee limits,
    rewards, estimate fees. 1 TRX = 1,000,000 SUN, which is exactly what the protocol uses, so
    nothing is rounded between the caller and the wire.
  - `TokenAmount` — every TRC20 amount, always in the token's own minimal units. Its scale comes
    from the contract's `decimals`, so it is deliberately **not** interchangeable with `SUN`:
    `1_000_000` is one USDT but a millionth of a TRX.
  - Convert only at the edges: `FromTRX` / `SUN.TRX()` and `FromTokenDecimal` / `FromTokenUnits` /
    `TokenAmount.Decimal(decimals)`. Those constructors are the single place that rejects
    unrepresentable values, so per-method overflow guards are neither needed nor wanted.
  - Resource units (energy, bandwidth) and percentages (brokerage, consume-user-resource) stay
    plain — they are not money. Vote counts are TRON POWER, also not money.
- **Addresses.** The `Client` layer uses base58check strings (`T...`); the transport layer uses raw
  `[]byte`. Convert with `tronutils.DecodeCheck` / `tronutils.EncodeCheck`.
- **HTTP transport is the tricky one.** java-tron's REST JSON is not protojson: bytes are hex (which
  protojson silently mis-reads as base64), maps are `[{"key","value"}]` arrays, `Any` is
  `{"type_url","value"}`, int64 must not pass through float64. Every read goes through **`fetchTron`**
  (`decodeTronJSON` in `tronjson.go`, driven by the message's protobuf descriptor) and is requested
  **without `visible`** — never hand a body to protojson directly. Transaction-creating endpoints
  return the tx at the top level and report validation failures as **HTTP 200 with an `Error`
  field**; they go through **`doTxRequest`**, which rebuilds the tx from `raw_data_hex` and turns
  `Error` into a `ContractValidateError`. `/wallet/getReward` and `/wallet/getBrokerage` are the only
  camelCase endpoints (405 in lowercase). Details and the accepted gRPC/HTTP differences are in the
  transport guide.
- **Typed errors.** Return/wrap the sentinels in `errors.go` (`ErrInvalidAddress`, `ErrInvalidAmount`,
  `ErrAccountNotFound`, `ErrNoHealthyNodes`, …) with `fmt.Errorf("%w: …", ErrX, …)` so callers can
  `errors.Is`.

## Testing conventions

- **Two layers.** Integration tests in `tests/` (`package tests`) hit live nodes and must have both a
  `_GRPC` and `_HTTP` variant (round-robin cases: `_MultiNode`). `tests/local_*_test.go` run against a
  local private network and hold the parity test (gRPC vs HTTP, `proto.Equal`) of every `Transport`
  method — a new method gets one. Deterministic transport/health logic
  is unit-tested in `pkg/client/` (`package client`) using Go 1.26 `testing/synctest` for
  virtual-time tests with no network or sleeps. Shared constants/helpers live in `tests/common_test.go`
  (`newGRPCClient`/`newHTTPClient`/`newMultiNodeClient`, known mainnet addresses like `testAddress`,
  `usdtContract`, `stakedAddress`).
- **Doc-consistency test.** `docs_test.go` (`TestDocsReferenceRealAPI`) parses `README.md`, `doc.go`,
  `docs/*.md` and every file of the skill, and fails if they reference a `pkg/address` or
  `pkg/tronutils` symbol that isn't in the compile-checked allow-list. **If you rename or document an
  API symbol, update `docs_test.go` and the docs together**, or this test breaks.
- **House style.** Tests are independent and cover every real external state — do not shape cases to
  match `if`-branches in the code. When asked for tests, adversarially hunt for real bugs and prove
  each with a failing test, rather than chasing coverage. There is a `go-test` skill and a
  `go-code-review` skill in this environment that encode this stance.

## Documentation

Two audiences, two places — keep them apart:

- **`docs/` — for working on gotron itself.** `architecture.md` (layers, client wiring, error
  internals, design rules), `transport.md` (the `Transport` implementations, HTTP decoding, accepted
  gRPC/HTTP differences, add-a-method checklist, endpoint map), `testing.md` (test layout, synctest,
  the local private network). Read the relevant one before non-trivial work in that area — they are
  more detailed than this file. Internal knowledge (how something works inside, why it was built
  that way, bugs it guards against) goes here.
- **`skills/gotron/` — for people integrating the library.** It is published (`skills repo add
  sxwebdev/gotron`), so it describes how to *use* gotron: `SKILL.md` (connect, the three rules,
  recipes, errors, transport choice, pitfalls) and `references/` (`api-surface.md` — the full public
  API; `constants.md` — errors, enums, metric names). No internal file names, unexported symbols or
  implementation history there. When a public method or its behaviour changes, update
  `api-surface.md` (and `SKILL.md` if a recipe uses it).
