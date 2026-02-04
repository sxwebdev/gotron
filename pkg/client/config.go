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

// NodeConfig represents configuration for a single node
type NodeConfig struct {
	// Protocol specifies the transport protocol (grpc or http)
	Protocol Protocol

	// Address is the node address
	// For gRPC: "grpc.trongrid.io:50051"
	// For HTTP: "https://api.trongrid.io"
	Address string

	// UseTLS enables TLS for gRPC connections
	UseTLS bool

	// DialOptions are additional gRPC dial options (gRPC only)
	DialOptions []grpc.DialOption

	// HTTPClient allows providing a custom HTTP client (HTTP only)
	HTTPClient *http.Client

	// HTTPHeaders are custom headers for HTTP requests (HTTP only)
	HTTPHeaders map[string]string
}

// Config holds the configuration for the Tron client
type Config struct {
	// Nodes is a list of node configurations for multi-node support
	// When multiple nodes are provided, round-robin load balancing is used
	// This takes precedence over single node configuration fields below
	Nodes []NodeConfig

	// Protocol specifies the transport protocol (grpc or http)
	// Default: grpc (for backward compatibility)
	// Used when Nodes is empty
	Protocol Protocol

	// GRPCAddress is the address of the Tron node gRPC endpoint
	// Used when Protocol is ProtocolGRPC and Nodes is empty
	GRPCAddress string

	// HTTPAddress is the base URL of the Tron node HTTP endpoint
	// Used when Protocol is ProtocolHTTP and Nodes is empty
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

// HasMultipleNodes returns true if multiple nodes are configured
func (c Config) HasMultipleNodes() bool {
	return len(c.Nodes) > 1
}

// Validate validates the client configuration
func (c Config) Validate() error {
	// If Nodes is provided, validate each node
	if len(c.Nodes) > 0 {
		for i, node := range c.Nodes {
			if err := node.Validate(); err != nil {
				return fmt.Errorf("%w: node %d: %v", ErrInvalidConfig, i, err)
			}
		}
		return nil
	}

	// Validate single node configuration
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

// Validate validates the node configuration
func (n NodeConfig) Validate() error {
	if n.Address == "" {
		return fmt.Errorf("address is required")
	}

	protocol := n.Protocol
	if protocol == "" {
		protocol = ProtocolGRPC
	}

	switch protocol {
	case ProtocolGRPC, ProtocolHTTP:
		// valid protocols
	default:
		return fmt.Errorf("invalid protocol %s", n.Protocol)
	}

	return nil
}

// GetProtocol returns the node protocol, defaulting to gRPC
func (n NodeConfig) GetProtocol() Protocol {
	if n.Protocol == "" {
		return ProtocolGRPC
	}
	return n.Protocol
}
