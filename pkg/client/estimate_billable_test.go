package client

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/types/known/anypb"
)

func dec(v int64) decimal.Decimal { return decimal.NewFromInt(v) }

// recipientBytes is the raw form of the address the estimates send to.
func recipientBytes(t *testing.T) []byte {
	t.Helper()

	b, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)
	return b
}

// Bandwidth is charged to one pool or to none: the staked pool for the whole
// transaction, else the free allowance for the whole transaction, else TRX for
// every byte. The two cases this pins down are the ones a "needed minus
// available" formula gets wrong - a transaction larger than both pools is billed
// in full, and two pools that only cover it together cover nothing.
func TestBillableBandwidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		needed          int64
		staked, free    int64
		want            int64
		whyItIsBillable string
	}{
		{name: "free allowance covers it", needed: 267, staked: 0, free: 600, want: 0},
		{name: "staked pool covers it", needed: 267, staked: 5_000, free: 0, want: 0},
		{name: "both pools cover it", needed: 267, staked: 5_000, free: 600, want: 0},
		{
			name: "neither pool covers it: the whole transaction is billed",
			// 400+400 would "cover" 500 only if the pools added up. They do not.
			needed: 500, staked: 400, free: 400, want: 500,
			whyItIsBillable: "a partially covered transaction is billed for every byte, not for the shortfall",
		},
		{
			name: "no resources at all", needed: 267, staked: 0, free: 0, want: 267,
		},
		{
			name: "needed exactly equals the free allowance", needed: 600, staked: 0, free: 600, want: 0,
			whyItIsBillable: "the boundary is inclusive - the pool must cover it, not exceed it",
		},
		{
			name: "needed exactly equals the staked pool", needed: 600, staked: 600, free: 0, want: 0,
		},
		{
			name: "one byte past both pools", needed: 601, staked: 600, free: 600, want: 601,
		},
		{name: "nothing needed", needed: 0, staked: 0, free: 0, want: 0},
		{name: "nothing needed, pools present", needed: 0, staked: 100, free: 100, want: 0},
		{
			name: "negative pools are not a discount", needed: 100, staked: -50, free: -50, want: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := billableBandwidth(dec(tt.needed), dec(tt.staked), dec(tt.free))
			require.True(t, dec(tt.want).Equal(got),
				"billableBandwidth(%d, staked=%d, free=%d) = %s, want %d. %s",
				tt.needed, tt.staked, tt.free, got, tt.want, tt.whyItIsBillable)
		})
	}
}

// Energy is additive - the staked pool covers what it can and the rest is bought
// with TRX - so here the shortfall really is what gets billed.
func TestBillableEnergy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		needed, available int64
		want              int64
	}{
		{name: "fully covered", needed: 30_000, available: 100_000, want: 0},
		{name: "partially covered: only the shortfall is billed", needed: 130_000, available: 100_000, want: 30_000},
		{name: "not covered at all", needed: 130_000, available: 0, want: 130_000},
		{name: "needed exactly equals available", needed: 100_000, available: 100_000, want: 0},
		{name: "one unit past available", needed: 100_001, available: 100_000, want: 1},
		{name: "nothing needed", needed: 0, available: 0, want: 0},
		{name: "nothing needed, pool present", needed: 0, available: 100_000, want: 0},
		{name: "negative pool is not a surcharge", needed: 100, available: -50, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := billableEnergy(dec(tt.needed), dec(tt.available))
			require.True(t, dec(tt.want).Equal(got),
				"billableEnergy(%d, available=%d) = %s, want %d", tt.needed, tt.available, got, tt.want)
		})
	}
}

// estimateFake wires a Client whose transport answers every call an estimate
// makes, so the whole path runs without a node.
type estimateFake struct {
	freeNetLimit, freeNetUsed int64
	netLimit, netUsed         int64
	energyLimit, energyUsed   int64
	recipientActivated        bool
	energyUsedByCall          int64
}

