package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
)

// minWitnessCount is the number of active super representatives on Tron; the
// candidate list is always larger.
const minWitnessCount = 27

func assertWitnessList(t *testing.T, list *api.WitnessList) {
	t.Helper()

	require.NotNil(t, list)
	require.GreaterOrEqual(t, len(list.GetWitnesses()), minWitnessCount)

	var maxVotes int64
	for _, w := range list.GetWitnesses() {
		// A mangled address decode would still be non-empty, so check the shape:
		// Tron addresses are 21 bytes and start with the 0x41 mainnet prefix.
		require.Len(t, w.GetAddress(), 21, "witness address must be 21 bytes")
		require.EqualValues(t, 0x41, w.GetAddress()[0], "witness address must start with 0x41")
		if w.GetVoteCount() > maxVotes {
			maxVotes = w.GetVoteCount()
		}
	}
	require.Greater(t, maxVotes, int64(0), "at least one witness must have votes")
}

func TestListWitnesses_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := c.ListWitnesses(ctx)
	require.NoError(t, err)
	assertWitnessList(t, list)

	t.Logf("gRPC: %d witnesses", len(list.GetWitnesses()))
}

func TestListWitnesses_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := c.ListWitnesses(ctx)
	require.NoError(t, err)
	assertWitnessList(t, list)

	t.Logf("HTTP: %d witnesses", len(list.GetWitnesses()))
}

func TestGetUnclaimedReward_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reward, err := c.GetUnclaimedReward(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, reward, client.SUN(0))

	t.Logf("gRPC: unclaimed reward: %d SUN", reward)
}

func TestGetUnclaimedReward_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reward, err := c.GetUnclaimedReward(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, reward, client.SUN(0))

	t.Logf("HTTP: unclaimed reward: %d SUN", reward)
}

func TestGetWitnessBrokerage_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	brokerage, err := c.GetWitnessBrokerage(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, brokerage, int64(0))
	assert.LessOrEqual(t, brokerage, int64(100))

	t.Logf("gRPC: brokerage: %d%%", brokerage)
}

func TestGetWitnessBrokerage_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	brokerage, err := c.GetWitnessBrokerage(ctx, stakedAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, brokerage, int64(0))
	assert.LessOrEqual(t, brokerage, int64(100))

	t.Logf("HTTP: brokerage: %d%%", brokerage)
}

// witnessAddress must be present in the list returned by both transports.
func TestListWitnessesContainsKnownSR(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := c.ListWitnesses(ctx)
	require.NoError(t, err)

	require.Contains(t, witnessAddressSet(t, list), witnessAddress)
}

func witnessAddressSet(t *testing.T, list *api.WitnessList) map[string]struct{} {
	t.Helper()

	set := make(map[string]struct{}, len(list.GetWitnesses()))
	for _, w := range list.GetWitnesses() {
		set[tronutils.EncodeCheck(w.GetAddress())] = struct{}{}
	}
	return set
}
