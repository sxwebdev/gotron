package units_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
)

func equal(t *testing.T, got, want decimal.Decimal) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestTRX(t *testing.T) {
	trx := units.NewTRX(decimal.NewFromInt(5))

	equal(t, trx.ToDecimal(), decimal.NewFromInt(5))

	// 1 TRX = 1_000_000 SUN
	equal(t, trx.ToSUN(), decimal.NewFromInt(5_000_000))

	// value / energyFee * 1e6 => 5 / 100 * 1e6 = 50_000
	equal(t, trx.ToEnergy(100), decimal.NewFromInt(50_000))

	// value * transactionFee => 5 * 1000 = 5_000
	equal(t, trx.ToBandwidth(1000), decimal.NewFromInt(5_000))
}

func TestEnergy(t *testing.T) {
	e := units.NewEnergy(decimal.NewFromInt(50_000))

	equal(t, e.ToDecimal(), decimal.NewFromInt(50_000))

	// value * energyFee / 1e6 => 50_000 * 100 / 1e6 = 5 ; inverse of TRX.ToEnergy
	equal(t, e.ToTRX(100).ToDecimal(), decimal.NewFromInt(5))
}

func TestBandwidth(t *testing.T) {
	b := units.NewBandwidth(decimal.NewFromInt(5_000))

	equal(t, b.ToDecimal(), decimal.NewFromInt(5_000))

	// value / transactionFee => 5_000 / 1000 = 5 ; inverse of TRX.ToBandwidth
	equal(t, b.ToTRX(1000).ToDecimal(), decimal.NewFromInt(5))
}

// A zero fee must not panic (shopspring/decimal panics on division by zero).
func TestZeroFeeDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic on zero fee: %v", r)
		}
	}()

	if got := units.NewTRX(decimal.NewFromInt(10)).ToEnergy(0); !got.IsZero() {
		t.Errorf("TRX.ToEnergy(0) = %s, want 0", got)
	}
	if got := units.NewBandwidth(decimal.NewFromInt(10)).ToTRX(0); !got.ToDecimal().IsZero() {
		t.Errorf("Bandwidth.ToTRX(0) = %s, want 0", got.ToDecimal())
	}
}
