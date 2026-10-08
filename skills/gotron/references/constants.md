# Gotron Constants, Errors, and Enums

## Sentinel Errors

```go
// Common
ErrInvalidConfig            = errors.New("invalid client configuration")
ErrNotConnected             = errors.New("client not connected")
ErrInvalidParams            = errors.New("invalid parameters")
ErrNilResponse              = errors.New("nil response from server")

// Address
ErrInvalidAddress           = errors.New("invalid address")
ErrEmptyAddress             = errors.New("address is empty")
ErrAccountNotActivated      = errors.New("account is not activated")

// Transaction
ErrInvalidAmount            = errors.New("invalid amount")
ErrInvalidTransaction       = errors.New("invalid transaction")
ErrInvalidPrivateKey        = errors.New("invalid private key")
ErrTransactionNotFound      = errors.New("transaction not found")
ErrTransactionInfoNotFound  = errors.New("transaction info not found")

// Resources
ErrInvalidResourceType      = errors.New("invalid resource type")

// Delegation refusals (matched on *ContractValidateError by errors.Is)
ErrDelegateStakeShort       = errors.New("delegate balance exceeds the owner's available stake")
ErrDelegateBelowMinimum     = errors.New("delegate balance below the minimum delegation")

// Account
ErrAccountNotFound          = errors.New("account not found")

// Account permissions
ErrInvalidPermissionID     = errors.New("invalid permission id")
ErrInvalidPermission       = errors.New("invalid permission")
ErrPermissionNotFound      = errors.New("permission not found")
ErrPermissionDenied        = errors.New("permission denied")

// Transport (HTTP only)
ErrNodeRefusedRequest       = errors.New("node refused the request")

// Health-checker / tier fallback
ErrNoHealthyNodes           = errors.New("no healthy nodes available in any tier")
```

`ErrNoHealthyNodes` is returned when every node of every tier is currently
marked unhealthy. The background probe loop
keeps retrying — callers should retry with backoff. Detect with
`errors.Is(err, client.ErrNoHealthyNodes)`.

`ErrNodeRefusedRequest` marks a request an HTTP node would not process at all —
a malformed address, an unparseable number. The `/wallet` endpoints report that
as **HTTP 200 with an `"Error"` field** rather than a status code, so without
the sentinel it is only distinguishable from a real transport failure by
substring-matching a Java class name. It says the request was wrong, not the
node: node health is untouched and retrying elsewhere returns the same refusal.
The transaction-creating endpoints report the same class of refusal as a
`*ContractValidateError`, which additionally carries the code gRPC gives.

**Address package errors** (`pkg/address/address.go`):

```go
ErrInvalidMnemonic   = errors.New("invalid mnemonic")
ErrInvalidPrivateKey = errors.New("invalid private key")
ErrInvalidAddress    = errors.New("invalid address")
```

## TransportError

```go
type TransportError struct {
    Host     string  // "grpc.trongrid.io:50051" or "https://api.trongrid.io"
    Protocol string  // "grpc" or "http"
    Method   string  // "/protocol.Wallet/GetAccount" or "/wallet/getaccount"
    Err      error   // original error
}
```

Usage: `errors.AsType[*TransportError](err)` (Go 1.26+) to inspect which node failed.

## HTTPStatusError

```go
type HTTPStatusError struct {
    Code int    // HTTP status (e.g. 503)
    Body string // raw response body
}
```

Returned by an HTTP node that answers with a non-2xx status, wrapped inside a
`TransportError`. The default health classifier treats 5xx, 408 and 429 as
network-level failures (they count toward a node's unhealthy threshold) and
other 4xx codes as logical errors (they do not affect node health). Inspect with
`errors.AsType[*HTTPStatusError](err)` (Go 1.26+).

`ErrDelegateStakeShort` and `ErrDelegateBelowMinimum` are never returned bare:
a `*ContractValidateError` from `DelegateResource` matches them with
`errors.Is` when its message is the node's verdict — the owner's stake of the
resource, less what its own usage holds, is under the amount; or the amount is
under `MinDelegateBalance`. Both wordings java-tron has used (before and since
GreatVoyage-v4.7.3) and both transports' prefixes match. The client refuses an
amount under the minimum itself, without asking a node.

## Network Types

```go
type Network string

NetworkMainnet Network = "mainnet"
NetworkShasta  Network = "shasta"
NetworkNile    Network = "nile"
```

## Account Permission IDs

```go
const (
    OwnerPermissionID       int32 = 0
    WitnessPermissionID     int32 = 1
    FirstActivePermissionID int32 = 2
    LastActivePermissionID  int32 = 9
)
```

## Protocol Types

```go
type Protocol string

ProtocolGRPC Protocol = "grpc"
ProtocolHTTP Protocol = "http"
```

## Resource Types

```go
type ResourceType int32

ResourceTypeBandwidth ResourceType = 0
ResourceTypeEnergy    ResourceType = 1
```

Methods: `Validate()`, `String()` ("BANDWIDTH"/"ENERGY"), `ToProto()` -> `core.ResourceCode`

## TRX Constants

```go
TrxDecimals        = 6            // 1 TRX = 1,000,000 SUN
TrxAssetIdentifier = "trx"

MinDelegateBalance SUN = 1_000_000 // least stake a DelegateResource may lend (1 TRX)
```

## TRC20 Method Signatures

Exported from `pkg/client`:

```go
Trc20TransferFromMethodSignature = "0x23b872dd" // transferFrom(address,address,uint256)
Trc20TransferEventSignature      = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" // Transfer event topic[0]
```

The other selectors (`transfer`, `approve`, `balanceOf`, `name`, `symbol`, `decimals`) are used by
the `TRC20*` methods and are not exported; call those methods instead of building the data
yourself.

## Address Constants

- Address length: 21 bytes, prefix byte `0x41` (base58check renders it as a leading `T`).
- BIP44 derivation path used by `address.FromMnemonic` / `address.NewGenerator`: `m/44'/195'/0'/0/{index}`.

## Prometheus Metric Names

| Metric                        | Type      | Labels                     |
| ----------------------------- | --------- | -------------------------- |
| `gotron_rpc_requests_total`   | Counter   | blockchain, method, status |
| `gotron_rpc_duration_seconds` | Histogram | blockchain, method         |
| `gotron_rpc_in_flight`        | Gauge     | (none)                     |
| `gotron_rpc_retries_total`    | Counter   | blockchain, method         |
| `gotron_rpc_pool_total`       | Gauge     | blockchain                 |
| `gotron_rpc_pool_healthy`     | Gauge     | blockchain                 |
| `gotron_rpc_pool_disabled`    | Gauge     | blockchain                 |
