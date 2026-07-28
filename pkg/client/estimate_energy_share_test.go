package client

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// A Tron contract can pay for its own calls. consume_user_resource_percent is
// the share the *caller* pays; the owner covers the rest from its staked
// energy, capped per call by origin_energy_limit, and anything the owner cannot
// cover falls back to the caller.
func TestContractEnergyShare(t *testing.T) {
	cases := []struct {
		name            string
		total           decimal.Decimal
		percent         int64 // consume_user_resource_percent
		originLimit     int64
		originAvailable decimal.Decimal
		want            decimal.Decimal
	}{
		{
			// The case from Nile tx e3e1633c…: the owner paid all 14650 and the
			// sender's receipt showed fee 0.
			name: "owner pays everything", total: dec(14650), percent: 0,
			originLimit: 1_000_000_000, originAvailable: dec(227_459_357), want: dec(14650),
		},
		{
			name: "caller pays everything", total: dec(14650), percent: 100,
			originLimit: 1_000_000_000, originAvailable: dec(227_459_357), want: decimal.Zero,
		},
		{
			name: "split down the middle", total: dec(14650), percent: 50,
			originLimit: 1_000_000_000, originAvailable: dec(227_459_357), want: dec(7325),
		},
		{
			// 14650 * 70 / 100 = 10255 exactly; 30% stays with the caller.
			name: "caller pays 30 percent", total: dec(14650), percent: 30,
			originLimit: 1_000_000_000, originAvailable: dec(227_459_357), want: dec(10255),
		},
		{
			// java-tron divides with integer truncation: 101 * 99 / 100 = 99,
			// not 99.99. Rounding up would hand the caller a share of zero that
			// the chain does not give.
			name: "share truncates like the chain", total: dec(101), percent: 1,
			originLimit: 1_000_000_000, originAvailable: dec(1_000_000), want: dec(99),
		},
		{
			// The owner is willing but broke, so its share collapses to what it
			// has and the caller pays the rest.
			name: "owner out of energy", total: dec(14650), percent: 0,
			originLimit: 1_000_000_000, originAvailable: dec(4000), want: dec(4000),
		},
		{
			name: "owner has nothing", total: dec(14650), percent: 0,
			originLimit: 1_000_000_000, originAvailable: decimal.Zero, want: decimal.Zero,
		},
		{
			name: "owner energy negative", total: dec(14650), percent: 0,
			originLimit: 1_000_000_000, originAvailable: dec(-5), want: decimal.Zero,
		},
		{
			// origin_energy_limit caps a single call regardless of the balance,
			// which is what stops one caller draining the owner.
			name: "origin energy limit caps the share", total: dec(14650), percent: 0,
			originLimit: 1000, originAvailable: dec(227_459_357), want: dec(1000),
		},
		{
			name: "origin energy limit zero", total: dec(14650), percent: 0,
			originLimit: 0, originAvailable: dec(227_459_357), want: decimal.Zero,
		},
		{
			name: "exactly enough", total: dec(14650), percent: 0,
			originLimit: 14650, originAvailable: dec(14650), want: dec(14650),
		},
		{
			name: "one short", total: dec(14650), percent: 0,
			originLimit: 14649, originAvailable: dec(14650), want: dec(14649),
		},
		{
			name: "no energy at all", total: decimal.Zero, percent: 0,
			originLimit: 1_000_000_000, originAvailable: dec(1_000_000), want: decimal.Zero,
		},
		{
			// Percentages outside 0..100 should not produce a share bigger than
			// the call or a negative one.
			name: "percent above 100", total: dec(14650), percent: 150,
			originLimit: 1_000_000_000, originAvailable: dec(1_000_000), want: decimal.Zero,
		},
		{
			name: "percent negative", total: dec(14650), percent: -50,
			originLimit: 1_000_000_000, originAvailable: dec(1_000_000_000), want: dec(14650),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := contractEnergyShare(tc.total, tc.percent, tc.originLimit, tc.originAvailable)
			require.True(t, got.Equal(tc.want), "want %s, got %s", tc.want, got)
			// The owner can never be billed for more than the call used.
			require.True(t, got.LessThanOrEqual(tc.total.Add(decimal.Zero)) || !tc.total.IsPositive(),
				"share %s exceeds the call's total %s", got, tc.total)
		})
	}
}

func TestSenderEnergy(t *testing.T) {
	cases := []struct {
		name     string
		total    decimal.Decimal
		contract decimal.Decimal
		want     decimal.Decimal
	}{
		{"owner pays all", dec(14650), dec(14650), decimal.Zero},
		{"owner pays none", dec(14650), decimal.Zero, dec(14650)},
		{"owner pays part", dec(14650), dec(4000), dec(10650)},
		// A node reporting more subsidy than usage must not produce negative
		// energy, which would turn into a negative fee.
		{"owner share exceeds total", dec(100), dec(500), decimal.Zero},
		{"no energy", decimal.Zero, decimal.Zero, decimal.Zero},
		{"negative total", dec(-5), decimal.Zero, decimal.Zero},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := ResourceUsage{Energy: tc.total, ContractEnergy: tc.contract}
			require.True(t, u.SenderEnergy().Equal(tc.want), "want %s, got %s", tc.want, u.SenderEnergy())
		})
	}
}

