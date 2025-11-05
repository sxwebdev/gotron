// Package gotron provides a comprehensive SDK for interacting with the Tron blockchain.
//
// This package offers a complete implementation of Tron's gRPC API with a clean,
// idiomatic Go interface. It handles address generation, transaction creation and signing,
// TRC20 token operations, resource management, and blockchain queries.
//
// # Features
//
//   - Full gRPC client for Tron Wallet API
//   - BIP39/BIP44 mnemonic and address generation
//   - Transaction creation, signing, and broadcasting
//   - TRC20 token support (transfer, balance, metadata)
//   - Resource delegation (bandwidth and energy)
//   - Account operations and activation
//   - Block and transaction queries
//   - Multi-network support (Mainnet, Shasta, Nile)
//   - Type-safe operations with validation
//
// # Quick Start
//
// Create a client and query an account balance:
//
//	import (
//	    "context"
//	    "fmt"
//	    "log"
//
//	    "github.com/sxwebdev/gotron"
//	)
//
//	func main() {
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
//	    balance, err := tron.GetAccountBalance(ctx, "TYourAddress")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    fmt.Printf("Balance: %s TRX\n", balance.String())
//	}
//
// # Address Generation
//
// Generate addresses from mnemonic phrases using BIP39/BIP44:
//
//	import "github.com/sxwebdev/gotron/pkg/address"
//
//	// Generate a new 12-word mnemonic
//	mnemonic, err := address.GenerateMnemonic(128)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Derive address at index 0 (m/44'/195'/0'/0/0)
//	addr, err := address.FromMnemonic(mnemonic, "", 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Address: %s\n", addr.Address)
//	fmt.Printf("Private Key: %s\n", addr.PrivateKey)
//
// # Transactions
//
// Create and broadcast a TRX transfer:
//
//	import (
//	    "github.com/shopspring/decimal"
//	    "github.com/sxwebdev/gotron/pkg/address"
//	)
//
//	ctx := context.Background()
//
//	// Create transaction (amount in TRX)
//	tx, err := tron.CreateTransferTransaction(
//	    ctx,
//	    "TFromAddress",
//	    "TToAddress",
//	    decimal.NewFromFloat(1.5), // 1.5 TRX
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Sign with private key
//	privateKey, _ := address.PrivateKeyFromHex("your-hex-private-key")
//	err = tron.SignTransaction(tx.Transaction, privateKey)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Broadcast to network
//	result, err := tron.BroadcastTransaction(ctx, tx.Transaction)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # TRC20 Tokens
//
// Work with TRC20 tokens:
//
//	const usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
//
//	// Get token info
//	name, _ := tron.TRC20GetName(ctx, usdtContract)
//	symbol, _ := tron.TRC20GetSymbol(ctx, usdtContract)
//	decimals, _ := tron.TRC20GetDecimals(ctx, usdtContract)
//
//	// Get balance
//	balance, err := tron.TRC20ContractBalance(ctx, "TAddress", usdtContract)
//
//	// Transfer tokens
//	tx, err := tron.TRC20Send(
//	    ctx,
//	    "TFromAddress",
//	    "TToAddress",
//	    usdtContract,
//	    decimal.NewFromInt(1000000), // Amount in smallest unit
//	    100_000_000,                  // Fee limit in SUN
//	)
//
// # Resource Management
//
// Delegate and reclaim resources:
//
//	// Delegate energy
//	tx, err := tron.DelegateResource(
//	    ctx,
//	    "TOwnerAddress",
//	    "TReceiverAddress",
//	    gotron.Energy,
//	    1000_000_000, // 1000 TRX in SUN
//	    false,         // lock
//	    0,             // lock period
//	)
//
//	// Reclaim resources
//	tx, err = tron.ReclaimResource(
//	    ctx,
//	    "TOwnerAddress",
//	    "TReceiverAddress",
//	    gotron.Energy,
//	    1000_000_000,
//	)
//
// # Network Configuration
//
// The package supports multiple networks:
//
//	// Mainnet
//	cfg := gotron.Config{Network: gotron.Mainnet, APIKey: "key"}
//
//	// Shasta Testnet
//	cfg := gotron.Config{Network: gotron.Shasta, APIKey: "key"}
//
//	// Nile Testnet
//	cfg := gotron.Config{Network: gotron.Nile, APIKey: "key"}
//
//	// Custom node
//	cfg := client.Config{
//	    GRPCAddress: "custom-node:50051",
//	    UseTLS:      true,
//	    Network:     client.NetworkMainnet,
//	}
//
// # Package Organization
//
// The SDK is organized into several packages:
//
//   - gotron: High-level wrapper and convenience functions
//   - pkg/client: Core gRPC client with all blockchain operations
//   - pkg/address: Address generation, validation, and key management
//   - pkg/utils: Utility functions for encoding, formatting, etc.
//   - schema/pb: Protocol buffer definitions from Tron protocol
//
// # Error Handling
//
// The package provides typed errors for common scenarios:
//
//	import "github.com/sxwebdev/gotron/pkg/client"
//
//	_, err := tron.GetAccount(ctx, "address")
//	if errors.Is(err, client.ErrAccountNotFound) {
//	    // Handle account not found
//	}
//
// Common errors:
//   - client.ErrAccountNotFound
//   - client.ErrInvalidAddress
//   - client.ErrInvalidAmount
//   - client.ErrTransactionNotFound
//   - client.ErrInvalidConfig
//
// # Advanced Usage
//
// For advanced use cases, access the underlying gRPC client directly:
//
//	// Get the raw Wallet API client
//	walletAPI := tron.API()
//
//	// Use any Wallet API method
//	nodeInfo, err := walletAPI.GetNodeInfo(ctx, &api.EmptyMessage{})
//
// Generate multiple addresses from a single mnemonic:
//
//	generator := address.NewAddressGenerator(mnemonic, "")
//	for i := uint32(0); i < 10; i++ {
//	    addr, err := generator.Generate(i)
//	    // Use addr...
//	}
//
// # Documentation
//
// For detailed API documentation, visit https://pkg.go.dev/github.com/sxwebdev/gotron
package gotron
