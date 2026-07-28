package client

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/units"
)

// Network-free validation coverage (the package_test sibling needs live nodes).
func TestEstimateTRXTransferValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		from, to string
		amount   SUN
		wantErr  error
	}{
		{"empty from", "", testAddr2, 1, ErrInvalidAddress},
		{"empty to", testAddr, "", 1, ErrInvalidAddress},
		{"zero amount", testAddr, testAddr2, 0, ErrInvalidAmount},
		{"negative amount", testAddr, testAddr2, -1, ErrInvalidAmount},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(&fakeTransport{})
			res, err := c.EstimateTRXTransfer(t.Context(), tc.from, tc.to, tc.amount)
			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, res)
		})
	}
}

func TestEstimateTRC20TransferValidation(t *testing.T) {
	t.Parallel()

	one, err := units.FromTokenUnits(big.NewInt(1))
	require.NoError(t, err)

	cases := []struct {
		name               string
		from, to, contract string
		amount             TokenAmount
		wantErr            error
	}{
		{"empty from", "", testAddr2, testAddr, one, ErrInvalidAddress},
		{"empty to", testAddr, "", testAddr, one, ErrInvalidAddress},
		{"empty contract", testAddr, testAddr2, "", one, ErrInvalidAddress},
		{"zero amount", testAddr, testAddr2, testAddr, TokenAmount{}, ErrInvalidAmount},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(&fakeTransport{})
			res, err := c.EstimateTRC20Transfer(t.Context(), tc.from, tc.to, tc.contract, tc.amount)
			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, res)
		})
	}
}
