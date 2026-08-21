package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sxwebdev/gotron/pkg/address"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// Stake freezes TRX to obtain bandwidth or energy (Stake 2.0, FreezeBalanceV2).
//
// The amount is in SUN. Convert from human-facing TRX with units.FromTRX.
func (c *Client) Stake(ctx context.Context, owner string, resource ResourceType, amount SUN) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	if err := resource.Validate(); err != nil {
		return nil, err
	}

	if amount <= 0 {
		return nil, fmt.Errorf("%w: stake amount must be greater than zero", ErrInvalidAmount)
	}

	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	contract := &core.FreezeBalanceV2Contract{
		OwnerAddress:  ownerBytes,
		FrozenBalance: amount.Int64(),
		Resource:      resource.ToProto(),
	}

	tx, err := c.transport.FreezeBalanceV2(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// Unstake begins the unfreeze period for staked TRX (UnfreezeBalanceV2). The TRX
// becomes withdrawable via WithdrawUnstaked once the network's unfreeze delay passes.
//
// The amount is in SUN. Convert from human-facing TRX with units.FromTRX.
func (c *Client) Unstake(ctx context.Context, owner string, resource ResourceType, amount SUN) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	if err := resource.Validate(); err != nil {
		return nil, err
	}

	if amount <= 0 {
		return nil, fmt.Errorf("%w: unstake amount must be greater than zero", ErrInvalidAmount)
	}

	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	contract := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    ownerBytes,
		UnfreezeBalance: amount.Int64(),
		Resource:        resource.ToProto(),
	}

	tx, err := c.transport.UnfreezeBalanceV2(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// WithdrawUnstaked withdraws all TRX whose unfreeze period has expired
// (WithdrawExpireUnfreeze).
func (c *Client) WithdrawUnstaked(ctx context.Context, owner string) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	contract := &core.WithdrawExpireUnfreezeContract{OwnerAddress: ownerBytes}

	tx, err := c.transport.WithdrawExpireUnfreeze(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// CancelAllUnstakes cancels every pending unstake and re-stakes the TRX
// (CancelAllUnfreezeV2). Already expired entries are withdrawn instead.
func (c *Client) CancelAllUnstakes(ctx context.Context, owner string) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	contract := &core.CancelAllUnfreezeV2Contract{OwnerAddress: ownerBytes}

	tx, err := c.transport.CancelAllUnfreezeV2(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// GetAvailableUnstakeCount returns how many more Unstake calls the account may make
// before it has to withdraw or cancel pending unstakes.
func (c *Client) GetAvailableUnstakeCount(ctx context.Context, owner string) (int64, error) {
	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return 0, err
	}

	msg := &api.GetAvailableUnfreezeCountRequestMessage{OwnerAddress: ownerBytes}

	res, err := c.transport.GetAvailableUnfreezeCount(ctx, msg)
	if err != nil {
		return 0, err
	}

	return res.GetCount(), nil
}

// GetWithdrawableUnstaked returns the amount in SUN that WithdrawUnstaked would
// release right now, as reported by the node. It should agree with
// StakeInfo.WithdrawableNow, which is computed locally without an extra round-trip.
func (c *Client) GetWithdrawableUnstaked(ctx context.Context, owner string) (SUN, error) {
	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return 0, err
	}

	msg := &api.CanWithdrawUnfreezeAmountRequestMessage{
		OwnerAddress: ownerBytes,
		Timestamp:    time.Now().UnixMilli(),
	}

	res, err := c.transport.GetCanWithdrawUnfreezeAmount(ctx, msg)
	if err != nil {
		return 0, err
	}

	return SUN(res.GetAmount()), nil
}

// GetStakeInfo returns an aggregated view of an account's Stake 2.0 position:
// staked balances per resource and pending unstakes with their expiry.
func (c *Client) GetStakeInfo(ctx context.Context, addr string) (*StakeInfo, error) {
	account, err := c.GetAccount(ctx, addr)
	if err != nil {
		return nil, err
	}

	// Captured once so that WithdrawableNow and every ExpireTime are consistent
	// against a single instant.
	nowMs := time.Now().UnixMilli()

	info := &StakeInfo{}

	for _, item := range account.GetFrozenV2() {
		// TRON_POWER entries are always present and are not a separate stake -
		// counting them would double the total.
		switch item.GetType() {
		case core.ResourceCode_BANDWIDTH:
			info.StakedBandwidth += SUN(item.GetAmount())
		case core.ResourceCode_ENERGY:
			info.StakedEnergy += SUN(item.GetAmount())
		}
	}
	info.TotalStaked = info.StakedBandwidth + info.StakedEnergy

	for _, item := range account.GetUnfrozenV2() {
		amount := SUN(item.GetUnfreezeAmount())
		if amount <= 0 {
			continue
		}

		expireMs := item.GetUnfreezeExpireTime()

		info.UnstakingTotal += amount
		if expireMs <= nowMs {
			info.WithdrawableNow += amount
		}

		info.PendingUnstakes = append(info.PendingUnstakes, PendingUnstake{
			Resource:   ResourceType(item.GetType()),
			Amount:     amount,
			ExpireTime: time.UnixMilli(expireMs),
		})
	}

	return info, nil
}

// checkTransaction rejects transactions the node did not actually build: an empty
// response, or one carrying a contract validation error.
//
// The validation error is returned as *ContractValidateError so that callers can
// tell "my request is wrong" from "this node is unwell" with errors.As instead
// of by matching on the message.
func checkTransaction(tx *api.TransactionExtention) error {
	if proto.Size(tx) == 0 {
		return ErrInvalidTransaction
	}
	if code := tx.GetResult().GetCode(); code != 0 {
		return &ContractValidateError{
			Code:    code,
			Message: string(tx.GetResult().GetMessage()),
		}
	}
	if tx.GetTransaction() == nil || tx.GetTransaction().GetRawData() == nil {
		return ErrInvalidTransaction
	}
	return nil
}
