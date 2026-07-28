package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestStakeValidation(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		resource ResourceType
		amount   int64
		wantErr  error
	}{
		{"invalid owner", "bad!", ResourceTypeEnergy, 1, ErrInvalidAddress},
		{"empty owner", "", ResourceTypeEnergy, 1, ErrInvalidAddress},
		{"invalid resource", testAddr, ResourceType(9), 1, ErrInvalidResourceType},
		{"zero amount", testAddr, ResourceTypeEnergy, 0, ErrInvalidAmount},
		{"negative amount", testAddr, ResourceTypeEnergy, -5, ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			c := newTestClient(&fakeTransport{
				freezeBalanceV2: func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
					calls++
					return okTx(), nil
				},
				unfreezeBalanceV2: func(context.Context, *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
					calls++
					return okTx(), nil
				},
			})

			_, err := c.Stake(t.Context(), tt.owner, tt.resource, tt.amount)
			require.ErrorIs(t, err, tt.wantErr)

			_, err = c.Unstake(t.Context(), tt.owner, tt.resource, tt.amount)
			require.ErrorIs(t, err, tt.wantErr)

			require.Zero(t, calls, "transport must not be called for invalid input")
		})
	}
}

func TestStakeBuildsContract(t *testing.T) {
	tests := []struct {
		name     string
		resource ResourceType
		want     core.ResourceCode
	}{
		{"energy", ResourceTypeEnergy, core.ResourceCode_ENERGY},
		{"bandwidth", ResourceTypeBandwidth, core.ResourceCode_BANDWIDTH},
	}

	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *core.FreezeBalanceV2Contract
			c := newTestClient(&fakeTransport{
				freezeBalanceV2: func(_ context.Context, ct *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
					got = ct
					return okTx(), nil
				},
			})

			_, err := c.Stake(t.Context(), testAddr, tt.resource, 1_000_000)
			require.NoError(t, err)
			require.Equal(t, wantOwner, got.GetOwnerAddress())
			require.Equal(t, tt.want, got.GetResource())
			require.EqualValues(t, 1_000_000, got.GetFrozenBalance())
		})
	}
}

func TestUnstakeBuildsContract(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	var got *core.UnfreezeBalanceV2Contract
	c := newTestClient(&fakeTransport{
		unfreezeBalanceV2: func(_ context.Context, ct *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			got = ct
			return okTx(), nil
		},
	})

	_, err = c.Unstake(t.Context(), testAddr, ResourceTypeBandwidth, 2_500_000)
	require.NoError(t, err)
	require.Equal(t, wantOwner, got.GetOwnerAddress())
	require.Equal(t, core.ResourceCode_BANDWIDTH, got.GetResource())
	// UnfreezeBalanceV2Contract names this field differently from
	// FreezeBalanceV2Contract.FrozenBalance - a copy-paste would leave it zero.
	require.EqualValues(t, 2_500_000, got.GetUnfreezeBalance())
}

func TestOwnerOnlyStakeContracts(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	t.Run("withdraw unstaked", func(t *testing.T) {
		var got *core.WithdrawExpireUnfreezeContract
		c := newTestClient(&fakeTransport{
			withdrawExpireUnfreeze: func(_ context.Context, ct *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
				got = ct
				return okTx(), nil
			},
		})

		_, err := c.WithdrawUnstaked(t.Context(), "bad!")
		require.ErrorIs(t, err, ErrInvalidAddress)

		_, err = c.WithdrawUnstaked(t.Context(), testAddr)
		require.NoError(t, err)
		require.Equal(t, wantOwner, got.GetOwnerAddress())
	})

	t.Run("cancel all unstakes", func(t *testing.T) {
		var got *core.CancelAllUnfreezeV2Contract
		c := newTestClient(&fakeTransport{
			cancelAllUnfreezeV2: func(_ context.Context, ct *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
				got = ct
				return okTx(), nil
			},
		})

		_, err := c.CancelAllUnstakes(t.Context(), "bad!")
		require.ErrorIs(t, err, ErrInvalidAddress)

		_, err = c.CancelAllUnstakes(t.Context(), testAddr)
		require.NoError(t, err)
		require.Equal(t, wantOwner, got.GetOwnerAddress())
	})
}

