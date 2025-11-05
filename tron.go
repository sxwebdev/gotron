// Package gotron provides a high-level client for interacting with the Tron blockchain.
//
// Example usage:
//
//	package main
//
//	import (
//	    "fmt"
//	    "log"
//	    "github.com/sxwebdev/gotron"
//	    "github.com/shopspring/decimal"
//	)
//
//	func main() {
//	    // Create client for mainnet
//	    client, err := gotron.NewClient(gotron.Mainnet)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer client.Close()
//
//	    // Get balance
//	    balance, err := client.GetBalance("TYourAddress")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("Balance: %s TRX\n", balance)
//
//	    // Transfer TRX
//	    txID, err := client.Transfer("TFromAddress", "TToAddress",
//	        decimal.NewFromInt(1000000), "privatekey")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("Transaction ID: %s\n", txID)
//	}
package gotron

import (
	"github.com/sxwebdev/gotron/pkg/client"
)

// Network types
const (
	Mainnet = client.NetworkMainnet
	Shasta  = client.NetworkShasta
	Nile    = client.NetworkNile
)

// Resource types
const (
	Bandwidth = client.ResourceTypeBandwidth
	Energy    = client.ResourceTypeEnergy
)

// Constants
const (
	TrxDecimals                 = client.TrxDecimals
	Trc20TransferEventSignature = client.Trc20TransferEventSignature
)

// Tron is the high-level Tron blockchain client
type Tron struct {
	*client.Client
}

// New creates a new Tron client with custom configuration
func New(cfg client.Config) (*Tron, error) {
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Tron{Client: c}, nil
}
