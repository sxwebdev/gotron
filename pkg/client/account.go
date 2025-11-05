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

	balance, err := decimal.NewFromString(utils.FormatPrecision(res.GetBalance(), TrxDecimals))
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

type EstimateActivateAccountResult struct {
	Energy    decimal.Decimal `json:"energy"`
	Bandwidth decimal.Decimal `json:"bandwidth"`
	Trx       decimal.Decimal `json:"trx"`
}

// EstimateActivateAccount estimates the activation fee for a Tron address.
// It checks the available bandwidth and adds the activation fee accordingly.
// The fee is returned in TRX (1 TRX = 1_000_000 SUN).
// We assume that fromAddress is ALWAYS activated, being it processing address.
// Simple swap of arg to BlackHoleAddress on fakeTx creation will always return valid tx.
func (c *Client) EstimateActivateAccount(ctx context.Context, fromAddress, toAddress string) (*EstimateActivateAccountResult, error) {
	estimate := &EstimateActivateAccountResult{}

	isActivated, err := c.IsAccountActivated(ctx, toAddress)
	if err != nil {
		return nil, fmt.Errorf("check is wallet activated: %w", err)
	}

	if isActivated {
		return estimate, nil
	}

	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain params: %w", err)
	}

	// Add activation constant fee
	estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateNewAccountFeeInSystemContract))

	accountResources, err := c.AvailableForDelegateResources(ctx, fromAddress)
	if err != nil {
		return nil, err
	}

	// Estimate bandwidth required from staked bandwidth for activation transaction
	// BlackHoleAddress is safe to be used here on main/test networks.
	fakeTx, err := CreateFakeCreateAccountTransaction(fromAddress, toAddress)
	if err != nil {
		return nil, fmt.Errorf("fake create account tx: %w", err)
	}

	estimatedBandwidth, err := c.EstimateBandwidth(fakeTx)
	if err != nil {
		return nil, fmt.Errorf("estimate fake create account bandwidth: %w", err)
	}

	// Add 0.1 TRX when address does not have any staked bandwidth, nor does it have enough to activate account.
	// Or add actual bandwidth required if enough bandwidth
	if accountResources.Bandwidth.LessThan(estimatedBandwidth) {
		estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateAccountFee))
	} else {
		// We add coefficient to be safe.
		estimate.Bandwidth = estimate.Bandwidth.Add(estimatedBandwidth)
	}

	// Convert from SUN to TRX
	estimate.Trx = estimate.Trx.Div(decimal.NewFromInt(1e6))

	return estimate, nil
}