func (f estimateFake) client(t *testing.T) *Client {
	t.Helper()

	return newTestClient(&fakeTransport{
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getTransactionFee", Value: 1_000},
				{Key: "getEnergyFee", Value: 100},
				{Key: "getCreateAccountFee", Value: 100_000},
				{Key: "getCreateNewAccountFeeInSystemContract", Value: 1_000_000},
				{Key: "getTotalEnergyCurrentLimit", Value: 180_000_000_000},
			}}, nil
		},
		getAccountResource: func(context.Context, *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{
				FreeNetLimit:      f.freeNetLimit,
				FreeNetUsed:       f.freeNetUsed,
				NetLimit:          f.netLimit,
				NetUsed:           f.netUsed,
				EnergyLimit:       f.energyLimit,
				EnergyUsed:        f.energyUsed,
				TotalNetLimit:     43_200_000_000,
				TotalNetWeight:    68_365_362_784,
				TotalEnergyWeight: 2_438_540_483,
			}, nil
		},
		getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
			// GetAccount reports "not found" by answering with an address that
			// does not match the request, which is how the node signals an
			// unactivated account. Only the recipient may look unactivated, so
			// the fake has to answer per address rather than per flag.
			if !f.recipientActivated && bytes.Equal(a.GetAddress(), recipientBytes(t)) {
				return &core.Account{}, nil
			}
			return &core.Account{Address: a.GetAddress(), Balance: 1, CreateTime: 1}, nil
		},
		createTransaction: func(_ context.Context, ct *core.TransferContract) (*api.TransactionExtention, error) {
			return okTransferTx(ct), nil
		},
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
				EnergyUsed:  f.energyUsedByCall,
			}, nil
		},
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
			}, nil
		},
	})
}

func okTransferTx(ct *core.TransferContract) *api.TransactionExtention {
	param, _ := anypb.New(ct)
	return &api.TransactionExtention{
		Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
		Txid:   []byte{0x01},
		Transaction: &core.Transaction{RawData: &core.TransactionRaw{
			Timestamp: 1,
			Contract: []*core.Transaction_Contract{{
				Type:      core.Transaction_Contract_TransferContract,
				Parameter: param,
			}},
		}},
	}
}

// The scenario the whole billable calculation started from: on Nile a 267-byte
// transfer to an uncreated address, from an account with 600 free bandwidth and
// no stake, was reported at 1.367 TRX while the chain takes 1.1. The 0.267 TRX
// difference is the entire bandwidth cost, charged for a transaction that is
// never billed by the byte.
func TestNileScenarioCostsWhatTheChainTakes(t *testing.T) {
	t.Parallel()

	c := estimateFake{freeNetLimit: 600, recipientActivated: false}.client(t)

	res, err := c.EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	require.True(t, res.Usage.Bandwidth.IsPositive(), "the transfer really does consume bandwidth")
	require.Equal(t, SUN(0), res.Charges.Bandwidth,
		"none of which is billed: creating an account is settled by the flat fee")
	require.Equal(t, SUN(0), res.Charges.Energy)

	// Itemised, so a caller can say which part is the flat creation fee and
	// which is the surcharge for having no stake.
	require.Equal(t, MustFromTRX(decimal.NewFromInt(1)), res.Charges.AccountCreation)
	require.Equal(t, MustFromTRX(decimal.RequireFromString("0.1")), res.Charges.UnstakedCreation)
	require.Equal(t, MustFromTRX(decimal.RequireFromString("1.1")), res.Fee)
}

// The free allowance covering a transfer is a property of the ordinary path, so
// it has to be shown on a recipient that already exists - on the creation path
// the allowance is not consulted at all and a zero bandwidth charge would prove
// nothing.
func TestFreeAllowanceCoversAnOrdinaryTransfer(t *testing.T) {
	t.Parallel()

	c := estimateFake{freeNetLimit: 600, recipientActivated: true}.client(t)

	res, err := c.EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	require.True(t, res.Usage.Bandwidth.IsPositive())
	require.True(t, res.Usage.Bandwidth.LessThanOrEqual(res.Available.FreeBandwidth),
		"the allowance must really cover it, otherwise this passes for the wrong reason")
	require.True(t, res.Available.StakedBandwidth.IsZero(), "and the stake must not be what covers it")

	require.Equal(t, SUN(0), res.Charges.Bandwidth, "bandwidth covered by the allowance is not charged")
	require.Equal(t, SUN(0), res.Fee)
}

// With an activated recipient nothing is created, so a transfer inside the free
// allowance costs nothing at all.
func TestChargesAreZeroWhenTheAllowanceCoversAnActivatedRecipient(t *testing.T) {
	t.Parallel()

	c := estimateFake{freeNetLimit: 600, recipientActivated: true}.client(t)

	res, err := c.EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	require.True(t, res.Usage.Bandwidth.IsPositive(), "the transfer still consumes bandwidth")
	require.Equal(t, SUN(0), res.Fee, "nothing is burned when the allowance covers the transfer")
	require.Equal(t, TransferCharges{}, res.Charges)
}

