// Package transaction provides functionality for creating and signing Tron transactions.
package transaction

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/crypto"
)

var (
	// ErrInvalidTransaction is returned when a transaction is invalid
	ErrInvalidTransaction = errors.New("invalid transaction")
	// ErrInvalidPrivateKey is returned when the private key is invalid
	ErrInvalidPrivateKey = errors.New("invalid private key")
	// ErrInvalidAddress is returned when an address is invalid
	ErrInvalidAddress = errors.New("invalid address")
	// ErrInvalidAmount is returned when an amount is invalid
	ErrInvalidAmount = errors.New("invalid amount")
)

// TransferParams represents parameters for a TRX transfer
type TransferParams struct {
	From       string
	To         string
	Amount     decimal.Decimal
	PrivateKey string
}

// Validate validates the transfer parameters
func (p *TransferParams) Validate() error {
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
	if rawData == nil || len(rawData) == 0 {
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

// EstimateBandwidth estimates the bandwidth required for a transaction
// Note: This is a placeholder. Full implementation requires proto definitions.
func EstimateBandwidth(txSize int64) (int64, error) {
	if txSize <= 0 {
		return 0, ErrInvalidTransaction
	}

	// Bandwidth is approximately the size of the transaction in bytes plus overhead
	bandwidth := txSize + 64

	return bandwidth, nil
}

// GetTransactionID calculates the transaction ID from raw transaction data
// Note: This is a placeholder. Full implementation requires proto definitions.
func GetTransactionID(rawData []byte) (string, error) {
	if rawData == nil || len(rawData) == 0 {
		return "", ErrInvalidTransaction
	}

	// Hash the raw data to get transaction ID
	hash := crypto.HashSHA256(rawData)
	return fmt.Sprintf("%x", hash), nil
}
