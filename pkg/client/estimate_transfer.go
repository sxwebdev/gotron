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

// EstimateTransferResourcesResult contains the estimated cost of a TRX/TRC20
// transfer broken down into the transfer itself and the recipient activation
// (when toAddress is not yet activated). Total is the sum of Transfer and
// Activation per resource.
//
// For activated recipients Activation is zero-valued and Total equals Transfer.
//
// Note: in Tron, when sending to an unactivated address the activation fee is
// consumed by the transfer transaction itself rather than a separate
// CreateAccount call. Total is therefore a conservative upper bound — the
// real on-chain cost is typically slightly lower than Total.
type EstimateTransferResourcesResult struct {
	Total      EstimateResult `json:"total"`
	Transfer   EstimateResult `json:"transfer"`
	Activation EstimateResult `json:"activation"`
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

	var transfer EstimateResult
	var err error

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

	transfer.Bandwidth, err = c.EstimateBandwidth(tx)
	if err != nil {
		return nil, err
	}

	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	if contractAddress == TrxAssetIdentifier {
		transfer.Trx = units.NewBandwidth(transfer.Bandwidth).ToTRX(chainParams.TransactionFee).ToDecimal()
	} else {
		jsonString := fmt.Sprintf(`[{"address":"%s"},{"uint256":"%s"}]`, toAddress, amount.BigInt())

		data, err = c.TriggerConstantContractCustom(ctx, fromAddress, contractAddress, "transfer(address,uint256)", jsonString)
		if err != nil && !strings.Contains(err.Error(), "reset by peer") {
			return nil, fmt.Errorf("cannot trigger contract: %w", err)
		}

		transfer.Energy = decimal.NewFromInt(data.EnergyUsed)
		transfer.Trx = units.NewEnergy(transfer.Energy).
			ToTRX(chainParams.EnergyFee).
			ToDecimal().
			Add(
				units.NewBandwidth(transfer.Bandwidth).
					ToTRX(chainParams.TransactionFee).
					ToDecimal(),
			)
	}

	activation, err := c.EstimateSystemContractActivation(ctx, fromAddress, toAddress)
	if err != nil {
		return nil, fmt.Errorf("estimate activation: %w", err)
	}

	return &EstimateTransferResourcesResult{
		Transfer:   transfer,
		Activation: *activation,
		Total: EstimateResult{
			Energy:    transfer.Energy.Add(activation.Energy),
			Bandwidth: transfer.Bandwidth.Add(activation.Bandwidth),
			Trx:       transfer.Trx.Add(activation.Trx),
		},
	}, nil
}
