package client

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/crypto"
)

// TransactionParams represents parameters for a TRX transfer
type TransactionParams struct {
	From       string
	To         string
	Amount     decimal.Decimal
	PrivateKey string
}

// Validate validates the transfer parameters
func (p *TransactionParams) Validate() error {
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
		return fmt.Errorf("%w: private key is required", ErrInvalidPrivateKey)
	}

	return nil
}

// SignTransaction signs a raw transaction with the given private key
// Note: This is a placeholder. Full implementation requires proto definitions.
func SignTransaction(rawData []byte, privateKey string) ([]byte, error) {
	if len(rawData) == 0 {
		return nil, ErrInvalidTransaction
	}

	if privateKey == "" {
		return nil, ErrInvalidPrivateKey
	}

	signature, err := crypto.SignData(rawData, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signature, nil
}

// Transfer sends TRX from one address to another
func (t *Client) Transfer(from, to string, amount decimal.Decimal, privateKey string) (string, error) {
	params := TransactionParams{
		From:       from,
		To:         to,
		Amount:     amount,
		PrivateKey: privateKey,
	}

	if err := params.Validate(); err != nil {
		return "", err
	}

	// TODO: Implement actual transaction creation and broadcasting
	return "", nil
}