// A node that answers with an empty TransactionExtention has not built anything.
// Returning it as a success would only fail later, at signing or broadcast.
func TestStakeRejectsEmptyTransaction(t *testing.T) {
	empty := func() (*api.TransactionExtention, error) { return &api.TransactionExtention{}, nil }

	ft := &fakeTransport{
		freezeBalanceV2: func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return empty()
		},
		unfreezeBalanceV2: func(context.Context, *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return empty()
		},
		withdrawExpireUnfreeze: func(context.Context, *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
			return empty()
		},
		cancelAllUnfreezeV2: func(context.Context, *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
			return empty()
		},
	}
	c := newTestClient(ft)

	calls := map[string]func() error{
		"Stake":             func() error { _, err := c.Stake(t.Context(), testAddr, ResourceTypeEnergy, 1); return err },
		"Unstake":           func() error { _, err := c.Unstake(t.Context(), testAddr, ResourceTypeEnergy, 1); return err },
		"WithdrawUnstaked":  func() error { _, err := c.WithdrawUnstaked(t.Context(), testAddr); return err },
		"CancelAllUnstakes": func() error { _, err := c.CancelAllUnstakes(t.Context(), testAddr); return err },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, call(), ErrInvalidTransaction)
		})
	}
}

func TestStakeSurfacesContractValidateError(t *testing.T) {
	c := newTestClient(&fakeTransport{
		freezeBalanceV2: func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_CONTRACT_VALIDATE_ERROR,
					Message: []byte("no frozenBalance(Energy)"),
				},
			}, nil
		},
	})

	_, err := c.Stake(t.Context(), testAddr, ResourceTypeEnergy, 1)
	require.ErrorContains(t, err, "no frozenBalance(Energy)")
}

func TestStakeTransportErrorPropagates(t *testing.T) {
	sentinel := errors.New("node down")
	c := newTestClient(&fakeTransport{
		freezeBalanceV2: func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return nil, sentinel
		},
	})

	_, err := c.Stake(t.Context(), testAddr, ResourceTypeEnergy, 1)
	require.ErrorIs(t, err, sentinel)
}

func TestGetAvailableUnstakeCount(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	t.Run("value", func(t *testing.T) {
		var got *api.GetAvailableUnfreezeCountRequestMessage
		c := newTestClient(&fakeTransport{
			getAvailableUnfreezeCount: func(_ context.Context, m *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
				got = m
				return &api.GetAvailableUnfreezeCountResponseMessage{Count: 7}, nil
			},
		})

		count, err := c.GetAvailableUnstakeCount(t.Context(), testAddr)
		require.NoError(t, err)
		require.EqualValues(t, 7, count)
		require.Equal(t, wantOwner, got.GetOwnerAddress())
	})

	t.Run("invalid address", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetAvailableUnstakeCount(t.Context(), "bad!")
		require.Error(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		sentinel := errors.New("boom")
		c := newTestClient(&fakeTransport{
			getAvailableUnfreezeCount: func(context.Context, *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
				return nil, sentinel
			},
		})
		_, err := c.GetAvailableUnstakeCount(t.Context(), testAddr)
		require.ErrorIs(t, err, sentinel)
	})
}

func TestGetWithdrawableUnstakedSendsMillisecondTimestamp(t *testing.T) {
	var got *api.CanWithdrawUnfreezeAmountRequestMessage
	c := newTestClient(&fakeTransport{
		getCanWithdrawUnfreezeAmount: func(_ context.Context, m *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
			got = m
			return &api.CanWithdrawUnfreezeAmountResponseMessage{Amount: 42}, nil
		},
	})

	before := time.Now().UnixMilli()
	amount, err := c.GetWithdrawableUnstaked(t.Context(), testAddr)
	require.NoError(t, err)
	after := time.Now().UnixMilli()

	require.EqualValues(t, 42, amount)
	// Seconds instead of milliseconds, or an unset zero, would fall outside this range.
	require.GreaterOrEqual(t, got.GetTimestamp(), before)
	require.LessOrEqual(t, got.GetTimestamp(), after)
}

