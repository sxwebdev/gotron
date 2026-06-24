package client

import (
	"testing"

	"github.com/shopspring/decimal"
)

// Formulas are exercised for correctness and for round-trip consistency.
func TestConvertEnergyRoundTrip(t *testing.T) {
	c := &Client{}

	const (
		limit  = int64(100_000)
		weight = int64(1_000)
		staked = int64(1_000_000) // 1 TRX in SUN
	)

	energy := c.ConvertStackedTRXToEnergy(limit, weight, staked)
	// staked/1e6 / weight * limit = 1/1000*100000 = 100
	if !energy.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("ConvertStackedTRXToEnergy = %s, want 100", energy)
	}

	back := c.ConvertEnergyToStackedTRX(limit, weight, energy)
	if !back.Equal(decimal.NewFromInt(staked)) {
		t.Errorf("ConvertEnergyToStackedTRX = %s, want %d (round-trip)", back, staked)
	}
}

func TestConvertBandwidthRoundTrip(t *testing.T) {
	c := &Client{}

	const (
		limit  = int64(43_200_000_000)
		weight = int64(5_000)
		staked = int64(1_000_000)
	)

	bw := c.ConvertStackedTRXToBandwidth(weight, limit, staked)
	back := c.ConvertBandwidthToStackedTRX(weight, limit, bw)
	if !back.Equal(decimal.NewFromInt(staked)) {
		t.Errorf("bandwidth round-trip = %s, want %d", back, staked)
	}
}

// Regression: a zero weight/limit (e.g. a fresh chain or an empty node
// response) must not panic via decimal division by zero.
func TestConvertZeroDivisorDoesNotPanic(t *testing.T) {
	c := &Client{}

	cases := []struct {
		name string
		fn   func() decimal.Decimal
	}{
		{"StackedTRXToEnergy zero weight", func() decimal.Decimal { return c.ConvertStackedTRXToEnergy(100, 0, 1_000_000) }},
		{"EnergyToStackedTRX zero limit", func() decimal.Decimal { return c.ConvertEnergyToStackedTRX(0, 100, decimal.NewFromInt(1)) }},
		{"StackedTRXToBandwidth zero weight", func() decimal.Decimal { return c.ConvertStackedTRXToBandwidth(0, 100, 1_000_000) }},
		{"BandwidthToStackedTRX zero limit", func() decimal.Decimal { return c.ConvertBandwidthToStackedTRX(100, 0, decimal.NewFromInt(1)) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("unexpected panic: %v", r)
				}
			}()
			if got := tc.fn(); !got.IsZero() {
				t.Errorf("got %s, want 0 for zero divisor", got)
			}
		})
	}
}
