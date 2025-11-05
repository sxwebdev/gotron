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