func TestGetStakeInfo(t *testing.T) {
	const hourMs = int64(60 * 60 * 1000)

	// Fixtures are relative to now so the expired/pending split is stable.
	now := time.Now().UnixMilli()

	tests := []struct {
		name    string
		account *core.Account
		want    StakeInfo
	}{
		{
			// Literal mainnet response for an account with no stake: three
			// frozenV2 entries, all zero, BANDWIDTH's type field omitted.
			name: "no stake",
			account: &core.Account{FrozenV2: []*core.Account_FreezeV2{
				{},
				{Type: core.ResourceCode_ENERGY},
				{Type: core.ResourceCode_TRON_POWER},
			}},
			want: StakeInfo{},
		},
		{
			name: "bandwidth type omitted",
			account: &core.Account{FrozenV2: []*core.Account_FreezeV2{
				{Amount: 13_000_000_000},
			}},
			want: StakeInfo{StakedBandwidth: 13_000_000_000, TotalStaked: 13_000_000_000},
		},
		{
			name: "tron power excluded from total",
			account: &core.Account{FrozenV2: []*core.Account_FreezeV2{
				{Type: core.ResourceCode_BANDWIDTH, Amount: 100},
				{Type: core.ResourceCode_ENERGY, Amount: 200},
				{Type: core.ResourceCode_TRON_POWER, Amount: 999},
			}},
			want: StakeInfo{StakedBandwidth: 100, StakedEnergy: 200, TotalStaked: 300},
		},
		{
			name: "pending not expired",
			account: &core.Account{UnfrozenV2: []*core.Account_UnFreezeV2{
				{Type: core.ResourceCode_ENERGY, UnfreezeAmount: 500, UnfreezeExpireTime: now + hourMs},
			}},
			want: StakeInfo{
				UnstakingTotal: 500,
				PendingUnstakes: []PendingUnstake{
					{Resource: ResourceTypeEnergy, Amount: 500, ExpireTime: time.UnixMilli(now + hourMs)},
				},
			},
		},
		{
			name: "pending expired is withdrawable",
			account: &core.Account{UnfrozenV2: []*core.Account_UnFreezeV2{
				{Type: core.ResourceCode_BANDWIDTH, UnfreezeAmount: 500, UnfreezeExpireTime: now - hourMs},
			}},
			want: StakeInfo{
				UnstakingTotal:  500,
				WithdrawableNow: 500,
				PendingUnstakes: []PendingUnstake{
					{Resource: ResourceTypeBandwidth, Amount: 500, ExpireTime: time.UnixMilli(now - hourMs)},
				},
			},
		},
		{
			// One millisecond in the past must already count. Paired with the
			// far-future case below this pins the unit: a seconds-based
			// comparison passes here but fails there.
			name: "millisecond boundary",
			account: &core.Account{UnfrozenV2: []*core.Account_UnFreezeV2{
				{UnfreezeAmount: 1, UnfreezeExpireTime: now - 1},
			}},
			want: StakeInfo{
				UnstakingTotal:  1,
				WithdrawableNow: 1,
				PendingUnstakes: []PendingUnstake{
					{Resource: ResourceTypeBandwidth, Amount: 1, ExpireTime: time.UnixMilli(now - 1)},
				},
			},
		},
		{
			// A seconds-based comparison would read this millisecond value as a
			// far-past second and wrongly call it withdrawable.
			name: "far future not withdrawable",
			account: &core.Account{UnfrozenV2: []*core.Account_UnFreezeV2{
				{UnfreezeAmount: 1, UnfreezeExpireTime: now + 365*24*hourMs},
			}},
			want: StakeInfo{
				UnstakingTotal: 1,
				PendingUnstakes: []PendingUnstake{
					{Resource: ResourceTypeBandwidth, Amount: 1, ExpireTime: time.UnixMilli(now + 365*24*hourMs)},
				},
			},
		},
		{
			name: "zero amount unfreeze entries skipped",
			account: &core.Account{UnfrozenV2: []*core.Account_UnFreezeV2{
				{},
				{Type: core.ResourceCode_ENERGY},
			}},
			want: StakeInfo{},
		},
		{
			name: "mixed",
			account: &core.Account{
				FrozenV2: []*core.Account_FreezeV2{
					{Amount: 1_000},
					{Type: core.ResourceCode_ENERGY, Amount: 2_000},
					{Type: core.ResourceCode_TRON_POWER},
				},
				UnfrozenV2: []*core.Account_UnFreezeV2{
					{Type: core.ResourceCode_ENERGY, UnfreezeAmount: 300, UnfreezeExpireTime: now - hourMs},
					{Type: core.ResourceCode_BANDWIDTH, UnfreezeAmount: 700, UnfreezeExpireTime: now + hourMs},
				},
			},
			want: StakeInfo{
				StakedBandwidth: 1_000,
				StakedEnergy:    2_000,
				TotalStaked:     3_000,
				UnstakingTotal:  1_000,
				WithdrawableNow: 300,
				PendingUnstakes: []PendingUnstake{
					{Resource: ResourceTypeEnergy, Amount: 300, ExpireTime: time.UnixMilli(now - hourMs)},
					{Resource: ResourceTypeBandwidth, Amount: 700, ExpireTime: time.UnixMilli(now + hourMs)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(&fakeTransport{
				getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
					tt.account.Address = a.GetAddress()
					return tt.account, nil
				},
			})

			got, err := c.GetStakeInfo(t.Context(), testAddr)
			require.NoError(t, err)
			require.Equal(t, tt.want, *got)
		})
	}
}

func TestGetStakeInfoErrors(t *testing.T) {
	t.Run("account not found", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getAccount: func(context.Context, *core.Account) (*core.Account, error) { return nil, nil },
		})

		_, err := c.GetStakeInfo(t.Context(), testAddr)
		require.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("invalid address does not reach transport", func(t *testing.T) {
		calls := 0
		c := newTestClient(&fakeTransport{
			getAccount: func(context.Context, *core.Account) (*core.Account, error) {
				calls++
				return &core.Account{}, nil
			},
		})

		_, err := c.GetStakeInfo(t.Context(), "bad!")
		require.Error(t, err)
		require.Zero(t, calls)
	})
}
