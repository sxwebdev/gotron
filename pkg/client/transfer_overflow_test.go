package client

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// decimal.IntPart() is big.Int.Int64(), which silently keeps only the low 64
// bits. An amount past the int64 ceiling used to be rebuilt as an unrelated
// number - often small, sometimes negative - and then signed and broadcast as a
// perfectly valid transfer of the wrong size.
func TestCreateTransferTransactionRejectsUnrepresentableAmount(t *testing.T) {
	t.Parallel()

	// 1e13 TRX scales to 1e19 SUN; int64 tops out near 9.22e18.
	overflowing := decimal.NewFromInt(10_000_000_000_000)
	require.False(t, overflowing.Mul(decimal.NewFromInt(1e6)).BigInt().IsInt64(),
		"fixture must actually exceed int64 after scaling")

	tests := []struct {
		name   string
		amount decimal.Decimal
	}{
		{"just above int64", decimal.NewFromBigInt(new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1)), -6)},
		{"1e13 TRX", overflowing},
		{"astronomically large", decimal.NewFromBigInt(new(big.Int).Lsh(big.NewInt(1), 200), 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			c := newTestClient(&fakeTransport{
				createTransaction: func(context.Context, *core.TransferContract) (*api.TransactionExtention, error) {
					called = true
					return &api.TransactionExtention{Txid: []byte{0x01}}, nil
				},
			})

			_, err := c.CreateTransferTransaction(t.Context(), testAddr, testAddr2, tt.amount)
			require.ErrorIs(t, err, ErrInvalidAmount)
			require.False(t, called, "no transaction may be built for an unrepresentable amount")
		})
	}
}

// The largest representable amount must still work, and must arrive unchanged.
func TestCreateTransferTransactionAcceptsMaxInt64SUN(t *testing.T) {
	t.Parallel()

	// MaxInt64 SUN expressed in TRX.
	maxTRX := decimal.NewFromBigInt(big.NewInt(math.MaxInt64), -6)

	var gotAmount int64
	c := newTestClient(&fakeTransport{
		createTransaction: func(_ context.Context, ct *core.TransferContract) (*api.TransactionExtention, error) {
			gotAmount = ct.GetAmount()
			return &api.TransactionExtention{Txid: []byte{0x01}}, nil
		},
	})

	_, err := c.CreateTransferTransaction(t.Context(), testAddr, testAddr2, maxTRX)
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64), gotAmount)
}

// Ordinary amounts must keep their existing truncating behaviour: sub-SUN
// precision is dropped, not rejected.
func TestCreateTransferTransactionTruncatesSubSun(t *testing.T) {
	t.Parallel()

	var gotAmount int64
	c := newTestClient(&fakeTransport{
		createTransaction: func(_ context.Context, ct *core.TransferContract) (*api.TransactionExtention, error) {
			gotAmount = ct.GetAmount()
			return &api.TransactionExtention{Txid: []byte{0x01}}, nil
		},
	})

	// 1.0000005 TRX = 1_000_000.5 SUN
	_, err := c.CreateTransferTransaction(t.Context(), testAddr, testAddr2, decimal.RequireFromString("1.0000005"))
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000), gotAmount)
}
