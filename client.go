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
	"context"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/address"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/resources"
	"github.com/sxwebdev/gotron/pkg/transaction"
	"github.com/sxwebdev/gotron/pkg/trc20"
)

// Network types
const (
	Mainnet = client.Mainnet
	Testnet = client.Testnet
	Nile    = client.Nile
)

// Resource types
const (
	Bandwidth = resources.Bandwidth
	Energy    = resources.Energy
)

// Client is the high-level Tron blockchain client
type Client struct {
	*client.Client
}

// NewClient creates a new Tron client for the specified network
func NewClient(ctx context.Context, network client.Network) (*Client, error) {
	c, err := client.NewClient(ctx, network)
	if err != nil {
		return nil, err
	}

	return &Client{Client: c}, nil
}

// NewClientWithConfig creates a new Tron client with custom configuration
func NewClientWithConfig(ctx context.Context, cfg *client.Config) (*Client, error) {
	c, err := client.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Client{Client: c}, nil
}

// Transfer sends TRX from one address to another
func (c *Client) Transfer(from, to string, amount decimal.Decimal, privateKey string) (string, error) {
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
func (c *Client) TransferTRC20(contractAddress, from, to string, amount decimal.Decimal, privateKey string, feeLimit int64) (string, error) {
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
func (c *Client) DelegateResource(from, to string, balance decimal.Decimal, resourceType resources.ResourceType, privateKey string, lock bool, lockPeriod int64) (string, error) {
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
func (c *Client) UndelegateResource(from, to string, balance decimal.Decimal, resourceType resources.ResourceType, privateKey string) (string, error) {
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

// Address operations

// GenerateMnemonic generates a new BIP39 mnemonic
func GenerateMnemonic(strength int) (string, error) {
	return address.GenerateMnemonic(strength)
}

// GenerateAddress creates a new random Tron address
func GenerateAddress() (*address.Address, error) {
	return address.Generate()
}

// AddressFromMnemonic creates a Tron address from a mnemonic and passphrase
func AddressFromMnemonic(mnemonic, passphrase string, index uint32) (*address.Address, error) {
	return address.FromMnemonic(mnemonic, passphrase, index)
}

// AddressFromPrivateKey creates a Tron address from a private key
func AddressFromPrivateKey(privateKey string) (*address.Address, error) {
	return address.FromPrivateKey(privateKey)
}

// ValidateAddress validates a Tron address
func ValidateAddress(addr string) error {
	return address.Validate(addr)
}
