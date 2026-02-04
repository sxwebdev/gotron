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
	// Only available when using gRPC transport
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
