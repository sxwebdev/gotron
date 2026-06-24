package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// Regression: a transaction with a nil Result (common on success) must not
// panic via tx.Result.Code.
func TestTriggerContractNilResultDoesNotPanic(t *testing.T) {
	c := newTestClient(&fakeTransport{
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Transaction: &core.Transaction{}}, nil // Result == nil
		},
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TriggerContract panicked on nil Result: %v", r)
		}
	}()

	// feeLimit 0 so the FeeLimit/UpdateHash branch is skipped.
	_, err := c.TriggerContract(context.Background(), testAddr, testAddr2, "name()", "", 0, 0, "", 0)
	require.NoError(t, err)
}

func TestTriggerContractResultCodeError(t *testing.T) {
	c := newTestClient(&fakeTransport{
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("reverted")},
			}, nil
		},
	})
	_, err := c.TriggerContract(context.Background(), testAddr, testAddr2, "name()", "", 0, 0, "", 0)
	require.Error(t, err)
	require.ErrorContains(t, err, "reverted")
}

func TestGetContract(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getContract: func(context.Context, []byte) (*core.SmartContract, error) { return nil, nil },
		})
		_, err := c.GetContract(context.Background(), testAddr)
		require.ErrorContains(t, err, "contract not found")
	})

	t.Run("invalid address", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetContract(context.Background(), "bad!")
		require.Error(t, err)
	})
}

func TestGetContractABI(t *testing.T) {
	t.Run("no abi", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getContract: func(context.Context, []byte) (*core.SmartContract, error) {
				return &core.SmartContract{}, nil // Abi == nil
			},
		})
		_, err := c.GetContractABI(context.Background(), testAddr)
		require.ErrorContains(t, err, "no ABI")
	})

	t.Run("with abi", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getContract: func(context.Context, []byte) (*core.SmartContract, error) {
				return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
			},
		})
		got, err := c.GetContractABI(context.Background(), testAddr)
		require.NoError(t, err)
		require.NotNil(t, got)
	})
}

func TestDeployContractValidation(t *testing.T) {
	c := newTestClient(&fakeTransport{})
	ctx := context.Background()

	tests := []struct {
		name       string
		from       string
		curPercent int64
		oeLimit    int64
	}{
		{"invalid from", "bad!", 50, 1},
		{"percent too high", testAddr, 101, 1},
		{"percent negative", testAddr, -1, 1},
		{"zero origin energy limit", testAddr, 50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.DeployContract(ctx, tt.from, "name", nil, "00", 0, tt.curPercent, tt.oeLimit)
			require.Error(t, err)
		})
	}
}
