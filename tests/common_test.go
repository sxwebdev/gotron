package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

const (
	// Public nodes for testing
	grpcAddress = "tron-grpc.publicnode.com:443"
	httpAddress = "https://tron-rpc.publicnode.com"

	// Known mainnet addresses for testing
	testAddress  = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g" // Binance hot wallet
	usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT contract

	// Known block with transactions for testing
	testBlockNum = uint64(79831098)
)

func newGRPCClient(t *testing.T) *client.Client {
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolGRPC,
				Address:  grpcAddress,
				UseTLS:   true,
			},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	return c
}

func newHTTPClient(t *testing.T) *client.Client {
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolHTTP,
				Address:  httpAddress,
			},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	return c
}

func newMultiNodeClient(t *testing.T) *client.Client {
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolGRPC,
				Address:  grpcAddress,
				UseTLS:   true,
			},
			{
				Protocol: client.ProtocolHTTP,
				Address:  httpAddress,
			},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	return c
}
