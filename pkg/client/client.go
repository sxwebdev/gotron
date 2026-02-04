// Package client provides a client for interacting with Tron nodes via gRPC or HTTP.
package client

import (
	"fmt"
)

// Client represents a Tron blockchain client
type Client struct {
	transport Transport
	config    Config
}

// New creates a new Tron client with the given configuration
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	transports := make([]Transport, 0, len(cfg.Nodes))
	for i, nodeCfg := range cfg.Nodes {
		transport, err := createTransportFromNode(nodeCfg)
		if err != nil {
			// Close already created transports on error
			for _, t := range transports {
				t.Close()
			}
			return nil, fmt.Errorf("failed to create transport for node %d: %w", i, err)
		}
		transports = append(transports, transport)
	}

	return &Client{
		config:    cfg,
		transport: NewRoundRobinTransport(transports),
	}, nil
}

// createTransportFromNode creates a Transport from NodeConfig
func createTransportFromNode(nodeCfg NodeConfig) (Transport, error) {
	switch nodeCfg.GetProtocol() {
	case ProtocolGRPC:
		return NewGRPCTransport(nodeCfg)

	case ProtocolHTTP:
		return NewHTTPTransport(nodeCfg)

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", nodeCfg.Protocol)
	}
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}

// GetNetwork returns the current network based on configuration
func (c *Client) GetNetwork() Network {
	return c.config.Network
}
