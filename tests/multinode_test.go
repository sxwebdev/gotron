package tests

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

func TestMultiNode_RoundRobin(t *testing.T) {
	c := newMultiNodeClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make multiple requests - they should alternate between gRPC and HTTP
	for i := range 4 {
		block, err := c.GetLastBlock(ctx)
		require.NoError(t, err)
		require.NotNil(t, block)

		blockNum := block.GetBlockHeader().GetRawData().GetNumber()
		assert.Greater(t, blockNum, int64(0))
		t.Logf("MultiNode request %d: Latest block number: %d", i+1, blockNum)
	}
}

func TestMultiNode_GetAccount(t *testing.T) {
	c := newMultiNodeClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make multiple account requests
	for i := range 4 {
		account, err := c.GetAccount(ctx, testAddress)
		require.NoError(t, err)
		require.NotNil(t, account)

		t.Logf("MultiNode request %d: Account balance: %d SUN", i+1, account.GetBalance())
	}
}

func TestMultiNode_GetAccountBalance(t *testing.T) {
	c := newMultiNodeClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make multiple balance requests
	for i := range 4 {
		balance, err := c.GetAccountBalance(ctx, testAddress)
		require.NoError(t, err)
		assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))

		t.Logf("MultiNode request %d: Account balance: %s TRX", i+1, balance.String())
	}
}

func TestMultiNode_TRC20GetName(t *testing.T) {
	c := newMultiNodeClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make multiple TRC20 name requests
	for i := range 4 {
		name, err := c.TRC20GetName(ctx, usdtContract)
		require.NoError(t, err)
		assert.Equal(t, "Tether USD", name)

		t.Logf("MultiNode request %d: Token name: %s", i+1, name)
	}
}

func TestMultiNode_ChainParams(t *testing.T) {
	c := newMultiNodeClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make multiple chain params requests
	for i := range 4 {
		params, err := c.ChainParams(ctx)
		require.NoError(t, err)
		require.NotNil(t, params)

		assert.Greater(t, params.EnergyFee, int64(0))
		t.Logf("MultiNode request %d: Energy fee: %d", i+1, params.EnergyFee)
	}
}

func TestMultiNode_SameProtocol(t *testing.T) {
	// Test with multiple HTTP nodes
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolHTTP,
				Address:  httpAddress,
			},
			{
				Protocol: client.ProtocolHTTP,
				Address:  httpAddress, // Same node for testing
			},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := range 4 {
		block, err := c.GetLastBlock(ctx)
		require.NoError(t, err)
		require.NotNil(t, block)

		blockNum := block.GetBlockHeader().GetRawData().GetNumber()
		t.Logf("MultiNode HTTP request %d: Latest block number: %d", i+1, blockNum)
	}
}
