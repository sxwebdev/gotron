package client

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestCreateFakeCreateAccountTransaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tx, err := CreateFakeCreateAccountTransaction(testAddr, testAddr2)
		require.NoError(t, err)
		require.Len(t, tx.GetRawData().GetContract(), 1)
		require.Equal(t, core.Transaction_Contract_AccountCreateContract, tx.RawData.Contract[0].Type)
	})
	t.Run("invalid from", func(t *testing.T) {
		_, err := CreateFakeCreateAccountTransaction("bad!", testAddr2)
		require.Error(t, err)
	})
	t.Run("invalid to", func(t *testing.T) {
		_, err := CreateFakeCreateAccountTransaction(testAddr, "bad!")
		require.Error(t, err)
	})
}

func TestCreateFakeResourceTransaction(t *testing.T) {
	t.Run("delegate", func(t *testing.T) {
		tx, err := CreateFakeResourceTransaction(testAddr, testAddr2, 1, core.ResourceCode_ENERGY, false)
		require.NoError(t, err)
		require.Equal(t, core.Transaction_Contract_DelegateResourceContract, tx.RawData.Contract[0].Type)
	})
	t.Run("reclaim", func(t *testing.T) {
		tx, err := CreateFakeResourceTransaction(testAddr, testAddr2, 1, core.ResourceCode_ENERGY, true)
		require.NoError(t, err)
		require.Equal(t, core.Transaction_Contract_UnDelegateResourceContract, tx.RawData.Contract[0].Type)
	})
	t.Run("invalid address", func(t *testing.T) {
		_, err := CreateFakeResourceTransaction("bad!", testAddr2, 1, core.ResourceCode_ENERGY, false)
		require.Error(t, err)
	})
}

func TestCreateFakeStakeTransaction(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	t.Run("stake", func(t *testing.T) {
		tx, err := CreateFakeStakeTransaction(testAddr, 1_000_000, core.ResourceCode_ENERGY, false)
		require.NoError(t, err)
		require.Len(t, tx.GetRawData().GetContract(), 1)
		require.Equal(t, core.Transaction_Contract_FreezeBalanceV2Contract, tx.RawData.Contract[0].Type)

		// The contract body must match the type tag, otherwise the size estimate
		// is computed over the wrong payload.
		contract := &core.FreezeBalanceV2Contract{}
		require.NoError(t, tx.RawData.Contract[0].Parameter.UnmarshalTo(contract))
		require.Equal(t, wantOwner, contract.GetOwnerAddress())
		require.EqualValues(t, 1_000_000, contract.GetFrozenBalance())
		require.Equal(t, core.ResourceCode_ENERGY, contract.GetResource())
	})

	t.Run("unstake", func(t *testing.T) {
		tx, err := CreateFakeStakeTransaction(testAddr, 2_000_000, core.ResourceCode_BANDWIDTH, true)
		require.NoError(t, err)
		require.Equal(t, core.Transaction_Contract_UnfreezeBalanceV2Contract, tx.RawData.Contract[0].Type)

		contract := &core.UnfreezeBalanceV2Contract{}
		require.NoError(t, tx.RawData.Contract[0].Parameter.UnmarshalTo(contract))
		require.Equal(t, wantOwner, contract.GetOwnerAddress())
		require.EqualValues(t, 2_000_000, contract.GetUnfreezeBalance())
		require.Equal(t, core.ResourceCode_BANDWIDTH, contract.GetResource())
	})

	t.Run("invalid address", func(t *testing.T) {
		_, err := CreateFakeStakeTransaction("bad!", 1, core.ResourceCode_ENERGY, false)
		require.Error(t, err)
	})
}

func TestCreateFakeWithdrawUnstakedTransaction(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		tx, err := CreateFakeWithdrawUnstakedTransaction(testAddr)
		require.NoError(t, err)
		require.Equal(t, core.Transaction_Contract_WithdrawExpireUnfreezeContract, tx.RawData.Contract[0].Type)

		contract := &core.WithdrawExpireUnfreezeContract{}
		require.NoError(t, tx.RawData.Contract[0].Parameter.UnmarshalTo(contract))
		require.Equal(t, wantOwner, contract.GetOwnerAddress())
	})

	t.Run("invalid address", func(t *testing.T) {
		_, err := CreateFakeWithdrawUnstakedTransaction("bad!")
		require.Error(t, err)
	})
}

func TestEstimateBandwidthForStakeTransactions(t *testing.T) {
	c := &Client{}

	// EstimateBandwidth mutates tx (it appends a fake signature), so every call
	// needs a freshly built transaction.
	estimate := func(t *testing.T, build func() (*core.Transaction, error)) decimal.Decimal {
		t.Helper()
		tx, err := build()
		require.NoError(t, err)
		bw, err := c.EstimateBandwidth(tx)
		require.NoError(t, err)
		require.True(t, bw.GreaterThan(decimal.NewFromInt(64)), "got %s", bw)
		return bw
	}

	small := estimate(t, func() (*core.Transaction, error) {
		return CreateFakeStakeTransaction(testAddr, 1, core.ResourceCode_ENERGY, false)
	})
	large := estimate(t, func() (*core.Transaction, error) {
		return CreateFakeStakeTransaction(testAddr, 1_000_000_000_000, core.ResourceCode_ENERGY, false)
	})
	estimate(t, func() (*core.Transaction, error) {
		return CreateFakeWithdrawUnstakedTransaction(testAddr)
	})

	// The amount is a varint, so a larger one must serialize wider. A builder
	// that dropped the contract body would produce identical sizes here.
	require.True(t, large.GreaterThan(small), "large %s must exceed small %s", large, small)
}

func TestEstimateBandwidth(t *testing.T) {
	c := &Client{}
	tx, err := CreateFakeCreateAccountTransaction(testAddr, testAddr2)
	require.NoError(t, err)

	bw, err := c.EstimateBandwidth(tx)
	require.NoError(t, err)
	// proto.Size + 64 overhead, must be a sane positive number.
	require.True(t, bw.GreaterThan(decimal.NewFromInt(64)), "got %s", bw)
}
