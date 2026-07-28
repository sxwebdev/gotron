package client

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
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

	tx, err := c.transport.UpdateEnergyLimit(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.GetResult().GetCode() > 0 {
		return nil, fmt.Errorf("%s", string(tx.GetResult().GetMessage()))
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

	tx, err := c.transport.UpdateSetting(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.GetResult().GetCode() > 0 {
		return nil, fmt.Errorf("%s", string(tx.GetResult().GetMessage()))
	}

	return tx, err
}

// TriggerConstantContractCustom and return tx result
func (c *Client) TriggerConstantContractCustom(ctx context.Context, from, contractAddress, method, jsonString string) (*api.TransactionExtention, error) {
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

	return c.TriggerConstantContract(ctx, ct)
}

// TriggerConstantContract runs a contract call without broadcasting it and
// returns the node's answer.
//
// A call the VM refused is reported as an error wrapping ErrContractCallFailed,
// and the extention is returned alongside it so the caller can still read
// constant_result and the energy burned before the failure. Detection is by
// result.message, not by the result code: a revert arrives with code SUCCESS.
func (c *Client) TriggerConstantContract(ctx context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	tx, err := c.transport.TriggerConstantContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	if msg := tx.GetResult().GetMessage(); len(msg) > 0 {
		return tx, fmt.Errorf("%w: %s", ErrContractCallFailed, msg)
	}
	if code := tx.GetResult().GetCode(); code != 0 {
		return tx, fmt.Errorf("%w: %s", ErrContractCallFailed, code)
	}

	return tx, nil
}

// TriggerContract and return tx result
func (c *Client) TriggerContract(ctx context.Context, from, contractAddress, method, jsonString string,
	feeLimit SUN, tAmount int64, tTokenID string, tTokenAmount int64,
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

	return c.triggerContract(ctx, ct, feeLimit.Int64())
}

// triggerContract and return tx result
func (c *Client) triggerContract(ctx context.Context, ct *core.TriggerSmartContract, feeLimit int64) (*api.TransactionExtention, error) {
	tx, err := c.transport.TriggerContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.GetResult().GetCode() > 0 {
		return nil, fmt.Errorf("%s", string(tx.GetResult().GetMessage()))
	}

	if feeLimit > 0 {
		if tx.GetTransaction().GetRawData() == nil {
			return nil, ErrInvalidTransaction
		}
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		err = c.UpdateHash(tx)
	}

	return tx, err
}

// DeployContractRequest describes a contract deployment.
//
// It is a struct rather than a parameter list because the deployment already
// carries four strings in a row, which no compiler can tell apart if two are
// swapped, and because Tron keeps adding to CreateSmartContract - a payable
// constructor's call value and an attached TRC10 both belong here eventually,
// and a field can be added without breaking callers.
type DeployContractRequest struct {
	// From is the deploying account in base58check form.
	From string
	// Name is stored on-chain with the contract; it is metadata, not an identifier.
	Name string
	// ABI is the contract's interface. Build it from the compiler's output with
	// abi.LoadContractABI. A contract deploys without one, but then nothing can
	// decode its calls afterwards.
	ABI *core.SmartContract_ABI
	// Bytecode is the compiled contract as hex, with or without an 0x prefix.
	// Do not append constructor arguments to it - use ConstructorParams, which
	// encodes them the way the chain expects.
	Bytecode string
	// ConstructorParams are the constructor's arguments in the pkg/client/abi
	// JSON form, e.g. `[{"address":"T..."},{"uint256":"1000"}]`. Leave empty
	// when the constructor takes none.
	ConstructorParams string
	// FeeLimit caps what the deployment may burn. Zero leaves the field unset,
	// which lets the node apply its own default - usually far below what
	// deploying costs, so set it.
	FeeLimit SUN
	// ConsumeUserResourcePercent is the share (0..100) of each future call's
	// cost paid by the caller instead of by this contract's own resources.
	ConsumeUserResourcePercent int64
	// OriginEnergyLimit caps the energy the contract's owner pays per call. The
	// chain rejects a deployment that does not set it.
	OriginEnergyLimit int64
}

// Validate reports whether the request is one the chain would accept, without
// spending a round-trip to find out.
func (r DeployContractRequest) Validate() error {
	_, err := r.build()
	return err
}

// build validates the request and turns it into the contract the transport
// takes. Validation and construction are the same pass so that the checks
// cannot drift from what is actually sent, and so that nothing is decoded twice.
func (r DeployContractRequest) build() (*core.CreateSmartContract, error) {
	if r.From == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrEmptyAddress)
	}
	owner, err := tronutils.DecodeCheck(r.From)
	if err != nil {
		return nil, fmt.Errorf("%w: from address: %s", ErrInvalidAddress, err)
	}

	// The prefix is stripped before the emptiness check, so that "0x" - hex for
	// nothing at all - is rejected like an empty string rather than deploying a
	// contract with no code.
	code := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(r.Bytecode), "0x"), "0X")
	if code == "" {
		return nil, fmt.Errorf("%w: bytecode is required", ErrInvalidParams)
	}
	// tronutils.FromHex left-pads an odd-length string with a zero nibble, which
	// for bytecode is worse than an error: every byte after the pad shifts by
	// half a byte, so truncated input deploys code that compiles to something
	// else entirely instead of failing. Bytecode is always even-length.
	if len(code)%2 == 1 {
		return nil, fmt.Errorf("%w: bytecode has an odd number of hex digits, so it is truncated", ErrInvalidParams)
	}
	bytecode, err := tronutils.FromHex(code)
	if err != nil {
		return nil, fmt.Errorf("%w: bytecode: %s", ErrInvalidParams, err)
	}

	if r.ConsumeUserResourcePercent < 0 || r.ConsumeUserResourcePercent > 100 {
		return nil, fmt.Errorf("%w: consume_user_resource_percent must be between 0 and 100", ErrInvalidParams)
	}
	if r.OriginEnergyLimit <= 0 {
		return nil, fmt.Errorf("%w: origin_energy_limit must be greater than zero", ErrInvalidParams)
	}
	if r.FeeLimit < 0 {
		return nil, fmt.Errorf("%w: fee limit cannot be negative", ErrInvalidAmount)
	}

	// Tron has no separate field for constructor arguments: they are ABI-encoded
	// and appended to the bytecode, which is where the constructor reads them.
	if r.ConstructorParams != "" {
		params, err := abi.LoadFromJSON(r.ConstructorParams)
		if err != nil {
			return nil, fmt.Errorf("%w: constructor params: %s", ErrInvalidParams, err)
		}
		encoded, err := abi.GetPaddedParam(params)
		if err != nil {
			return nil, fmt.Errorf("%w: encode constructor params: %s", ErrInvalidParams, err)
		}
		bytecode = append(bytecode, encoded...)
	}

	return &core.CreateSmartContract{
		OwnerAddress: owner,
		NewContract: &core.SmartContract{
			OriginAddress:              owner,
			Abi:                        r.ABI,
			Name:                       r.Name,
			ConsumeUserResourcePercent: r.ConsumeUserResourcePercent,
			OriginEnergyLimit:          r.OriginEnergyLimit,
			Bytecode:                   bytecode,
		},
	}, nil
}

