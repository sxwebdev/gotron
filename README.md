# Gotron SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/sxwebdev/gotron.svg)](https://pkg.go.dev/github.com/sxwebdev/gotron)
[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://go.dev/)
[![License](https://img.shields.io/github/license/sxwebdev/gotron)](LICENSE)

A comprehensive Go SDK for the Tron blockchain. This library provides a complete client implementation for interacting with Tron nodes via gRPC or HTTP, managing addresses, creating and signing transactions, and working with TRC20 tokens.

## Features

- **Dual Transport Support** - gRPC and HTTP REST API
- **Round-Robin Load Balancing** - Automatic load balancing across multiple nodes
- **Complete API Client** - Full implementation of Tron Wallet API
- **Address Management** - BIP39/BIP44 mnemonic support, address generation and validation
- **Transaction Handling** - Create, sign, and broadcast transactions
- **TRC20 Token Support** - Transfer, approve, balance queries, token info
- **Resource Management** - Delegate/undelegate bandwidth and energy
- **Account Operations** - Balance queries, account info, activation
- **Block & Transaction Queries** - Get blocks, transactions, and receipts
- **Multi-Network Support** - Mainnet, Shasta testnet, Nile testnet
- **Precision Arithmetic** - Uses `decimal.Decimal` for accurate calculations
- **Type Safety** - Full type definitions with validation
- **Native Implementation** - Built on official Tron protocol buffers

## Installation

```bash
go get github.com/sxwebdev/gotron
```

## Quick Start

### Initialize Client

```go
package main

import (
  "context"
  "fmt"
  "log"

  "github.com/sxwebdev/gotron/pkg/client"
)

func main() {
  // Create client with gRPC node
  cfg := client.Config{
    Nodes: []client.NodeConfig{
      {
        Protocol: client.ProtocolGRPC,
        Address:  "grpc.trongrid.io:50051",
        UseTLS:   true,
      },
    },
  }

  tron, err := client.New(cfg)
  if err != nil {
    log.Fatal(err)
  }
  defer tron.Close()

  ctx := context.Background()

  // Get account balance (in TRX)
  balance, err := tron.GetAccountBalance(ctx, "TYourAddress")
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("Balance: %s TRX\n", balance.String())
}
```

### Generate Addresses

```go
import "github.com/sxwebdev/gotron/pkg/address"

// Generate a new 12-word mnemonic
mnemonic, err := address.GenerateMnemonic(128)
if err != nil {
  log.Fatal(err)
}

// Derive address from mnemonic (BIP44 path: m/44'/195'/0'/0/0)
addr, err := address.FromMnemonic(mnemonic, "", 0)
if err != nil {
  log.Fatal(err)
}

fmt.Printf("Address: %s\n", addr.Address)
fmt.Printf("Private Key: %s\n", addr.PrivateKey)
fmt.Printf("Mnemonic: %s\n", addr.Mnemonic)

// Generate random address
randomAddr, err := address.Generate()
if err != nil {
  log.Fatal(err)
}

// Import from private key
importedAddr, err := address.FromPrivateKey("your-hex-private-key")
if err != nil {
  log.Fatal(err)
}
```

### Transfer TRX

```go
import (
  "context"
  "github.com/shopspring/decimal"
  "github.com/sxwebdev/gotron/pkg/address"
)

ctx := context.Background()

// Create transfer transaction (amount in TRX)
tx, err := tron.CreateTransferTransaction(
  ctx,
  "TFromAddress",
  "TToAddress",
  decimal.NewFromFloat(1.5), // 1.5 TRX
)
if err != nil {
  log.Fatal(err)
}

// Import private key
privateKey, err := address.PrivateKeyFromHex("your-hex-private-key")
if err != nil {
  log.Fatal(err)
}

// Sign transaction
err = tron.SignTransaction(tx.Transaction, privateKey)
if err != nil {
  log.Fatal(err)
}

// Broadcast transaction
result, err := tron.BroadcastTransaction(ctx, tx.Transaction)
if err != nil {
  log.Fatal(err)
}

fmt.Printf("Transaction broadcasted: %s\n", result.Message)
```

### TRC20 Token Operations

```go
const (
  usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT on Tron
)

ctx := context.Background()

// Get token info
name, _ := tron.TRC20GetName(ctx, usdtContract)
symbol, _ := tron.TRC20GetSymbol(ctx, usdtContract)
decimals, _ := tron.TRC20GetDecimals(ctx, usdtContract)

fmt.Printf("Token: %s (%s), Decimals: %d\n", name, symbol, decimals)

// Get token balance
balance, err := tron.TRC20ContractBalance(ctx, "TYourAddress", usdtContract)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Balance: %s\n", balance.String())

// Transfer tokens (amount in smallest unit, 1 USDT = 1000000)
tx, err := tron.TRC20Send(
  ctx,
  "TFromAddress",
  "TToAddress",
  usdtContract,
  decimal.NewFromInt(1000000), // 1 USDT
  100_000_000, // Fee limit in SUN (100 TRX)
)
if err != nil {
    log.Fatal(err)
}

// Sign and broadcast...
privateKey, _ := address.PrivateKeyFromHex("your-private-key")
tron.SignTransaction(tx.Transaction, privateKey)
result, _ := tron.BroadcastTransaction(ctx, tx.Transaction)
```

### Delegate & Reclaim Resources

```go
import "github.com/sxwebdev/gotron"

ctx := context.Background()

// Delegate 1000 TRX worth of energy to another address
tx, err := tron.DelegateResource(
  ctx,
  "TOwnerAddress",
  "TReceiverAddress",
  gotron.Energy,              // Resource type
  1000_000_000,               // Balance in SUN (1000 TRX)
  false,                      // Lock
  0,                          // Lock period
)
if err != nil {
    log.Fatal(err)
}

// Sign and broadcast...

// Reclaim delegated resources
reclaimTx, err := tron.ReclaimResource(
  ctx,
  "TOwnerAddress",
  "TReceiverAddress",
  gotron.Energy,
  1000_000_000, // Amount in SUN
)
```

### Query Blocks and Transactions

```go
ctx := context.Background()

// Get latest block
block, err := tron.GetLastBlock(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Latest block: %d\n", block.BlockHeader.RawData.Number)

// Get block by height
specificBlock, err := tron.GetBlockByHeight(ctx, 12345678)

// Get transaction by hash
tx, err := tron.GetTransactionByHash(ctx, "transaction-hash")

// Get transaction receipt
receipt, err := tron.GetTransactionInfoByHash(ctx, "transaction-hash")
fmt.Printf("Result: %s\n", receipt.Result)
```

### Account Operations

```go
ctx := context.Background()

// Check if account is activated
isActivated, err := tron.IsAccountActivated(ctx, "TAddress")

// Estimate activation cost
estimate, err := tron.EstimateActivateAccount(ctx, "TFromAddress", "TToAddress")
fmt.Printf("Activation cost: %s TRX\n", estimate.Trx.String())

// Get account resources
resources, err := tron.TotalAvailableResources(ctx, "TAddress")
fmt.Printf("Energy: %s, Bandwidth: %s\n",
  resources.Energy.String(),
  resources.Bandwidth.String(),
)
```

## Configuration

The SDK uses a unified `Nodes` configuration with round-robin load balancing.

### Single gRPC Node

```go
import "github.com/sxwebdev/gotron/pkg/client"

cfg := client.Config{
    Nodes: []client.NodeConfig{
        {
            Protocol: client.ProtocolGRPC,
            Address:  "grpc.trongrid.io:50051",
            UseTLS:   true,
        },
    },
}

tron, err := client.New(cfg)
```

### Single HTTP Node

```go
import "github.com/sxwebdev/gotron/pkg/client"

cfg := client.Config{
    Nodes: []client.NodeConfig{
        {
            Protocol: client.ProtocolHTTP,
            Address:  "https://api.trongrid.io",
            Headers: map[string]string{
                "TRON-PRO-API-KEY": "your-api-key",
            },
        },
    },
}

tron, err := client.New(cfg)
```

### Multiple Nodes (Round-Robin)

```go
import "github.com/sxwebdev/gotron/pkg/client"

cfg := client.Config{
    Nodes: []client.NodeConfig{
        {
            Protocol: client.ProtocolGRPC,
            Address:  "grpc.trongrid.io:50051",
            UseTLS:   true,
        },
        {
            Protocol: client.ProtocolHTTP,
            Address:  "https://api.trongrid.io",
            Headers: map[string]string{
                "TRON-PRO-API-KEY": "your-api-key",
            },
        },
        {
            Protocol: client.ProtocolHTTP,
            Address:  "https://tron-rpc.publicnode.com",
        },
    },
}

tron, err := client.New(cfg)
```

Each request uses the next node in round-robin fashion. Errors are returned as-is without retries.

### TronGrid with API Key (gRPC)

The `Headers` field works for both HTTP and gRPC transports:

```go
import "github.com/sxwebdev/gotron/pkg/client"

cfg := client.Config{
    Nodes: []client.NodeConfig{
        {
            Protocol: client.ProtocolGRPC,
            Address:  "grpc.trongrid.io:50051",
            UseTLS:   true,
            Headers: map[string]string{
                "TRON-PRO-API-KEY": "your-api-key",
            },
        },
    },
}

tron, err := client.New(cfg)
```

### Custom HTTP Client

```go
import (
    "net/http"
    "time"
    "github.com/sxwebdev/gotron/pkg/client"
)

cfg := client.Config{
    Nodes: []client.NodeConfig{
        {
            Protocol: client.ProtocolHTTP,
            Address:  "https://your-custom-node",
            HTTPClient: &http.Client{
                Timeout: 60 * time.Second,
            },
            Headers: map[string]string{
                "Authorization": "Bearer your-token",
            },
        },
    },
}

tron, err := client.New(cfg)
```

## Prometheus Metrics

The SDK supports optional metrics for monitoring RPC performance. You can use the built-in Prometheus metrics or provide a custom `MetricsCollector` implementation.

### Built-in Prometheus Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/sxwebdev/gotron/pkg/client"
)

// Create built-in metrics collector
metrics := client.NewMetrics(prometheus.DefaultRegisterer)

// Create client with metrics
cfg := client.Config{
    Nodes: []client.NodeConfig{
        {
            Address: "grpc.trongrid.io:50051",
            UseTLS:  true,
        },
    },
    Blockchain: "tron", // Label for metrics (default: "tron")
    Metrics:    metrics,
}

tron, err := client.New(cfg)
```

### Custom MetricsCollector

Implement the `MetricsCollector` interface to use your own metrics:

```go
type MetricsCollector interface {
    RecordRequest(blockchain, method, status string, duration time.Duration)
    RecordRetry(blockchain, method string)
    SetPoolHealth(blockchain string, total, healthy, disabled int)
}
```

Example with external metrics:

```go
cfg := client.Config{
    Nodes:      nodes,
    Blockchain: "tron",
    Metrics:    myExternalMetrics, // any MetricsCollector implementation
}
```

### Available Metrics

| Metric                        | Type      | Labels                           | Description                              |
| ----------------------------- | --------- | -------------------------------- | ---------------------------------------- |
| `gotron_rpc_requests_total`   | Counter   | `blockchain`, `method`, `status` | Total number of RPC requests             |
| `gotron_rpc_duration_seconds` | Histogram | `blockchain`, `method`           | RPC request duration in seconds          |
| `gotron_rpc_in_flight`        | Gauge     | -                                | Number of requests currently in progress |
| `gotron_rpc_retries_total`    | Counter   | `blockchain`, `method`           | Total number of RPC retries              |
| `gotron_rpc_pool_total`       | Gauge     | `blockchain`                     | Total number of nodes in the pool        |
| `gotron_rpc_pool_healthy`     | Gauge     | `blockchain`                     | Number of healthy nodes in the pool      |
| `gotron_rpc_pool_disabled`    | Gauge     | `blockchain`                     | Number of disabled nodes in the pool     |

### Example Prometheus Queries

```promql
# Request rate per method
rate(gotron_rpc_requests_total{blockchain="tron"}[5m])

# Error rate
rate(gotron_rpc_requests_total{blockchain="tron",status="error"}[5m])

# P99 latency
histogram_quantile(0.99, rate(gotron_rpc_duration_seconds_bucket{blockchain="tron"}[5m]))

# Current in-flight requests
gotron_rpc_in_flight
```

## Package Structure

```text
gotron/
├── tron.go                  # High-level wrapper and constants
├── pkg/
│   ├── address/             # Address generation, validation, BIP39/BIP44
│   ├── client/              # Client implementation
│   │   ├── client.go        # Client initialization
│   │   ├── config.go        # Configuration (Nodes, NodeConfig)
│   │   ├── transport.go     # Transport interface
│   │   ├── transport_grpc.go # gRPC transport implementation
│   │   ├── transport_http.go # HTTP transport implementation
│   │   ├── transport_roundrobin.go # Round-robin load balancer
│   │   ├── account.go       # Account operations
│   │   ├── transfer.go      # TRX transfers
│   │   ├── trc20.go         # TRC20 token operations
│   │   ├── resources.go     # Resource delegation
│   │   ├── block.go         # Block queries
│   │   ├── transactions.go  # Transaction operations
│   │   └── ...
│   └── tronutils/           # Utility functions
└── schema/pb/               # Protocol buffer definitions
    ├── api/                 # Tron API definitions
    └── core/                # Core protocol types
```

## Constants

```go
// Resource Types
gotron.Bandwidth
gotron.Energy

// Decimals
gotron.TrxDecimals = 6

// Event Signatures
gotron.Trc20TransferEventSignature
```

## Error Handling

The library provides typed errors for common scenarios:

```go
import "github.com/sxwebdev/gotron/pkg/client"

_, err := tron.GetAccount(ctx, "invalid-address")
if errors.Is(err, client.ErrAccountNotFound) {
  fmt.Println("Account not found")
}

// Other errors:
// - client.ErrInvalidAddress
// - client.ErrInvalidAmount
// - client.ErrInvalidConfig
// - client.ErrTransactionNotFound
// - client.ErrInvalidResourceType
```

## Advanced Usage

### Address Generator with Custom Derivation

```go
import "github.com/sxwebdev/gotron/pkg/address"

// Create address generator
generator := address.NewAddressGenerator(mnemonic, "passphrase")

// Customize BIP44 path
generator.
  SetBipPurpose(44).
  SetCoinType(195).
  SetAccount(0)

// Generate multiple addresses
for i := uint32(0); i < 10; i++ {
  addr, err := generator.Generate(i)
  if err != nil {
      log.Fatal(err)
  }
  fmt.Printf("Address %d: %s\n", i, addr.Address)
}
```

## Testing

```bash
go test ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## Resources

- [Tron Documentation](https://developers.tron.network/)
- [TronGrid API](https://www.trongrid.io/)
- [pkg.go.dev Documentation](https://pkg.go.dev/github.com/sxwebdev/gotron)
