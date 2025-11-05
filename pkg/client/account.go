package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/utils"
	pbtron "github.com/sxwebdev/gotron/schema/pb/core"
)

var ErrAccountNotFound = errors.New("account not found")

func (t *Client) GetAccount(ctx context.Context, addr string) (*pbtron.Account, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}

	account := new(pbtron.Account)
	var err error

	account.Address, err = utils.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	acc, err := t.walletClient.GetAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(acc.Address, account.Address) {
		return nil, ErrAccountNotFound
	}

	return acc, nil
}

// GetAccountBalance retrieves the TRX balance of the specified address
func (t *Client) GetAccountBalance(ctx context.Context, address string) (decimal.Decimal, error) {
	res, err := t.GetAccount(ctx, address)
	if err != nil {
		return decimal.Zero, err
	}

	balance, err := decimal.NewFromString(utils.FormatPrecision(res.GetBalance(), TRXDecimals))
	if err != nil {
		return decimal.Zero, fmt.Errorf("convert balance to decimal: %w", err)
	}

	return balance, nil
}
