package client_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

const (
	estimateFromAddress = "TVCEYdpK6o8hBt71h82aUVbgfyyxJNMYfe"

	activatedAddressWithUSDT       = "TFTWNgDBkQ5wQoP8RXpRznnHvAVV8x5jLu"
	estimateToActivatedWithoutUSDT = "TXi3DQDPvDeLBHCJdKye32mWdTUwJdbJqL"
	emptyNotActivatedAddress       = "TWtfgTXy7ycYWu9hBCV62nP7pXnSQM1tTB"

	usdtTRC20Contract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	usdtDecimals      = 6
)

func TestEstimateTransfer_Validation(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	one, err := client.FromTokenUnits(big.NewInt(1))
	require.NoError(t, err)

	t.Run("trx", func(t *testing.T) {
		cases := []struct {
			name     string
			from, to string
			amount   client.SUN
			wantErr  error
		}{
			{"empty from", "", activatedAddressWithUSDT, 1, client.ErrInvalidAddress},
			{"empty to", estimateFromAddress, "", 1, client.ErrInvalidAddress},
			{"zero amount", estimateFromAddress, activatedAddressWithUSDT, 0, client.ErrInvalidAmount},
			{"negative amount", estimateFromAddress, activatedAddressWithUSDT, -1, client.ErrInvalidAmount},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				res, err := c.EstimateTRXTransfer(context.Background(), tc.from, tc.to, tc.amount)
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, res)
			})
		}
	})

	t.Run("trc20", func(t *testing.T) {
		cases := []struct {
			name               string
			from, to, contract string
			amount             client.TokenAmount
			wantErr            error
		}{
			{"empty from", "", activatedAddressWithUSDT, usdtTRC20Contract, one, client.ErrInvalidAddress},
			{"empty to", estimateFromAddress, "", usdtTRC20Contract, one, client.ErrInvalidAddress},
			{"empty contract", estimateFromAddress, activatedAddressWithUSDT, "", one, client.ErrInvalidAddress},
			{"zero amount", estimateFromAddress, activatedAddressWithUSDT, usdtTRC20Contract, client.TokenAmount{}, client.ErrInvalidAmount},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				res, err := c.EstimateTRC20Transfer(context.Background(), tc.from, tc.to, tc.contract, tc.amount)
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, res)
			})
		}
	})
}

func TestEstimateTransfer_TRX(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name              string
		to                string
		wantActivationGT0 bool
	}{
		{name: "to activated with USDT", to: activatedAddressWithUSDT},
		{name: "to activated without USDT", to: estimateToActivatedWithoutUSDT},
		{name: "to not activated", to: emptyNotActivatedAddress, wantActivationGT0: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateTRXTransfer(
				context.Background(),
				estimateFromAddress,
				tc.to,
				client.MustFromTRX(decimal.NewFromInt(1)),
			)

			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Transfer.Bandwidth.IsPositive(), "transfer bandwidth must be > 0, got %s", res.Transfer.Bandwidth.String())
			require.True(t, res.Transfer.Energy.Equal(decimal.Zero), "transfer energy must be 0 for TRX, got %s", res.Transfer.Energy.String())
			require.GreaterOrEqual(t, res.Transfer.Fee, client.SUN(0), "transfer fee must be >= 0")

			if tc.wantActivationGT0 {
				require.GreaterOrEqual(t, res.Activation.Fee, client.MustFromTRX(decimal.NewFromInt(1)),
					"activation fee must be >= 1 TRX for an unactivated address")
			} else {
				require.Zero(t, res.Activation.Fee, "activation fee must be 0 for an activated address")
				require.True(t, res.Activation.Bandwidth.Equal(decimal.Zero),
					"activation bandwidth must be 0 for activated address, got %s", res.Activation.Bandwidth.String())
				require.True(t, res.Activation.Energy.Equal(decimal.Zero),
					"activation energy must be 0 for activated address, got %s", res.Activation.Energy.String())
			}

			require.True(t, res.Total.Bandwidth.Equal(res.Transfer.Bandwidth.Add(res.Activation.Bandwidth)),
				"total bandwidth must equal transfer + activation, got total=%s", res.Total.Bandwidth.String())
			require.True(t, res.Total.Energy.Equal(res.Transfer.Energy.Add(res.Activation.Energy)),
				"total energy must equal transfer + activation, got total=%s", res.Total.Energy.String())
			require.Equal(t, res.Transfer.Fee+res.Activation.Fee, res.Total.Fee,
				"total fee must equal transfer + activation")

			t.Logf(
				"TRX → %s: total=(b=%s e=%s fee=%s) transfer=(b=%s e=%s fee=%s) activation=(b=%s e=%s fee=%s)",
				tc.to,
				res.Total.Bandwidth, res.Total.Energy, res.Total.Fee,
				res.Transfer.Bandwidth, res.Transfer.Energy, res.Transfer.Fee,
				res.Activation.Bandwidth, res.Activation.Energy, res.Activation.Fee,
			)
		})
	}
}

