package units_test

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/units"
)

func TestFromTRX(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		trx     string
		want    units.SUN
		wantErr string
	}{
		{name: "whole", trx: "5", want: 5_000_000},
		{name: "fractional", trx: "1.5", want: 1_500_000},
		{name: "one sun", trx: "0.000001", want: 1},
		{name: "zero", trx: "0", want: 0},
		{name: "negative", trx: "-1.5", want: -1_500_000},
		{
			// int64 tops out near 9.22e18 SUN, i.e. ~9.22e12 TRX.
			name: "max representable",
			trx:  "9223372036854.775807",
			want: 9_223_372_036_854_775_807,
		},
		{name: "sub-SUN precision", trx: "0.0000001", wantErr: "sub-SUN precision"},
		{name: "above int64", trx: "10000000000000", wantErr: "does not fit in int64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := units.FromTRX(decimal.RequireFromString(tt.trx))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSUNRoundTrip(t *testing.T) {
	t.Parallel()

	for _, raw := range []units.SUN{0, 1, 1_500_000, -1_500_000, 9_223_372_036_854_775_807} {
		got, err := units.FromTRX(raw.TRX())
		require.NoError(t, err)
		require.Equal(t, raw, got, "SUN -> TRX -> SUN must be lossless")
	}
}

func TestSUNAccessors(t *testing.T) {
	t.Parallel()

	s := units.SUN(1_500_000)
	require.Equal(t, "1.5", s.TRX().String())
	require.Equal(t, int64(1_500_000), s.Int64())
	require.Equal(t, "1.5 TRX", s.String())
}

func TestMustFromTRXPanicsOnUnrepresentable(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		units.MustFromTRX(decimal.RequireFromString("0.0000001"))
	})
	require.NotPanics(t, func() {
		require.Equal(t, units.SUN(1_500_000), units.MustFromTRX(decimal.RequireFromString("1.5")))
	})
}

func TestFromTokenUnits(t *testing.T) {
	t.Parallel()

	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	tests := []struct {
		name    string
		amount  *big.Int
		wantErr string
	}{
		{name: "zero", amount: big.NewInt(0)},
		{name: "positive", amount: big.NewInt(1_000_000)},
		{name: "max uint256", amount: maxUint256},
		{name: "nil", amount: nil, wantErr: "required"},
		{name: "negative", amount: big.NewInt(-1), wantErr: "negative"},
		{name: "wider than uint256", amount: new(big.Int).Lsh(big.NewInt(1), 256), wantErr: "needs 33 bytes, an ABI uint256 holds 32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := units.FromTokenUnits(tt.amount)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 0, got.TokenUnits().Cmp(tt.amount))
		})
	}
}

// The constructor must copy, otherwise a caller mutating its big.Int would
// change an amount that has already been validated.
func TestFromTokenUnitsCopies(t *testing.T) {
	t.Parallel()

	raw := big.NewInt(100)
	amount, err := units.FromTokenUnits(raw)
	require.NoError(t, err)

	raw.SetInt64(999)
	require.Equal(t, "100", amount.String())

	amount.TokenUnits().SetInt64(777)
	require.Equal(t, "100", amount.String())
}

func TestFromTokenDecimal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   string
		decimals int32
		want     string
		wantErr  string
	}{
		{name: "usdt one", amount: "1", decimals: 6, want: "1000000"},
		{name: "usdt fractional", amount: "1.5", decimals: 6, want: "1500000"},
		{name: "smallest unit", amount: "0.000001", decimals: 6, want: "1"},
		{name: "zero decimals whole", amount: "5", decimals: 0, want: "5"},
		{name: "eighteen decimals", amount: "1", decimals: 18, want: "1000000000000000000"},
		{name: "zero", amount: "0", decimals: 6, want: "0"},
		// Truncating here is what made a 0.5 transfer send nothing.
		{name: "finer than token", amount: "0.5", decimals: 0, wantErr: "finer than the token's 0 decimals"},
		{name: "one digit too fine", amount: "0.0000001", decimals: 6, wantErr: "finer than the token's 6 decimals"},
		{name: "negative", amount: "-1", decimals: 6, wantErr: "negative"},
		{name: "negative decimals", amount: "1", decimals: -1, wantErr: "token decimals -1 is outside 0..78"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := units.FromTokenDecimal(decimal.RequireFromString(tt.amount), tt.decimals)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
			require.True(t, got.Decimal(tt.decimals).Equal(decimal.RequireFromString(tt.amount)))
		})
	}
}

func TestTokenAmountZeroValue(t *testing.T) {
	t.Parallel()

	var a units.TokenAmount
	require.True(t, a.IsZero())
	require.False(t, a.IsPositive())
	require.Equal(t, "0", a.String())
	require.Equal(t, int64(0), a.TokenUnits().Int64())
	require.Equal(t, "0", a.Decimal(6).String())
}