// An account with no resources pays for every byte.
func TestChargesBandwidthWhenNoPoolCoversIt(t *testing.T) {
	t.Parallel()

	c := estimateFake{recipientActivated: true}.client(t)

	res, err := c.EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	const transactionFee = 1_000
	require.Equal(t, SUN(res.Usage.Bandwidth.IntPart()*transactionFee), res.Charges.Bandwidth)
	require.Equal(t, res.Charges.Bandwidth, res.Fee)
}

// Two pools that only cover the transfer when added together cover nothing.
func TestChargesDoNotAddTheBandwidthPools(t *testing.T) {
	t.Parallel()

	probe, err := estimateFake{recipientActivated: true}.client(t).
		EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	size := probe.Usage.Bandwidth.IntPart()
	require.Greater(t, size, int64(2), "the fixture transfer must be big enough to split")

	// Each pool holds a bit more than half: together more than enough, alone
	// never enough.
	half := size/2 + 1
	res, err := estimateFake{
		freeNetLimit:       half,
		netLimit:           half,
		recipientActivated: true,
	}.client(t).EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	const transactionFee = 1_000
	require.Equal(t, SUN(size*transactionFee), res.Charges.Bandwidth,
		"free %d + staked %d must not cover a %d-byte transfer", half, half, size)
}

// The staked energy pool absorbs part of a contract call and the remainder is
// charged - the additive rule that bandwidth does not follow.
func TestChargesEnergyShortfallForTRC20(t *testing.T) {
	t.Parallel()

	const (
		energyNeeded = 130_000
		energyStaked = 100_000
		energyFee    = 100
	)

	c := estimateFake{
		freeNetLimit:       10_000, // large enough that bandwidth is never charged
		energyLimit:        energyStaked,
		recipientActivated: true,
		energyUsedByCall:   energyNeeded,
	}.client(t)

	amount, err := FromTokenUnits(big.NewInt(1_000_000))
	require.NoError(t, err)

	res, err := c.EstimateTRC20Transfer(t.Context(), testAddr, testAddr2, testAddr, amount)
	require.NoError(t, err)

	require.True(t, res.Usage.Energy.Equal(dec(energyNeeded)))
	require.True(t, res.Available.StakedEnergy.Equal(dec(energyStaked)))
	require.Equal(t, SUN((energyNeeded-energyStaked)*energyFee), res.Charges.Energy,
		"only the energy the stake does not cover is charged")
	require.Equal(t, SUN(0), res.Charges.Bandwidth, "the allowance covers the bandwidth here")
	require.Equal(t, res.Charges.Energy, res.Fee)
}

// Used bandwidth counts against the pool it belongs to.
func TestAvailableRespectsAlreadyUsedBandwidth(t *testing.T) {
	t.Parallel()

	probe, err := estimateFake{recipientActivated: true}.client(t).
		EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)
	size := probe.Usage.Bandwidth.IntPart()

	// The limit alone would cover the transfer; what is left of it does not.
	res, err := estimateFake{
		freeNetLimit:       size + 10,
		freeNetUsed:        11,
		recipientActivated: true,
	}.client(t).EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)

	require.True(t, res.Available.FreeBandwidth.Equal(dec(size-1)), "a spent allowance is not still available")
	require.True(t, res.Charges.Bandwidth > 0, "and therefore no longer covers the transfer")
}

// Fee is exactly the sum of the itemised charges - the property a caller relies
// on to decompose a bill and have it add back up.
func TestFeeIsTheSumOfItsCharges(t *testing.T) {
	t.Parallel()

	fakes := map[string]estimateFake{
		"nothing covered":         {},
		"allowance covers it":     {freeNetLimit: 10_000},
		"stake covers it":         {netLimit: 10_000},
		"activated recipient":     {recipientActivated: true},
		"energy partially staked": {freeNetLimit: 10_000, energyLimit: 50_000, energyUsedByCall: 130_000},
	}

	for name, f := range fakes {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			res, err := f.client(t).EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
			require.NoError(t, err)

			require.Equal(t,
				res.Charges.Bandwidth+res.Charges.Energy+res.Charges.AccountCreation+res.Charges.UnstakedCreation,
				res.Fee)
			require.Equal(t, res.Charges.Total(), res.Fee)
		})
	}
}

