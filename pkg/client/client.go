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

// New creates a new Tron client with the given configuration.
//
// By default, requests are routed by HealthAwareTransport: nodes are grouped
// by NodeConfig.Tier, the lowest-numbered tier with at least one healthy node
// is used, and a background health-checker tracks every node's status. To
// restore the legacy plain round-robin (no health-checking, NodeConfig.Tier
// ignored), set cfg.Health.Disabled = true.
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	blockchain := cfg.Blockchain
	if blockchain == "" {
		blockchain = "tron"
	}

	var transport Transport
	if cfg.Health.Disabled {
		transports := make([]Transport, 0, len(cfg.Nodes))
		for i, nodeCfg := range cfg.Nodes {
			t, err := createTransportFromNode(nodeCfg)
			if err != nil {
				for _, x := range transports {
					_ = x.Close()
				}
				return nil, fmt.Errorf("failed to create transport for node %d: %w", i, err)
			}
			transports = append(transports, t)
		}
		transport = NewRoundRobinTransport(transports)
	} else {
		ht, err := NewHealthAwareTransport(cfg.Nodes, createTransportFromNode, cfg.Health, cfg.Metrics, blockchain)
		if err != nil {
			return nil, err
		}
		transport = ht
	}

	// Wrap with metrics transport if metrics are configured
	if cfg.Metrics != nil {
		transport = NewMetricsTransport(transport, cfg.Metrics, blockchain)
	}

	return &Client{
		config:    cfg,
		transport: transport,
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
