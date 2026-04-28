package client_test

import (
	"context"
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

func TestEstimateTransferResources_Validation(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name      string
		from      string
		to        string
		contract  string
		amount    decimal.Decimal
		decimals  int64
		expectMsg string
	}{
		{
			name:      "empty from",
			from:      "",
			to:        activatedAddressWithUSDT,
			contract:  client.TrxAssetIdentifier,
			amount:    decimal.NewFromInt(1),
			decimals:  client.TrxDecimals,
			expectMsg: "from address is required",
		},
		{
			name:      "empty to",
			from:      estimateFromAddress,
			to:        "",
			contract:  client.TrxAssetIdentifier,
			amount:    decimal.NewFromInt(1),
			decimals:  client.TrxDecimals,
			expectMsg: "to address is required",
		},
		{
			name:      "empty contract",
			from:      estimateFromAddress,
			to:        activatedAddressWithUSDT,
			contract:  "",
			amount:    decimal.NewFromInt(1),
			decimals:  client.TrxDecimals,
			expectMsg: "contract address is required",
		},
		{
			name:      "zero amount",
			from:      estimateFromAddress,
			to:        activatedAddressWithUSDT,
			contract:  client.TrxAssetIdentifier,
			amount:    decimal.Zero,
			decimals:  client.TrxDecimals,
			expectMsg: "amount must be greater than 0",
		},
		{
			name:      "negative amount",
			from:      estimateFromAddress,
			to:        activatedAddressWithUSDT,
			contract:  client.TrxAssetIdentifier,
			amount:    decimal.NewFromInt(-1),
			decimals:  client.TrxDecimals,
			expectMsg: "amount must be greater than 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateTransferResources(context.Background(), tc.from, tc.to, tc.contract, tc.amount, tc.decimals)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.expectMsg)
			require.Nil(t, res)
		})
	}
}

func TestEstimateTransferResources_TRX(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name    string
		to      string
		wantErr error
	}{
		{name: "to activated with USDT", to: activatedAddressWithUSDT},
		{name: "to activated without USDT", to: estimateToActivatedWithoutUSDT},
		{name: "to not activated", to: emptyNotActivatedAddress, wantErr: client.ErrAccountNotActivated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateTransferResources(
				context.Background(),
				estimateFromAddress,
				tc.to,
				client.TrxAssetIdentifier,
				decimal.NewFromInt(1),
				client.TrxDecimals,
			)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, res)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Bandwidth.IsPositive(), "bandwidth must be > 0, got %s", res.Bandwidth.String())
			require.True(t, res.Energy.Equal(decimal.Zero), "energy must be 0 for TRX transfer, got %s", res.Energy.String())
			require.True(t, res.Trx.GreaterThanOrEqual(decimal.Zero), "trx must be >= 0, got %s", res.Trx.String())

			t.Logf("TRX → %s: bandwidth=%s energy=%s trx=%s", tc.to, res.Bandwidth, res.Energy, res.Trx)
		})
	}
}

func TestEstimateTransferResources_TRC20(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name    string
		to      string
		wantErr error
	}{
		{name: "to activated with USDT", to: activatedAddressWithUSDT},
		{name: "to activated without USDT", to: estimateToActivatedWithoutUSDT},
		{name: "to not activated", to: emptyNotActivatedAddress, wantErr: client.ErrAccountNotActivated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateTransferResources(
				context.Background(),
				estimateFromAddress,
				tc.to,
				usdtTRC20Contract,
				decimal.NewFromInt(1),
				usdtDecimals,
			)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, res)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Bandwidth.IsPositive(), "bandwidth must be > 0, got %s", res.Bandwidth.String())
			require.True(t, res.Energy.IsPositive(), "energy must be > 0 for TRC20 transfer, got %s", res.Energy.String())
			require.True(t, res.Trx.IsPositive(), "trx must be > 0, got %s", res.Trx.String())

			t.Logf("TRC20 → %s: bandwidth=%s energy=%s trx=%s", tc.to, res.Bandwidth, res.Energy, res.Trx)
		})
	}
}
