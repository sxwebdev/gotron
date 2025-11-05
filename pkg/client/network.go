package client

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
