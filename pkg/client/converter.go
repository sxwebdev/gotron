package client

import (
	"math/big"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
)

// ConvertStakedToEnergy converts a staked balance to the energy it yields.
//
// totalEnergyWeight is the network-wide staked total in TRX, so the staked
// amount is scaled down from SUN before the ratio is taken.
func (c *Client) ConvertStakedToEnergy(totalEnergyCurrentLimit, totalEnergyWeight int64, staked SUN) decimal.Decimal {
	if totalEnergyWeight == 0 {
		return decimal.Zero
	}
	return staked.TRX().
		Div(decimal.NewFromInt(totalEnergyWeight)).
		Mul(decimal.NewFromInt(totalEnergyCurrentLimit))
}

// ConvertEnergyToStaked converts an energy amount to the balance that must be
// staked to yield it: the exact share, rounded up to a SUN.
//
// That is the share in exact arithmetic, and the chain truncates its own
// figure: for what to delegate so a receiver gets the energy, use
// ResourceRates.StakeFor, or CallStakeFor for energy a contract call spends.
func (c *Client) ConvertEnergyToStaked(totalEnergyCurrentLimit, totalEnergyWeight int64, energy decimal.Decimal) SUN {
	return stakedShare(energy, totalEnergyWeight, totalEnergyCurrentLimit)
}

// ConvertStakedToBandwidth converts a staked balance to the bandwidth it yields.
//
// totalNetWeight is the network-wide staked total in TRX, so the staked amount
// is scaled down from SUN before the ratio is taken.
func (c *Client) ConvertStakedToBandwidth(totalNetWeight, totalNetLimit int64, staked SUN) decimal.Decimal {
	if totalNetWeight == 0 {
		return decimal.Zero
	}
	return staked.TRX().
		Div(decimal.NewFromInt(totalNetWeight)).
		Mul(decimal.NewFromInt(totalNetLimit))
}

// ConvertBandwidthToStaked converts a bandwidth amount to the balance that must
// be staked to yield it: the exact share, rounded up to a SUN.
//
// That is the share in exact arithmetic, and the chain truncates its own
// figure: for what to delegate so a receiver gets the bandwidth, use
// ResourceRates.StakeFor.
func (c *Client) ConvertBandwidthToStaked(totalNetWeight, totalNetLimit int64, bandwidth decimal.Decimal) SUN {
	return stakedShare(bandwidth, totalNetWeight, totalNetLimit)
}

// stakedShare is amount * weight * SunPerTRX / limit rounded up, computed as a
// fraction. Dividing first in decimals rounded the quotient to 16 digits and
// could lose the remainder the ceiling had to see, coming back a SUN short.
func stakedShare(amount decimal.Decimal, weight, limit int64) SUN {
	if limit == 0 {
		return 0
	}
	share := amount.Rat()
	share.Mul(share, new(big.Rat).SetInt64(weight))
	share.Mul(share, new(big.Rat).SetInt64(units.SunPerTRX))
	share.Quo(share, new(big.Rat).SetInt64(limit))
	// Ceiling of num/den (den > 0): floor division, then up one on a remainder.
	q, m := new(big.Int).DivMod(share.Num(), share.Denom(), new(big.Int))
	if m.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return units.CeilToSUN(decimal.NewFromBigInt(q, 0))
}
