# Gotron SDK

A comprehensive Go SDK for the Tron blockchain. This library provides a complete client implementation for interacting with Tron nodes via gRPC, managing addresses, creating and signing transactions, and working with TRC20 tokens.

## Features

- **Complete gRPC Client** - Full implementation of Tron Wallet API
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

    "github.com/sxwebdev/gotron"
)

func main() {
    // Create client with default mainnet configuration
    cfg := gotron.Config{
        Network: gotron.Mainnet,
    }

    tron, err := gotron.New(cfg)
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
    resources.Bandwidth.String())
```

## Network Configuration

### Predefined Networks

```go
// Mainnet (default)
cfg := gotron.Config{
    Network: gotron.Mainnet,
}

// Shasta Testnet
cfg := gotron.Config{
    Network: gotron.Shasta,
}

// Nile Testnet
cfg := gotron.Config{
    Network: gotron.Nile,
}
```

### Custom Configuration

```go
import (
    "github.com/sxwebdev/gotron/pkg/client"
    "google.golang.org/grpc"
)

cfg := client.Config{
    GRPCAddress: "your-custom-node:50051",
    UseTLS:      true,
    Network:     client.NetworkMainnet,
    DialOptions: []grpc.DialOption{
        // Add custom dial options here
    },
}

tron, err := gotron.New(cfg)
```

### TronGrid with API Key (Interceptor Pattern)

For production use with TronGrid, implement an interceptor to add the API key header:

```go
import (
    "context"

    "github.com/sxwebdev/gotron/pkg/client"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

// Create interceptor that adds TRON-PRO-API-KEY header
func tronGridAPIKeyInterceptor(apiKey string) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{},
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

        ctx = metadata.AppendToOutgoingContext(ctx, "TRON-PRO-API-KEY", apiKey)
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

func main() {
    apiKey := "your-trongrid-api-key"

    cfg := client.Config{
        Network: client.NetworkMainnet,
        DialOptions: []grpc.DialOption{
            grpc.WithUnaryInterceptor(tronGridAPIKeyInterceptor(apiKey)),
        },
    }

    tron, err := gotron.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer tron.Close()

    // All requests will include the API key header
    balance, err := tron.GetAccountBalance(ctx, "TAddress")
}
```

## Package Structure

```text
gotron/
├── tron.go              # High-level wrapper and constants
├── pkg/
│   ├── address/         # Address generation, validation, BIP39/BIP44
│   ├── client/          # Core gRPC client implementation
│   │   ├── client.go    # Client initialization
│   │   ├── account.go   # Account operations
│   │   ├── transfer.go  # TRX transfers
│   │   ├── trc20.go     # TRC20 token operations
│   │   ├── resources.go # Resource delegation
│   │   ├── block.go     # Block queries
│   │   ├── transactions.go # Transaction operations
│   │   └── ...
│   └── utils/           # Utility functions
└── schema/pb/           # Protocol buffer definitions
    ├── api/             # Tron API definitions
    └── core/            # Core protocol types
```

## Constants

```go
// Networks
gotron.Mainnet
gotron.Shasta
gotron.Nile

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

### Direct API Access

```go
// Access underlying gRPC client for advanced operations
walletAPI := tron.API()

// Use any Wallet API method directly
nodeInfo, err := walletAPI.GetNodeInfo(ctx, &api.EmptyMessage{})
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
