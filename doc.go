// Package gotron provides a comprehensive SDK for interacting with the Tron blockchain.
//
// This package includes functionality for:
//   - Address generation from mnemonic phrases using BIP39/BIP44
//   - Transaction creation, signing, and broadcasting
//   - TRC20 token operations
//   - Resource delegation and management
//   - Balance queries and account information
//
// Basic Usage:
//
//	// Generate a new address from mnemonic
//	mnemonic, err := address.GenerateMnemonic(128)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	addr, err := address.FromMnemonic(mnemonic, "")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create a client
//	cfg := &client.Config{
//		GRPCAddress: "grpc.trongrid.io:50051",
//		UseTLS:      false,
//	}
//	c, err := client.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer c.Close()
//
//	// Get balance
//	balance, err := c.GetBalance(addr.Address)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// For more examples and detailed documentation, see the package subdirectories.
package gotron