// A TRC20 balance lives in the token contract's storage, so moving it never
// creates a Tron account and the account-creation fees never apply. Proved on
// mainnet: TCw7QnN8bZKe1ErEaz8YbHt8ryegAA8Lw3 holds 20000 USDT while getaccount
// answers {}. The estimator used to charge 1.1 TRX to send USDT to exactly such
// an address - the common case of onboarding a new user.
func TestTRC20DoesNotChargeForAccountCreation(t *testing.T) {
	t.Parallel()

	f := estimateFake{freeNetLimit: 10_000, recipientActivated: false, energyUsedByCall: 130_285}
	c := f.client(t)

	amount, err := FromTokenUnits(big.NewInt(1_000_000))
	require.NoError(t, err)

	res, err := c.EstimateTRC20Transfer(t.Context(), testAddr, testAddr2, testAddr, amount)
	require.NoError(t, err)

	require.Equal(t, SUN(0), res.Charges.AccountCreation, "a TRC20 transfer creates no account")
	require.Equal(t, SUN(0), res.Charges.UnstakedCreation, "and so carries no creation surcharge")

	// The recipient really is unactivated - otherwise this passes for the wrong
	// reason - and the same recipient does cost 1.1 TRX to receive TRX.
	activated, err := c.IsAccountActivated(t.Context(), testAddr2)
	require.NoError(t, err)
	require.False(t, activated)

	trx, err := c.EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
	require.NoError(t, err)
	require.Equal(t, MustFromTRX(decimal.RequireFromString("1.1")), trx.Fee,
		"the same recipient costs 1.1 TRX to receive TRX")
}

// The cost of a fresh TRC20 recipient is energy, not a fee, and it arrives
// through the constant call rather than through any activation logic.
func TestTRC20FreshRecipientCostsEnergyNotAFee(t *testing.T) {
	t.Parallel()

	// Measured on mainnet USDT: an address that already holds the token against
	// one that does not. The gap is the new storage slot.
	const (
		energyExistingHolder = 64_285
		energyNewHolder      = 130_285
		energyFee            = 100
	)

	amount, err := FromTokenUnits(big.NewInt(1_000_000))
	require.NoError(t, err)

	estimate := func(energyUsed int64) *EstimateTransferResult {
		c := estimateFake{freeNetLimit: 10_000, recipientActivated: false, energyUsedByCall: energyUsed}.client(t)
		res, err := c.EstimateTRC20Transfer(t.Context(), testAddr, testAddr2, testAddr, amount)
		require.NoError(t, err)
		return res
	}

	existing, fresh := estimate(energyExistingHolder), estimate(energyNewHolder)

	require.True(t, fresh.Usage.Energy.Sub(existing.Usage.Energy).Equal(dec(energyNewHolder-energyExistingHolder)),
		"the extra cost of a fresh recipient must show up as energy")
	require.Equal(t, SUN((energyNewHolder-energyExistingHolder)*energyFee), fresh.Fee-existing.Fee,
		"and must be charged as energy, since the sender has no staked energy")
	require.Equal(t, SUN(0), fresh.Charges.AccountCreation)
}

