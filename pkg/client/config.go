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
	// Default: grpc
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

	// Headers are custom headers/metadata for requests (HTTP headers and gRPC metadata)
	Headers map[string]string
}

// Config holds the configuration for the Tron client
type Config struct {
	// Nodes is a list of node configurations
	// Round-robin load balancing is used across all nodes
	Nodes []NodeConfig

	// Network specifies the Tron network type (informational only)
	Network Network
}

// Validate validates the client configuration
func (c Config) Validate() error {
	if len(c.Nodes) == 0 {
		return fmt.Errorf("%w: at least one node is required", ErrInvalidConfig)
	}

	for i, node := range c.Nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("%w: node %d: %v", ErrInvalidConfig, i, err)
		}
	}

	return nil
}

// Validate validates the node configuration
func (n NodeConfig) Validate() error {
	if n.Address == "" {
		return fmt.Errorf("address is required")
	}

	switch n.GetProtocol() {
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
