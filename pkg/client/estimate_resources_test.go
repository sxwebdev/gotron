package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// Regression: an EstimateEnergy result with a nil Result must not panic.
func TestEstimateEnergyNilResultDoesNotPanic(t *testing.T) {
	c := newTestClient(&fakeTransport{
		estimateEnergy: func(context.Context, *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
			return &api.EstimateEnergyMessage{}, nil // Result == nil
		},
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("EstimateEnergy panicked on nil Result: %v", r)
		}
	}()

	res, err := c.EstimateEnergy(context.Background(), testAddr, testAddr2, "name()", "", 0, "", 0)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestEstimateEnergyResultCodeError(t *testing.T) {
	c := newTestClient(&fakeTransport{
		estimateEnergy: func(context.Context, *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
			return &api.EstimateEnergyMessage{
				Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("bad call")},
			}, nil
		},
	})
	_, err := c.EstimateEnergy(context.Background(), testAddr, testAddr2, "name()", "", 0, "", 0)
	require.Error(t, err)
	require.ErrorContains(t, err, "bad call")
}
