package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

// TRC20 tests

func TestTRC20GetName_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	name, err := c.TRC20GetName(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)

	t.Logf("gRPC: Token name: %s", name)
}

func TestTRC20GetName_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	name, err := c.TRC20GetName(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)

	t.Logf("HTTP: Token name: %s", name)
}

func TestTRC20GetSymbol_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	symbol, err := c.TRC20GetSymbol(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "USDT", symbol)

	t.Logf("gRPC: Token symbol: %s", symbol)
}

func TestTRC20GetSymbol_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	symbol, err := c.TRC20GetSymbol(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, "USDT", symbol)

	t.Logf("HTTP: Token symbol: %s", symbol)
}

func TestTRC20GetDecimals_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	decimals, err := c.TRC20GetDecimals(ctx, usdtContract)
	require.NoError(t, err)
	assert.Equal(t, int64(6), decimals.Int64())

	t.Logf("gRPC: Token decimals: %d", decimals.Int64())
}

func TestTRC20GetDecimals_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

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
	defer func() { _ = c.Close() }()

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
	defer func() { _ = c.Close() }()

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
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	abi, err := c.GetContractABI(ctx, usdtContract)
	require.NoError(t, err)
	require.NotNil(t, abi)

	t.Logf("gRPC: Contract ABI has %d entries", len(abi.GetEntrys()))
}

func TestGetContractABI_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	abi, err := c.GetContractABI(ctx, usdtContract)
	require.NoError(t, err)
	require.NotNil(t, abi)

	t.Logf("HTTP: Contract ABI has %d entries", len(abi.GetEntrys()))
}

// Constant call failures

// revertingCall is a transfer of more than any TRC20's total supply, so it
// reverts for insufficient balance whoever the sender is.
const revertingTransferParams = `[{"address":"TFTWNgDBkQ5wQoP8RXpRznnHvAVV8x5jLu"},{"uint256":"1000000000000000000000000000000000"}]`

func TestTriggerConstantContractRevert_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := c.TriggerConstantContractCustom(ctx, testAddress, usdtContract,
		"transfer(address,uint256)", revertingTransferParams)

	require.ErrorIs(t, err, client.ErrContractCallFailed)
	require.ErrorContains(t, err, "REVERT")
	// The evidence survives the error, so a caller can see how far the VM got.
	require.NotNil(t, tx)

	t.Logf("gRPC: %v (energy burned before the revert: %d)", err, tx.GetEnergyUsed())
}

// The same call over HTTP. This pair is the point: the HTTP transport used to
// drop result.code and result.message, so a revert arrived as a plain success
// with the pre-revert energy looking like the real cost. Only comparing the two
// transports shows it - either test alone passes on its own terms.
func TestTriggerConstantContractRevert_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := c.TriggerConstantContractCustom(ctx, testAddress, usdtContract,
		"transfer(address,uint256)", revertingTransferParams)

	require.ErrorIs(t, err, client.ErrContractCallFailed)
	require.ErrorContains(t, err, "REVERT")
	require.NotNil(t, tx)

	t.Logf("HTTP: %v (energy burned before the revert: %d)", err, tx.GetEnergyUsed())
}

// A read that succeeds must not be mistaken for a failure by the same check.
func TestTriggerConstantContractSuccess_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := c.TriggerConstantContractCustom(ctx, testAddress, usdtContract,
		"balanceOf(address)", `[{"address":"`+testAddress+`"}]`)
	require.NoError(t, err)
	require.NotEmpty(t, tx.GetConstantResult())
}

func TestTriggerConstantContractSuccess_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := c.TriggerConstantContractCustom(ctx, testAddress, usdtContract,
		"balanceOf(address)", `[{"address":"`+testAddress+`"}]`)
	require.NoError(t, err)
	require.NotEmpty(t, tx.GetConstantResult())
}
