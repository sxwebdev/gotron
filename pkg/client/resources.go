package client

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/address"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// GetAccountResource retrieves resource information for the specified account address
func (c *Client) GetAccountResource(ctx context.Context, addr string) (*api.AccountResourceMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = tronutils.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	return c.transport.GetAccountResource(ctx, account)
}

// GetDelegatedResources lists what address has delegated out, using the Stake
// 1.0 index. Most accounts want GetDelegatedResourcesV2.
func (c *Client) GetDelegatedResources(ctx context.Context, address string) ([]Delegation, error) {
	return c.delegationsOut(ctx, address,
		c.transport.GetDelegatedResourceAccountIndex, c.transport.GetDelegatedResource)
}

// GetDelegatedResourcesV2 lists what address has delegated out under Stake 2.0.
func (c *Client) GetDelegatedResourcesV2(ctx context.Context, address string) ([]Delegation, error) {
	return c.delegationsOut(ctx, address,
		c.transport.GetDelegatedResourceAccountIndexV2, c.transport.GetDelegatedResourceV2)
}

// delegationsOut walks the account index and reads each delegation, converting
// the chain's raw form - byte addresses, SUN as int64, Unix-millisecond lock
// expiries - into the same string/SUN/time.Time shape GetStakeInfo uses. The
// two Stake versions differ only in which pair of RPCs they call.
func (c *Client) delegationsOut(
	ctx context.Context,
	address string,
	index func(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error),
	fetch func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error),
) ([]Delegation, error) {
	if address == "" {
		return nil, fmt.Errorf("%w: address is required", ErrEmptyAddress)
	}

	addrBytes, err := tronutils.DecodeCheck(address)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddress, err)
	}

	ai, err := index(ctx, addrBytes)
	if err != nil {
		return nil, err
	}

	var result []Delegation
	for _, addrTo := range ai.GetToAccounts() {
		list, err := fetch(ctx, &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		})
		if err != nil {
			return nil, err
		}

		// One index entry can hold several records, and the node returns
		// placeholder entries with nothing delegated; those are dropped rather
		// than reported as zero-amount delegations.
		for _, d := range list.GetDelegatedResource() {
			if d.GetFrozenBalanceForBandwidth() <= 0 && d.GetFrozenBalanceForEnergy() <= 0 {
				continue
			}
			result = append(result, Delegation{
				From:               tronutils.EncodeCheck(d.GetFrom()),
				To:                 tronutils.EncodeCheck(d.GetTo()),
				Bandwidth:          SUN(d.GetFrozenBalanceForBandwidth()),
				BandwidthExpiresAt: msToTime(d.GetExpireTimeForBandwidth()),
				Energy:             SUN(d.GetFrozenBalanceForEnergy()),
				EnergyExpiresAt:    msToTime(d.GetExpireTimeForEnergy()),
			})
		}
	}

	return result, nil
}

// msToTime converts a Unix-millisecond field to a time.Time, mapping the
// chain's "not set" zero to the zero time rather than to 1970.
func msToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// GetCanDelegatedMaxSize returns the largest stake the account may still
// delegate for the given resource.
//
// The answer is a staked TRX amount, not an amount of the resource: delegation
// moves the stake, and how much bandwidth or energy that yields depends on the
// network weights at the time. Convert with ConvertStakedToBandwidth or
// ConvertStakedToEnergy.
func (c *Client) GetCanDelegatedMaxSize(ctx context.Context, addr string, resource ResourceType) (SUN, error) {
	if addr == "" {
		return 0, fmt.Errorf("%w: address is required", ErrEmptyAddress)
	}

	if err := resource.Validate(); err != nil {
		return 0, err
	}

	addrBytes, err := tronutils.DecodeCheck(addr)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidAddress, err)
	}

	response, err := c.transport.GetCanDelegatedMaxSize(ctx, &api.CanDelegatedMaxSizeRequestMessage{
		OwnerAddress: addrBytes,
		Type:         int32(resource.ToProto()),
	})
	if err != nil {
		return 0, err
	}

	return SUN(response.GetMaxSize()), nil
}

// DelegateResource delegates a resource from one account to another
func (c *Client) DelegateResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance SUN, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	if err := address.Validate(receiver); err != nil {
		return nil, fmt.Errorf("%w: receiver address is required", ErrInvalidAddress)
	}

	if err := resource.Validate(); err != nil {
		return nil, err
	}

	if delegateBalance <= 0 {
		return nil, fmt.Errorf("%w: delegate balance must be greater than zero", ErrInvalidAmount)
	}

	addrFromBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := tronutils.DecodeCheck(receiver)
	if err != nil {
		return nil, err
	}

	contract := &core.DelegateResourceContract{}

	contract.Resource = resource.ToProto()
	contract.OwnerAddress = addrFromBytes
	contract.ReceiverAddress = addrToBytes
	contract.Balance = delegateBalance.Int64()
	contract.Lock = lock
	contract.LockPeriod = lockPeriod

	response, err := c.transport.DelegateResource(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(response); err != nil {
		return nil, err
	}

	return response, nil
}

