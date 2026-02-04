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

const (
	// Public nodes for testing
	grpcAddress = "tron-grpc.publicnode.com:443"
	httpAddress = "https://tron-rpc.publicnode.com"

	// Known mainnet addresses for testing
	testAddress  = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g" // Binance hot wallet
	usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT contract
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

func TestGetNowBlock_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	block, err := c.GetLastBlock(ctx)
	require.NoError(t, err)
	require.NotNil(t, block)

	blockNum := block.GetBlockHeader().GetRawData().GetNumber()
	assert.Greater(t, blockNum, int64(0))
	t.Logf("gRPC: Latest block number: %d", blockNum)
}

func TestGetNowBlock_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	block, err := c.GetLastBlock(ctx)
	require.NoError(t, err)
	require.NotNil(t, block)

	blockNum := block.GetBlockHeader().GetRawData().GetNumber()
	assert.Greater(t, blockNum, int64(0))
	t.Logf("HTTP: Latest block number: %d", blockNum)
}

func TestGetAccount_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account, err := c.GetAccount(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, account)

	t.Logf("gRPC: Account balance: %d SUN", account.GetBalance())
}

func TestGetAccount_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account, err := c.GetAccount(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, account)

	t.Logf("HTTP: Account balance: %d SUN", account.GetBalance())
}

func TestGetAccountBalance_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := c.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))

	t.Logf("gRPC: Account balance: %s TRX", balance.String())
}

func TestGetAccountBalance_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := c.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))

	t.Logf("HTTP: Account balance: %s TRX", balance.String())
}

func TestGetChainParameters_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params, err := c.ChainParams(ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Greater(t, params.EnergyFee, int64(0))
	t.Logf("gRPC: Energy fee: %d, Transaction fee: %d", params.EnergyFee, params.TransactionFee)
}

func TestGetChainParameters_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params, err := c.ChainParams(ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Greater(t, params.EnergyFee, int64(0))
	t.Logf("HTTP: Energy fee: %d, Transaction fee: %d", params.EnergyFee, params.TransactionFee)
}

func TestTRC20GetName_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	name, err := c.TRC20GetName(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)

	t.Logf("gRPC: Token name: %s", name)
}

func TestTRC20GetName_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	name, err := c.TRC20GetName(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)

	t.Logf("HTTP: Token name: %s", name)
}

func TestTRC20GetSymbol_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	symbol, err := c.TRC20GetSymbol(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "USDT", symbol)

	t.Logf("gRPC: Token symbol: %s", symbol)
}

func TestTRC20GetSymbol_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	symbol, err := c.TRC20GetSymbol(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "USDT", symbol)

	t.Logf("HTTP: Token symbol: %s", symbol)
}

func TestTRC20GetDecimals_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	decimals, err := c.TRC20GetDecimals(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, int64(6), decimals.Int64())

	t.Logf("gRPC: Token decimals: %d", decimals.Int64())
}

func TestTRC20GetDecimals_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	decimals, err := c.TRC20GetDecimals(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, int64(6), decimals.Int64())

	t.Logf("HTTP: Token decimals: %d", decimals.Int64())
}

func TestGetBlockByHeight_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get a recent block
	latestHeight, err := c.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	block, err := c.GetBlockByHeight(ctx, latestHeight-10)
	require.NoError(t, err)
	require.NotNil(t, block)

	t.Logf("gRPC: Block %d has %d transactions", latestHeight-10, len(block.GetTransactions()))
}

func TestGetBlockByHeight_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get a recent block
	latestHeight, err := c.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	block, err := c.GetBlockByHeight(ctx, latestHeight-10)
	require.NoError(t, err)
	require.NotNil(t, block)

	t.Logf("HTTP: Block %d has %d transactions", latestHeight-10, len(block.GetTransactions()))
}

func TestCompareResults_BlockHeight(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcHeight, err := grpcClient.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	httpHeight, err := httpClient.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	// Allow for a small difference due to timing
	diff := int64(grpcHeight) - int64(httpHeight)
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, int64(5), "Block height difference should be small")

	t.Logf("gRPC height: %d, HTTP height: %d, diff: %d", grpcHeight, httpHeight, diff)
}

func TestCompareResults_ChainParams(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcParams, err := grpcClient.ChainParams(ctx)
	require.NoError(t, err)

	httpParams, err := httpClient.ChainParams(ctx)
	require.NoError(t, err)

	assert.Equal(t, grpcParams.EnergyFee, httpParams.EnergyFee)
	assert.Equal(t, grpcParams.TransactionFee, httpParams.TransactionFee)
	assert.Equal(t, grpcParams.CreateAccountFee, httpParams.CreateAccountFee)

	t.Logf("Chain params match: EnergyFee=%d, TransactionFee=%d", grpcParams.EnergyFee, grpcParams.TransactionFee)
}

func TestCompareResults_AccountBalance(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcBalance, err := grpcClient.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)

	httpBalance, err := httpClient.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)

	// Balances should be very close (may differ slightly due to timing)
	diff := grpcBalance.Sub(httpBalance).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromInt(1)), "Balance difference should be < 1 TRX")

	t.Logf("gRPC balance: %s TRX, HTTP balance: %s TRX", grpcBalance.String(), httpBalance.String())
}

func TestGetAccountResource_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources, err := c.GetAccountResource(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, resources)

	t.Logf("gRPC: Energy limit: %d, Net limit: %d", resources.EnergyLimit, resources.NetLimit)
}

func TestGetAccountResource_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources, err := c.GetAccountResource(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, resources)

	t.Logf("HTTP: Energy limit: %d, Net limit: %d", resources.EnergyLimit, resources.NetLimit)
}

func TestIsAccountActivated_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isActivated, err := c.IsAccountActivated(ctx, testAddress)
	require.NoError(t, err)
	assert.True(t, isActivated)

	t.Logf("gRPC: Account %s is activated: %v", testAddress, isActivated)
}

func TestIsAccountActivated_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isActivated, err := c.IsAccountActivated(ctx, testAddress)
	require.NoError(t, err)
	assert.True(t, isActivated)

	t.Logf("HTTP: Account %s is activated: %v", testAddress, isActivated)
}

// Multi-node tests

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
	for i := 0; i < 4; i++ {
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
	for i := 0; i < 4; i++ {
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
