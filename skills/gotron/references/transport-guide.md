# Transport Layer Guide

## Table of Contents

- [Transport interface](#transport-interface)
- [GRPCTransport](#grpctransport)
- [HTTPTransport](#httptransport)
- [RoundRobinTransport](#roundrobintransport)
- [MetricsTransport](#metricstransport)
- [Adding a new transport method](#adding-a-new-transport-method)

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

**Key challenge:** Tron's HTTP API returns non-standard JSON that differs from protobuf JSON:

- Hex strings instead of base64 for bytes fields
- `type_url`/`value` instead of `@type` for Any types
- `blockID` instead of `blockid`, `txID` instead of `txid`
- Transactions not wrapped in `TransactionExtention` for block responses
- Some endpoints return arrays directly instead of wrapper objects

**Request methods (choose based on response format):**

| Method                 | When to use                                                              |
| ---------------------- | ------------------------------------------------------------------------ |
| `doRequest`            | Standard responses that work with `protojson.Unmarshal` directly         |
| `doRequestTransformed` | Responses with Any types needing `transformTronJSON`                     |
| `doBlockRequest`       | Block responses needing transaction wrapping into `TransactionExtention` |
| `doBlockListRequest`   | Block list responses (array -> `{block: [...]}`)                         |
| `doRequestRaw`         | Custom parsing needed (returns raw `[]byte`)                             |

**Custom parsing examples:**

- `GetAccount` — uses `httpAccount` helper struct because account JSON is incompatible with protojson
- `GetAccountResource` — uses `httpAccountResourceMessage` helper struct
- `TriggerConstantContract` — uses `httpTriggerConstantContractResponse` for `constant_result` parsing
- `GetTransactionInfoByBlockNum` — array wrapping + hex->base64 transform

**HTTP endpoints map to `/wallet/<methodname>` paths:**

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

No retry logic, no health checking. Errors propagated as-is.

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

Choose the right request method based on response format:

```go
func (t *HTTPTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    reqBody := map[string]interface{}{
        "field": value,
        "visible": true,  // include for address-related endpoints
    }
    result := &OutputProto{}
    if err := t.doRequest(ctx, "/wallet/newmethod", reqBody, result); err != nil {
        return nil, err
    }
    return result, nil
}
```

### 4. RoundRobinTransport (`transport_roundrobin.go`)

```go
func (t *RoundRobinTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    return t.next().NewMethod(ctx, input)
}
```

### 5. MetricsTransport (`transport_metrics.go`)

```go
func (t *MetricsTransport) NewMethod(ctx context.Context, input *InputProto) (*OutputProto, error) {
    start := time.Now()
    result, err := t.transport.NewMethod(ctx, input)
    t.after("NewMethod", start, err)
    return result, err
}
```

### 6. Client method (`pkg/client/<domain>.go`)

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
