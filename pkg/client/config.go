package client

import (
	"fmt"
	"strings"
)

// Config holds the configuration for the Tron client
type Config struct {
	// GRPCAddress is the address of the Tron node gRPC endpoint
	GRPCAddress string
	// UseTLS enables TLS for the gRPC connection
	UseTLS bool
	// APIKey is an optional API key for TronGrid
	APIKey string
	// Network specifies the Tron network type
	Network Network
}

// Validate validates the client configuration
func (c Config) Validate() error {
	if c.GRPCAddress == "" {
		return fmt.Errorf("%w: gRPC address is required", ErrInvalidConfig)
	}

	if strings.Contains(c.GRPCAddress, "trongrid") && c.APIKey == "" {
		return fmt.Errorf("%w: API key is required for TronGrid", ErrInvalidConfig)
	}

	return nil
}
