package client

import (
	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/transaction"
	"github.com/sxwebdev/gotron/pkg/trc20"
)

// Transfer sends TRX from one address to another
func (t *Client) Transfer(from, to string, amount decimal.Decimal, privateKey string) (string, error) {
	params := transaction.TransferParams{
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

// TransferTRC20 transfers TRC20 tokens
func (t *Client) TransferTRC20(contractAddress, from, to string, amount decimal.Decimal, privateKey string, feeLimit int64) (string, error) {
	params := trc20.TransferParams{
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
