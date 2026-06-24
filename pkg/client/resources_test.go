package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestDelegateResourceValidation(t *testing.T) {
	c := newTestClient(&fakeTransport{})
	ctx := context.Background()

	tests := []struct {
		name              string
		owner, receiver   string
		resource          ResourceType
		balance           int64
	}{
		{"invalid owner", "bad!", testAddr2, ResourceTypeEnergy, 1},
		{"invalid receiver", testAddr, "bad!", ResourceTypeEnergy, 1},
		{"invalid resource", testAddr, testAddr2, ResourceType(9), 1},
		{"zero balance", testAddr, testAddr2, ResourceTypeEnergy, 0},
		{"negative balance", testAddr, testAddr2, ResourceTypeEnergy, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.DelegateResource(ctx, tt.owner, tt.receiver, tt.resource, tt.balance, false, 0)
			require.Error(t, err)
		})
	}
}

func TestDelegateResourceSuccess(t *testing.T) {
	var got *core.DelegateResourceContract
	c := newTestClient(&fakeTransport{
		delegateResource: func(_ context.Context, ct *core.DelegateResourceContract) (*api.TransactionExtention, error) {
			got = ct
			return &api.TransactionExtention{}, nil
		},
	})

	_, err := c.DelegateResource(context.Background(), testAddr, testAddr2, ResourceTypeEnergy, 1_000_000, true, 100)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, core.ResourceCode_ENERGY, got.Resource)
	require.Equal(t, int64(1_000_000), got.Balance)
	require.True(t, got.Lock)
	require.Equal(t, int64(100), got.LockPeriod)
}

func TestReclaimResourceValidationAndSuccess(t *testing.T) {
	t.Run("invalid balance", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.ReclaimResource(context.Background(), testAddr, testAddr2, ResourceTypeBandwidth, 0)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		var got *core.UnDelegateResourceContract
		c := newTestClient(&fakeTransport{
			unDelegateResource: func(_ context.Context, ct *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
				got = ct
				return &api.TransactionExtention{}, nil
			},
		})
		_, err := c.ReclaimResource(context.Background(), testAddr, testAddr2, ResourceTypeBandwidth, 500)
		require.NoError(t, err)
		require.Equal(t, core.ResourceCode_BANDWIDTH, got.Resource)
		require.Equal(t, int64(500), got.Balance)
	})
}

func TestAvailableResourceCalcs(t *testing.T) {
	c := &Client{}
	res := &api.AccountResourceMessage{
		EnergyLimit:  1000,
		EnergyUsed:   400,
		NetLimit:     500,
		NetUsed:      100,
		FreeNetLimit: 600,
		FreeNetUsed:  50,
	}
	require.Equal(t, int64(600), c.AvailableEnergy(res).IntPart())                 // 1000-400
	require.Equal(t, int64(1000), c.TotalEnergyLimit(res).IntPart())               // 1000
	require.Equal(t, int64(950), c.AvailableBandwidth(res).IntPart())              // 500+600-100-50
	require.Equal(t, int64(400), c.AvailableBandwidthWithoutFree(res).IntPart())   // 500-100
	require.Equal(t, int64(1100), c.TotalBandwidthLimit(res).IntPart())            // 500+600
}
