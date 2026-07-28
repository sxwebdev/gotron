package gotron

import (
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/units"
)

// Network types for Tron blockchain
const (
	// Mainnet is the Tron mainnet
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
	// SunPerTRX is the fixed scale of TRX: 1 TRX = 1,000,000 SUN
	SunPerTRX = units.SunPerTRX
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

// Amount types. Every TRX-denominated value in the SDK is a SUN and every TRC20
// amount is a TokenAmount, so the unit is always part of the signature.
type (
	// SUN is an amount of TRX in the chain's native unit: 1 TRX = 1,000,000 SUN.
	SUN = client.SUN
	// TokenAmount is an amount of a TRC20 token in the token's minimal units.
	TokenAmount = client.TokenAmount
)

// Amount constructors, re-exported for convenience.
var (
	// FromTRX converts a TRX amount to SUN, rejecting values that cannot be
	// represented exactly.
	FromTRX = client.FromTRX
	// MustFromTRX is FromTRX for constants and tests; it panics on bad input.
	MustFromTRX = client.MustFromTRX
	// FromTokenUnits wraps a raw minimal-unit token amount.
	FromTokenUnits = client.FromTokenUnits
	// FromTokenDecimal converts a human-facing token amount using the token's decimals.
	FromTokenDecimal = client.FromTokenDecimal
)

// New creates a new Tron client with the specified configuration.
//
// Example:
//
//	cfg := gotron.Config{
//	    Nodes: []client.NodeConfig{
//	        {
//	            Protocol: client.ProtocolGRPC,
//	            Address:  "grpc.trongrid.io:50051",
//	            UseTLS:   true,
//	        },
//	    },
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
