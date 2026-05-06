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
//   - Tier-based fallback: primary/fallback node groups via NodeConfig.Tier
//   - Background health checking with automatic node recovery
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
//	            Headers: map[string]string{
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
//	            Headers: map[string]string{
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
// Each request uses the next node in round-robin fashion. By default a
// background health-checker probes every node and excludes unhealthy ones
// from selection; failed live requests count toward the per-node failure
// threshold. The error of the failed call is still returned to the caller —
// there is no automatic retry on the same request, the next request just
// goes to the next healthy node.
//
// # Tier-based Fallback with Health Checking
//
// Nodes can be partitioned into priority tiers via NodeConfig.Tier (0 = primary,
// 1 = fallback, 2+ = next). Requests are routed to the lowest-numbered tier
// that still has at least one healthy node; a higher tier is used only when
// every node of every lower tier is currently unhealthy. As soon as one
// primary recovers, traffic returns to it.
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        // Primary group
//	        {Protocol: client.ProtocolGRPC, Address: "grpc.trongrid.io:50051", UseTLS: true, Tier: 0},
//	        {Protocol: client.ProtocolHTTP, Address: "https://api.trongrid.io",  Tier: 0},
//	        // Fallback group — only used when every primary is unhealthy
//	        {Protocol: client.ProtocolHTTP, Address: "https://tron-rpc.publicnode.com", Tier: 1},
//	    },
//	    Health: client.HealthConfig{
//	        FailureThreshold:     2,
//	        SuccessThreshold:     2,
//	        HealthyInterval:      30 * time.Second, // active tier
//	        UnhealthyInterval:    5 * time.Second,  // any unhealthy node
//	        InactiveTierInterval: 5 * time.Minute,  // healthy fallbacks
//	        ProbeTimeout:         5 * time.Second,
//	    },
//	}
//
// When every node of every tier is unhealthy, calls return ErrNoHealthyNodes —
// detect with errors.Is(err, client.ErrNoHealthyNodes). The health-checker
// keeps probing all nodes (at UnhealthyInterval) and they re-enter the pool
// the moment they recover.
//
// To disable the health-checker entirely (legacy round-robin without health),
// set Health.Disabled = true. NodeConfig.Tier is then ignored.
//
// # Using TronGrid with API Key
//
// TronGrid requires an API key via the TRON-PRO-API-KEY header.
// Use the Headers field in NodeConfig - it works for both HTTP and gRPC transports:
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        {
//	            Protocol: client.ProtocolGRPC,
//	            Address:  "grpc.trongrid.io:50051",
//	            UseTLS:   true,
//	            Headers: map[string]string{
//	                "TRON-PRO-API-KEY": "your-api-key",
//	            },
//	        },
//	    },
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
// The client package uses a Transport interface pattern. The default
// transport stack is HealthAwareTransport (tier-based fallback + per-node
// health checking) wrapped by an optional MetricsTransport. Setting
// Config.Health.Disabled = true falls back to a plain RoundRobinTransport.
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
//   - client.ErrNoHealthyNodes
//
// # Prometheus Metrics
//
// The SDK supports optional metrics for monitoring RPC performance.
// Use the built-in Prometheus implementation or provide a custom MetricsCollector.
//
// Built-in Prometheus metrics:
//
//	metrics := client.NewMetrics(prometheus.DefaultRegisterer)
//
//	cfg := client.Config{
//	    Nodes: []client.NodeConfig{
//	        {Address: "grpc.trongrid.io:50051", UseTLS: true},
//	    },
//	    Blockchain: "tron",
//	    Metrics:    metrics,
//	}
//
//	tron, err := client.New(cfg)
//
// Available built-in metrics:
//   - gotron_rpc_requests_total: Counter with labels blockchain, method, status
//   - gotron_rpc_duration_seconds: Histogram with labels blockchain, method
//   - gotron_rpc_in_flight: Gauge for current active requests
//   - gotron_rpc_retries_total: Counter with labels blockchain, method
//   - gotron_rpc_pool_total: Gauge with label blockchain
//   - gotron_rpc_pool_healthy: Gauge with label blockchain
//   - gotron_rpc_pool_disabled: Gauge with label blockchain
//
// The pool_* gauges are kept up to date by HealthAwareTransport on every
// node state transition (and once at construction).
//
// Custom MetricsCollector:
//
//	type myMetrics struct{}
//	func (m *myMetrics) RecordRequest(blockchain, method, status string, duration time.Duration) { ... }
//	func (m *myMetrics) RecordRetry(blockchain, method string) { ... }
//	func (m *myMetrics) SetPoolHealth(blockchain string, total, healthy, disabled int) { ... }
//
//	cfg := client.Config{
//	    Nodes:      nodes,
//	    Blockchain: "tron",
//	    Metrics:    &myMetrics{},
//	}
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
