package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// delegationFake serves one index entry pointing at testAddr2 and returns the
// given records for it.
func delegationFake(t *testing.T, records ...*core.DelegatedResource) *fakeTransport {
	t.Helper()

	to, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	index := func(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) {
		return &core.DelegatedResourceAccountIndex{ToAccounts: [][]byte{to}}, nil
	}
	fetch := func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
		return &api.DelegatedResourceList{DelegatedResource: records}, nil
	}

	return &fakeTransport{
		getDelegatedResourceAccountIndex:   index,
		getDelegatedResourceAccountIndexV2: index,
		getDelegatedResource:               fetch,
		getDelegatedResourceV2:             fetch,
	}
}

func delegationRecord(t *testing.T, bandwidth, energy, bandwidthExpiry, energyExpiry int64) *core.DelegatedResource {
	t.Helper()

	from, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	to, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	return &core.DelegatedResource{
		From:                      from,
		To:                        to,
		FrozenBalanceForBandwidth: bandwidth,
		FrozenBalanceForEnergy:    energy,
		ExpireTimeForBandwidth:    bandwidthExpiry,
		ExpireTimeForEnergy:       energyExpiry,
	}
}

func TestGetDelegatedResourcesConvertsTheChainsForm(t *testing.T) {
	lockedAt := time.Now().Add(72 * time.Hour).Truncate(time.Millisecond)

	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			ft := delegationFake(t, delegationRecord(t, 1_000_000_000, 2_000_000_000, lockedAt.UnixMilli(), 0))
			c := newTestClient(ft)

			get := c.GetDelegatedResources
			if version == "v2" {
				get = c.GetDelegatedResourcesV2
			}

			got, err := get(context.Background(), testAddr)
			require.NoError(t, err)
			require.Len(t, got, 1)

			d := got[0]
			// Addresses come back base58, not raw bytes.
			require.Equal(t, testAddr, d.From)
			require.Equal(t, testAddr2, d.To)
			// Amounts are SUN, so 1000 TRX reads as 1000 TRX.
			require.Equal(t, SUN(1_000_000_000), d.Bandwidth)
			require.Equal(t, SUN(2_000_000_000), d.Energy)
			require.Equal(t, "1000 TRX", d.Bandwidth.String())

			// The expiry is Unix milliseconds on the wire. Reading it as
			// seconds would put this lock in 1970 and make it look expired.
			require.True(t, d.BandwidthExpiresAt.Equal(lockedAt),
				"want %s, got %s", lockedAt, d.BandwidthExpiresAt)
			require.True(t, d.BandwidthExpiresAt.After(time.Now()))

			// An unlocked delegation has no expiry, and the zero time says so -
			// time.UnixMilli(0) would claim it expired in 1970.
			require.True(t, d.EnergyExpiresAt.IsZero(), "got %s", d.EnergyExpiresAt)
		})
	}
}

func TestGetDelegatedResourcesSkipsEmptyRecords(t *testing.T) {
	// The node returns placeholders with nothing delegated; reporting them as
	// zero-amount delegations makes an account with none look like it has some.
	c := newTestClient(delegationFake(
		t,
		delegationRecord(t, 0, 0, 0, 0),
		delegationRecord(t, 5_000_000, 0, 0, 0),
		delegationRecord(t, 0, 7_000_000, 0, 0),
	))

	got, err := c.GetDelegatedResourcesV2(context.Background(), testAddr)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, SUN(5_000_000), got[0].Bandwidth)
	require.Equal(t, SUN(7_000_000), got[1].Energy)
}

func TestGetDelegatedResourcesMultipleRecordsPerAccount(t *testing.T) {
	// One index entry can carry several records; returning only the first would
	// silently under-report.
	c := newTestClient(delegationFake(
		t,
		delegationRecord(t, 1_000_000, 0, 0, 0),
		delegationRecord(t, 2_000_000, 0, 0, 0),
	))

	got, err := c.GetDelegatedResourcesV2(context.Background(), testAddr)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestGetDelegatedResourcesValidation(t *testing.T) {
	cases := []struct {
		name    string
		addr    string
		wantErr error
	}{
		{"empty", "", ErrEmptyAddress},
		{"invalid", "bad!", ErrInvalidAddress},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			c := newTestClient(&fakeTransport{
				getDelegatedResourceAccountIndexV2: func(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) {
					called = true
					return nil, nil
				},
			})

			_, err := c.GetDelegatedResourcesV2(context.Background(), tc.addr)
			require.ErrorIs(t, err, tc.wantErr)
			require.False(t, called)
		})
	}
}

func TestGetCanDelegatedMaxSize(t *testing.T) {
	t.Run("returns SUN and sends the resource type", func(t *testing.T) {
		var got *api.CanDelegatedMaxSizeRequestMessage
		c := newTestClient(&fakeTransport{
			getCanDelegatedMaxSize: func(_ context.Context, m *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
				got = m
				return &api.CanDelegatedMaxSizeResponseMessage{MaxSize: 1_500_000_000}, nil
			},
		})

		max, err := c.GetCanDelegatedMaxSize(context.Background(), testAddr, ResourceTypeEnergy)
		require.NoError(t, err)

		// The answer is a staked TRX amount, so it reads as TRX rather than as
		// a bare number that could be mistaken for energy units.
		require.Equal(t, SUN(1_500_000_000), max)
		require.Equal(t, "1500 TRX", max.String())

		wantAddr, err := tronutils.DecodeCheck(testAddr)
		require.NoError(t, err)
		require.Equal(t, wantAddr, got.GetOwnerAddress())
		require.Equal(t, int32(core.ResourceCode_ENERGY), got.GetType())
	})

	t.Run("bandwidth maps to the zero enum, not to a missing field", func(t *testing.T) {
		var got *api.CanDelegatedMaxSizeRequestMessage
		c := newTestClient(&fakeTransport{
			getCanDelegatedMaxSize: func(_ context.Context, m *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
				got = m
				return &api.CanDelegatedMaxSizeResponseMessage{}, nil
			},
		})

		max, err := c.GetCanDelegatedMaxSize(context.Background(), testAddr, ResourceTypeBandwidth)
		require.NoError(t, err)
		require.Equal(t, SUN(0), max)
		require.Equal(t, int32(core.ResourceCode_BANDWIDTH), got.GetType())
	})

	t.Run("validation", func(t *testing.T) {
		cases := []struct {
			name     string
			addr     string
			resource ResourceType
			wantErr  error
		}{
			{"empty address", "", ResourceTypeEnergy, ErrEmptyAddress},
			{"invalid address", "bad!", ResourceTypeEnergy, ErrInvalidAddress},
			{"unknown resource", testAddr, ResourceType(9), ErrInvalidResourceType},
			{"negative resource", testAddr, ResourceType(-1), ErrInvalidResourceType},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				called := false
				c := newTestClient(&fakeTransport{
					getCanDelegatedMaxSize: func(context.Context, *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
						called = true
						return nil, nil
					},
				})

				_, err := c.GetCanDelegatedMaxSize(context.Background(), tc.addr, tc.resource)
				require.ErrorIs(t, err, tc.wantErr)
				require.False(t, called)
			})
		}
	})
}
