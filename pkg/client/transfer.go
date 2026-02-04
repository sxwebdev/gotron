package client

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// CreateTransferTransaction creates a TRX transfer transaction
//
// Important! The amount is specified in TRX.
func (c *Client) CreateTransferTransaction(ctx context.Context, from, to string, amount decimal.Decimal) (*api.TransactionExtention, error) {
	if from == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if to == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	// Convert TRX to SUN
	amount = amount.Mul(decimal.NewFromInt(1e6))

	var err error
	contract := &core.TransferContract{}
	if contract.OwnerAddress, err = tronutils.DecodeCheck(from); err != nil {
		return nil, err
	}

	if contract.ToAddress, err = tronutils.DecodeCheck(to); err != nil {
		return nil, err
	}

	contract.Amount = amount.IntPart()

	// Create the transaction
	tx, err := c.transport.CreateTransaction(ctx, contract)
	if err != nil {
		return nil, err
	}

	if proto.Size(tx) == 0 {
		return nil, ErrInvalidTransaction
	}

	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}

	return tx, nil
}
