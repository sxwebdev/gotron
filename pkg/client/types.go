package client

import "fmt"

// Network represents the Tron network type
type Network string

const (
	// NetworkMainnet represents the Tron mainnet
	NetworkMainnet Network = "mainnet"
	// NetworkShasta represents the Tron testnet (Shasta)
	NetworkShasta Network = "shasta"
	// NetworkNile represents the Tron testnet (Nile)
	NetworkNile Network = "nile"
)

// Validate validates the network type
func (n Network) Validate() error {
	switch n {
	case NetworkMainnet, NetworkShasta, NetworkNile:
		return nil
	default:
		return fmt.Errorf("invalid network: %s", n)
	}
}

// String returns the string representation of the network type
func (n Network) String() string {
	return string(n)
}

// ResourceType represents the type of resource to delegate
type ResourceType int32

const (
	// ResourceTypeBandwidth represents bandwidth resource
	ResourceTypeBandwidth ResourceType = 0
	// ResourceTypeEnergy represents energy resource
	ResourceTypeEnergy ResourceType = 1
)

// Validate validates the resource type
func (r ResourceType) Validate() error {
	if r != ResourceTypeBandwidth && r != ResourceTypeEnergy {
		return fmt.Errorf("%w: must be Bandwidth or Energy", ErrInvalidResourceType)
	}
	return nil
}

// String returns the string representation of the resource type
func (r ResourceType) String() string {
	switch r {
	case ResourceTypeBandwidth:
		return "BANDWIDTH"
	case ResourceTypeEnergy:
		return "ENERGY"
	default:
		return "UNKNOWN"
	}
}