// subsidyFake wires a TRC20 estimate against a contract with the given owner
// settings, plus a sender with no resources at all.
func subsidyFake(t *testing.T, owner string, percent, originLimit, originEnergy int64) *Client {
	t.Helper()

	ownerBytes, err := tronutils.DecodeCheck(owner)
	require.NoError(t, err)

	senderBytes, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	return newTestClient(&fakeTransport{
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return okTx(), nil
		},
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:     &api.Return{Result: true},
				EnergyUsed: 14650,
			}, nil
		},
		getContract: func(context.Context, []byte) (*core.SmartContract, error) {
			return &core.SmartContract{
				OriginAddress:              ownerBytes,
				ConsumeUserResourcePercent: percent,
				OriginEnergyLimit:          originLimit,
			}, nil
		},
		getAccountResource: func(_ context.Context, a *core.Account) (*api.AccountResourceMessage, error) {
			// The contract's owner is asked about separately from the sender -
			// except in the self-call case, where they are the same account and
			// the one answer has to serve both roles.
			if string(a.GetAddress()) == string(ownerBytes) {
				return &api.AccountResourceMessage{EnergyLimit: originEnergy, FreeNetLimit: 600}, nil
			}
			require.Equal(t, senderBytes, a.GetAddress())
			return &api.AccountResourceMessage{FreeNetLimit: 600}, nil
		},
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getEnergyFee", Value: 100},
				{Key: "getTransactionFee", Value: 1000},
				{Key: "getCreateAccountFee", Value: 100_000},
				{Key: "getCreateNewAccountFeeInSystemContract", Value: 1_000_000},
			}}, nil
		},
	})
}

// The bug this covers: the estimate billed the sender for the whole call's
// energy, so a transfer the chain settles for nothing was quoted at 1.465 TRX.
func TestEstimateTRC20TransferCreditsTheContractsEnergy(t *testing.T) {
	amount, err := FromTokenUnits(big.NewInt(1_000_000))
	require.NoError(t, err)

	cases := []struct {
		name         string
		owner        string
		percent      int64
		originLimit  int64
		originEnergy int64
		wantContract decimal.Decimal
		wantFee      SUN
	}{
		{
			// Nile USDT: percent 0, owner rich. Receipt for e3e1633c… shows
			// origin_energy_usage 14650 and fee 0.
			name: "owner pays it all", owner: testAddr, percent: 0,
			originLimit: 1_000_000_000, originEnergy: 227_459_357,
			wantContract: dec(14650), wantFee: 0,
		},
		{
			name: "owner pays nothing", owner: testAddr, percent: 100,
			originLimit: 1_000_000_000, originEnergy: 227_459_357,
			wantContract: decimal.Zero, wantFee: SUN(14650 * 100),
		},
		{
			name: "owner is broke, sender pays", owner: testAddr, percent: 0,
			originLimit: 1_000_000_000, originEnergy: 0,
			wantContract: decimal.Zero, wantFee: SUN(14650 * 100),
		},
		{
			name: "half each", owner: testAddr, percent: 50,
			originLimit: 1_000_000_000, originEnergy: 227_459_357,
			wantContract: dec(7325), wantFee: SUN(7325 * 100),
		},
		{
			// Calling one's own contract is a self-call: java-tron skips the
			// split and bills the caller for everything, however generous the
			// contract's settings. Owner and sender are one account here, so its
			// 5000 staked energy covers 5000 of the call either way - but only
			// once, as the sender. Treating it as a subsidy too would credit the
			// same 5000 twice and quote 0.465 TRX instead of 0.965.
			name: "sender owns the contract", owner: testAddr2, percent: 0,
			originLimit: 1_000_000_000, originEnergy: 5000,
			wantContract: decimal.Zero, wantFee: SUN((14650 - 5000) * 100),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := subsidyFake(t, tc.owner, tc.percent, tc.originLimit, tc.originEnergy)

			res, err := c.EstimateTRC20Transfer(context.Background(), testAddr2, testAddr, testAddr, amount)
			require.NoError(t, err)

			// Usage.Energy stays the call's total, matching the receipt's
			// energy_usage_total, so the split is decomposable by the caller.
			require.True(t, res.Usage.Energy.Equal(dec(14650)), "usage energy %s", res.Usage.Energy)
			require.True(t, res.Usage.ContractEnergy.Equal(tc.wantContract),
				"contract energy: want %s, got %s", tc.wantContract, res.Usage.ContractEnergy)
			require.Equal(t, tc.wantFee, res.Charges.Energy)
			require.Equal(t, tc.wantFee, res.Fee)
		})
	}
}

// A TRX transfer never touches a contract, so nothing subsidises it.
func TestEstimateTRXTransferHasNoContractEnergy(t *testing.T) {
	c := newTestClient(&fakeTransport{
		createTransaction: func(context.Context, *core.TransferContract) (*api.TransactionExtention, error) {
			return okTx(), nil
		},
		getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
			return &core.Account{Address: a.GetAddress(), CreateTime: 1}, nil
		},
		getAccountResource: func(context.Context, *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{FreeNetLimit: 600}, nil
		},
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getEnergyFee", Value: 100},
				{Key: "getTransactionFee", Value: 1000},
			}}, nil
		},
	})

	res, err := c.EstimateTRXTransfer(context.Background(), testAddr2, testAddr, SUN(1_000_000))
	require.NoError(t, err)
	require.True(t, res.Usage.ContractEnergy.IsZero())
	require.True(t, res.Usage.SenderEnergy().IsZero())
	require.Zero(t, res.Charges.Energy)
}
