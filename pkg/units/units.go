// Package units carries the SDK's amount types.
//
// Tron has two unrelated notions of "amount" and mixing them is the single
// biggest source of bugs when integrating: TRX amounts live on a fixed 1e6
// scale, while TRC20 amounts live on whatever scale the token's own decimals
// declare. Both are represented here by distinct types so the unit is visible
// in every signature and the compiler rejects the mix-up.
package units

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/shopspring/decimal"
)

// SunPerTRX is the fixed scale of TRX: 1 TRX = 1,000,000 SUN.
const SunPerTRX = 1_000_000

// maxTokenBytes is the width of an ABI uint256 word. A larger amount cannot be
// encoded into TRC20 calldata.
const maxTokenBytes = 32

// maxTokenDecimals is the number of decimal digits in the largest ABI uint256,
// and therefore the point past which a token's scale cannot describe a real
// balance. The bound exists because decimals is reported by the token contract
// itself: without it, a contract answering 1e9 would make the conversion below
// materialise a billion-digit integer before anything could reject it.
const maxTokenDecimals = 78

// ErrInvalidAmount marks every amount this package refuses to build. The client
// package re-exports it, so callers match amount failures with a single
// errors.Is regardless of which layer produced them.
var ErrInvalidAmount = errors.New("invalid amount")

// CeilToSUN rounds a SUN-denominated decimal up to a whole SUN, saturating at
// the int64 bounds.
//
// decimal.IntPart is big.Int.Int64, which keeps only the low 64 bits: an
// out-of-range result comes back as an unrelated - frequently negative - number
// that reads like a real amount. Saturating instead keeps an impossible input
// visibly impossible. Rounding up matches the rest of the pricing helpers, so a
// cost estimate is never short.
func CeilToSUN(d decimal.Decimal) SUN {
	i := d.Ceil().BigInt()
	if i.IsInt64() {
		return SUN(i.Int64())
	}
	if i.Sign() < 0 {
		return SUN(math.MinInt64)
	}
	return SUN(math.MaxInt64)
}

// SUN is an amount of TRX in the chain's native unit.
//
// Every TRX-denominated value in the SDK - transfer amounts, balances, staked
// balances, fee limits, rewards - is a SUN. It is the exact type the protocol
// uses, so no rounding happens anywhere between the caller and the wire.
// Convert to and from human-facing TRX only at the edges, via FromTRX and TRX.
type SUN int64

// FromTRX converts a TRX amount to SUN.
//
// It rejects amounts that cannot be represented exactly: more than six decimal
// places (there is no sub-SUN precision on Tron) or a scaled value outside
// int64. The latter matters because decimal.IntPart silently keeps only the low
// 64 bits, which turns an oversized amount into an unrelated - sometimes
// negative - number that still builds and signs into a valid transaction.
func FromTRX(trx decimal.Decimal) (SUN, error) {
	scaled := trx.Mul(decimal.NewFromInt(SunPerTRX))

	if !scaled.IsInteger() {
		return 0, fmt.Errorf("%w: %s TRX has sub-SUN precision, at most 6 decimal places are representable", ErrInvalidAmount, trx)
	}

	sun := scaled.BigInt()
	if !sun.IsInt64() {
		return 0, fmt.Errorf("%w: %s TRX does not fit in int64 SUN", ErrInvalidAmount, trx)
	}

	return SUN(sun.Int64()), nil
}

// MustFromTRX is FromTRX for constants and tests. It panics on values that
// FromTRX rejects, so never use it on caller-supplied input.
func MustFromTRX(trx decimal.Decimal) SUN {
	sun, err := FromTRX(trx)
	if err != nil {
		panic(err)
	}
	return sun
}

// TRX returns the amount as a decimal number of TRX.
func (s SUN) TRX() decimal.Decimal {
	return decimal.New(int64(s), -6)
}

// Int64 returns the raw SUN value for protocol calls.
func (s SUN) Int64() int64 { return int64(s) }

// String renders the amount in TRX, the unit humans read.
func (s SUN) String() string { return s.TRX().String() + " TRX" }

// TokenAmount is an amount of a TRC20 token in the token's own minimal units.
//
// Its scale is set by the token contract's decimals, so it is deliberately not
// interchangeable with SUN: 1_000_000 is one USDT but a millionth of a TRX.
// The zero value is a valid zero amount.
type TokenAmount struct{ v *big.Int }