// DeployContract builds a deployment transaction. The transaction is unsigned
// and is not broadcast: sign it with SignTransaction and send it with
// BroadcastTransaction.
//
// The address the contract will occupy is already determined at this point -
// pass the returned transaction to DeployedContractAddress to read it without
// waiting for the receipt.
func (c *Client) DeployContract(ctx context.Context, req DeployContractRequest) (*api.TransactionExtention, error) {
	ct, err := req.build()
	if err != nil {
		return nil, err
	}

	tx, err := c.transport.DeployContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	// Without this a node that refused the deployment comes back as a
	// transaction carrying an error Result and a nil error, and the caller signs
	// and broadcasts nothing.
	if err := checkTransaction(tx); err != nil {
		return nil, err
	}

	if req.FeeLimit > 0 {
		if tx.GetTransaction().GetRawData() == nil {
			return nil, ErrInvalidTransaction
		}
		tx.Transaction.RawData.FeeLimit = req.FeeLimit.Int64()
		// The fee limit is part of raw_data, so it changes the txID - and with
		// it the contract's address, which is derived from that txID.
		if err := c.UpdateHash(tx); err != nil {
			return nil, err
		}
	}

	return tx, nil
}

// DeployedContractAddress returns the address a deployment transaction creates.
//
// Tron derives it from the transaction rather than from an account nonce:
// keccak256(txID || ownerAddress), keeping the low 20 bytes and putting the
// address prefix back. Nothing about it depends on the signature or on the
// transaction being mined, so the address can be recorded before broadcasting -
// but it does depend on raw_data, so read it only after the last edit to the
// transaction (DeployContract sets the fee limit before returning).
//
// Verified against mainnet: USDT (TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t) and
// TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY both reproduce from their creation
// transactions.
func DeployedContractAddress(tx *core.Transaction) (string, error) {
	contracts := tx.GetRawData().GetContract()
	if len(contracts) != 1 {
		return "", fmt.Errorf("%w: expected exactly one contract, got %d", ErrInvalidTransaction, len(contracts))
	}
	if contracts[0].GetType() != core.Transaction_Contract_CreateSmartContract {
		return "", fmt.Errorf("%w: not a contract deployment: %s", ErrInvalidTransaction, contracts[0].GetType())
	}

	var deploy core.CreateSmartContract
	if err := contracts[0].GetParameter().UnmarshalTo(&deploy); err != nil {
		return "", fmt.Errorf("%w: decode deployment: %s", ErrInvalidTransaction, err)
	}
	if len(deploy.GetOwnerAddress()) == 0 {
		return "", fmt.Errorf("%w: deployment has no owner address", ErrInvalidTransaction)
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", fmt.Errorf("%w: marshal raw data: %s", ErrInvalidTransaction, err)
	}
	txID := sha256.Sum256(rawData)

	return contractAddressFromTxID(txID[:], deploy.GetOwnerAddress()), nil
}

// contractAddressFromTxID is the derivation itself, kept apart from the
// transaction plumbing so it can be pinned against real mainnet deployments.
func contractAddressFromTxID(txID, ownerAddress []byte) string {
	seed := make([]byte, 0, len(txID)+len(ownerAddress))
	seed = append(seed, txID...)
	seed = append(seed, ownerAddress...)

	hash := tronutils.Keccak256(seed)

	return tronutils.EncodeCheck(append([]byte{tronutils.TronBytePrefix}, hash[len(hash)-20:]...))
}

// UpdateHash after local changes
func (c *Client) UpdateHash(tx *api.TransactionExtention) error {
	return tx.UpdateHash()
}

// GetContract returns smart contract information by address
func (c *Client) GetContract(ctx context.Context, contractAddress string) (*core.SmartContract, error) {
	contractDesc, err := tronutils.DecodeCheck(contractAddress)
	if err != nil {
		return nil, err
	}

	sm, err := c.transport.GetContract(ctx, contractDesc)
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, fmt.Errorf("contract not found")
	}

	return sm, nil
}

// GetContractABI return smartContract
func (c *Client) GetContractABI(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error) {
	sm, err := c.GetContract(ctx, contractAddress)
	if err != nil {
		return nil, err
	}
	if sm.Abi == nil {
		return nil, fmt.Errorf("contract has no ABI")
	}

	return sm.Abi, nil
}