// A contract call that creates an account is charged for it in energy, not in
// the system-contract fees, so the estimate must pass that energy through and
// add nothing on top.
//
// Measured against the activator contract TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY on
// mainnet: 8227 energy for a target that already exists, 33227 for a fresh one.
// The 25000 gap is java-tron's NEW_ACCT_CALL. Its on-chain receipt
// (c0c64a67d0453db9e953646ec3c4a91d83e06ab7654e32bc75092ca3054535e0) shows
// energy_usage 33227, net_usage 313 and no fee whatsoever.
func TestContractActivationIsChargedAsEnergyNotAFee(t *testing.T) {
	t.Parallel()

	const (
		energyExistingTarget = 8_227
		energyFreshTarget    = 33_227
		newAccountCall       = 25_000
		energyFee            = 100
	)

	require.Equal(t, newAccountCall, energyFreshTarget-energyExistingTarget,
		"the fixture must really differ by NEW_ACCT_CALL")

	amount, err := FromTokenUnits(big.NewInt(1_000_000))
	require.NoError(t, err)

	estimate := func(energyUsed int64) *EstimateTransferResult {
		// Unactivated recipient: the case where a fee would wrongly be added.
		c := estimateFake{freeNetLimit: 10_000, recipientActivated: false, energyUsedByCall: energyUsed}.client(t)
		res, err := c.EstimateTRC20Transfer(t.Context(), testAddr, testAddr2, testAddr, amount)
		require.NoError(t, err)
		return res
	}

	existing, fresh := estimate(energyExistingTarget), estimate(energyFreshTarget)

	require.True(t, fresh.Usage.Energy.Sub(existing.Usage.Energy).Equal(dec(newAccountCall)),
		"creating the account must surface as 25000 energy of usage")
	require.Equal(t, SUN(newAccountCall*energyFee), fresh.Fee-existing.Fee,
		"and must be paid for as energy")
	require.Equal(t, SUN(0), fresh.Charges.AccountCreation,
		"the system-contract creation fee does not apply to a contract call")
	require.Equal(t, SUN(0), fresh.Charges.UnstakedCreation)
}

// Creating an account does not follow the ordinary bandwidth rules: the chain
// tries the staked pool alone and otherwise charges a flat fee, so the free
// allowance neither helps nor matters and the per-byte burn never applies.
func TestCreateAccountNeedsFee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		needed int64
		staked int64
		want   bool
	}{
		{name: "staked covers it", needed: 267, staked: 5_000, want: false},
		{name: "staked exactly covers it", needed: 267, staked: 267, want: false},
		{name: "staked one short", needed: 267, staked: 266, want: true},
		{name: "no stake at all", needed: 267, staked: 0, want: true},
		{name: "negative staked pool", needed: 267, staked: -10, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, createAccountNeedsFee(dec(tt.needed), dec(tt.staked)))
		})
	}
}

// The defect: an account-creating transfer whose bandwidth neither pool covers
// was billed for every byte on top of the creation fees. The chain never does
// that - it settles the whole transaction with the flat 0.1 TRX already counted
// as UnstakedCreation.
//
// Verified on mainnet over 12 account-creating transfers: senders with NetLimit
// 0 paid exactly 100000 SUN of net_fee and nothing per byte, senders with staked
// bandwidth paid nothing beyond the 1 TRX, and no sample paid both.
func TestAccountCreationIsNeverBilledPerByte(t *testing.T) {
	t.Parallel()

	oneTRX := MustFromTRX(decimal.NewFromInt(1))
	pointOne := MustFromTRX(decimal.RequireFromString("0.1"))

	tests := []struct {
		name                 string
		freeNetLimit         int64
		netLimit             int64
		wantUnstakedCreation SUN
	}{
		{
			// The case that used to add a per-byte burn: neither pool covers the
			// transfer, so the old code charged bandwidth as well as the fees.
			name:         "neither pool covers the transfer",
			freeNetLimit: 100, netLimit: 0, wantUnstakedCreation: pointOne,
		},
		{
			// A full free allowance does not spare the surcharge: creation reads
			// the staked pool only.
			name:         "free allowance is ample but there is no stake",
			freeNetLimit: 10_000, netLimit: 0, wantUnstakedCreation: pointOne,
		},
		{
			name:         "staked bandwidth covers the creation",
			freeNetLimit: 0, netLimit: 10_000, wantUnstakedCreation: 0,
		},
		{
			name:         "no resources at all",
			freeNetLimit: 0, netLimit: 0, wantUnstakedCreation: pointOne,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := estimateFake{
				freeNetLimit:       tt.freeNetLimit,
				netLimit:           tt.netLimit,
				recipientActivated: false,
			}.client(t)

			res, err := c.EstimateTRXTransfer(t.Context(), testAddr, testAddr2, MustFromTRX(decimal.NewFromInt(1)))
			require.NoError(t, err)

			require.True(t, res.Usage.Bandwidth.IsPositive(), "the transfer must really consume bandwidth")
			require.Equal(t, SUN(0), res.Charges.Bandwidth,
				"an account-creating transfer is never billed by the byte")
			require.Equal(t, oneTRX, res.Charges.AccountCreation)
			require.Equal(t, tt.wantUnstakedCreation, res.Charges.UnstakedCreation)
			require.Equal(t, oneTRX+tt.wantUnstakedCreation, res.Fee)
		})
	}
}
