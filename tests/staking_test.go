package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

// maxUnstakeCount is Tron's MAX_UNFREEZE_V2_SIZE chain parameter.
const maxUnstakeCount = 32

func assertStakeInfo(t *testing.T, info *client.StakeInfo) {
	t.Helper()

	require.NotNil(t, info)
	require.Equal(t, info.StakedBandwidth+info.StakedEnergy, info.TotalStaked)
	require.GreaterOrEqual(t, info.StakedBandwidth, client.SUN(0))
	require.GreaterOrEqual(t, info.StakedEnergy, client.SUN(0))

	var pendingSum client.SUN
	for _, p := range info.PendingUnstakes {
		require.Greater(t, p.Amount, client.SUN(0), "zero-amount entries must be dropped")
		require.False(t, p.ExpireTime.IsZero())
		pendingSum += p.Amount
	}
	require.Equal(t, pendingSum, info.UnstakingTotal)
	require.LessOrEqual(t, info.WithdrawableNow, info.UnstakingTotal)
}

func TestGetStakeInfo_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.GetStakeInfo(ctx, stakedAddress)
	require.NoError(t, err)
	assertStakeInfo(t, info)
	require.Greater(t, info.TotalStaked, client.SUN(0), "fixture account is expected to have an active stake")

	t.Logf("gRPC: staked bandwidth=%d energy=%d, unstaking=%d, withdrawable=%d",
		info.StakedBandwidth, info.StakedEnergy, info.UnstakingTotal, info.WithdrawableNow)
}

func TestGetStakeInfo_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.GetStakeInfo(ctx, stakedAddress)
	require.NoError(t, err)
	assertStakeInfo(t, info)
	require.Greater(t, info.TotalStaked, client.SUN(0), "fixture account is expected to have an active stake")

	t.Logf("HTTP: staked bandwidth=%d energy=%d, unstaking=%d, withdrawable=%d",
		info.StakedBandwidth, info.StakedEnergy, info.UnstakingTotal, info.WithdrawableNow)
}

func TestGetAvailableUnstakeCount_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := c.GetAvailableUnstakeCount(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
	assert.LessOrEqual(t, count, int64(maxUnstakeCount))

	t.Logf("gRPC: available unstake count: %d", count)
}

func TestGetAvailableUnstakeCount_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := c.GetAvailableUnstakeCount(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(0))
	assert.LessOrEqual(t, count, int64(maxUnstakeCount))

	t.Logf("HTTP: available unstake count: %d", count)
}

func TestGetWithdrawableUnstaked_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	amount, err := c.GetWithdrawableUnstaked(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, amount, client.SUN(0))

	// The node's answer must not exceed what the account actually has unstaking.
	info, err := c.GetStakeInfo(ctx, stakedAddress)
	require.NoError(t, err)
	assert.LessOrEqual(t, amount, info.UnstakingTotal)

	t.Logf("gRPC: withdrawable unstaked: %d SUN", amount)
}

func TestGetWithdrawableUnstaked_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	amount, err := c.GetWithdrawableUnstaked(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, amount, client.SUN(0))

	info, err := c.GetStakeInfo(ctx, stakedAddress)
	require.NoError(t, err)
	assert.LessOrEqual(t, amount, info.UnstakingTotal)

	t.Logf("HTTP: withdrawable unstaked: %d SUN", amount)
}
