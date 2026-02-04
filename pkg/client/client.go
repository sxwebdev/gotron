// Package client provides a client for interacting with Tron nodes via gRPC or HTTP.
package client

import (
	"fmt"

	"github.com/sxwebdev/gotron/schema/pb/api"
)

// Client represents a Tron blockchain client
type Client struct {
	transport Transport
	config    Config

	// walletClient is kept for backward compatibility with API() method
	// Only available when using single gRPC transport
	walletClient api.WalletClient
}

// New creates a new Tron client with the given configuration
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		config: cfg,
	}

	// Handle multiple nodes configuration
	if len(cfg.Nodes) > 0 {
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

		if len(transports) == 1 {
			// Single node - use it directly
			client.transport = transports[0]
			// For backward compatibility with single gRPC node
			if grpcTransport, ok := transports[0].(*GRPCTransport); ok {
				client.walletClient = grpcTransport.WalletClient()
			}
		} else {
			// Multiple nodes - use round-robin
			client.transport = NewRoundRobinTransport(transports)
		}

		return client, nil
	}

	// Handle legacy single node configuration
	var err error

	switch cfg.GetProtocol() {
	case ProtocolGRPC:
		grpcTransport, err := NewGRPCTransport(cfg)
		if err != nil {
			return nil, err
		}
		client.transport = grpcTransport
		// For backward compatibility, expose walletClient
		client.walletClient = grpcTransport.WalletClient()

	case ProtocolHTTP:
		client.transport, err = NewHTTPTransport(cfg)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("%w: unsupported protocol %s", ErrInvalidConfig, cfg.Protocol)
	}

	return client, nil
}

// createTransportFromNode creates a Transport from NodeConfig
func createTransportFromNode(nodeCfg NodeConfig) (Transport, error) {
	switch nodeCfg.GetProtocol() {
	case ProtocolGRPC:
		// Convert NodeConfig to Config for gRPC transport
		cfg := Config{
			Protocol:    ProtocolGRPC,
			GRPCAddress: nodeCfg.Address,
			UseTLS:      nodeCfg.UseTLS,
			DialOptions: nodeCfg.DialOptions,
		}
		return NewGRPCTransport(cfg)

	case ProtocolHTTP:
		// Convert NodeConfig to Config for HTTP transport
		cfg := Config{
			Protocol:    ProtocolHTTP,
			HTTPAddress: nodeCfg.Address,
			HTTPClient:  nodeCfg.HTTPClient,
			HTTPHeaders: nodeCfg.HTTPHeaders,
		}
		return NewHTTPTransport(cfg)

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

// GetProtocol returns the current transport protocol
func (c *Client) GetProtocol() Protocol {
	return c.config.GetProtocol()
}

// API returns the underlying WalletClient API (gRPC only)
// Deprecated: This method only works with gRPC transport.
// Use the high-level Client methods instead for protocol-agnostic code.
func (c *Client) API() api.WalletClient {
	return c.walletClient
}
