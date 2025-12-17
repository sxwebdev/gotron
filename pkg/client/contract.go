package client

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// UpdateEnergyLimitContract update contract enery limit
func (c *Client) UpdateEnergyLimitContract(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error) {
	fromDesc, err := tronutils.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.UpdateEnergyLimitContract{
		OwnerAddress:      fromDesc,
		ContractAddress:   contractDesc,
		OriginEnergyLimit: value,
	}

	tx, err := c.walletClient.UpdateEnergyLimit(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// UpdateSettingContract change contract owner consumption ratio
func (c *Client) UpdateSettingContract(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error) {
	fromDesc, err := tronutils.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.UpdateSettingContract{
		OwnerAddress:               fromDesc,
		ContractAddress:            contractDesc,
		ConsumeUserResourcePercent: value,
	}

	tx, err := c.walletClient.UpdateSetting(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// TriggerConstantContract and return tx result
func (c *Client) TriggerConstantContract(ctx context.Context, from, contractAddress, method, jsonString string) (*api.TransactionExtention, error) {
	var err error
	fromDesc, err := tronutils.FromHex("410000000000000000000000000000000000000000")
	if err != nil {
		return nil, err
	}

	if len(from) > 0 {
		fromDesc, err = tronutils.DecodeCheck(from)
		if err != nil {
			return nil, err
		}
	}
	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc,
		ContractAddress: contractDesc,
		Data:            dataBytes,
	}

	return c.triggerConstantContract(ctx, ct)
}

// triggerConstantContract and return tx result
func (c *Client) triggerConstantContract(ctx context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return c.walletClient.TriggerConstantContract(ctx, ct)
}

// TriggerContract and return tx result
func (c *Client) TriggerContract(ctx context.Context, from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64,
) (*api.TransactionExtention, error) {
	fromDesc, err := tronutils.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc,
		ContractAddress: contractDesc,
		Data:            dataBytes,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return c.triggerContract(ctx, ct, feeLimit)
}

// triggerContract and return tx result
func (c *Client) triggerContract(ctx context.Context, ct *core.TriggerSmartContract, feeLimit int64) (*api.TransactionExtention, error) {
	tx, err := c.walletClient.TriggerContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		err = c.UpdateHash(tx)
	}

	return tx, err
}

// EstimateEnergy returns enery required
func (c *Client) EstimateEnergy(ctx context.Context, from, contractAddress, method, jsonString string,
	tAmount int64, tTokenID string, tTokenAmount int64,
) (*api.EstimateEnergyMessage, error) {
	fromDesc, err := tronutils.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc,
		ContractAddress: contractDesc,
		Data:            dataBytes,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return c.estimateEnergy(ctx, ct)
}

// triggerContract and return tx result
func (c *Client) estimateEnergy(ctx context.Context, ct *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	tx, err := c.walletClient.EstimateEnergy(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// DeployContract and return tx result
func (c *Client) DeployContract(ctx context.Context, from, contractName string,
	abi *core.SmartContract_ABI, codeStr string,
	feeLimit, curPercent, oeLimit int64,
) (*api.TransactionExtention, error) {
	var err error

	fromDesc, err := tronutils.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	if curPercent > 100 || curPercent < 0 {
		return nil, fmt.Errorf("consume_user_resource_percent should be >= 0 and <= 100")
	}
	if oeLimit <= 0 {
		return nil, fmt.Errorf("origin_energy_limit must > 0")
	}

	bc, err := tronutils.FromHex(codeStr)
	if err != nil {
		return nil, err
	}

	ct := &core.CreateSmartContract{
		OwnerAddress: fromDesc,
		NewContract: &core.SmartContract{
			OriginAddress:              fromDesc,
			Abi:                        abi,
			Name:                       contractName,
			ConsumeUserResourcePercent: curPercent,
			OriginEnergyLimit:          oeLimit,
			Bytecode:                   bc,
		},
	}

	tx, err := c.walletClient.DeployContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		err = c.UpdateHash(tx)
	}

	return tx, err
}

// UpdateHash after local changes
func (c *Client) UpdateHash(tx *api.TransactionExtention) error {
	return tx.UpdateHash()
}

// GetContractABI return smartContract
func (c *Client) GetContractABI(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error) {
	var err error
	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	sm, err := c.walletClient.GetContract(ctx, GetMessageBytes(contractDesc))
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, fmt.Errorf("invalid contract abi")
	}

	return sm.Abi, nil
}
