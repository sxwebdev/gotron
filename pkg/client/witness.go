package client

import (
	"context"
	"fmt"

	"github.com/sxwebdev/gotron/pkg/address"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// VoteWitnesses replaces the account's entire vote set (VoteWitnessAccount).
// Votes not listed here are dropped, so always pass the full desired set.
//
// Vote counts are in TRON POWER, which an account obtains by staking TRX.
func (c *Client) VoteWitnesses(ctx context.Context, owner string, votes []Vote) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	if len(votes) == 0 {
		return nil, fmt.Errorf("%w: at least one vote is required", ErrInvalidParams)
	}

	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	contract := &core.VoteWitnessContract{
		OwnerAddress: ownerBytes,
		Votes:        make([]*core.VoteWitnessContract_Vote, len(votes)),
	}

	for i, vote := range votes {
		if err := address.Validate(vote.WitnessAddress); err != nil {
			return nil, fmt.Errorf("%w: witness address is invalid", ErrInvalidAddress)
		}

		if vote.Count <= 0 {
			return nil, fmt.Errorf("%w: vote count must be greater than zero", ErrInvalidAmount)
		}

		witnessBytes, err := tronutils.DecodeCheck(vote.WitnessAddress)
		if err != nil {
			return nil, err
		}

		contract.Votes[i] = &core.VoteWitnessContract_Vote{
			VoteAddress: witnessBytes,
			VoteCount:   vote.Count,
		}
	}

	tx, err := c.transport.VoteWitnessAccount(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// ClaimRewards withdraws the account's accumulated voting or SR rewards
// (WithdrawBalance). Rewards can only be claimed once every 24 hours.
func (c *Client) ClaimRewards(ctx context.Context, owner string) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	ownerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	contract := &core.WithdrawBalanceContract{OwnerAddress: ownerBytes}

	tx, err := c.transport.WithdrawBalance(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

// ListWitnesses returns all super representatives and candidates.
func (c *Client) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	return c.transport.ListWitnesses(ctx)
}

// GetUnclaimedReward returns the account's unclaimed reward in SUN.
func (c *Client) GetUnclaimedReward(ctx context.Context, addr string) (int64, error) {
	addrBytes, err := tronutils.DecodeCheck(addr)
	if err != nil {
		return 0, err
	}

	res, err := c.transport.GetRewardInfo(ctx, addrBytes)
	if err != nil {
		return 0, err
	}

	return res.GetNum(), nil
}

// GetWitnessBrokerage returns the witness commission rate as a percentage (0-100).
// Addresses that are not registered witnesses report 0.
func (c *Client) GetWitnessBrokerage(ctx context.Context, witness string) (int64, error) {
	witnessBytes, err := tronutils.DecodeCheck(witness)
	if err != nil {
		return 0, err
	}

	res, err := c.transport.GetBrokerageInfo(ctx, witnessBytes)
	if err != nil {
		return 0, err
	}

	return res.GetNum(), nil
}
