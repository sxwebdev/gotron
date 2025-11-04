package client

import "github.com/shopspring/decimal"

// ConvertStackedTRXToEnergy converts stacked TRX to energy.
func (c *Client) ConvertStackedTRXToEnergy(totalEnergyCurrentLimit, totalEnergyWeight, stackedTrx int64) decimal.Decimal {
	return decimal.NewFromInt(stackedTrx).
		Div(decimal.NewFromInt(1e6)).
		Div(decimal.NewFromInt(totalEnergyWeight)).
		Mul(decimal.NewFromInt(totalEnergyCurrentLimit))
}

// ConvertEnergyToStackedTRX converts energy to stacked TRX. Returns value in SUN.
func (c *Client) ConvertEnergyToStackedTRX(totalEnergyCurrentLimit, totalEnergyWeight int64, energy decimal.Decimal) decimal.Decimal {
	return energy.
		Div(decimal.NewFromInt(totalEnergyCurrentLimit)).
		Mul(decimal.NewFromInt(totalEnergyWeight)).
		Mul(decimal.NewFromInt(1e6))
}

// ConvertStackedTRXToBandwidth converts stacked TRX to bandwidth.
func (c *Client) ConvertStackedTRXToBandwidth(totalNetWeight, totalNetLimit, stackedTrx int64) decimal.Decimal {
	return decimal.NewFromInt(stackedTrx).
		Div(decimal.NewFromInt(1e6)).
		Div(decimal.NewFromInt(totalNetWeight)).
		Mul(decimal.NewFromInt(totalNetLimit))
}

// ConvertBandwidthToStackedTRX converts bandwidth to stacked TRX. Returns value in SUN.
func (c *Client) ConvertBandwidthToStackedTRX(totalNetWeight, totalNetLimit int64, bandwidth decimal.Decimal) decimal.Decimal {
	return bandwidth.
		Div(decimal.NewFromInt(totalNetLimit)).
		Mul(decimal.NewFromInt(totalNetWeight)).
		Mul(decimal.NewFromInt(1e6))
}
