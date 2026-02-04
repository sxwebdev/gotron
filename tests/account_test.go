package tests

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
