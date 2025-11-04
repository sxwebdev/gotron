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
	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/resources"
	"github.com/sxwebdev/gotron/pkg/transaction"
	"github.com/sxwebdev/gotron/pkg/trc20"
)

// Resource types
const (
	Bandwidth = resources.Bandwidth
	Energy    = resources.Energy
)

// Tron is the high-level Tron blockchain client
type Tron struct {
	*Client
}

// New creates a new Tron client with custom configuration
func New(cfg *Config) (*Tron, error) {
	c, err := newClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Tron{Client: c}, nil
}

// Transfer sends TRX from one address to another
func (t *Tron) Transfer(from, to string, amount decimal.Decimal, privateKey string) (string, error) {
	params := transaction.TransferParams{
		From:       from,
		To:         to,
		Amount:     amount,
		PrivateKey: privateKey,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual transaction creation and broadcasting
	return "", nil
}

// TransferTRC20 transfers TRC20 tokens
func (t *Tron) TransferTRC20(contractAddress, from, to string, amount decimal.Decimal, privateKey string, feeLimit int64) (string, error) {
	params := trc20.TransferParams{
		ContractAddress: contractAddress,
		From:            from,
		To:              to,
		Amount:          amount,
		PrivateKey:      privateKey,
		FeeLimit:        feeLimit,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual TRC20 transfer
	return "", nil
}

// DelegateResource delegates bandwidth or energy to another address
func (t *Tron) DelegateResource(from, to string, balance decimal.Decimal, resourceType resources.ResourceType, privateKey string, lock bool, lockPeriod int64) (string, error) {
	params := resources.DelegateResourceParams{
		From:         from,
		To:           to,
		Balance:      balance,
		ResourceType: resourceType,
		PrivateKey:   privateKey,
		Lock:         lock,
		LockPeriod:   lockPeriod,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual resource delegation
	return "", nil
}

// UndelegateResource undelegates bandwidth or energy from an address
func (t *Tron) UndelegateResource(from, to string, balance decimal.Decimal, resourceType resources.ResourceType, privateKey string) (string, error) {
	params := resources.UndelegateResourceParams{
		From:         from,
		To:           to,
		Balance:      balance,
		ResourceType: resourceType,
		PrivateKey:   privateKey,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual resource undelegation
	return "", nil
}
