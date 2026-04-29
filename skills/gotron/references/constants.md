# Gotron Constants, Errors, and Enums

## Sentinel Errors

**File:** `pkg/client/errors.go`

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

// Account (file: account.go)
ErrAccountNotFound          = errors.New("account not found")
```

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

Usage: `errors.As(err, &transportErr)` to inspect which node failed.

## Network Types

**File:** `pkg/client/types.go`

```go
type Network string

NetworkMainnet Network = "mainnet"
NetworkShasta  Network = "shasta"
NetworkNile    Network = "nile"
```

## Protocol Types

**File:** `pkg/client/config.go`

```go
type Protocol string

ProtocolGRPC Protocol = "grpc"
ProtocolHTTP Protocol = "http"
```

## Resource Types

**File:** `pkg/client/types.go`

```go
type ResourceType int32

ResourceTypeBandwidth ResourceType = 0
ResourceTypeEnergy    ResourceType = 1
```

Methods: `Validate()`, `String()` ("BANDWIDTH"/"ENERGY"), `ToProto()` -> `core.ResourceCode`

## TRX Constants

**File:** `pkg/client/constants.go`

```go
TrxDecimals        = 6            // 1 TRX = 1,000,000 SUN
TrxAssetIdentifier = "trx"
```

## TRC20 Method Signatures

**File:** `pkg/client/trc20.go`

```go
trc20TransferMethodSignature     = "0xa9059cbb"   // transfer(address,uint256)
trc20ApproveMethodSignature      = "0x095ea7b3"   // approve(address,uint256)
Trc20TransferFromMethodSignature = "0x23b872dd"   // transferFrom(address,address,uint256)
trc20BalanceOf                   = "0x70a08231"   // balanceOf(address)
trc20NameSignature               = "0x06fdde03"   // name()
trc20SymbolSignature             = "0x95d89b41"   // symbol()
trc20DecimalsSignature           = "0x313ce567"   // decimals()

// Event signature (exported)
Trc20TransferEventSignature = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
```

## Address Constants

**File:** `pkg/address/address.go`

```go
bip44Purpose   = 44
tronCoinType   = 195
defaultAccount = 0
defaultChange  = 0
addressLength  = 21
prefixByte     = 0x41  // Tron mainnet address prefix
```

BIP44 derivation path: `m/44'/195'/0'/0/{index}`

## gRPC Constants

**File:** `pkg/client/transport_grpc.go`

```go
defaultMaxSizeOption = grpc.MaxCallRecvMsgSize(32 * 10e6)  // ~320MB
// Also configured: grpc.MaxCallRecvMsgSize(1024*1024*100)  // 100MB in dial options
```

## Prometheus Metric Names

**File:** `pkg/client/metrics.go`

| Metric                        | Type      | Labels                     |
| ----------------------------- | --------- | -------------------------- |
| `gotron_rpc_requests_total`   | Counter   | blockchain, method, status |
| `gotron_rpc_duration_seconds` | Histogram | blockchain, method         |
| `gotron_rpc_in_flight`        | Gauge     | (none)                     |
| `gotron_rpc_retries_total`    | Counter   | blockchain, method         |
| `gotron_rpc_pool_total`       | Gauge     | blockchain                 |
| `gotron_rpc_pool_healthy`     | Gauge     | blockchain                 |
| `gotron_rpc_pool_disabled`    | Gauge     | blockchain                 |

## HTTP Endpoint Mapping

Key endpoints used by `HTTPTransport`:

| Client Method                | HTTP Endpoint                          |
| ---------------------------- | -------------------------------------- |
| GetAccount                   | `/wallet/getaccount`                   |
| GetAccountResource           | `/wallet/getaccountresource`           |
| CreateAccount                | `/wallet/createaccount`                |
| GetNowBlock                  | `/wallet/getnowblock`                  |
| GetBlockByNum                | `/wallet/getblockbynum`                |
| GetBlockById                 | `/wallet/getblockbyid`                 |
| GetBlockByLimitNext          | `/wallet/getblockbylimitnext`          |
| GetBlockByLatestNum          | `/wallet/getblockbylatestnum`          |
| GetTransactionById           | `/wallet/gettransactionbyid`           |
| GetTransactionInfoById       | `/wallet/gettransactioninfobyid`       |
| GetTransactionInfoByBlockNum | `/wallet/gettransactioninfobyblocknum` |
| BroadcastTransaction         | `/wallet/broadcasttransaction`         |
| CreateTransaction            | `/wallet/createtransaction`            |
| TriggerContract              | `/wallet/triggersmartcontract`         |
| TriggerConstantContract      | `/wallet/triggerconstantcontract`      |
| EstimateEnergy               | `/wallet/estimateenergy`               |
| DeployContract               | `/wallet/deploycontract`               |
| GetContract                  | `/wallet/getcontract`                  |
| DelegateResource             | `/wallet/delegateresource`             |
| UnDelegateResource           | `/wallet/undelegateresource`           |
| ListNodes                    | `/wallet/listnodes`                    |
| GetChainParameters           | `/wallet/getchainparameters`           |
