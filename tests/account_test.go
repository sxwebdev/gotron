package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestGetAccount_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account, err := c.GetAccount(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, account)

	t.Logf("gRPC: Account balance: %d SUN", account.GetBalance())
}

func TestGetAccount_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account, err := c.GetAccount(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, account)

	t.Logf("HTTP: Account balance: %d SUN", account.GetBalance())
}

func TestGetAccountPermission_GRPC(t *testing.T) {
	testGetAccountPermission(t, newGRPCClient(t))
}

func TestGetAccountPermission_HTTP(t *testing.T) {
	testGetAccountPermission(t, newHTTPClient(t))
}

func testGetAccountPermission(t *testing.T, c *client.Client) {
	t.Helper()
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	permission, err := c.GetAccountPermission(ctx, testAddress, client.FirstActivePermissionID)
	require.NoError(t, err)
	require.Equal(t, client.FirstActivePermissionID, permission.GetId())
	require.NotEmpty(t, permission.GetKeys())
}

func TestGetWitnessPermission_GRPC(t *testing.T) {
	testGetWitnessPermission(t, newGRPCClient(t))
}

func TestGetWitnessPermission_HTTP(t *testing.T) {
	testGetWitnessPermission(t, newHTTPClient(t))
}

func testGetWitnessPermission(t *testing.T, c *client.Client) {
	t.Helper()
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	permission, err := c.GetAccountPermission(ctx, witnessAddress, client.WitnessPermissionID)
	require.NoError(t, err)
	require.Equal(t, core.Permission_Witness, permission.GetType())
	require.Equal(t, client.WitnessPermissionID, permission.GetId())
	require.Len(t, permission.GetKeys(), 1)
}

func TestUpdateAccountPermissions_GRPC(t *testing.T) {
	testUpdateAccountPermissions(t, newGRPCClient(t))
}

func TestUpdateAccountPermissions_HTTP(t *testing.T) {
	testUpdateAccountPermissions(t, newHTTPClient(t))
}

func testUpdateAccountPermissions(t *testing.T, c *client.Client) {
	t.Helper()
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	account, err := c.GetAccount(ctx, witnessAddress)
	require.NoError(t, err)
	// is_witness is what java-tron consults to allow a witness permission at
	// all, and the HTTP account parser used to drop it while gRPC kept it.
	require.True(t, account.GetIsWitness())
	require.NotNil(t, account.GetOwnerPermission())
	require.NotNil(t, account.GetWitnessPermission())
	require.NotEmpty(t, account.GetActivePermission())

	tx, err := c.UpdateAccountPermissions(ctx, client.AccountPermissionUpdateRequest{
		Account: witnessAddress,
		Owner:   account.GetOwnerPermission(),
		Witness: account.GetWitnessPermission(),
		Actives: account.GetActivePermission(),
	})
	require.NoError(t, err)
	require.Equal(t, core.Transaction_Contract_AccountPermissionUpdateContract, tx.GetTransaction().GetRawData().GetContract()[0].GetType())
}

func TestGetAccountBalance_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := c.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, balance, client.SUN(0))

	t.Logf("gRPC: Account balance: %s", balance)
}

func TestGetAccountBalance_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := c.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, balance, client.SUN(0))

	t.Logf("HTTP: Account balance: %s", balance)
}

func TestGetAccountResource_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources, err := c.GetAccountResource(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, resources)

	t.Logf("gRPC: Energy limit: %d, Net limit: %d", resources.EnergyLimit, resources.NetLimit)
}

func TestGetAccountResource_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources, err := c.GetAccountResource(ctx, testAddress)
	require.NoError(t, err)
	require.NotNil(t, resources)

	t.Logf("HTTP: Energy limit: %d, Net limit: %d", resources.EnergyLimit, resources.NetLimit)
}

func TestIsAccountActivated_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isActivated, err := c.IsAccountActivated(ctx, testAddress)
	require.NoError(t, err)
	assert.True(t, isActivated)

	t.Logf("gRPC: Account %s is activated: %v", testAddress, isActivated)
}

func TestIsAccountActivated_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isActivated, err := c.IsAccountActivated(ctx, testAddress)
	require.NoError(t, err)
	assert.True(t, isActivated)

	t.Logf("HTTP: Account %s is activated: %v", testAddress, isActivated)
}

func TestGetChainParameters_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

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
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params, err := c.ChainParams(ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Greater(t, params.EnergyFee, int64(0))
	t.Logf("HTTP: Energy fee: %d, Transaction fee: %d", params.EnergyFee, params.TransactionFee)
}
