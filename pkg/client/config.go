package client

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc"
)

// Protocol represents the transport protocol type
type Protocol string

const (
	// ProtocolGRPC uses gRPC transport (default)
	ProtocolGRPC Protocol = "grpc"
	// ProtocolHTTP uses HTTP REST API transport
	ProtocolHTTP Protocol = "http"
)

// Config holds the configuration for the Tron client
type Config struct {
	// Protocol specifies the transport protocol (grpc or http)
	// Default: grpc (for backward compatibility)
	Protocol Protocol

	// GRPCAddress is the address of the Tron node gRPC endpoint
	// Used when Protocol is ProtocolGRPC
	GRPCAddress string

	// HTTPAddress is the base URL of the Tron node HTTP endpoint
	// Used when Protocol is ProtocolHTTP
	// Examples: https://api.trongrid.io, https://api.shasta.trongrid.io
	HTTPAddress string

	// UseTLS enables TLS for the gRPC connection
	UseTLS bool

	// Network specifies the Tron network type
	Network Network

	// DialOptions are additional gRPC dial options
	DialOptions []grpc.DialOption

	// HTTPClient allows providing a custom HTTP client
	// Used when Protocol is ProtocolHTTP
	HTTPClient *http.Client

	// HTTPHeaders are custom headers to add to every HTTP request
	// Used when Protocol is ProtocolHTTP
	// Example: map[string]string{"TRON-PRO-API-KEY": "your-api-key"}
	HTTPHeaders map[string]string
}

// GetProtocol returns the configured protocol, defaulting to gRPC
func (c Config) GetProtocol() Protocol {
	if c.Protocol == "" {
		return ProtocolGRPC
	}
	return c.Protocol
}

// Validate validates the client configuration
func (c Config) Validate() error {
	switch c.GetProtocol() {
	case ProtocolGRPC:
		if c.GRPCAddress == "" {
			return fmt.Errorf("%w: gRPC address is required", ErrInvalidConfig)
		}
	case ProtocolHTTP:
		if c.HTTPAddress == "" {
			return fmt.Errorf("%w: HTTP address is required", ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: invalid protocol %s", ErrInvalidConfig, c.Protocol)
	}

	return nil
}
