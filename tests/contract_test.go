package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TRC20 tests

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

// Smart contract tests

func TestGetContract_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contract, err := c.GetContract(ctx, usdtContract)
	require.NoError(t, err)
	require.NotNil(t, contract)

	t.Logf("gRPC: Contract name: %s, consume_user_resource_percent: %d",
		contract.GetName(), contract.GetConsumeUserResourcePercent())
}

func TestGetContract_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contract, err := c.GetContract(ctx, usdtContract)
	require.NoError(t, err)
	require.NotNil(t, contract)

	t.Logf("HTTP: Contract name: %s, consume_user_resource_percent: %d",
		contract.GetName(), contract.GetConsumeUserResourcePercent())
}

func TestGetContractABI_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	abi, err := c.GetContractABI(ctx, usdtContract)
	require.NoError(t, err)
	require.NotNil(t, abi)

	t.Logf("gRPC: Contract ABI has %d entries", len(abi.GetEntrys()))
}

func TestGetContractABI_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	abi, err := c.GetContractABI(ctx, usdtContract)
	require.NoError(t, err)
	require.NotNil(t, abi)

	t.Logf("HTTP: Contract ABI has %d entries", len(abi.GetEntrys()))
}
