// Package client provides a gRPC client for interacting with Tron nodes.
package client

import (
	"crypto/tls"
	"fmt"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a Tron blockchain client
type Client struct {
	conn   *grpc.ClientConn
	config Config

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

	opts := append(
		cfg.DialOptions,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)

	if cfg.UseTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS13,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(cfg.GRPCAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}

	client.conn = conn

	client.walletClient = api.NewWalletClient(client.conn)

	return client, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// GetNetwork returns the current network based on gRPC address
func (c *Client) GetNetwork() Network {
	return c.config.Network
}

// API returns the underlying WalletClient API
func (c *Client) API() api.WalletClient {
	return c.walletClient
}
