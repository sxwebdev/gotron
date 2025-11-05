package client

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// TransferTRC20Params represents parameters for a TRC20 token transfer
type TransferTRC20Params struct {
	ContractAddress string
	From            string
	To              string
	Amount          decimal.Decimal
	PrivateKey      string
	FeeLimit        int64 // Maximum TRX to spend on energy (in SUN)
}

// Validate validates the transfer parameters
func (p *TransferTRC20Params) Validate() error {
	if p.ContractAddress == "" {
		return fmt.Errorf("%w: contract address is required", ErrInvalidAddress)
	}

	if p.From == "" {
		return fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if p.To == "" {
		return fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if p.Amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	if p.PrivateKey == "" {
		return fmt.Errorf("%w: private key is required", ErrInvalidParams)
	}

	if p.FeeLimit <= 0 {
		return fmt.Errorf("%w: fee limit must be greater than zero", ErrInvalidParams)
	}

	return nil
}

// TransferTRC20 transfers TRC20 tokens
func (t *Client) TransferTRC20(contractAddress, from, to string, amount decimal.Decimal, privateKey string, feeLimit int64) (string, error) {
	params := TransferTRC20Params{
		ContractAddress: contractAddress,
		From:            from,
		To:              to,
		Amount:          amount,
		PrivateKey:      privateKey,
		FeeLimit:        feeLimit,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual TRC20 transfer
	return "", nil
}
