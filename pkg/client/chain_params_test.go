package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func chainParamsFake() *fakeTransport {
	return &fakeTransport{
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getEnergyFee", Value: 420},
				{Key: "getTransactionFee", Value: 1000},
				{Key: "getTotalEnergyCurrentLimit", Value: 90_000_000_000},
				{Key: "getFreeNetLimit", Value: 600},
				{Key: "getCreateAccountFee", Value: 100_000},
				{Key: "getCreateNewAccountFeeInSystemContract", Value: 1_000_000},
				{Key: "unknownKey", Value: 5}, // must be ignored
			}}, nil
		},
	}
}

func TestChainParams(t *testing.T) {
	c := newTestClient(chainParamsFake())
	p, err := c.ChainParams(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(420), p.EnergyFee)
	require.Equal(t, int64(1000), p.TransactionFee)
	require.Equal(t, int64(90_000_000_000), p.TotalEnergyCurrentLimit)
	require.Equal(t, int64(600), p.FreeNetLimit)
	require.Equal(t, int64(100_000), p.CreateAccountFee)
	require.Equal(t, int64(1_000_000), p.CreateNewAccountFeeInSystemContract)
}

func TestChainParam(t *testing.T) {
	c := newTestClient(chainParamsFake())

	got, err := c.ChainParam(context.Background(), "getEnergyFee")
	require.NoError(t, err)
	require.Equal(t, int64(420), got.Value)

	_, err = c.ChainParam(context.Background(), "doesNotExist")
	require.ErrorContains(t, err, "not found")
}
