package client

import (
	"fmt"

	"google.golang.org/grpc"
)

// Config holds the configuration for the Tron client
type Config struct {
	// GRPCAddress is the address of the Tron node gRPC endpoint
	GRPCAddress string
	// UseTLS enables TLS for the gRPC connection
	UseTLS bool
	// Network specifies the Tron network type
	Network Network
	// DialOptions are additional gRPC dial options
	DialOptions []grpc.DialOption
}

// Validate validates the client configuration
func (c Config) Validate() error {
	if c.GRPCAddress == "" {
		return fmt.Errorf("%w: gRPC address is required", ErrInvalidConfig)
	}

	return nil
}