// The resource pricing parameters are quoted in SUN per unit, so the conversion
// multiplies. The previous implementation divided by transactionFee, which
// agrees only while the parameter equals 1000 - the value it happens to have
// today. Any governance change to it would have silently skewed every estimate,
// and the old round-trip test could not catch that because TRX.ToBandwidth was
// wrong in exactly the inverse way.
func TestResourcePricing(t *testing.T) {
	t.Parallel()

	t.Run("bandwidth at the current chain parameter", func(t *testing.T) {
		t.Parallel()
		// 5000 bytes at 1000 SUN/byte = 5_000_000 SUN = 5 TRX
		require.Equal(t, units.SUN(5_000_000), units.NewBandwidth(decimal.NewFromInt(5_000)).ToSUN(1_000))
	})

	t.Run("bandwidth at a different fee", func(t *testing.T) {
		t.Parallel()
		// 5000 bytes at 10 SUN/byte = 50_000 SUN. Dividing would yield 500 TRX.
		require.Equal(t, units.SUN(50_000), units.NewBandwidth(decimal.NewFromInt(5_000)).ToSUN(10))
	})

	t.Run("energy", func(t *testing.T) {
		t.Parallel()
		// 50_000 energy at 100 SUN/energy = 5_000_000 SUN = 5 TRX
		require.Equal(t, units.SUN(5_000_000), units.NewEnergy(decimal.NewFromInt(50_000)).ToSUN(100))
	})

	t.Run("fractional costs round up so an estimate is never short", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, units.SUN(3), units.NewBandwidth(decimal.RequireFromString("2.5")).ToSUN(1))
		require.Equal(t, units.SUN(3), units.NewEnergy(decimal.RequireFromString("2.1")).ToSUN(1))
	})

	t.Run("zero fee", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, units.SUN(0), units.NewBandwidth(decimal.NewFromInt(5_000)).ToSUN(0))
		require.Equal(t, units.SUN(0), units.NewEnergy(decimal.NewFromInt(5_000)).ToSUN(0))
	})

	t.Run("accessors", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "5000", units.NewBandwidth(decimal.NewFromInt(5_000)).ToDecimal().String())
		require.Equal(t, "5000", units.NewEnergy(decimal.NewFromInt(5_000)).ToDecimal().String())
	})
}

// decimal.IntPart is big.Int.Int64: it keeps the low 64 bits of anything that
// does not fit, so an out-of-range price came back as an unrelated - and for
// half the inputs negative - amount. A negative fee estimate or a negative
// "you must stake this much" reads like a real answer and propagates.
func TestCeilToSUNSaturatesInsteadOfWrapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want units.SUN
	}{
		{"within range", "1000000", 1_000_000},
		{"rounds up", "0.1", 1},
		{"exactly max", "9223372036854775807", math.MaxInt64},
		{"one past max", "9223372036854775808", math.MaxInt64},
		{"far past max", "1e30", math.MaxInt64},
		{"exactly min", "-9223372036854775808", math.MinInt64},
		{"far below min", "-1e30", math.MinInt64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, units.CeilToSUN(decimal.RequireFromString(tt.in)))
		})
	}
}

// The pricing helpers go through CeilToSUN, so an absurd resource amount must
// not come back as a small or negative cost.
func TestResourcePricingDoesNotWrap(t *testing.T) {
	t.Parallel()

	huge := decimal.RequireFromString("1e30")
	require.Equal(t, units.SUN(math.MaxInt64), units.NewEnergy(huge).ToSUN(1))
	require.Equal(t, units.SUN(math.MaxInt64), units.NewBandwidth(huge).ToSUN(1))
}

// decimals is whatever the token contract reports, so it is attacker-controlled
// in practice. Without a bound, Shift+BigInt materialises an integer with that
// many digits - and the rejection message used to print every one of them -
// before any check can run.
func TestFromTokenDecimalBoundsDecimals(t *testing.T) {
	t.Parallel()

	t.Run("absurd decimals are rejected cheaply", func(t *testing.T) {
		t.Parallel()

		start := time.Now()
		_, err := units.FromTokenDecimal(decimal.NewFromInt(1), 1_000_000_000)
		require.ErrorIs(t, err, units.ErrInvalidAmount)
		require.Less(t, time.Since(start), time.Second, "the bound must be checked before the value is scaled")
		require.Less(t, len(err.Error()), 200, "the message must not embed the scaled value")
	})

	t.Run("negative decimals are rejected", func(t *testing.T) {
		t.Parallel()

		_, err := units.FromTokenDecimal(decimal.NewFromInt(1), -1)
		require.ErrorIs(t, err, units.ErrInvalidAmount)
	})

	t.Run("the widest real token still works", func(t *testing.T) {
		t.Parallel()

		// 18 decimals is the practical maximum among real TRC20 tokens.
		amount, err := units.FromTokenDecimal(decimal.RequireFromString("1.5"), 18)
		require.NoError(t, err)
		require.Equal(t, "1500000000000000000", amount.String())
	})
}

// An oversized raw amount is described by its width, not printed: its decimal
// expansion is unbounded and would be the largest thing in the process.
func TestFromTokenUnitsRejectionMessageIsBounded(t *testing.T) {
	t.Parallel()

	huge := new(big.Int).Lsh(big.NewInt(1), 100_000)
	_, err := units.FromTokenUnits(huge)
	require.ErrorIs(t, err, units.ErrInvalidAmount)
	require.Less(t, len(err.Error()), 200)
}

// Every rejection carries the shared sentinel, so a caller matches amount
// failures with one errors.Is no matter which constructor produced them.
func TestAmountErrorsCarryTheSentinel(t *testing.T) {
	t.Parallel()

	_, err := units.FromTRX(decimal.RequireFromString("0.0000001"))
	require.ErrorIs(t, err, units.ErrInvalidAmount)

	_, err = units.FromTRX(decimal.RequireFromString("1e30"))
	require.ErrorIs(t, err, units.ErrInvalidAmount)

	_, err = units.FromTokenUnits(nil)
	require.ErrorIs(t, err, units.ErrInvalidAmount)

	_, err = units.FromTokenUnits(big.NewInt(-1))
	require.ErrorIs(t, err, units.ErrInvalidAmount)

	_, err = units.FromTokenDecimal(decimal.RequireFromString("0.5"), 0)
	require.ErrorIs(t, err, units.ErrInvalidAmount)
}
