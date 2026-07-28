package client_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

func TestEstimateActivationFee(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name             string
		from             string
		to               string
		wantFee          client.SUN
		wantBandwidthGT0 bool
		wantZeros        bool
	}{
		{
			name:    "to not activated, from has no own staked bandwidth",
			from:    "TFTWNgDBkQ5wQoP8RXpRznnHvAVV8x5jLu",
			to:      emptyNotActivatedAddress,
			wantFee: client.MustFromTRX(decimal.NewFromFloat(1.1)),
		},
		{
			name:    "to not activated, from has TRX but no staked bandwidth",
			from:    "TXum2J87saPTaTGLwGDeABUF7aDeTGMtyg",
			to:      emptyNotActivatedAddress,
			wantFee: client.MustFromTRX(decimal.NewFromFloat(1.1)),
		},
		{
			name:             "to not activated, from has staked bandwidth and energy",
			from:             "TGs6btfMT4sgKLmNATu3ce2Y9ENeACB65t",
			to:               emptyNotActivatedAddress,
			wantFee:          client.MustFromTRX(decimal.NewFromInt(1)),
			wantBandwidthGT0: true,
		},
		{
			name:      "to already activated",
			from:      estimateFromAddress,
			to:        activatedAddressWithUSDT,
			wantZeros: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateActivationFee(context.Background(), tc.from, tc.to)
			require.NoError(t, err)
			require.NotNil(t, res)

			if tc.wantZeros {
				require.Zero(t, res.Fee, "fee must be 0, got %s", res.Fee)
				require.True(t, res.Bandwidth.Equal(decimal.Zero), "bandwidth must be 0, got %s", res.Bandwidth.String())
				require.True(t, res.Energy.Equal(decimal.Zero), "energy must be 0, got %s", res.Energy.String())
			} else {
				require.Equal(t, tc.wantFee, res.Fee, "fee must be %s, got %s", tc.wantFee, res.Fee)
				require.True(t, res.Energy.Equal(decimal.Zero), "energy must be 0, got %s", res.Energy.String())
				if tc.wantBandwidthGT0 {
					require.True(t, res.Bandwidth.IsPositive(), "bandwidth must be > 0, got %s", res.Bandwidth.String())
				}
			}

			t.Logf("activation %s → %s: bandwidth=%s energy=%s fee=%s", tc.from, tc.to, res.Bandwidth, res.Energy, res.Fee)
		})
	}
}

func TestEstimateSystemContractActivation(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	cases := []struct {
		name             string
		caller           string
		receiver         string
		wantFee          client.SUN
		wantBandwidthGT0 bool
		wantZeros        bool
	}{
		{
			name:     "receiver not activated, caller has no own staked bandwidth",
			caller:   "TFTWNgDBkQ5wQoP8RXpRznnHvAVV8x5jLu",
			receiver: emptyNotActivatedAddress,
			wantFee:  client.MustFromTRX(decimal.NewFromFloat(1.1)),
		},
		{
			name:     "receiver not activated, caller has TRX but no staked bandwidth",
			caller:   "TXum2J87saPTaTGLwGDeABUF7aDeTGMtyg",
			receiver: emptyNotActivatedAddress,
			wantFee:  client.MustFromTRX(decimal.NewFromFloat(1.1)),
		},
		{
			name:             "receiver not activated, caller has staked bandwidth and energy",
			caller:           "TGs6btfMT4sgKLmNATu3ce2Y9ENeACB65t",
			receiver:         emptyNotActivatedAddress,
			wantFee:          client.MustFromTRX(decimal.NewFromInt(1)),
			wantBandwidthGT0: true,
		},
		{
			name:      "receiver already activated",
			caller:    estimateFromAddress,
			receiver:  activatedAddressWithUSDT,
			wantZeros: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateSystemContractActivation(context.Background(), tc.caller, tc.receiver)
			require.NoError(t, err)
			require.NotNil(t, res)

			if tc.wantZeros {
				require.Zero(t, res.Fee, "fee must be 0, got %s", res.Fee)
				require.True(t, res.Bandwidth.Equal(decimal.Zero), "bandwidth must be 0, got %s", res.Bandwidth.String())
				require.True(t, res.Energy.Equal(decimal.Zero), "energy must be 0, got %s", res.Energy.String())
			} else {
				require.Equal(t, tc.wantFee, res.Fee, "fee must be %s, got %s", tc.wantFee, res.Fee)
				require.True(t, res.Energy.Equal(decimal.Zero), "energy must be 0, got %s", res.Energy.String())
				if tc.wantBandwidthGT0 {
					require.True(t, res.Bandwidth.IsPositive(), "bandwidth must be > 0, got %s", res.Bandwidth.String())
				}
			}

			t.Logf("system activation %s → %s: bandwidth=%s energy=%s fee=%s", tc.caller, tc.receiver, res.Bandwidth, res.Energy, res.Fee)
		})
	}
}
