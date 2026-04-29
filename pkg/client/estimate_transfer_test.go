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

func TestEstimateTransfer_Validation(t *testing.T) {
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
			res, err := c.EstimateTransfer(context.Background(), tc.from, tc.to, tc.contract, tc.amount, tc.decimals)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.expectMsg)
			require.Nil(t, res)
		})
	}
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
			res, err := c.EstimateTransfer(
				context.Background(),
				estimateFromAddress,
				tc.to,
				client.TrxAssetIdentifier,
				decimal.NewFromInt(1),
				client.TrxDecimals,
			)

			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Transfer.Bandwidth.IsPositive(), "transfer bandwidth must be > 0, got %s", res.Transfer.Bandwidth.String())
			require.True(t, res.Transfer.Energy.Equal(decimal.Zero), "transfer energy must be 0 for TRX, got %s", res.Transfer.Energy.String())
			require.True(t, res.Transfer.Trx.GreaterThanOrEqual(decimal.Zero), "transfer trx must be >= 0, got %s", res.Transfer.Trx.String())

			if tc.wantActivationGT0 {
				require.True(t, res.Activation.Trx.GreaterThanOrEqual(decimal.NewFromInt(1)),
					"activation trx must be >= 1 for unactivated address, got %s", res.Activation.Trx.String())
			} else {
				require.True(t, res.Activation.Trx.Equal(decimal.Zero),
					"activation trx must be 0 for activated address, got %s", res.Activation.Trx.String())
				require.True(t, res.Activation.Bandwidth.Equal(decimal.Zero),
					"activation bandwidth must be 0 for activated address, got %s", res.Activation.Bandwidth.String())
				require.True(t, res.Activation.Energy.Equal(decimal.Zero),
					"activation energy must be 0 for activated address, got %s", res.Activation.Energy.String())
			}

			require.True(t, res.Total.Bandwidth.Equal(res.Transfer.Bandwidth.Add(res.Activation.Bandwidth)),
				"total bandwidth must equal transfer + activation, got total=%s", res.Total.Bandwidth.String())
			require.True(t, res.Total.Energy.Equal(res.Transfer.Energy.Add(res.Activation.Energy)),
				"total energy must equal transfer + activation, got total=%s", res.Total.Energy.String())
			require.True(t, res.Total.Trx.Equal(res.Transfer.Trx.Add(res.Activation.Trx)),
				"total trx must equal transfer + activation, got total=%s", res.Total.Trx.String())

			t.Logf("TRX → %s: total=(b=%s e=%s trx=%s) transfer=(b=%s e=%s trx=%s) activation=(b=%s e=%s trx=%s)",
				tc.to,
				res.Total.Bandwidth, res.Total.Energy, res.Total.Trx,
				res.Transfer.Bandwidth, res.Transfer.Energy, res.Transfer.Trx,
				res.Activation.Bandwidth, res.Activation.Energy, res.Activation.Trx,
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
			res, err := c.EstimateTransfer(
				context.Background(),
				estimateFromAddress,
				tc.to,
				usdtTRC20Contract,
				decimal.NewFromInt(1),
				usdtDecimals,
			)

			require.NoError(t, err)
			require.NotNil(t, res)
			require.True(t, res.Transfer.Bandwidth.IsPositive(), "transfer bandwidth must be > 0, got %s", res.Transfer.Bandwidth.String())
			require.True(t, res.Transfer.Energy.IsPositive(), "transfer energy must be > 0 for TRC20, got %s", res.Transfer.Energy.String())
			require.True(t, res.Transfer.Trx.IsPositive(), "transfer trx must be > 0, got %s", res.Transfer.Trx.String())

			if tc.wantActivationGT0 {
				require.True(t, res.Activation.Trx.GreaterThanOrEqual(decimal.NewFromInt(1)),
					"activation trx must be >= 1 for unactivated address, got %s", res.Activation.Trx.String())
			} else {
				require.True(t, res.Activation.Trx.Equal(decimal.Zero),
					"activation trx must be 0 for activated address, got %s", res.Activation.Trx.String())
				require.True(t, res.Activation.Bandwidth.Equal(decimal.Zero),
					"activation bandwidth must be 0 for activated address, got %s", res.Activation.Bandwidth.String())
				require.True(t, res.Activation.Energy.Equal(decimal.Zero),
					"activation energy must be 0 for activated address, got %s", res.Activation.Energy.String())
			}

			require.True(t, res.Total.Bandwidth.Equal(res.Transfer.Bandwidth.Add(res.Activation.Bandwidth)),
				"total bandwidth must equal transfer + activation, got total=%s", res.Total.Bandwidth.String())
			require.True(t, res.Total.Energy.Equal(res.Transfer.Energy.Add(res.Activation.Energy)),
				"total energy must equal transfer + activation, got total=%s", res.Total.Energy.String())
			require.True(t, res.Total.Trx.Equal(res.Transfer.Trx.Add(res.Activation.Trx)),
				"total trx must equal transfer + activation, got total=%s", res.Total.Trx.String())

			t.Logf("TRC20 → %s: total=(b=%s e=%s trx=%s) transfer=(b=%s e=%s trx=%s) activation=(b=%s e=%s trx=%s)",
				tc.to,
				res.Total.Bandwidth, res.Total.Energy, res.Total.Trx,
				res.Transfer.Bandwidth, res.Transfer.Energy, res.Transfer.Trx,
				res.Activation.Bandwidth, res.Activation.Energy, res.Activation.Trx,
			)
		})
	}
}