// FromTokenUnits wraps a raw minimal-unit amount.
//
// It rejects negative values, which big.Int.Bytes would silently encode as a
// large positive transfer, and amounts wider than an ABI uint256 word, which
// common.LeftPadBytes would pass through unpadded and misalign the calldata.
func FromTokenUnits(units *big.Int) (TokenAmount, error) {
	if units == nil {
		return TokenAmount{}, fmt.Errorf("%w: token amount is required", ErrInvalidAmount)
	}
	if units.Sign() < 0 {
		return TokenAmount{}, fmt.Errorf("%w: token amount is negative", ErrInvalidAmount)
	}
	// The oversized value is described, not printed: it can be arbitrarily wide,
	// and its decimal expansion would be the largest thing in the process.
	if size := len(units.Bytes()); size > maxTokenBytes {
		return TokenAmount{}, fmt.Errorf("%w: token amount needs %d bytes, an ABI uint256 holds %d", ErrInvalidAmount, size, maxTokenBytes)
	}

	return TokenAmount{v: new(big.Int).Set(units)}, nil
}

// FromTokenDecimal converts a human-facing token amount to minimal units using
// the token's decimals, which TRC20GetDecimals reports.
//
// It rejects amounts finer than the token can represent rather than truncating
// them: decimal.BigInt would turn 0.5 of a zero-decimal token into a transfer
// of nothing. decimals is bounded because the token contract chooses it, and an
// absurd value would otherwise be scaled into an integer of that many digits
// before any check could run.
func FromTokenDecimal(amount decimal.Decimal, decimals int32) (TokenAmount, error) {
	if decimals < 0 || decimals > maxTokenDecimals {
		return TokenAmount{}, fmt.Errorf("%w: token decimals %d is outside 0..%d", ErrInvalidAmount, decimals, maxTokenDecimals)
	}

	scaled := amount.Shift(decimals)
	if !scaled.IsInteger() {
		return TokenAmount{}, fmt.Errorf("%w: %s is finer than the token's %d decimals", ErrInvalidAmount, amount, decimals)
	}

	return FromTokenUnits(scaled.BigInt())
}

// TokenUnits returns the amount in the token's minimal units. The result is a
// copy, so callers cannot mutate the amount through it.
func (a TokenAmount) TokenUnits() *big.Int {
	if a.v == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(a.v)
}

// Decimal renders the amount using the token's decimals.
func (a TokenAmount) Decimal(decimals int32) decimal.Decimal {
	return decimal.NewFromBigInt(a.TokenUnits(), 0).Shift(-decimals)
}

// IsZero reports whether the amount is zero.
func (a TokenAmount) IsZero() bool { return a.v == nil || a.v.Sign() == 0 }

// IsPositive reports whether the amount is greater than zero.
func (a TokenAmount) IsPositive() bool { return a.v != nil && a.v.Sign() > 0 }

// String renders the raw minimal-unit amount, since the scale is not known
// without the token's decimals.
func (a TokenAmount) String() string { return a.TokenUnits().String() }

// Energy is an amount of the energy resource.
type Energy struct{ value decimal.Decimal }

// NewEnergy wraps an energy amount.
func NewEnergy(value decimal.Decimal) Energy { return Energy{value: value} }

// ToDecimal returns the energy amount.
func (e Energy) ToDecimal() decimal.Decimal { return e.value }

// ToSUN prices the energy at the chain's current energy fee, which is quoted in
// SUN per unit of energy. Fractional results are rounded up so a cost estimate
// is never short; see CeilToSUN for the out-of-range behaviour.
func (e Energy) ToSUN(energyFee int64) SUN {
	return CeilToSUN(e.value.Mul(decimal.NewFromInt(energyFee)))
}

// Bandwidth is an amount of the bandwidth resource.
type Bandwidth struct{ value decimal.Decimal }

// NewBandwidth wraps a bandwidth amount.
func NewBandwidth(value decimal.Decimal) Bandwidth { return Bandwidth{value: value} }

// ToDecimal returns the bandwidth amount.
func (b Bandwidth) ToDecimal() decimal.Decimal { return b.value }

// ToSUN prices the bandwidth at the chain's current transaction fee, which is
// quoted in SUN per byte. Fractional results are rounded up so a cost estimate
// is never short; see CeilToSUN for the out-of-range behaviour.
func (b Bandwidth) ToSUN(transactionFee int64) SUN {
	return CeilToSUN(b.value.Mul(decimal.NewFromInt(transactionFee)))
}
