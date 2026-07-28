package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
)

// estimateTransferFeeLimit is the fee limit used to build the throwaway TRC20
// transaction an estimate is measured on. It is never broadcast.
const estimateTransferFeeLimit = SUN(100 * units.SunPerTRX)

// EstimateTransferResult contains the estimated cost of a TRX/TRC20
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
type EstimateTransferResult struct {
	Total      EstimateResult `json:"total"`
	Transfer   EstimateResult `json:"transfer"`
	Activation EstimateResult `json:"activation"`
}

// EstimateTRXTransfer estimates the cost of sending TRX.
//
// TRX and TRC20 estimates are separate calls because their amounts are measured
// on different scales: SUN is fixed at 1e6, while a token's scale comes from its
// own decimals. A single entry point would have to take one amount meaning two
// different things depending on the asset.
func (c *Client) EstimateTRXTransfer(ctx context.Context, fromAddress, toAddress string, amount SUN) (*EstimateTransferResult, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if toAddress == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	data, err := c.CreateTransferTransaction(ctx, fromAddress, toAddress, amount)
	if err != nil && !strings.Contains(err.Error(), "reset by peer") {
		return nil, fmt.Errorf("transfer: %w", err)
	}

	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	var transfer EstimateResult
	transfer.Bandwidth, err = c.EstimateBandwidth(data.GetTransaction())
	if err != nil {
		return nil, err
	}
	transfer.Fee = units.NewBandwidth(transfer.Bandwidth).ToSUN(chainParams.TransactionFee)

	return c.withActivation(ctx, fromAddress, toAddress, transfer)
}

// EstimateTRC20Transfer estimates the cost of sending a TRC20 token.
//
// The amount is in the token's minimal units; build it with
// units.FromTokenDecimal and the decimals TRC20GetDecimals reports.
func (c *Client) EstimateTRC20Transfer(ctx context.Context, fromAddress, toAddress, contractAddress string, amount TokenAmount) (*EstimateTransferResult, error) {
	if fromAddress == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if toAddress == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("%w: contract address is required", ErrInvalidAddress)
	}

	if !amount.IsPositive() {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	data, err := c.TRC20Send(ctx, fromAddress, toAddress, contractAddress, amount, estimateTransferFeeLimit)
	if err != nil && !strings.Contains(err.Error(), "reset by peer") {
		return nil, fmt.Errorf("cannot make tron transaction: %w", err)
	}

	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	var transfer EstimateResult
	transfer.Bandwidth, err = c.EstimateBandwidth(data.GetTransaction())
	if err != nil {
		return nil, err
	}

	jsonString := fmt.Sprintf(`[{"address":"%s"},{"uint256":"%s"}]`, toAddress, amount)

	data, err = c.TriggerConstantContractCustom(ctx, fromAddress, contractAddress, "transfer(address,uint256)", jsonString)
	if err != nil && !strings.Contains(err.Error(), "reset by peer") {
		return nil, fmt.Errorf("cannot trigger contract: %w", err)
	}

	transfer.Energy = decimal.NewFromInt(data.GetEnergyUsed())
	transfer.Fee = units.NewEnergy(transfer.Energy).ToSUN(chainParams.EnergyFee) +
		units.NewBandwidth(transfer.Bandwidth).ToSUN(chainParams.TransactionFee)

	return c.withActivation(ctx, fromAddress, toAddress, transfer)
}

// withActivation adds the recipient activation cost, if any, to a transfer
// estimate.
func (c *Client) withActivation(ctx context.Context, fromAddress, toAddress string, transfer EstimateResult) (*EstimateTransferResult, error) {
	activation, err := c.EstimateSystemContractActivation(ctx, fromAddress, toAddress)
	if err != nil {
		return nil, fmt.Errorf("estimate activation: %w", err)
	}

	return &EstimateTransferResult{
		Transfer:   transfer,
		Activation: *activation,
		Total: EstimateResult{
			Energy:    transfer.Energy.Add(activation.Energy),
			Bandwidth: transfer.Bandwidth.Add(activation.Bandwidth),
			Fee:       transfer.Fee + activation.Fee,
		},
	}, nil
}
