package client

import (
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
// staked to yield it.
func (c *Client) ConvertEnergyToStaked(totalEnergyCurrentLimit, totalEnergyWeight int64, energy decimal.Decimal) SUN {
	if totalEnergyCurrentLimit == 0 {
		return 0
	}
	return units.CeilToSUN(energy.
		Div(decimal.NewFromInt(totalEnergyCurrentLimit)).
		Mul(decimal.NewFromInt(totalEnergyWeight)).
		Mul(decimal.NewFromInt(units.SunPerTRX)))
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
// be staked to yield it.
func (c *Client) ConvertBandwidthToStaked(totalNetWeight, totalNetLimit int64, bandwidth decimal.Decimal) SUN {
	if totalNetLimit == 0 {
		return 0
	}
	return units.CeilToSUN(bandwidth.
		Div(decimal.NewFromInt(totalNetLimit)).
		Mul(decimal.NewFromInt(totalNetWeight)).
		Mul(decimal.NewFromInt(units.SunPerTRX)))
}
