// Package client provides a gRPC client for interacting with Tron nodes.
package gotron

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	pbtronapi "github.com/sxwebdev/gotron/pb/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// ErrInvalidConfig is returned when the client configuration is invalid
	ErrInvalidConfig = errors.New("invalid client configuration")
	// ErrNotConnected is returned when the client is not connected
	ErrNotConnected = errors.New("client not connected")
	// ErrInvalidAddress is returned when an address is invalid
	ErrInvalidAddress = errors.New("invalid address")
	// ErrInvalidAmount is returned when an amount is invalid
	ErrInvalidAmount = errors.New("invalid amount")
)

// Config holds the configuration for the Tron client
type Config struct {
	// GRPCAddress is the address of the Tron node gRPC endpoint
	GRPCAddress string
	// UseTLS enables TLS for the gRPC connection
	UseTLS bool
	// Timeout is the default timeout for RPC calls
	Timeout time.Duration
	// APIKey is an optional API key for TronGrid
	APIKey string
}

// Network represents the Tron network type
type Network string

const (
	// Mainnet represents the Tron mainnet
	Mainnet Network = "mainnet"
	// Testnet represents the Tron testnet (Shasta)
	Testnet Network = "testnet"
	// Nile represents the Tron Nile testnet
	Nile Network = "nile"
)

// Client represents a Tron blockchain client
type Client struct {
	conn   *grpc.ClientConn
	config *Config

	tronClient pbtronapi.WalletClient
}

// New creates a new Tron client with the given configuration
func newClient(cfg *Config) (*Client, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Set default timeout if not specified
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	client := &Client{
		config: cfg,
	}

	if err := client.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client.tronClient = pbtronapi.NewWalletClient(client.conn)

	return client, nil
}

// validateConfig validates the client configuration
func validateConfig(cfg *Config) error {
	if cfg == nil {
		return ErrInvalidConfig
	}

	if cfg.GRPCAddress == "" {
		return fmt.Errorf("%w: gRPC address is required", ErrInvalidConfig)
	}

	return nil
}

// connect establishes a connection to the Tron node
func (c *Client) connect() error {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 100)),
	}

	if c.config.UseTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(c.config.GRPCAddress, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC: %w", err)
	}

	c.conn = conn

	return nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
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

// GetNetwork returns the current network based on gRPC address
func (c *Client) GetNetwork() Network {
	switch c.config.GRPCAddress {
	case "grpc.trongrid.io:50051":
		return Mainnet
	case "grpc.shasta.trongrid.io:50051":
		return Testnet
	case "grpc.nile.trongrid.io:50051":
		return Nile
	default:
		return Mainnet
	}
}
