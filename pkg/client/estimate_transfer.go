package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

type EstimateTransferResourcesResult struct {
	Energy    decimal.Decimal `json:"energy"`
	Bandwidth decimal.Decimal `json:"bandwidth"`
	Trx       decimal.Decimal `json:"trx"`
}

func (c *Client) EstimateTransferResources(
	ctx context.Context,
	fromAddress, toAddress, contractAddress string,
	amount decimal.Decimal,
	decimals int64,
) (*EstimateTransferResourcesResult, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("from address is required")
	}

	if toAddress == "" {
		return nil, fmt.Errorf("to address is required")
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	isActivated, err := c.IsAccountActivated(ctx, toAddress)
	if err != nil {
		return nil, fmt.Errorf("check to address activation: %w", err)
	}
	if !isActivated {
		return nil, fmt.Errorf("%w: %s", ErrAccountNotActivated, toAddress)
	}

	var res EstimateTransferResourcesResult

	var tx *core.Transaction
	var data *api.TransactionExtention
	if contractAddress == TrxAssetIdentifier { //nolint:nestif
		data, err = c.CreateTransferTransaction(ctx, fromAddress, toAddress, amount)
		if err != nil && !strings.Contains(err.Error(), "reset by peer") {
			return nil, fmt.Errorf("transfer: %w", err)
		}

		tx = data.GetTransaction()
	} else {
		amount = amount.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(decimals)))

		data, err = c.TRC20Send(ctx, fromAddress, toAddress, contractAddress, amount, 100*1e6)
		if err != nil && !strings.Contains(err.Error(), "reset by peer") {
			return nil, fmt.Errorf("cannot make tron transaction: %w", err)
		}

		tx = data.GetTransaction()
	}

	res.Bandwidth, err = c.EstimateBandwidth(tx)
	if err != nil {
		return nil, err
	}

	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	if contractAddress == TrxAssetIdentifier {
		res.Trx = units.NewBandwidth(res.Bandwidth).ToTRX(chainParams.TransactionFee).ToDecimal()
	} else {
		jsonString := fmt.Sprintf(`[{"address":"%s"},{"uint256":"%s"}]`, toAddress, amount.BigInt())

		var err error
		data, err = c.TriggerConstantContractCustom(ctx, fromAddress, contractAddress, "transfer(address,uint256)", jsonString)
		if err != nil && !strings.Contains(err.Error(), "reset by peer") {
			return nil, fmt.Errorf("cannot trigger contract: %w", err)
		}

		res.Energy = decimal.NewFromInt(data.EnergyUsed)
		res.Trx = units.NewEnergy(res.Energy).
			ToTRX(chainParams.EnergyFee).
			ToDecimal().
			Add(
				units.NewBandwidth(res.Bandwidth).
					ToTRX(chainParams.TransactionFee).
					ToDecimal(),
			)
	}

	return &res, nil
}