// ReclaimResource reclaims a delegated resource from one account to another
func (c *Client) ReclaimResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance SUN) (*api.TransactionExtention, error) {
	if err := address.Validate(owner); err != nil {
		return nil, fmt.Errorf("%w: owner address is required", ErrInvalidAddress)
	}

	if err := address.Validate(receiver); err != nil {
		return nil, fmt.Errorf("%w: receiver address is required", ErrInvalidAddress)
	}

	if err := resource.Validate(); err != nil {
		return nil, err
	}

	if delegateBalance <= 0 {
		return nil, fmt.Errorf("%w: delegate balance must be greater than zero", ErrInvalidAmount)
	}

	addrOwnerBytes, err := tronutils.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	addrReceiverBytes, err := tronutils.DecodeCheck(receiver)
	if err != nil {
		return nil, err
	}

	contract := &core.UnDelegateResourceContract{}

	contract.Resource = resource.ToProto()
	contract.OwnerAddress = addrOwnerBytes
	contract.ReceiverAddress = addrReceiverBytes
	contract.Balance = delegateBalance.Int64()

	response, err := c.transport.UnDelegateResource(ctx, contract)
	if err != nil {
		return nil, err
	}

	if err := checkTransaction(response); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) AvailableForDelegateResources(ctx context.Context, addr string) (*AvailableResources, error) {
	chainParams, err := c.ChainParams(ctx)
	if err != nil {
		return nil, err
	}

	account, err := c.GetAccount(ctx, addr)
	if err != nil {
		return nil, err
	}

	accountResources, err := c.GetAccountResource(ctx, addr)
	if err != nil {
		return nil, err
	}

	stackedEnergy, stackedBandwidth := decimal.Zero, decimal.Zero
	for _, item := range account.FrozenV2 {
		if item.Type == core.ResourceCode_BANDWIDTH {
			stackedBandwidth = stackedBandwidth.Add(c.ConvertStakedToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, SUN(item.Amount)))
		}
		if item.Type == core.ResourceCode_ENERGY {
			stackedEnergy = stackedEnergy.Add(c.ConvertStakedToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, SUN(item.Amount)))
		}
	}

	resources := &AvailableResources{
		Energy:         c.AvailableEnergy(accountResources),
		TotalEnergy:    c.TotalEnergyLimit(accountResources),
		Bandwidth:      c.AvailableBandwidth(accountResources),
		TotalBandwidth: c.TotalBandwidthLimit(accountResources),
	}

	if stackedEnergy.LessThan(resources.Energy) {
		resources.Energy = stackedEnergy
	}

	if stackedBandwidth.LessThan(resources.Bandwidth) {
		resources.Bandwidth = stackedBandwidth
	}

	return resources, nil
}

// AvailableEnergy calculates the available energy.
func (c *Client) AvailableEnergy(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.EnergyLimit - res.EnergyUsed)
}

// AvailableBandwidth calculates the available bandwidth.
func (c *Client) AvailableBandwidth(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.GetNetLimit() + res.GetFreeNetLimit() - res.GetNetUsed() - res.GetFreeNetUsed())
}

func (c *Client) AvailableBandwidthWithoutFree(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.GetNetLimit() - res.GetNetUsed())
}

// AvailableFreeBandwidth returns what is left of the daily free allowance.
//
// It is deliberately separate from the staked pool rather than folded into
// AvailableBandwidth: Tron charges a transaction to one pool or the other, never
// to both, so the two numbers answer different questions and their sum answers
// neither. See billableBandwidth.
func (c *Client) AvailableFreeBandwidth(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.GetFreeNetLimit() - res.GetFreeNetUsed())
}

func (c *Client) TotalEnergyLimit(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.EnergyLimit)
}

func (c *Client) TotalBandwidthLimit(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.NetLimit + res.FreeNetLimit)
}

// TotalAvailableResources calculates the total available resources for the account.
func (c *Client) TotalAvailableResources(ctx context.Context, addr string) (*AvailableResources, error) {
	accountResources, err := c.GetAccountResource(ctx, addr)
	if err != nil {
		return nil, err
	}

	resources := &AvailableResources{
		Energy:         c.AvailableEnergy(accountResources),
		Bandwidth:      c.AvailableBandwidth(accountResources),
		TotalEnergy:    c.TotalEnergyLimit(accountResources),
		TotalBandwidth: c.TotalBandwidthLimit(accountResources),
	}

	return resources, nil
}
