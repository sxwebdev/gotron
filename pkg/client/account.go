package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/utils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	pbtron "github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

var ErrAccountNotFound = errors.New("account not found")

func (c *Client) GetAccount(ctx context.Context, addr string) (*pbtron.Account, error) {
	if addr == "" {
		return nil, ErrEmptyAddress
	}

	account := new(pbtron.Account)
	var err error

	account.Address, err = utils.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	acc, err := c.walletClient.GetAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(acc.Address, account.Address) {
		return nil, ErrAccountNotFound
	}

	return acc, nil
}

// GetAccountBalance retrieves the TRX balance of the specified address
func (c *Client) GetAccountBalance(ctx context.Context, address string) (decimal.Decimal, error) {
	res, err := c.GetAccount(ctx, address)
	if err != nil {
		return decimal.Zero, err
	}

	balance, err := decimal.NewFromString(utils.FormatPrecision(res.GetBalance(), TRXDecimals))
	if err != nil {
		return decimal.Zero, fmt.Errorf("convert balance to decimal: %w", err)
	}

	return balance, nil
}

// IsAccountActivated checks if the account with the given address is activated
func (c *Client) IsAccountActivated(ctx context.Context, address string) (bool, error) {
	_, err := c.GetAccount(ctx, address)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CreateAccount creates a new account on the blockchain.
func (c *Client) CreateAccount(ctx context.Context, from, addr string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.AccountCreateContract{}
	if contract.OwnerAddress, err = utils.DecodeCheck(from); err != nil {
		return nil, err
	}

	if contract.AccountAddress, err = utils.DecodeCheck(addr); err != nil {
		return nil, err
	}

	tx, err := c.walletClient.CreateAccount2(ctx, contract)
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
