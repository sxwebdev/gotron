package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// stateClient answers GetAccount with acc, under the address asked for.
func stateClient(acc *core.Account) *Client {
	return newTestClient(&fakeTransport{
		getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
			acc.Address = a.Address
			return acc, nil
		},
	})
}

// Every stake a limit counts, of either Stake version, own or acquired, and
// nothing else: what the account delegated out, its tron power and a frozen
// supply of a token it issued give it no resources.
func TestGetAccountStateSumsTheStake(t *testing.T) {
	t.Parallel()
	c := stateClient(&core.Account{
		Frozen:       []*core.Account_Frozen{{FrozenBalance: 1_000_000}, {FrozenBalance: 2_000_000}},
		FrozenSupply: []*core.Account_Frozen{{FrozenBalance: 900_000_000}},
		AcquiredDelegatedFrozenBalanceForBandwidth:   4_000_000,
		AcquiredDelegatedFrozenV2BalanceForBandwidth: 8_000_000,
		DelegatedFrozenBalanceForBandwidth:           700_000_000,
		DelegatedFrozenV2BalanceForBandwidth:         710_000_000,
		FrozenV2: []*core.Account_FreezeV2{
			{Type: core.ResourceCode_BANDWIDTH, Amount: 16_000_000},
			{Type: core.ResourceCode_ENERGY, Amount: 32_000_000},
			{Type: core.ResourceCode_TRON_POWER, Amount: 720_000_000},
			{Type: core.ResourceCode_BANDWIDTH, Amount: 64_000_000},
		},
		NetUsage:     11,
		FreeNetUsage: 13,
		AccountResource: &core.Account_AccountResource{
			EnergyUsage:                               17,
			FrozenBalanceForEnergy:                    &core.Account_Frozen{FrozenBalance: 128_000_000},
			AcquiredDelegatedFrozenBalanceForEnergy:   256_000_000,
			AcquiredDelegatedFrozenV2BalanceForEnergy: 512_000_000,
			DelegatedFrozenBalanceForEnergy:           730_000_000,
			DelegatedFrozenV2BalanceForEnergy:         740_000_000,
		},
		TronPower: &core.Account_Frozen{FrozenBalance: 750_000_000},
	})

	s, err := c.GetAccountState(t.Context(), testAddr)
	require.NoError(t, err)
	require.Equal(t, SUN(1_000_000+2_000_000+4_000_000+8_000_000+16_000_000+64_000_000), s.FrozenForBandwidth)
	require.Equal(t, SUN(32_000_000+128_000_000+256_000_000+512_000_000), s.FrozenForEnergy)
	require.Equal(t, int64(11), s.NetUsed)
	require.Equal(t, int64(13), s.FreeNetUsed)
	require.Equal(t, int64(17), s.EnergyUsed)
	require.NotNil(t, s.Account)
}

func TestGetAccountStateErrors(t *testing.T) {
	t.Parallel()
	boom := errors.New("rpc down")
	c := newTestClient(&fakeTransport{
		getAccount: func(context.Context, *core.Account) (*core.Account, error) { return nil, boom },
	})
	_, err := c.GetAccountState(t.Context(), testAddr)
	require.ErrorIs(t, err, boom)

	// No account answers as an empty message: not a state of zeros.
	_, err = newTestClient(&fakeTransport{}).GetAccountState(t.Context(), testAddr)
	require.ErrorIs(t, err, ErrAccountNotFound)
}

// ConsumedAt is the latest of the three consume times, whichever it is.
func TestAccountStateConsumedAt(t *testing.T) {
	t.Parallel()
	const at = int64(1_778_753_271_000)
	for _, tc := range []struct {
		name                string
		staked, free, power int64
		want                int64
	}{
		{"none consumed", 0, 0, 0, 0},
		{"staked latest", at, at - 3000, at - 6000, at},
		{"free latest", at - 3000, at, at - 6000, at},
		{"energy latest", at - 6000, at - 3000, at, at},
		{"only energy", 0, 0, at, at},
		{"only free", 0, at, 0, at},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			acc := &core.Account{LatestConsumeTime: tc.staked, LatestConsumeFreeTime: tc.free}
			if tc.power != 0 {
				acc.AccountResource = &core.Account_AccountResource{LatestConsumeTimeForEnergy: tc.power}
			}
			s, err := stateClient(acc).GetAccountState(t.Context(), testAddr)
			require.NoError(t, err)
			if tc.want == 0 {
				require.True(t, s.ConsumedAt.IsZero(), "ConsumedAt = %v, want zero", s.ConsumedAt)
				return
			}
			require.Equal(t, time.UnixMilli(tc.want), s.ConsumedAt)
		})
	}
}

// A consume stamped with a block's own time proves that block; a millisecond
// later does not. The bounds are the whole of the rule.
func TestAccountStateHoldsBlock(t *testing.T) {
	t.Parallel()
	consumed := time.UnixMilli(1_778_753_271_000)
	s := &AccountState{ConsumedAt: consumed}
	for _, tc := range []struct {
		name  string
		s     *AccountState
		block time.Time
		want  bool
	}{
		{"the block of the consume's head", s, consumed, true},
		{"an earlier block", s, consumed.Add(-time.Millisecond), true},
		{"a block a slot earlier", s, consumed.Add(-3 * time.Second), true},
		{"a later block", s, consumed.Add(time.Millisecond), false},
		{"the account's latest transaction's block", s, consumed.Add(3 * time.Second), false},
		{"an unknown block time", s, time.Time{}, false},
		{"nothing consumed", &AccountState{}, consumed, false},
		{"no state", nil, consumed, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, tc.s.HoldsBlock(tc.block))
		})
	}
}

