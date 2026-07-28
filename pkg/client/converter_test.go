package client

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Formulas are exercised for correctness and for round-trip consistency.
func TestConvertEnergyRoundTrip(t *testing.T) {
	c := &Client{}

	const (
		limit  = int64(100_000)
		weight = int64(1_000)
		staked = SUN(1_000_000) // 1 TRX
	)

	energy := c.ConvertStakedToEnergy(limit, weight, staked)
	// staked/1e6 / weight * limit = 1/1000*100000 = 100
	if !energy.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("ConvertStakedToEnergy = %s, want 100", energy)
	}

	back := c.ConvertEnergyToStaked(limit, weight, energy)
	if back != staked {
		t.Errorf("ConvertEnergyToStaked = %s, want %s (round-trip)", back, staked)
	}
}

func TestConvertBandwidthRoundTrip(t *testing.T) {
	c := &Client{}

	const (
		limit  = int64(43_200_000_000)
		weight = int64(5_000)
		staked = SUN(1_000_000)
	)

	bw := c.ConvertStakedToBandwidth(weight, limit, staked)
	back := c.ConvertBandwidthToStaked(weight, limit, bw)
	if back != staked {
		t.Errorf("bandwidth round-trip = %s, want %s", back, staked)
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
		{"StakedToEnergy zero weight", func() decimal.Decimal { return c.ConvertStakedToEnergy(100, 0, 1_000_000) }},
		{"EnergyToStaked zero limit", func() decimal.Decimal { return c.ConvertEnergyToStaked(0, 100, decimal.NewFromInt(1)).TRX() }},
		{"StakedToBandwidth zero weight", func() decimal.Decimal { return c.ConvertStakedToBandwidth(0, 100, 1_000_000) }},
		{"BandwidthToStaked zero limit", func() decimal.Decimal { return c.ConvertBandwidthToStaked(100, 0, decimal.NewFromInt(1)).TRX() }},
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

// The Convert*ToStaked pair lands on int64. It used to do so with
// decimal.IntPart, which keeps only the low 64 bits: a large energy or
// bandwidth request came back as an unrelated - and here negative - staked
// amount, which is a nonsensical answer that still looks like one.
func TestConvertToStakedDoesNotWrap(t *testing.T) {
	t.Parallel()

	c := &Client{}
	huge := decimal.RequireFromString("1e30")

	tests := []struct {
		name string
		got  SUN
	}{
		{"energy", c.ConvertEnergyToStaked(1, 1, huge)},
		{"bandwidth", c.ConvertBandwidthToStaked(1, 1, huge)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, SUN(math.MaxInt64), tt.got)
			require.Positive(t, tt.got, "an out-of-range result must never read as a negative stake")
		})
	}
}