func TestEstimateTransfer_TRC20(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name              string
		to                string
		wantActivationGT0 bool
	}{
		{name: "to activated with USDT", to: activatedAddressWithUSDT},
		{name: "to activated without USDT", to: estimateToActivatedWithoutUSDT},
		{name: "to not activated", to: emptyNotActivatedAddress, wantActivationGT0: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			amount, err := client.FromTokenDecimal(decimal.NewFromInt(1), usdtDecimals)
			require.NoError(t, err)

			res, err := c.EstimateTRC20Transfer(
				context.Background(),
				estimateFromAddress,
				tc.to,
				usdtTRC20Contract,
				amount,
			)

			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Transfer.Bandwidth.IsPositive(), "transfer bandwidth must be > 0, got %s", res.Transfer.Bandwidth.String())
			require.True(t, res.Transfer.Energy.IsPositive(), "transfer energy must be > 0 for TRC20, got %s", res.Transfer.Energy.String())
			require.Greater(t, res.Transfer.Fee, client.SUN(0), "transfer fee must be > 0")

			if tc.wantActivationGT0 {
				require.GreaterOrEqual(t, res.Activation.Fee, client.MustFromTRX(decimal.NewFromInt(1)),
					"activation fee must be >= 1 TRX for an unactivated address")
			} else {
				require.Zero(t, res.Activation.Fee, "activation fee must be 0 for an activated address")
				require.True(t, res.Activation.Bandwidth.Equal(decimal.Zero),
					"activation bandwidth must be 0 for activated address, got %s", res.Activation.Bandwidth.String())
				require.True(t, res.Activation.Energy.Equal(decimal.Zero),
					"activation energy must be 0 for activated address, got %s", res.Activation.Energy.String())
			}

			require.True(t, res.Total.Bandwidth.Equal(res.Transfer.Bandwidth.Add(res.Activation.Bandwidth)),
				"total bandwidth must equal transfer + activation, got total=%s", res.Total.Bandwidth.String())
			require.True(t, res.Total.Energy.Equal(res.Transfer.Energy.Add(res.Activation.Energy)),
				"total energy must equal transfer + activation, got total=%s", res.Total.Energy.String())
			require.Equal(t, res.Transfer.Fee+res.Activation.Fee, res.Total.Fee,
				"total fee must equal transfer + activation")

			t.Logf(
				"TRC20 → %s: total=(b=%s e=%s fee=%s) transfer=(b=%s e=%s fee=%s) activation=(b=%s e=%s fee=%s)",
				tc.to,
				res.Total.Bandwidth, res.Total.Energy, res.Total.Fee,
				res.Transfer.Bandwidth, res.Transfer.Energy, res.Transfer.Fee,
				res.Activation.Bandwidth, res.Activation.Energy, res.Activation.Fee,
			)
		})
	}
}
