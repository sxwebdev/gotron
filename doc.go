// Package gotron provides a comprehensive SDK for interacting with the Tron blockchain.
//
// This package offers a complete implementation of Tron's API with a clean,
// idiomatic Go interface. It supports both gRPC and HTTP REST API transports
// with round-robin load balancing across multiple nodes.
//
// # Features
//
//   - Dual transport support: gRPC and HTTP REST API
//   - Round-robin load balancing across multiple nodes
//   - Full client for Tron Wallet API
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
//	    "github.com/sxwebdev/gotron/pkg/client"
//	)
//
//	func main() {
//	    cfg := client.Config{
//	        Nodes: []client.NodeConfig{
//	            {
//	                Protocol: client.ProtocolGRPC,
//	                Address:  "grpc.trongrid.io:50051",
//	                UseTLS:   true,
//	            },
//	        },
//	    }
//
//	    tron, err := client.New(cfg)
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
// # Configuration
//
// The SDK uses a unified configuration with a list of nodes.
// Round-robin load balancing is always used across all configured nodes.
//
// Single gRPC node:
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        {
//	            Protocol: client.ProtocolGRPC,
//	            Address:  "grpc.trongrid.io:50051",
//	            UseTLS:   true,
//	        },
//	    },
//	}
//
// Single HTTP node:
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        {
//	            Protocol: client.ProtocolHTTP,
//	            Address:  "https://api.trongrid.io",
//	            HTTPHeaders: map[string]string{
//	                "TRON-PRO-API-KEY": "your-api-key",
//	            },
//	        },
//	    },
//	}
//
// Multiple nodes with mixed protocols:
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        {
//	            Protocol: client.ProtocolGRPC,
//	            Address:  "grpc.trongrid.io:50051",
//	            UseTLS:   true,
//	        },
//	        {
//	            Protocol: client.ProtocolHTTP,
//	            Address:  "https://api.trongrid.io",
//	            HTTPHeaders: map[string]string{
//	                "TRON-PRO-API-KEY": "your-api-key",
//	            },
//	        },
//	        {
//	            Protocol: client.ProtocolHTTP,
//	            Address:  "https://tron-rpc.publicnode.com",
//	        },
//	    },
//	}
//
// Each request uses the next node in round-robin fashion.
// Errors are returned as-is without retries.
//
// # Using TronGrid with API Key
//
// TronGrid requires an API key via the TRON-PRO-API-KEY header.
//
// For HTTP transport, use HTTPHeaders in NodeConfig:
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        {
//	            Protocol: client.ProtocolHTTP,
//	            Address:  "https://api.trongrid.io",
//	            HTTPHeaders: map[string]string{
//	                "TRON-PRO-API-KEY": "your-api-key",
//	            },
//	        },
//	    },
//	}
//
// For gRPC transport, use DialOptions with an interceptor:
//
//	import (
//	    "context"
//
//	    "github.com/sxwebdev/gotron/pkg/client"
//	    "google.golang.org/grpc"
//	    "google.golang.org/grpc/metadata"
//	)
//
//	func apiKeyInterceptor(apiKey string) grpc.UnaryClientInterceptor {
//	    return func(ctx context.Context, method string, req, reply interface{},
//	        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
//
//	        ctx = metadata.AppendToOutgoingContext(ctx, "TRON-PRO-API-KEY", apiKey)
//	        return invoker(ctx, method, req, reply, cc, opts...)
//	    }
//	}
//
//	func main() {
//	    cfg := client.Config{
//	        Nodes: []client.NodeConfig{
//	            {
//	                Protocol: client.ProtocolGRPC,
//	                Address:  "grpc.trongrid.io:50051",
//	                UseTLS:   true,
//	                DialOptions: []grpc.DialOption{
//	                    grpc.WithUnaryInterceptor(apiKeyInterceptor("your-api-key")),
//	                },
//	            },
//	        },
//	    }
//
//	    tron, err := client.New(cfg)
//	    // Use client...
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
// # Package Organization
//
// The SDK is organized into several packages:
//
//   - gotron: High-level wrapper and convenience functions
//   - pkg/client: Core client with gRPC and HTTP transport implementations
//   - pkg/address: Address generation, validation, and key management
//   - pkg/tronutils: Utility functions for encoding, formatting, etc.
//   - schema/pb: Protocol buffer definitions from Tron protocol
//
// The client package uses a Transport interface pattern with round-robin
// load balancing across all configured nodes.
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
