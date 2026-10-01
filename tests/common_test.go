package tests

import (
	"context"
	"testing"
	"time"

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

	// Account with an active Stake 2.0 position (both bandwidth and energy),
	// a non-zero brokerage and active SR votes.
	stakedAddress = "TUFaFimz7DYk8DVzUznvBgzBAFGppLEJaL"
	// Registered super representative (huobiwallet).
	witnessAddress = "TN2W4cc7a4dsYyTLiLMWa9m7jVpdLjGvYs"

	// finalizedDepth is how far below the head recentBlockNum reads: well past
	// the 19 confirmations that make a block irreversible.
	finalizedDepth = 100
)

// recentBlockNum returns the number of a block that can no longer change but
// that the public nodes still serve. They keep no old history: the fixed
// block the tests used to read now answers {} over both transports, so every
// comparison built on it compared two empty blocks and passed.
func recentBlockNum(t *testing.T) uint64 {
	t.Helper()
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	head, err := c.GetLastBlockHeight(ctx)
	require.NoError(t, err)
	return head - finalizedDepth
}

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
