package client

import (
	"context"
	"time"

	"github.com/sxwebdev/gotron/schema/pb/core"
)

// AccountState is one GetAccount answer read for an account's resources: the
// stake its own limits come from, what it has used of them, and how far the
// node had applied the chain when it answered. Everything in it comes from the
// same answer, so its pools and ConsumedAt describe one state of the account,
// even behind a load balancer that sends each call to another node.
type AccountState struct {
	Account *core.Account
	// NetUsed, FreeNetUsed and EnergyUsed are the account's usage recovered
	// to the node's head, as GetAccountResource reports them: Wallet.getAccount
	// runs the same updateUsage on the account before it answers.
	NetUsed, FreeNetUsed, EnergyUsed int64
	// FrozenForBandwidth and FrozenForEnergy are the stake the account's own
	// limits are computed from (AccountCapsule.getAllFrozenBalanceForBandwidth,
	// getAllFrozenBalanceForEnergy): its Stake 1.0 and 2.0 frozen balances and
	// what others delegated to it under either. What it delegated out has left
	// its frozen balance and is not among them.
	FrozenForBandwidth, FrozenForEnergy SUN
	// ConsumedAt is the latest of the account's latest_consume_time,
	// latest_consume_free_time and latest_consume_time_for_energy, or zero
	// when the node reports none.
	//
	// java-tron stamps a transaction's consumption with the node's head slot
	// as it applies the transaction (BandwidthProcessor.consume through
	// useAccountNet, useFreeNet and consumeForCreateNewAccount,
	// EnergyProcessor.useEnergy: chainBaseManager.getHeadSlot()), and moves
	// the head to a block only after the block's transactions are applied —
	// so a transaction of block N is stamped with block N-1. Wallet.getAccount
	// renders the slots as the genesis time plus 3 s a slot, the time of that
	// head block. A ConsumedAt not earlier than a block's time therefore
	// proves the node had applied that block, and every one before it, when
	// it made this answer: the usage beside it counts their transactions
	// (HoldsBlock).
	//
	// It proves nothing about the block of the account's latest transaction,
	// which is stamped with the block before its own, nor about any block
	// later than ConsumedAt. An account that has consumed nothing reads the
	// genesis time or zero, which proves nothing. A block time off the 3 s
	// grid only makes it prove less. It speaks of the chain the node
	// answering follows.
	ConsumedAt time.Time
}

// GetAccountState reads addr's AccountState with one GetAccount call. An
// address with no account is ErrAccountNotFound.
func (c *Client) GetAccountState(ctx context.Context, addr string) (*AccountState, error) {
	acc, err := c.GetAccount(ctx, addr)
	if err != nil {
		return nil, err
	}
	return accountState(acc), nil
}

func accountState(acc *core.Account) *AccountState {
	res := acc.GetAccountResource()
	s := &AccountState{
		Account:     acc,
		NetUsed:     acc.GetNetUsage(),
		FreeNetUsed: acc.GetFreeNetUsage(),
		EnergyUsed:  res.GetEnergyUsage(),
		FrozenForBandwidth: SUN(acc.GetAcquiredDelegatedFrozenBalanceForBandwidth() +
			acc.GetAcquiredDelegatedFrozenV2BalanceForBandwidth()),
		FrozenForEnergy: SUN(res.GetFrozenBalanceForEnergy().GetFrozenBalance() +
			res.GetAcquiredDelegatedFrozenBalanceForEnergy() +
			res.GetAcquiredDelegatedFrozenV2BalanceForEnergy()),
		ConsumedAt: msToTime(max(acc.GetLatestConsumeTime(), acc.GetLatestConsumeFreeTime(),
			res.GetLatestConsumeTimeForEnergy())),
	}
	for _, f := range acc.GetFrozen() {
		s.FrozenForBandwidth += SUN(f.GetFrozenBalance())
	}
	for _, f := range acc.GetFrozenV2() {
		switch f.GetType() {
		case core.ResourceCode_BANDWIDTH:
			s.FrozenForBandwidth += SUN(f.GetAmount())
		case core.ResourceCode_ENERGY:
			s.FrozenForEnergy += SUN(f.GetAmount())
		}
	}
	return s
}

// Pools are what is left of the account's staked bandwidth, its free daily
// allowance and its energy, computed as GetAccountResource computes NetLimit,
// EnergyLimit and the free limit and as AvailableBandwidthWithoutFree,
// AvailableFreeBandwidth and AvailableEnergy subtract the usage from them —
// each may be negative, as there. r holds the network's totals (TotalNetLimit
// and TotalNetWeight of GetAccountResource, TotalEnergyCurrentLimit, the
// energy weight and Harden) and freeNetLimit the getFreeNetLimit parameter,
// neither of which an account carries. The limits follow Stake 2.0, as
// ResourceRates.Limit does.
//
// The totals may come from another call, and another node, than the state:
// they are the network's and move slowly, while the usage and ConsumedAt must
// come from one answer.
func (s *AccountState) Pools(r ResourceRates, freeNetLimit int64) (staked, free, energy int64) {
	return r.Limit(ResourceTypeBandwidth, s.FrozenForBandwidth) - s.NetUsed,
		freeNetLimit - s.FreeNetUsed,
		r.Limit(ResourceTypeEnergy, s.FrozenForEnergy) - s.EnergyUsed
}

// HoldsBlock reports whether the node had applied the block of blockTime when
// it made this answer, so that the state counts every transaction of that
// block: ConsumedAt is not earlier than blockTime. False proves nothing either
// way, and an unknown time (zero) is never held.
func (s *AccountState) HoldsBlock(blockTime time.Time) bool {
	return s != nil && !blockTime.IsZero() && !s.ConsumedAt.IsZero() && !s.ConsumedAt.Before(blockTime)
}
