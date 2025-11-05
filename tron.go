// Package gotron provides a high-level client for interacting with the Tron blockchain.
//
// # Quick Start
//
// Create a client and get account balance:
//
//	import (
//	    "context"
//	    "fmt"
//	    "log"
//
//	    "github.com/sxwebdev/gotron"
//	    "github.com/shopspring/decimal"
//	)
//
//	func main() {
//	    // Initialize client
//	    cfg := gotron.Config{
//	        Network: gotron.Mainnet,
//	        APIKey:  "your-trongrid-api-key",
//	    }
//
//	    tron, err := gotron.New(cfg)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer tron.Close()
//
//	    ctx := context.Background()
//
//	    // Get balance
//	    balance, err := tron.GetAccountBalance(ctx, "TYourAddress")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("Balance: %s TRX\n", balance.String())
//	}
//
// # Transfer TRX
//
//	import "github.com/sxwebdev/gotron/pkg/address"
//
//	// Create transfer transaction
//	tx, err := tron.CreateTransferTransaction(
//	    ctx,
//	    "TFromAddress",
//	    "TToAddress",
//	    decimal.NewFromFloat(1.5), // Amount in TRX
//	)
//
//	// Sign transaction
//	privateKey, _ := address.PrivateKeyFromHex("your-private-key")
//	tron.SignTransaction(tx.Transaction, privateKey)
//
//	// Broadcast
//	result, err := tron.BroadcastTransaction(ctx, tx.Transaction)
//
// # TRC20 Tokens
//
//	// Transfer USDT
//	const usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
//
//	tx, err := tron.TRC20Send(
//	    ctx,
//	    "TFromAddress",
//	    "TToAddress",
//	    usdtContract,
//	    decimal.NewFromInt(1000000), // 1 USDT (6 decimals)
//	    100_000_000,                  // Fee limit in SUN
//	)
//
// See package documentation for more examples and details.
package gotron

import (
	"github.com/sxwebdev/gotron/pkg/client"
)

// Network types for Tron blockchain
const (
	// Mainnet is the Tron mainnet (grpc.trongrid.io:50051)
	Mainnet = client.NetworkMainnet
	// Shasta is the Tron Shasta testnet (grpc.shasta.trongrid.io:50051)
	Shasta = client.NetworkShasta
	// Nile is the Tron Nile testnet (grpc.nile.trongrid.io:50051)
	Nile = client.NetworkNile
)

// Resource types for delegation operations
const (
	// Bandwidth represents bandwidth resource type
	Bandwidth = client.ResourceTypeBandwidth
	// Energy represents energy resource type
	Energy = client.ResourceTypeEnergy
)

// Blockchain constants
const (
	// TrxDecimals is the number of decimals for TRX (1 TRX = 1,000,000 SUN)
	TrxDecimals = client.TrxDecimals
	// Trc20TransferEventSignature is the event signature for TRC20 transfers
	Trc20TransferEventSignature = client.Trc20TransferEventSignature
)

// Tron is the high-level Tron blockchain client.
// It wraps the underlying gRPC client and provides convenient methods
// for common blockchain operations.
type Tron struct {
	*client.Client
}

// Config is an alias for client.Config for convenience
type Config = client.Config

// New creates a new Tron client with the specified configuration.
//
// Example:
//
//	cfg := gotron.Config{
//	    Network: gotron.Mainnet,
//	    APIKey:  "your-api-key",
//	}
//	tron, err := gotron.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tron.Close()
func New(cfg Config) (*Tron, error) {
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Tron{Client: c}, nil
}