// Pools computes the limits as ResourceRates.Limit does, in both of its modes,
// and subtracts the usage — unclamped, as GetAccountResource's figures are.
func TestAccountStatePools(t *testing.T) {
	t.Parallel()
	s := &AccountState{
		FrozenForBandwidth: 58_050_200_000 + 25_341_549_771,
		FrozenForEnergy:    49_000_000,
		NetUsed:            1_234,
		FreeNetUsed:        650,
		EnergyUsed:         0,
	}
	for _, harden := range []bool{false, true} {
		r := ResourceRates{
			TotalNetLimit: 43_200_000_000, TotalNetWeight: 68_422_218_480,
			TotalEnergyCurrentLimit: 1, TotalEnergyWeight: 49, Harden: harden,
		}
		staked, free, energy := s.Pools(r, 600)
		require.Equal(t, r.Limit(ResourceTypeBandwidth, s.FrozenForBandwidth)-1_234, staked)
		require.Equal(t, int64(52_651-1_234), staked, "harden %t", harden)
		require.Equal(t, int64(-50), free, "an allowance overspent stays negative")
		// 49 TRX at 1/49 is a unit hardened and none in doubles.
		require.Equal(t, r.Limit(ResourceTypeEnergy, s.FrozenForEnergy), energy)
		require.Equal(t, map[bool]int64{false: 0, true: 1}[harden], energy, "harden %t", harden)
	}
	_, _, energy := (&AccountState{EnergyUsed: 5}).Pools(ResourceRates{}, 0)
	require.Equal(t, int64(-5), energy)
}

// The same account at the same head, read both ways over HTTP: the pools
// GetAccountState gives with the network's totals are the ones
// GetAccountResource reports. The bandwidth stake, staked and acquired, is a
// Nile account's at the totals its node reported the NetLimit below under;
// its energy limit is java-tron's double formula, computed apart.
const (
	stateAccountJSON = `{
		"address": "41a614f803b6fd780986a42c78ec9c7f77e6ded13c",
		"balance": 4200000,
		"net_usage": 1234,
		"free_net_usage": 77,
		"latest_consume_time": 1778753265000,
		"latest_consume_free_time": 1778753268000,
		"net_window_size": 28800000,
		"net_window_optimized": true,
		"account_resource": {
			"energy_usage": 15000,
			"latest_consume_time_for_energy": 1778753271000,
			"energy_window_size": 28800000,
			"energy_window_optimized": true,
			"acquired_delegated_frozenV2_balance_for_energy": 3000000,
			"delegated_frozenV2_balance_for_energy": 9000000000
		},
		"frozenV2": [
			{"amount": 58050200000},
			{"type": "ENERGY", "amount": 10000000000},
			{"type": "TRON_POWER"}
		],
		"acquired_delegated_frozenV2_balance_for_bandwidth": 25341549771,
		"delegated_frozenV2_balance_for_bandwidth": 5000000000
	}`
	stateResourceJSON = `{
		"freeNetUsed": 77,
		"freeNetLimit": 600,
		"NetUsed": 1234,
		"NetLimit": 52651,
		"TotalNetLimit": 43200000000,
		"TotalNetWeight": 68422218480,
		"EnergyUsed": 15000,
		"EnergyLimit": 90586,
		"TotalEnergyLimit": 180000000000,
		"TotalEnergyWeight": 19876543210
	}`
)

func TestAccountStatePoolsMatchAccountResource(t *testing.T) {
	t.Parallel()
	accTr, _ := newStubTransportAtPath(t, "/wallet/getaccount", http.StatusOK, stateAccountJSON)
	resTr, _ := newStubTransportAtPath(t, "/wallet/getaccountresource", http.StatusOK, stateResourceJSON)

	s, err := (&Client{transport: accTr}).GetAccountState(t.Context(), testAddr)
	require.NoError(t, err)
	// The energy time is the latest, read from inside account_resource.
	require.Equal(t, time.UnixMilli(1_778_753_271_000), s.ConsumedAt)
	require.Equal(t, SUN(58_050_200_000+25_341_549_771), s.FrozenForBandwidth)
	require.Equal(t, SUN(10_000_000_000+3_000_000), s.FrozenForEnergy)

	rc := &Client{transport: resTr}
	res, err := rc.GetAccountResource(t.Context(), testAddr)
	require.NoError(t, err)
	r := ResourceRates{
		TotalNetLimit: res.GetTotalNetLimit(), TotalNetWeight: res.GetTotalNetWeight(),
		TotalEnergyCurrentLimit: res.GetTotalEnergyLimit(), TotalEnergyWeight: res.GetTotalEnergyWeight(),
	}
	staked, free, energy := s.Pools(r, res.GetFreeNetLimit())
	require.Equal(t, rc.AvailableBandwidthWithoutFree(res).IntPart(), staked)
	require.Equal(t, rc.AvailableFreeBandwidth(res).IntPart(), free)
	require.Equal(t, rc.AvailableEnergy(res).IntPart(), energy)
	require.Equal(t, int64(52_651-1_234), staked)
	require.Equal(t, int64(600-77), free)
	require.Equal(t, int64(90_586-15_000), energy)
}
