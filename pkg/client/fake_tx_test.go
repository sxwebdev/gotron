package client

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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

func TestEstimateBandwidth(t *testing.T) {
	c := &Client{}
	tx, err := CreateFakeCreateAccountTransaction(testAddr, testAddr2)
	require.NoError(t, err)

	bw, err := c.EstimateBandwidth(tx)
	require.NoError(t, err)
	// proto.Size + 64 overhead, must be a sane positive number.
	require.True(t, bw.GreaterThan(decimal.NewFromInt(64)), "got %s", bw)
}
