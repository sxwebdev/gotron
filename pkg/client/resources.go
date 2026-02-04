package client

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/address"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
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

// GetDelegatedResourcesV2 retrieves delegated resources for the specified account address
func (c *Client) GetDelegatedResources(ctx context.Context, address string) ([]*api.DelegatedResourceList, error) {
	addrBytes, err := tronutils.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ai, err := c.transport.GetDelegatedResourceAccountIndex(ctx, addrBytes)
	if err != nil {
		return nil, err
	}
	result := make([]*api.DelegatedResourceList, len(ai.GetToAccounts()))
	for i, addrTo := range ai.GetToAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		}
		resource, err := c.transport.GetDelegatedResource(ctx, dm)
		if err != nil {
			return nil, err
		}
		result[i] = resource
	}
	return result, nil
}

// GetDelegatedResourcesV2 retrieves delegated resources V2 for the specified account address
func (c *Client) GetDelegatedResourcesV2(ctx context.Context, address string) ([]*api.DelegatedResourceList, error) {
	addrBytes, err := tronutils.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ai, err := c.transport.GetDelegatedResourceAccountIndexV2(ctx, addrBytes)
	if err != nil {
		return nil, err
	}

	result := make([]*api.DelegatedResourceList, len(ai.GetToAccounts()))
	for i, addrTo := range ai.GetToAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		}
		resource, err := c.transport.GetDelegatedResourceV2(ctx, dm)
		if err != nil {
			return nil, err
		}
		result[i] = resource
	}
	return result, nil
}

// GetCanDelegatedMaxSize retrieves the maximum size that can be delegated for a given resource type
func (c *Client) GetCanDelegatedMaxSize(ctx context.Context, address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	addrBytes, err := tronutils.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	dm := &api.CanDelegatedMaxSizeRequestMessage{}

	dm.Type = resource
	dm.OwnerAddress = addrBytes

	response, err := c.transport.GetCanDelegatedMaxSize(ctx, dm)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DelegateResource delegates a resource from one account to another
func (c *Client) DelegateResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
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
	contract.Balance = delegateBalance
	contract.Lock = lock
	contract.LockPeriod = lockPeriod

	response, err := c.transport.DelegateResource(ctx, contract)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// ReclaimResource reclaims a delegated resource from one account to another
func (c *Client) ReclaimResource(ctx context.Context, owner, receiver string, resource ResourceType, delegateBalance int64) (*api.TransactionExtention, error) {
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
	contract.Balance = delegateBalance

	response, err := c.transport.UnDelegateResource(ctx, contract)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) AvailableForDelegateResources(ctx context.Context, addr string) (*Resources, error) {
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
			stackedBandwidth = stackedBandwidth.Add(c.ConvertStackedTRXToBandwidth(accountResources.TotalNetWeight, accountResources.TotalNetLimit, item.Amount))
		}
		if item.Type == core.ResourceCode_ENERGY {
			stackedEnergy = stackedEnergy.Add(c.ConvertStackedTRXToEnergy(chainParams.TotalEnergyCurrentLimit, accountResources.TotalEnergyWeight, item.Amount))
		}
	}

	resources := &Resources{
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
	return decimal.NewFromInt(res.NetLimit + res.GetFreeNetLimit() - res.GetNetUsed() - res.GetFreeNetUsed())
}

func (c *Client) AvailableBandwidthWithoutFree(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.NetLimit - res.GetNetUsed())
}

func (c *Client) TotalEnergyLimit(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.EnergyLimit)
}

func (c *Client) TotalBandwidthLimit(res *api.AccountResourceMessage) decimal.Decimal {
	return decimal.NewFromInt(res.NetLimit + res.FreeNetLimit)
}

// TotalAvailableResources calculates the total available resources for the account.
func (c *Client) TotalAvailableResources(ctx context.Context, addr string) (*Resources, error) {
	accountResources, err := c.GetAccountResource(ctx, addr)
	if err != nil {
		return nil, err
	}

	resources := &Resources{
		Energy:         c.AvailableEnergy(accountResources),
		Bandwidth:      c.AvailableBandwidth(accountResources),
		TotalEnergy:    c.TotalEnergyLimit(accountResources),
		TotalBandwidth: c.TotalBandwidthLimit(accountResources),
	}

	return resources, nil
}

// EstimateBandwidth calculates the estimated bandwidth.
func (c *Client) EstimateBandwidth(tx *core.Transaction) (decimal.Decimal, error) {
	if err := fillFakeTX(tx); err != nil {
		return decimal.Decimal{}, err
	}

	return decimal.NewFromInt(int64(proto.Size(tx))).Add(decimal.NewFromInt(64)), nil
}
