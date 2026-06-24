package client

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Network-free validation coverage (the package_test sibling needs live nodes).
func TestEstimateTransferValidationUnit(t *testing.T) {
	c := newTestClient(&fakeTransport{})
	ctx := context.Background()

	cases := []struct {
		name                       string
		from, to, contract         string
		amount                     decimal.Decimal
		expect                     string
	}{
		{"empty from", "", testAddr2, TrxAssetIdentifier, decimal.NewFromInt(1), "from address is required"},
		{"empty to", testAddr, "", TrxAssetIdentifier, decimal.NewFromInt(1), "to address is required"},
		{"empty contract", testAddr, testAddr2, "", decimal.NewFromInt(1), "contract address is required"},
		{"zero amount", testAddr, testAddr2, TrxAssetIdentifier, decimal.Zero, "amount must be greater than 0"},
		{"negative amount", testAddr, testAddr2, TrxAssetIdentifier, decimal.NewFromInt(-1), "amount must be greater than 0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.EstimateTransfer(ctx, tc.from, tc.to, tc.contract, tc.amount, TrxDecimals)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.expect)
			require.Nil(t, res)
		})
	}
}
