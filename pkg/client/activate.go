package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// EstimateActivationFee estimates the activation fee for a Tron address.
// It checks the available bandwidth and adds the activation fee accordingly.
// The fee is returned in TRX (1 TRX = 1_000_000 SUN).
// We assume that fromAddress is ALWAYS activated, being it processing address.
// Simple swap of arg to BlackHoleAddress on fakeTx creation will always return valid tx.
func (t *Client) EstimateActivationFee(ctx context.Context, fromAddress, toAddress string) (*EstimateResult, error) {
	estimate := &EstimateResult{}

	isActivated, err := t.IsAccountActivated(ctx, toAddress)
	if err != nil {
		return nil, fmt.Errorf("check wallet activation: %w", err)
	}

	if isActivated {
		return estimate, nil
	}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain params: %w", err)
	}

	// Add activation constant fee
	estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateNewAccountFeeInSystemContract))

	accountResources, err := t.AvailableForDelegateResources(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("get account resources: %w", err)
	}

	// Estimate bandwidth required for the activation transaction.
	// BlackHoleAddress is safe to be used here on main/test networks.
	fakeTx, err := CreateFakeCreateAccountTransaction(fromAddress, toAddress)
	if err != nil {
		return nil, fmt.Errorf("fake create account tx: %w", err)
	}
	estimatedBandwidth, err := t.EstimateBandwidth(fakeTx)
	if err != nil {
		return nil, fmt.Errorf("estimate fake create account bandwidth: %w", err)
	}

	// Per Tron docs, the initiator must have enough Bandwidth obtained by
	// staking TRX — free daily quota and bandwidth received via delegation
	// do NOT count. If the caller has no own staked bandwidth (or not enough
	// of it) to cover the activation transaction, 0.1 TRX is burned.
	if accountResources.Bandwidth.LessThan(estimatedBandwidth) {
		estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateAccountFee))
	} else {
		estimate.Bandwidth = estimate.Bandwidth.Add(estimatedBandwidth)
	}

	// Convert from SUN to TRX
	estimate.Trx = estimate.Trx.Div(decimal.NewFromInt(1e6))

	return estimate, nil
}

// EstimateSystemContractActivation estimates the activation fee for a Tron address
// by building a real CreateAccount transaction via the node API (instead of the
// local fake transaction used by EstimateActivationFee).
// The fee is returned in TRX (1 TRX = 1_000_000 SUN).
func (t *Client) EstimateSystemContractActivation(ctx context.Context, caller string, receiver string) (*EstimateResult, error) {
	estimate := &EstimateResult{}

	isActivated, err := t.IsAccountActivated(ctx, receiver)
	if err != nil {
		return nil, fmt.Errorf("check wallet activation: %w", err)
	}

	if isActivated {
		return estimate, nil
	}

	chainParams, err := t.ChainParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain params: %w", err)
	}

	// Add activation constant fee
	estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateNewAccountFeeInSystemContract))

	accountResources, err := t.AvailableForDelegateResources(ctx, caller)
	if err != nil {
		return nil, fmt.Errorf("get account resources: %w", err)
	}

	tx, err := t.CreateAccount(ctx, caller, receiver, core.AccountType_Normal)
	if err != nil {
		// Receiver became activated between IsAccountActivated and CreateAccount,
		// or any other case where the node refuses on already-existing account.
		if strings.Contains(err.Error(), "Account has existed") {
			return &EstimateResult{}, nil
		}
		return nil, fmt.Errorf("create account: %w", err)
	}

	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}

	if tx.GetResult().GetCode() != 0 {
		if strings.Contains(string(tx.GetResult().GetMessage()), "Account has existed") {
			return &EstimateResult{}, nil
		}
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}

	estimatedBandwidth, err := t.EstimateBandwidth(tx.GetTransaction())
	if err != nil {
		return nil, fmt.Errorf("estimate create account bandwidth: %w", err)
	}

	// Per Tron docs, the caller must have enough Bandwidth obtained by
	// staking TRX — free daily quota and bandwidth received via delegation
	// do NOT count. If the caller has no own staked bandwidth (or not enough
	// of it) to cover the activation transaction, 0.1 TRX is burned.
	if accountResources.Bandwidth.LessThan(estimatedBandwidth) {
		estimate.Trx = estimate.Trx.Add(decimal.NewFromInt(chainParams.CreateAccountFee))
	} else {
		estimate.Bandwidth = estimate.Bandwidth.Add(estimatedBandwidth)
	}

	// Convert from SUN to TRX
	estimate.Trx = estimate.Trx.Div(decimal.NewFromInt(1e6))

	return estimate, nil
}
