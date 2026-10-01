# Transport layer

How gotron talks to Tron nodes: the `Transport` interface, its five implementations, how the HTTP
transport decodes java-tron's JSON, the differences from gRPC that remain, and the checklist for
adding an RPC method. For the big picture see [architecture.md](architecture.md); for the tests
that hold the transports to each other see [testing.md](testing.md).

## Table of Contents

- [Transport interface](#transport-interface)
- [GRPCTransport](#grpctransport)
- [HTTPTransport](#httptransport)
- [RoundRobinTransport](#roundrobintransport)
- [HealthAwareTransport](#healthawaretransport)
- [MetricsTransport](#metricstransport)
- [Adding a new transport method](#adding-a-new-transport-method)
- [Endpoints and internal constants](#endpoints-and-internal-constants)

---

## Transport interface

**File:** `pkg/client/transport.go`

The `Transport` interface defines all low-level RPC operations. It operates on protobuf types (raw `[]byte` addresses, proto messages), while the `Client` layer provides the ergonomic string-based API.

```go
type Transport interface {
    // Account
    GetAccount(ctx context.Context, account *core.Account) (*core.Account, error)
    GetAccountResource(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
    CreateAccount(ctx context.Context, contract *core.AccountCreateContract) (*api.TransactionExtention, error)
    AccountPermissionUpdate(ctx context.Context, contract *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error)

    // Block
    GetNowBlock(ctx context.Context) (*api.BlockExtention, error)
    GetBlockByNum(ctx context.Context, num int64) (*api.BlockExtention, error)
    GetBlockById(ctx context.Context, id []byte) (*core.Block, error)
    GetBlockByLimitNext(ctx context.Context, start, end int64) (*api.BlockListExtention, error)
    GetBlockByLatestNum(ctx context.Context, num int64) (*api.BlockListExtention, error)
    GetTransactionInfoByBlockNum(ctx context.Context, num int64) (*api.TransactionInfoList, error)

    // Transaction
    GetTransactionById(ctx context.Context, id []byte) (*core.Transaction, error)
    GetTransactionInfoById(ctx context.Context, id []byte) (*core.TransactionInfo, error)
    BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error)
    CreateTransaction(ctx context.Context, contract *core.TransferContract) (*api.TransactionExtention, error)

    // Contract
    TriggerContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error)
    TriggerConstantContract(ctx context.Context, contract *core.TriggerSmartContract) (*api.TransactionExtention, error)
    EstimateEnergy(ctx context.Context, contract *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
    DeployContract(ctx context.Context, contract *core.CreateSmartContract) (*api.TransactionExtention, error)
    GetContract(ctx context.Context, address []byte) (*core.SmartContract, error)
    UpdateSetting(ctx context.Context, contract *core.UpdateSettingContract) (*api.TransactionExtention, error)
    UpdateEnergyLimit(ctx context.Context, contract *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error)

    // Resource
    GetAccountResourceMessage(ctx context.Context, account *core.Account) (*api.AccountResourceMessage, error)
    GetDelegatedResource(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
    GetDelegatedResourceV2(ctx context.Context, msg *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
    GetDelegatedResourceAccountIndex(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error)
    GetDelegatedResourceAccountIndexV2(ctx context.Context, address []byte) (*core.DelegatedResourceAccountIndex, error)
    GetCanDelegatedMaxSize(ctx context.Context, msg *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)
    DelegateResource(ctx context.Context, contract *core.DelegateResourceContract) (*api.TransactionExtention, error)
    UnDelegateResource(ctx context.Context, contract *core.UnDelegateResourceContract) (*api.TransactionExtention, error)

    // Staking (Stake 2.0)
    FreezeBalanceV2(ctx context.Context, contract *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error)
    UnfreezeBalanceV2(ctx context.Context, contract *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error)
    WithdrawExpireUnfreeze(ctx context.Context, contract *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error)
    CancelAllUnfreezeV2(ctx context.Context, contract *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error)
    GetAvailableUnfreezeCount(ctx context.Context, msg *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error)
    GetCanWithdrawUnfreezeAmount(ctx context.Context, msg *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)

    // Witness
    VoteWitnessAccount(ctx context.Context, contract *core.VoteWitnessContract) (*api.TransactionExtention, error)
    WithdrawBalance(ctx context.Context, contract *core.WithdrawBalanceContract) (*api.TransactionExtention, error)
    ListWitnesses(ctx context.Context) (*api.WitnessList, error)
    GetRewardInfo(ctx context.Context, address []byte) (*api.NumberMessage, error)
    GetBrokerageInfo(ctx context.Context, address []byte) (*api.NumberMessage, error)

    // Asset
    GetAssetIssueById(ctx context.Context, id []byte) (*core.AssetIssueContract, error)
    GetAssetIssueListByName(ctx context.Context, name []byte) (*api.AssetIssueList, error)

    // Network
    ListNodes(ctx context.Context) (*api.NodeList, error)
    GetChainParameters(ctx context.Context) (*core.ChainParameters, error)
    GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error)
    TotalTransaction(ctx context.Context) (*api.NumberMessage, error)

    Close() error
}
```

---

## GRPCTransport

**File:** `pkg/client/transport_grpc.go`

Calls `api.WalletClient` (generated gRPC client) directly.

**Key details:**

- `MaxCallRecvMsgSize` = 100MB for large block responses
- TLS via `credentials.NewTLS` with TLS 1.3 minimum when `UseTLS: true`
- Custom headers injected via `headersInterceptor` (gRPC metadata)
- Errors wrapped via `transportErrorInterceptor` into `TransportError{Protocol: "grpc"}`
- Uses `grpc.NewClient` (not deprecated `grpc.Dial`)

**Pattern for gRPC methods:**

```go
func (t *GRPCTransport) MethodName(ctx context.Context, param *SomeProto) (*ResultProto, error) {
    return t.walletClient.MethodName(ctx, param)
}
```

For methods that return large responses, pass `defaultMaxSizeOption`:

```go
return t.walletClient.GetBlockByNum2(ctx, req, defaultMaxSizeOption)
```

For methods with no input, use `new(api.EmptyMessage)`:

```go
return t.walletClient.GetNowBlock2(ctx, new(api.EmptyMessage))
```

---

## HTTPTransport

**File:** `pkg/client/transport_http.go`

Uses HTTP POST to Tron's REST API. All requests are JSON with `Content-Type: application/json`.

**Key challenge:** java-tron renders protobuf as JSON with its own printer (`JsonFormat`), not
protojson, and the two disagree in ways protojson cannot be configured around:

- bytes are **hex**, not base64. An even-length hex string is valid base64, so protojson decodes it
  _without an error_ into different bytes — a 21-byte address became 31 bytes of noise
- a protobuf **map** is an array of `{"key": …, "value": …}` objects, which protojson refuses
- an `Any` is `{"type_url": …, "value": {…}}`, not `{"@type": …}`
- `int64` is a JSON number — exact only if it never passes through `float64` (balances exceed 2^53)
- block endpoints answer with `core.Block`'s shape plus `blockID`/`txID`, not `BlockExtention`
- with `"visible": true` addresses become base58 and some names UTF-8 text, indistinguishable from hex

**Reads: `fetchTron` + `decodeTronJSON` (`pkg/client/tronjson.go`).** Every read endpoint is
requested **without** `visible` (addresses in the request as hex `41…`) and decoded by
`decodeTronJSON`, which walks the target message's protobuf **descriptor**: every field the schema
marks as bytes is read as hex and nothing else is, maps are read from the key/value array, an `Any`
is resolved through the registry, enums are read by name (or number), numbers stay `json.Number`.
Keys the message lacks (`raw_data_hex`, `visible`, …) are skipped and `null` is "not set".

It replaced a field-_name_ allow-list (`bytesFields` + `transformTronJSON` → protojson). A name does
not say what a field is — `extra` is bytes in one message and a string in another, `ref_block_hash`
was never listed — and that approach broke receipts (`GetTransactionInfoById` never matched the id,
so `GetTransactionInfoByHash` always returned `ErrTransactionInfoNotFound`), transactions (empty
contract parameter, garbage signatures), every map-carrying receipt, and silently dropped fields of
the hand-written structs (`GetAccount` lost `votes`, `account_name`, `net_usage`, `asset_issued_*`,
the bandwidth delegation totals …).

Rules the decoder encodes — do not "simplify" them away:

| Rule                                                                       | Why                                                                                                      |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| A `core.Transaction` takes `raw_data` from **`raw_data_hex`** when present | those are the bytes that were hashed and signed; re-encoding the JSON would have to reproduce every byte |
| `TransactionExtention.txid` falls back to `transaction.txID`               | the constant-call servlets drop the top-level `txid`                                                     |
| `Account.asset_issued_ID` is read as **text**                              | `/wallet/getaccount` re-prints it as UTF-8 after the hex pass (`Util.convertOutput`)                     |
| `Return.message` is hex, falling back to text                              | some error paths print it with `toStringUtf8`                                                            |
| An unknown enum _name_ is an error                                         | falling back to 0 makes an unknown permission type `Owner`                                               |

Block shapes are rebuilt by `decodeTronBlockExtention`: `core.Block` decoded generically, `blockID`
→ `Blockid`, and each transaction wrapped as `TransactionExtention{Transaction, Txid: txID,
Result: {result: true}}` — exactly what java-tron's gRPC `getBlockExtention` returns.
`/wallet/gettransactioninfobyblocknum` answers with a bare array (`{}` for an empty block) and is
unwrapped into `TransactionInfoList`.

**Request helpers:**

| Method                          | When to use                                                                        |
| ------------------------------- | ---------------------------------------------------------------------------------- |
| `fetchTron`                     | **Every read** — decodes into the proto message with `decodeTronJSON`              |
| `fetchBlock` / `fetchBlockList` | Block endpoints → `BlockExtention` / `BlockListExtention`                          |
| `fetchJSON`                     | Answers that are not a proto message (`getReward`, `getBrokerage`, `broadcasthex`) |
| `fetch`                         | Custom parsing (returns raw `[]byte`, `Error` already checked)                     |
| `doTxRequest`                   | **Every transaction-creating endpoint** — see below                                |
| `doTxRequestWrapped`            | `/wallet/triggersmartcontract`                                                     |
| `doRequestRaw`                  | **Only** the two tx helpers and `fetch`                                            |

**Every decoding path goes through `fetch`, which is `doRequestRaw` + `apiError`.**
A `/wallet` endpoint answers a request it will not process at all — a malformed address, an
unparseable integer — with **HTTP 200 and `{"Error":"class org.tron…"}`**. No decoder has a field
for it, so without the check "your request was wrong" reaches the caller as "there is nothing there".
`fetch` checks once, on behalf of every decoding path, and returns **`ErrNodeRefusedRequest`**. It
is a statement about the request, not the node: health stays untouched.

`doRequestRaw` is called by exactly three things: `fetch`, `doTxRequest` and `doTxRequestWrapped`.
The two tx helpers read `"Error"` themselves to give it a `ContractValidateError`. **Anything else
that calls `doRequestRaw` directly reintroduces the silently-empty-message bug.** `apiError` ignores
a body that is not a JSON object and tests for the literal `"Error"` bytes before parsing.

**`doTxRequest` — transaction-creating endpoints.**
`/wallet/createtransaction`, `/wallet/freezebalancev2` and friends return the transaction at the
**top level** (`{"raw_data":…,"raw_data_hex":…,"txID":…}`) and report validation failures as **HTTP
200 with an `Error` field**. `doTxRequest` rebuilds the transaction from `raw_data_hex` (so the
request may keep `visible: true`) and turns `Error` into a **`ContractValidateError`** — the type
the client also builds from gRPC's `Result.code`. `txID == sha256(raw_data_hex)`.

**`doTxRequestWrapped` — `/wallet/triggersmartcontract`** nests the transaction under
`"transaction"` and reports the outcome in `"result"` (code + hex message), so it reuses
`parseTxResponse` on the nested object.

**`DeployContract` sends `abi` as the bare array of entries.** `DeployContractServlet` wraps whatever
`"abi"` holds in `{"entrys": …}` itself; sending the whole ABI message nested it twice and the
contract was stored with **one empty entry** — deployed, uncallable by anything that reads its ABI.
Entries are rendered with protojson (`UseProtoNames`); the node's parser takes the protobuf enum
names (`"Function"`, `"Nonpayable"`). The servlet also takes `call_value`, `call_token_value` and
`token_id`, which must be sent. It **always** sets the ABI field, so a deployment without an ABI
carries an empty `abi {}` over HTTP where gRPC's has none — same meaning, but two more bytes, hence
a different txid and contract address (documented, `TestLocalParity_DeployContract`). The fee limit
is not part of the request: `Client.DeployContract` sets it afterwards and calls `UpdateHash`.

**`BroadcastTransaction` posts to `/wallet/broadcasthex`, not `/wallet/broadcasttransaction`.**
The hex endpoint takes `{"transaction": hex(proto.Marshal(tx))}`, so the signed bytes reach the node
exactly as they were signed. `/wallet/broadcasttransaction` rebuilds the transaction from the JSON
`raw_data` object and ignores `raw_data_hex`, so using it would mean re-rendering every contract in
Tron's own dialect. The response's `message` is plain text, decoded by `httpBroadcastResponse`.

**Endpoint notes:**

- `TriggerConstantContract` / `EstimateEnergy` — must **omit** `contract_address` when the proto's is
  empty (a deployment); an explicit `""` is refused. Send `call_value`, `call_token_value`, `token_id`.
  A revert answers `result.result = true` with the failure only in `result.message`, so the whole
  `Return` must survive decoding.
- `GetRewardInfo` / `GetBrokerageInfo` — camelCase paths; the fields are `reward` / `brokerage`, not
  `NumberMessage`'s `num`.
- `GetAssetIssueListByName` sends the name **hex-encoded**; `GetAssetIssueById` sends the decimal id
  as it stands.
- `ListNodes` — `{}` when the node has no peers is an empty list.

**Known, accepted differences from gRPC** (asserted by the local parity tests):

- A refused build: gRPC returns an extention with `Result.code`/`message`
  (`"Contract validate error : <reason>"`); HTTP returns a `*ContractValidateError` whose message is
  `"class org.tron.core.exception.ContractValidateException : <reason>"` and whose `Code` is unset.
- `TriggerConstantContract` / `EstimateEnergy` refusing a call before execution word the message
  `"Contract validate error : <reason>"` over gRPC and `"<reason>"` over HTTP.
- A deployment without an ABI carries `abi {}` over HTTP (see above).

**HTTP endpoints map to `/wallet/<methodname>` paths** — with two exceptions that are camelCase and
return HTTP 405 in lowercase: **`/wallet/getReward`** and **`/wallet/getBrokerage`**.

- `/wallet/getaccount`
- `/wallet/getnowblock`
- `/wallet/getblockbynum`
- `/wallet/gettransactionbyid`
- `/wallet/triggersmartcontract`
- `/wallet/triggerconstantcontract`
- etc.

**Error wrapping:**

```go
func (t *HTTPTransport) wrapErr(method string, err error) error {
    return &TransportError{Host: t.baseURL, Protocol: "http", Method: method, Err: err}
}
```

---

## RoundRobinTransport

**File:** `pkg/client/transport_roundrobin.go`

Distributes requests across multiple transports using an atomic counter.

```go
type RoundRobinTransport struct {
    transports []Transport
    counter    atomic.Uint64
}

func (t *RoundRobinTransport) next() Transport {
    idx := t.counter.Add(1) - 1
    return t.transports[idx%uint64(len(t.transports))]
}
```

**Pattern for every method:**

```go
func (t *RoundRobinTransport) MethodName(ctx context.Context, param *SomeProto) (*ResultProto, error) {
    return t.next().MethodName(ctx, param)
}
```

No retry logic, no health checking. Errors propagated as-is. Kept as a public
helper for users who explicitly opt out of health checking via
`cfg.Health.Disabled = true`.

---

## HealthAwareTransport

**File:** `pkg/client/health.go`

The default transport in the production stack. Groups nodes by `NodeConfig.Tier`
(0 = primary, 1 = fallback, 2+ = next), tracks per-node health, and runs one
background probe goroutine per node. Selection rule: pick a node from the
lowest-numbered tier that still has at least one healthy node; round-robin
within that tier; return `ErrNoHealthyNodes` when every tier is empty.

**Key types:**

```go
type HealthConfig struct {
    Disabled             bool
    FailureThreshold     int           // default 2
    SuccessThreshold     int           // default 2
    HealthyInterval      time.Duration // default 30s — active tier
    UnhealthyInterval    time.Duration // default 5s  — any unhealthy node
    InactiveTierInterval time.Duration // default 5m  — healthy fallbacks
    ProbeTimeout         time.Duration // default 5s
    Probe                func(ctx context.Context, t Transport) error // default = GetNowBlock
    ClassifyErr          func(err error) bool                         // default = isNetworkError
    Logger               Logger // interface { Infof(format string, args ...any) }; nil = no-op
}

type HealthAwareTransport struct { /* ... */ }

func NewHealthAwareTransport(
    nodes []NodeConfig,
    factory func(NodeConfig) (Transport, error),
    cfg HealthConfig,
    metrics MetricsCollector,
    blockchain string,
) (*HealthAwareTransport, error)
```

**Pattern for every method:**

```go
func (h *HealthAwareTransport) MethodName(ctx context.Context, p *In) (*Out, error) {
    n, err := h.next()
    if err != nil {
        return nil, err
    }
    res, callErr := n.transport.MethodName(ctx, p)
    h.recordOutcome(n, callErr)
    return res, callErr
}
```

`recordOutcome` runs the configured `ClassifyErr` (default `isNetworkError` —
gRPC `Unavailable`/`DeadlineExceeded`/`Aborted`/`ResourceExhausted`/`Internal`/`Unknown`,
HTTP 5xx/408/429, `context.DeadlineExceeded`, `net.Error` timeouts, `io.EOF`,
`net.ErrClosed`). Only network-level errors count toward the failure
threshold; logical errors (e.g. `InvalidArgument`, HTTP 4xx) leave node health
untouched. Successful live calls also feed the success counter, so traffic
itself contributes to recovery.

When the active tier shifts (last primary down, or first primary back up),
all per-node loops are pinged via `notifyCh` to recompute their probe
interval — fallbacks promoted to active flip from `InactiveTierInterval` to
`HealthyInterval` immediately, and back when the primary recovers.

`Close()` closes `stopCh` (under a `sync.Once`), waits for every health-loop
to exit, then closes every underlying transport.

---

## MetricsTransport

**File:** `pkg/client/transport_metrics.go`

Wraps another transport and records timing/status via `MetricsCollector`.

**Pattern for every method:**

```go
func (t *MetricsTransport) MethodName(ctx context.Context, param *SomeProto) (*ResultProto, error) {
    start := time.Now()
    result, err := t.transport.MethodName(ctx, param)
    t.after("MethodName", start, err)
    return result, err
}
```

The `after` helper records `RecordRequest(blockchain, method, "success"|"error", duration)`.

---

## Adding a new transport method

Checklist for adding `NewMethod(ctx, *InputProto) (*OutputProto, error)`:

### 1. Transport interface (`transport.go`)

Add the method signature under the appropriate section comment.

### 2. GRPCTransport (`transport_grpc.go`)

```go
func (t *GRPCTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    return t.walletClient.NewMethod(ctx, input)
}
```

### 3. HTTPTransport (`transport_http.go`)

A read decodes with `fetchTron`; the request must **not** set `visible` (addresses go as hex):

```go
func (t *HTTPTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    reqBody := map[string]any{
        "owner_address": hexAddress(input.OwnerAddress),
    }
    result := &OutputProto{}
    if err := t.fetchTron(ctx, "/wallet/newmethod", reqBody, result); err != nil {
        return nil, err
    }
    return result, nil
}
```

A transaction-creating endpoint goes through `doTxRequest` instead. Then add the method to
`TestLocalParity_*` (see [testing.md](testing.md#local-private-network)) — that comparison against gRPC on the same node is
what catches a field the node renders differently from what the decoder expects.

### 4. RoundRobinTransport (`transport_roundrobin.go`)

```go
func (t *RoundRobinTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    return t.next().NewMethod(ctx, input)
}
```

### 5. HealthAwareTransport (`health.go`)

```go
func (h *HealthAwareTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    n, err := h.next()
    if err != nil {
        return nil, err
    }
    res, callErr := n.transport.NewMethod(ctx, input)
    h.recordOutcome(n, callErr)
    return res, callErr
}
```

### 6. MetricsTransport (`transport_metrics.go`)

```go
func (t *MetricsTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    start := time.Now()
    result, err := t.transport.NewMethod(ctx, input)
    t.after("NewMethod", start, err)
    return result, err
}
```

### 7. Client method (`pkg/client/<domain>.go`)

```go
func (c *Client) NewMethod(ctx context.Context, humanFriendlyParam string) (*OutputProto, error) {
    // Convert string addresses to bytes
    addrBytes, err := tronutils.DecodeCheck(humanFriendlyParam)
    if err != nil {
        return nil, err
    }
    // Call transport
    result, err := c.transport.NewMethod(ctx, &InputProto{Address: addrBytes})
    if err != nil {
        return nil, err
    }
    // Validate response
    if result == nil {
        return nil, ErrNilResponse
    }
    return result, nil
}
```

### 8. Tests

- `tests/<domain>_test.go` — `TestNewMethod_GRPC` and `TestNewMethod_HTTP` against the public
  nodes, asserting real values rather than "no error".
- `tests/local_parity_*_test.go` — a parity case comparing the bare `GRPCTransport` and
  `HTTPTransport` answers with `proto.Equal` on the local node. This is what catches a field the
  node renders differently from what the decoder expects.
- `pkg/client/*_test.go` — unit tests with `newStubTransport` for the HTTP request body and for
  verbatim node answers, especially error shapes.
- Document the client method in the skill's `references/api-surface.md`.

See [testing.md](testing.md) for helpers and conventions.

---

## Endpoints and internal constants

### HTTP endpoint mapping

| Transport method             | HTTP endpoint                          |
| ---------------------------- | -------------------------------------- |
| GetAccount                   | `/wallet/getaccount`                   |
| GetAccountResource           | `/wallet/getaccountresource`           |
| CreateAccount                | `/wallet/createaccount`                |
| AccountPermissionUpdate      | `/wallet/accountpermissionupdate`      |
| GetNowBlock                  | `/wallet/getnowblock`                  |
| GetBlockByNum                | `/wallet/getblockbynum`                |
| GetBlockById                 | `/wallet/getblockbyid`                 |
| GetBlockByLimitNext          | `/wallet/getblockbylimitnext`          |
| GetBlockByLatestNum          | `/wallet/getblockbylatestnum`          |
| GetTransactionById           | `/wallet/gettransactionbyid`           |
| GetTransactionInfoById       | `/wallet/gettransactioninfobyid`       |
| GetTransactionInfoByBlockNum | `/wallet/gettransactioninfobyblocknum` |
| BroadcastTransaction         | `/wallet/broadcasthex`                 |
| CreateTransaction            | `/wallet/createtransaction`            |
| TriggerContract              | `/wallet/triggersmartcontract`         |
| TriggerConstantContract      | `/wallet/triggerconstantcontract`      |
| EstimateEnergy               | `/wallet/estimateenergy`               |
| DeployContract               | `/wallet/deploycontract`               |
| GetContract                  | `/wallet/getcontract`                  |
| UpdateSetting                | `/wallet/updatesetting`                |
| UpdateEnergyLimit            | `/wallet/updateenergylimit`            |
| DelegateResource             | `/wallet/delegateresource`             |
| UnDelegateResource           | `/wallet/undelegateresource`           |
| FreezeBalanceV2              | `/wallet/freezebalancev2`              |
| UnfreezeBalanceV2            | `/wallet/unfreezebalancev2`            |
| WithdrawExpireUnfreeze       | `/wallet/withdrawexpireunfreeze`       |
| CancelAllUnfreezeV2          | `/wallet/cancelallunfreezev2`          |
| VoteWitnessAccount           | `/wallet/votewitnessaccount`           |
| WithdrawBalance              | `/wallet/withdrawbalance`              |
| GetRewardInfo                | `/wallet/getReward` (camelCase)        |
| GetBrokerageInfo             | `/wallet/getBrokerage` (camelCase)     |
| ListNodes                    | `/wallet/listnodes`                    |
| GetChainParameters           | `/wallet/getchainparameters`           |

### gRPC message size

`pkg/client/transport_grpc.go`:

```go
defaultMaxSizeOption = grpc.MaxCallRecvMsgSize(32 * 10e6)  // ~320MB, passed on block calls
// Also configured: grpc.MaxCallRecvMsgSize(1024*1024*100)  // 100MB in the dial options
```

### Unexported TRC20 selectors

`pkg/client/trc20.go` (the exported ones are listed in the skill's constants reference):

```go
trc20TransferMethodSignature = "0xa9059cbb" // transfer(address,uint256)
trc20ApproveMethodSignature  = "0x095ea7b3" // approve(address,uint256)
trc20BalanceOf               = "0x70a08231" // balanceOf(address)
trc20NameSignature           = "0x06fdde03" // name()
trc20SymbolSignature         = "0x95d89b41" // symbol()
trc20DecimalsSignature       = "0x313ce567" // decimals()
```

### Address derivation constants

`pkg/address/address.go`:

```go
bip44Purpose   = 44
tronCoinType   = 195
defaultAccount = 0
defaultChange  = 0
addressLength  = 21
prefixByte     = 0x41 // Tron mainnet address prefix
```
