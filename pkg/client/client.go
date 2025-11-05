// Package client provides a gRPC client for interacting with Tron nodes.
package client

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pb/api"
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

	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 100)),
	}

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

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil
}

// GetBalance retrieves the TRX balance for an address
// Returns the balance in SUN (1 TRX = 1,000,000 SUN)
func (c *Client) GetBalance(address string) (decimal.Decimal, error) {
	if !c.IsConnected() {
		return decimal.Zero, ErrNotConnected
	}

	if address == "" {
		return decimal.Zero, ErrInvalidAddress
	}

	// TODO: Implement actual gRPC call to get account
	// This requires proto definitions from tronprotocol/protocol
	return decimal.Zero, fmt.Errorf("not implemented: requires proto definitions")
}

// GetTRC20Balance retrieves the TRC20 token balance for an address
func (c *Client) GetTRC20Balance(contractAddress, ownerAddress string) (decimal.Decimal, error) {
	if !c.IsConnected() {
		return decimal.Zero, ErrNotConnected
	}

	if contractAddress == "" {
		return decimal.Zero, errors.New("contract address is required")
	}

	if ownerAddress == "" {
		return decimal.Zero, errors.New("owner address is required")
	}

	// TODO: Implement actual contract call
	// This requires proto definitions and ABI encoding
	return decimal.Zero, fmt.Errorf("not implemented: requires proto definitions and ABI")
}

// IsActivated checks if an address is activated on the network
func (c *Client) IsActivated(address string) (bool, error) {
	if !c.IsConnected() {
		return false, ErrNotConnected
	}

	if address == "" {
		return false, ErrInvalidAddress
	}

	// TODO: Implement actual gRPC call
	// This requires proto definitions from tronprotocol/protocol
	return false, fmt.Errorf("not implemented: requires proto definitions")
}

// GetAccountResources retrieves the resource information for an address
func (c *Client) GetAccountResources(address string) (map[string]interface{}, error) {
	if !c.IsConnected() {
		return nil, ErrNotConnected
	}

	if address == "" {
		return nil, ErrInvalidAddress
	}

	// TODO: Implement actual gRPC call
	// This requires proto definitions from tronprotocol/protocol
	return nil, fmt.Errorf("not implemented: requires proto definitions")
}
