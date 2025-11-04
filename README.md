# Gotron

A comprehensive Go SDK for the Tron blockchain network. This library provides a high-level client for interacting with Tron nodes, managing addresses, and executing transactions.

## Features

- ✅ High-level client with simple API
- ✅ Address generation from BIP39 mnemonic phrases
- ✅ Random address generation
- ✅ Address validation
- ✅ Transaction creation and signing
- ✅ TRC20 token operations
- ✅ Resource delegation and management
- ✅ Balance queries
- ✅ Multi-network support (Mainnet, Testnet, Nile)
- ✅ Custom client configuration with TLS support
- ✅ Parameter validation throughout
- ✅ Uses `decimal.Decimal` for precise calculations
- ✅ No third-party Tron SDK dependencies

## Installation

```bash
go get github.com/sxwebdev/gotron
```

## Quick Start

### Create a Client and Get Balance

```go
package main

import (
    "fmt"
    "log"
    "github.com/sxwebdev/gotron"
)

func main() {
    // Create client for mainnet
    client, err := gotron.NewClient(gotron.Mainnet)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get balance
    balance, err := client.GetBalance("TYourTronAddress")
    if err != nil {
        log.Fatal(err)
    }

    // Convert from SUN to TRX
    trx := balance.Div(decimal.NewFromInt(1000000))
    fmt.Printf("Balance: %s TRX\n", trx)
}
```

### Generate Address

```go
import "github.com/sxwebdev/gotron"

// Generate a random address
addr, err := gotron.GenerateAddress()
if err != nil {
    log.Fatal(err)
}
fmt.Println("Address:", addr.Address)
fmt.Println("Private Key:", addr.PrivateKey)

// Or generate from mnemonic
mnemonic, _ := gotron.GenerateMnemonic(128) // 12 words
addr, err := gotron.AddressFromMnemonic(mnemonic, "")
if err != nil {
    log.Fatal(err)
}
```

### Transfer TRX

```go
import (
    "github.com/shopspring/decimal"
    "github.com/sxwebdev/gotron"
)

client, _ := gotron.NewClient(gotron.Mainnet)
defer client.Close()

// Transfer 1 TRX (1,000,000 SUN)
txID, err := client.Transfer(
    "TFromAddress",
    "TToAddress",
    decimal.NewFromInt(1000000),
    "your-private-key-hex",
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Transaction ID: %s\n", txID)
```

### Transfer TRC20 Tokens

```go
// Transfer USDT (or any TRC20 token)
txID, err := client.TransferTRC20(
    "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // USDT contract address
    "TFromAddress",
    "TToAddress",
    decimal.NewFromInt(1000000), // 1 USDT (6 decimals)
    "your-private-key-hex",
    100000000, // Fee limit: 100 TRX in SUN
)
```

### Delegate Resources

```go
// Delegate energy to another address
txID, err := client.DelegateResource(
    "TFromAddress",
    "TToAddress",
    decimal.NewFromInt(1000000000), // 1000 TRX in SUN
    gotron.Energy,
    "your-private-key-hex",
    false, // lock
    0,     // lock period in seconds
)
```

### Undelegate Resources

```go
// Undelegate energy from an address
txID, err := client.UndelegateResource(
    "TFromAddress",
    "TToAddress",
    decimal.NewFromInt(1000000000), // 1000 TRX in SUN
    gotron.Energy,
    "your-private-key-hex",
)
```

## Custom Configuration

You can create a client with custom configuration:

```go
import (
    "time"
    "github.com/sxwebdev/gotron"
    "github.com/sxwebdev/gotron/pkg/client"
)

cfg := &client.Config{
    GRPCAddress: "grpc.trongrid.io:50051",
    UseTLS:      false,
    Timeout:     30 * time.Second,
    APIKey:      "your-api-key", // Optional
}

client, err := gotron.NewClientWithConfig(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## Networks

The library supports three networks:

- `gotron.Mainnet` - Tron mainnet (grpc.trongrid.io:50051)
- `gotron.Testnet` - Shasta testnet (grpc.shasta.trongrid.io:50051)
- `gotron.Nile` - Nile testnet (grpc.nile.trongrid.io:50051)

## Resource Types

When delegating/undelegating resources:

- `gotron.Bandwidth` - Bandwidth resource
- `gotron.Energy` - Energy resource

## Package Structure

The library is organized into the following packages:

- **Root package** - High-level client API for easy integration
- `pkg/address` - Address generation, validation, and key management
- `pkg/client` - Low-level gRPC client with custom configuration
- `pkg/transaction` - Transaction creation and signing
- `pkg/resources` - Resource delegation and management
- `pkg/trc20` - TRC20 token operations
- `pkg/crypto` - Cryptographic operations

## Usage as a Module

This library is designed to be used as a Go module in your projects:

```bash
go get github.com/sxwebdev/gotron
```

Then import and use it:

```go
import "github.com/sxwebdev/gotron"

// Use the high-level API
client, _ := gotron.NewClient(gotron.Mainnet)
defer client.Close()
```

## Examples

See the `examples/` directory for more comprehensive examples:

- `examples/basic/` - Basic usage examples

## Current Status

✅ **Fully Implemented:**

- High-level client API
- Address generation and validation
- BIP39 mnemonic support
- Custom client configuration
- Parameter validation for all operations
- Type definitions for transactions, resources, and TRC20

⚠️ **In Progress:**

- Proto integration for transaction broadcasting
- gRPC method implementations for balance queries
- TRC20 contract interaction
- Resource delegation execution

The library provides a clean, high-level API similar to other blockchain SDKs, making it easy to integrate Tron functionality into your Go applications without dealing with low-level proto definitions.

## Dependencies

- `github.com/sxwebdev/go-bip39` - BIP39 mnemonic generation
- `github.com/shopspring/decimal` - Decimal arithmetic
- `github.com/mr-tron/base58` - Base58 encoding
- `github.com/ethereum/go-ethereum/crypto` - Cryptographic operations
- `github.com/btcsuite/btcd` - HD key derivation
- `google.golang.org/grpc` - gRPC client

## Documentation

For detailed documentation, visit [pkg.go.dev](https://pkg.go.dev/github.com/sxwebdev/gotron).

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
