package client

import (
	"context"
	"errors"
	"sync"
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

func TestGetDelegationsReceivedV2AsksEachLender(t *testing.T) {
	// The index of a receiver lists its lenders under FromAccounts; each pair
	// is asked lender first. Asking the receiver as the lender reads what it
	// lends, which is nothing here, and every lender's stake would read gone.
	receiver, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)
	lender, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	var asked []*api.DelegatedResourceMessage
	c := newTestClient(&fakeTransport{
		getDelegatedResourceAccountIndexV2: func(_ context.Context, addr []byte) (*core.DelegatedResourceAccountIndex, error) {
			require.Equal(t, receiver, addr)
			return &core.DelegatedResourceAccountIndex{FromAccounts: [][]byte{lender}, ToAccounts: [][]byte{receiver}}, nil
		},
		getDelegatedResourceV2: func(_ context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			asked = append(asked, m)
			return &api.DelegatedResourceList{DelegatedResource: []*core.DelegatedResource{delegationRecord(t, 0, 3_000_000, 0, 0)}}, nil
		},
	})

	got, err := c.GetDelegationsReceivedV2(context.Background(), testAddr2)
	require.NoError(t, err)
	require.Len(t, asked, 1)
	require.Equal(t, lender, asked[0].GetFromAddress())
	require.Equal(t, receiver, asked[0].GetToAddress())
	require.Len(t, got, 1)
	require.Equal(t, testAddr, got[0].From)
	require.Equal(t, testAddr2, got[0].To)
	require.Equal(t, SUN(3_000_000), got[0].Energy)
}

func TestGetDelegationsReceivedV2Validation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		addr    string
		wantErr error
	}{
		{"empty", "", ErrEmptyAddress},
		{"invalid", "bad!", ErrInvalidAddress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			c := newTestClient(&fakeTransport{
				getDelegatedResourceAccountIndexV2: func(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) {
					called = true
					return nil, nil
				},
			})

			_, err := c.GetDelegationsReceivedV2(context.Background(), tc.addr)
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

// lendersFake serves an index of n lenders to testAddr2, each lending its
// index+1 SUN of energy, through fetch.
func lendersFake(t *testing.T, n int, fetch func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)) (*fakeTransport, [][]byte) {
	t.Helper()
	receiver, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)
	lenders := make([][]byte, n)
	for i := range lenders {
		lender := make([]byte, 21)
		lender[0], lender[20] = 0x41, byte(i+1)
		lenders[i] = lender
	}
	return &fakeTransport{
		getDelegatedResourceAccountIndexV2: func(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) {
			return &core.DelegatedResourceAccountIndex{FromAccounts: lenders}, nil
		},
		getDelegatedResourceV2: func(ctx context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			if fetch != nil {
				if _, err := fetch(ctx, m); err != nil {
					return nil, err
				}
			}
			return &api.DelegatedResourceList{DelegatedResource: []*core.DelegatedResource{{
				From: m.GetFromAddress(), To: receiver, FrozenBalanceForEnergy: int64(m.GetFromAddress()[20]),
			}}}, nil
		},
	}, lenders
}

func TestGetDelegationsReceivedV2GivesEachCallItsOwnDeadline(t *testing.T) {
	// One deadline over the walk failed an account with many lenders on a
	// slow node; each call gets its own.
	const perCall = 200 * time.Millisecond
	var deadlines []time.Duration
	var mu sync.Mutex
	ft, _ := lendersFake(t, 3, func(ctx context.Context, _ *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "a lender's call without a deadline of its own")
		mu.Lock()
		deadlines = append(deadlines, time.Until(deadline))
		mu.Unlock()
		return nil, nil
	})
	index := ft.getDelegatedResourceAccountIndexV2
	ft.getDelegatedResourceAccountIndexV2 = func(ctx context.Context, addr []byte) (*core.DelegatedResourceAccountIndex, error) {
		_, ok := ctx.Deadline()
		require.True(t, ok, "the index call without a deadline of its own")
		return index(ctx, addr)
	}
	got, err := newTestClient(ft).GetDelegationsReceivedV2(context.Background(), testAddr2, WithCallTimeout(perCall))
	require.NoError(t, err)
	require.Len(t, got, 3)
	for _, d := range deadlines {
		require.LessOrEqual(t, d, perCall)
		require.Greater(t, d, time.Duration(0))
	}
}

func TestGetDelegationsReceivedV2ReadsLendersAtOnceInTheIndexOrder(t *testing.T) {
	var running, most int
	var mu sync.Mutex
	ft, lenders := lendersFake(t, 12, func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
		mu.Lock()
		running++
		most = max(most, running)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		running--
		mu.Unlock()
		return nil, nil
	})
	got, err := newTestClient(ft).GetDelegationsReceivedV2(context.Background(), testAddr2, WithConcurrency(4))
	require.NoError(t, err)
	require.Len(t, got, len(lenders))
	for i, d := range got {
		require.Equal(t, tronutils.EncodeCheck(lenders[i]), d.From, "result out of the index's order")
	}
	require.LessOrEqual(t, most, 4)
	require.Greater(t, most, 1, "lenders read one after another")
}

func TestGetDelegationsReceivedV2ReturnsTheCallThatFailed(t *testing.T) {
	// The first failure cancels the calls still out; their cancellation is
	// not the listing's error.
	boom := errors.New("node refused")
	ft, lenders := lendersFake(t, 6, nil)
	ft.getDelegatedResourceV2 = func(ctx context.Context, m *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
		if m.GetFromAddress()[20] == byte(len(lenders)) {
			return nil, boom
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	// Bounded, so a walk that does not cancel the rest fails rather than hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := newTestClient(ft).GetDelegationsReceivedV2(ctx, testAddr2, WithConcurrency(len(lenders)))
	require.ErrorIs(t, err, boom)
}
