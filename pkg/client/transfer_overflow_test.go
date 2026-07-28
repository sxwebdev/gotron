package client

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// CreateTransferTransaction used to take TRX and scale it with decimal.IntPart,
// which keeps only the low 64 bits when the value does not fit: an amount past
// ~9.22e12 TRX was rebuilt as an unrelated number and broadcast as a valid
// transfer of the wrong size.
//
// The amount is now a SUN, so an unrepresentable value cannot be constructed in
// the first place - it fails at units.FromTRX, before any client method is
// reachable. This test pins that boundary end to end.
func TestUnrepresentableTRXNeverReachesATransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		trx  string
	}{
		{name: "just above int64 SUN", trx: "9223372036854.775808"},
		{name: "1e13 TRX", trx: "10000000000000"},
		{name: "astronomically large", trx: "1e30"},
		{name: "finer than one SUN", trx: "0.0000001"},
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

			amount, err := units.FromTRX(decimal.RequireFromString(tt.trx))
			require.Error(t, err, "the amount must be rejected at construction")
			require.ErrorIs(t, err, ErrInvalidAmount)
			require.Zero(t, amount)

			// A caller who ignores that error is left holding the zero value,
			// which the transfer itself refuses - so no transaction is built
			// either way.
			_, err = c.CreateTransferTransaction(t.Context(), testAddr, testAddr2, amount)
			require.ErrorIs(t, err, ErrInvalidAmount)
			require.False(t, called, "no transaction may be built for an unrepresentable amount")
		})
	}
}

// The largest representable amount must still go through unchanged.
func TestCreateTransferTransactionAcceptsMaxInt64SUN(t *testing.T) {
	t.Parallel()

	maxTRX := decimal.NewFromBigInt(big.NewInt(math.MaxInt64), -6)

	amount, err := units.FromTRX(maxTRX)
	require.NoError(t, err)
	require.Equal(t, SUN(math.MaxInt64), amount)

	var gotAmount int64
	c := newTestClient(&fakeTransport{
		createTransaction: func(_ context.Context, ct *core.TransferContract) (*api.TransactionExtention, error) {
			gotAmount = ct.GetAmount()
			return &api.TransactionExtention{Txid: []byte{0x01}}, nil
		},
	})

	_, err = c.CreateTransferTransaction(t.Context(), testAddr, testAddr2, amount)
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64), gotAmount)
}

// Sub-SUN precision is now an error rather than a silent truncation: 1.0000005
// TRX used to become 1_000_000 SUN with the half-SUN quietly dropped.
func TestSubSunPrecisionIsRejectedRatherThanTruncated(t *testing.T) {
	t.Parallel()

	_, err := units.FromTRX(decimal.RequireFromString("1.0000005"))
	require.ErrorContains(t, err, "sub-SUN precision")

	// The representable neighbour is accepted.
	amount, err := units.FromTRX(decimal.RequireFromString("1.000001"))
	require.NoError(t, err)
	require.Equal(t, SUN(1_000_001), amount)
}
