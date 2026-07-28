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

// Deployment estimates

// deployEstimateCode is 22 bytes of creation code that returns 10 bytes of
// runtime code, so a real deployment pays 200 x 10 energy for the code deposit
// plus a little execution - 2021 in total. A ballpark far below that would mean
// the constructor was not run at all.
const deployEstimateCode = "600a600c600039600a6000f3602a60805260206080f3"

func deployEstimateRequest() client.DeployContractRequest {
	return client.DeployContractRequest{
		From:                       testAddress,
		Name:                       "Toy",
		Bytecode:                   deployEstimateCode,
		ConsumeUserResourcePercent: 100,
		OriginEnergyLimit:          10_000_000,
	}
}

func assertDeployEstimate(t *testing.T, c *client.Client, label string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := c.EstimateDeployContract(ctx, deployEstimateRequest())
	require.NoError(t, err)

	require.True(t, res.Usage.Bandwidth.IsPositive(), "bandwidth must be > 0")
	// The code deposit alone is 2000; anything near zero means the node priced
	// the transaction without executing the constructor.
	require.GreaterOrEqual(t, res.Usage.Energy.IntPart(), int64(2000),
		"energy %s is too low to include the code deposit", res.Usage.Energy)

	// A deployment is billed to the deployer in full, and the contract account
	// it creates is paid for inside the energy, not by the activation fee.
	require.True(t, res.Usage.ContractEnergy.IsZero())
	require.Zero(t, res.Charges.AccountCreation)
	require.Zero(t, res.Charges.UnstakedCreation)

	require.Equal(t, res.Charges.Total(), res.Fee)

	t.Logf("%s: bandwidth=%s energy=%s charges=(b=%s e=%s) fee=%s",
		label, res.Usage.Bandwidth, res.Usage.Energy,
		res.Charges.Bandwidth, res.Charges.Energy, res.Fee)
}

func TestEstimateDeployContract_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	assertDeployEstimate(t, c, "gRPC")
}

// The same over HTTP. The transports express "this is a deployment" differently
// - gRPC leaves ContractAddress unset on the proto, HTTP has to omit the JSON
// field entirely - so only running both shows that either works.
func TestEstimateDeployContract_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	assertDeployEstimate(t, c, "HTTP")
}

func assertConstructorArgsArePriced(t *testing.T, c *client.Client, label string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	plain, err := c.EstimateDeployContract(ctx, deployEstimateRequest())
	require.NoError(t, err)

	withArgs := deployEstimateRequest()
	withArgs.ConstructorParams = `[{"uint256":"1000000"},{"address":"` + usdtContract + `"}]`

	loaded, err := c.EstimateDeployContract(ctx, withArgs)
	require.NoError(t, err)

	// Two 32-byte words are appended to the bytecode, so the transaction is at
	// least 64 bytes bigger. An estimate that priced the bytecode alone - the
	// mistake this catches - would report the same size for both.
	grew := loaded.Usage.Bandwidth.Sub(plain.Usage.Bandwidth)
	require.GreaterOrEqual(t, grew.IntPart(), int64(64),
		"bandwidth grew by only %s for 64 bytes of constructor arguments", grew)

	// Energy deliberately not asserted: this contract ignores its constructor
	// arguments, so running it costs the same either way. Bandwidth is what
	// always moves.
	t.Logf("%s: bandwidth %s -> %s with constructor arguments",
		label, plain.Usage.Bandwidth, loaded.Usage.Bandwidth)
}

func TestEstimateDeployContractPricesConstructorArgs_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	assertConstructorArgsArePriced(t, c, "gRPC")
}

func TestEstimateDeployContractPricesConstructorArgs_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	assertConstructorArgsArePriced(t, c, "HTTP")
}
