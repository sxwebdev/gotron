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
			require.True(t, res.Usage.Bandwidth.IsPositive(), "transfer bandwidth must be > 0, got %s", res.Usage.Bandwidth.String())
			require.True(t, res.Usage.Energy.Equal(decimal.Zero), "a TRX transfer uses no energy, got %s", res.Usage.Energy.String())

			if tc.wantActivationGT0 {
				require.GreaterOrEqual(t, res.Charges.AccountCreation, client.MustFromTRX(decimal.NewFromInt(1)),
					"creating an account costs at least 1 TRX")
			} else {
				require.Zero(t, res.Charges.AccountCreation, "no account is created for an activated address")
				require.Zero(t, res.Charges.UnstakedCreation, "the surcharge only applies alongside a creation")
			}

			// Fee accounts for every charge and for nothing else.
			require.Equal(t, res.Charges.Total(), res.Fee)
			require.Zero(t, res.Charges.Energy, "a TRX transfer burns no energy")

			// A bandwidth charge and a covering pool are mutually exclusive: the
			// pools are all-or-nothing, so being charged means neither covered it.
			if res.Charges.Bandwidth > 0 {
				require.True(t, res.Usage.Bandwidth.GreaterThan(res.Available.FreeBandwidth))
				require.True(t, res.Usage.Bandwidth.GreaterThan(res.Available.StakedBandwidth))
			}

			t.Logf(
				"TRX → %s: usage=(b=%s e=%s) available=(free=%s staked=%s energy=%s) charges=(b=%s e=%s create=%s unstaked=%s) fee=%s",
				tc.to,
				res.Usage.Bandwidth, res.Usage.Energy,
				res.Available.FreeBandwidth, res.Available.StakedBandwidth, res.Available.StakedEnergy,
				res.Charges.Bandwidth, res.Charges.Energy, res.Charges.AccountCreation, res.Charges.UnstakedCreation,
				res.Fee,
			)
		})
	}
}

func TestEstimateTransfer_TRC20(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name string
		to   string
	}{
		{name: "to activated with USDT", to: activatedAddressWithUSDT},
		{name: "to activated without USDT", to: estimateToActivatedWithoutUSDT},
		{name: "to not activated", to: emptyNotActivatedAddress},
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
			require.True(t, res.Usage.Bandwidth.IsPositive(), "transfer bandwidth must be > 0, got %s", res.Usage.Bandwidth.String())
			require.True(t, res.Usage.Energy.IsPositive(), "a TRC20 transfer must use energy, got %s", res.Usage.Energy.String())

			// A TRC20 balance is contract storage, so no recipient state creates
			// a Tron account and none of them carries a creation fee - not even
			// the unactivated one.
			require.Zero(t, res.Charges.AccountCreation, "a TRC20 transfer creates no account")
			require.Zero(t, res.Charges.UnstakedCreation)

			require.Equal(t, res.Charges.Total(), res.Fee)

			// Energy is additive, so a charge means the staked pool fell short by
			// exactly what is charged - never more, never a rounded-up whole.
			if res.Charges.Energy > 0 {
				require.True(t, res.Usage.Energy.GreaterThan(res.Available.StakedEnergy))
			} else {
				require.True(t, res.Usage.Energy.LessThanOrEqual(res.Available.StakedEnergy))
			}

			t.Logf(
				"TRC20 → %s: usage=(b=%s e=%s) available=(free=%s staked=%s energy=%s) charges=(b=%s e=%s create=%s unstaked=%s) fee=%s",
				tc.to,
				res.Usage.Bandwidth, res.Usage.Energy,
				res.Available.FreeBandwidth, res.Available.StakedBandwidth, res.Available.StakedEnergy,
				res.Charges.Bandwidth, res.Charges.Energy, res.Charges.AccountCreation, res.Charges.UnstakedCreation,
				res.Fee,
			)
		})
	}
}
