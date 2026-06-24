package client

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

const testAddr2 = "TVCEYdpK6o8hBt71h82aUVbgfyyxJNMYfe"

func TestCreateTransferTransactionValidation(t *testing.T) {
	c := newTestClient(&fakeTransport{})
	ctx := context.Background()

	tests := []struct {
		name           string
		from, to       string
		amount         decimal.Decimal
		expect         string
	}{
		{"empty from", "", testAddr2, decimal.NewFromInt(1), "from address is required"},
		{"empty to", testAddr, "", decimal.NewFromInt(1), "to address is required"},
		{"zero amount", testAddr, testAddr2, decimal.Zero, "amount must be greater than zero"},
		{"negative amount", testAddr, testAddr2, decimal.NewFromInt(-1), "amount must be greater than zero"},
		{"invalid from", "bad!", testAddr2, decimal.NewFromInt(1), ""},
		{"invalid to", testAddr, "bad!", decimal.NewFromInt(1), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.CreateTransferTransaction(ctx, tt.from, tt.to, tt.amount)
			require.Error(t, err)
			if tt.expect != "" {
				require.ErrorContains(t, err, tt.expect)
			}
		})
	}
}

func TestCreateTransferTransactionConvertsToSUN(t *testing.T) {
	var gotAmount int64
	var gotFrom, gotTo []byte
	c := newTestClient(&fakeTransport{
		createTransaction: func(_ context.Context, ct *core.TransferContract) (*api.TransactionExtention, error) {
			gotAmount, gotFrom, gotTo = ct.Amount, ct.OwnerAddress, ct.ToAddress
			return &api.TransactionExtention{Txid: []byte{0x01}}, nil
		},
	})

	_, err := c.CreateTransferTransaction(context.Background(), testAddr, testAddr2, decimal.RequireFromString("1.5"))
	require.NoError(t, err)

	// 1.5 TRX = 1_500_000 SUN
	require.Equal(t, int64(1_500_000), gotAmount)

	wantFrom, _ := tronutils.DecodeCheck(testAddr)
	wantTo, _ := tronutils.DecodeCheck(testAddr2)
	require.Equal(t, wantFrom, gotFrom)
	require.Equal(t, wantTo, gotTo)
}

func TestCreateTransferTransactionEmptyResult(t *testing.T) {
	c := newTestClient(&fakeTransport{
		createTransaction: func(context.Context, *core.TransferContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil // proto.Size == 0
		},
	})
	_, err := c.CreateTransferTransaction(context.Background(), testAddr, testAddr2, decimal.NewFromInt(1))
	require.ErrorIs(t, err, ErrInvalidTransaction)
}

func TestCreateTransferTransactionResultCodeError(t *testing.T) {
	c := newTestClient(&fakeTransport{
		createTransaction: func(context.Context, *core.TransferContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("validation failed")},
			}, nil
		},
	})
	_, err := c.CreateTransferTransaction(context.Background(), testAddr, testAddr2, decimal.NewFromInt(1))
	require.Error(t, err)
	require.ErrorContains(t, err, "validation failed")
}
