package client

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// EstimateBandwidth returns the bandwidth points a transaction will consume:
// its serialized size plus Tron's 64-byte protocol overhead.
//
// tx is left untouched. Sizing it needs a signature, and filling one in is
// destructive - it clears Ret and appends a throwaway signature - so the
// measurement is taken on a clone. Doing it in place corrupted the very
// transaction the caller was about to sign, which is the only reason anyone
// calls this.
func (c *Client) EstimateBandwidth(tx *core.Transaction) (decimal.Decimal, error) {
	if tx == nil {
		return decimal.Decimal{}, fmt.Errorf("%w: transaction is nil", ErrInvalidTransaction)
	}

	probe := proto.CloneOf(tx)
	if err := fillFakeTX(probe); err != nil {
		return decimal.Decimal{}, err
	}

	return decimal.NewFromInt(int64(proto.Size(probe))).Add(decimal.NewFromInt(64)), nil
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

	tx, err := c.transport.EstimateEnergy(ctx, ct)
	if err != nil {
		return nil, err
	}

	if code := tx.GetResult().GetCode(); code != 0 {
		msg := string(tx.GetResult().GetMessage())
		// The node would not run the estimate: a request it refuses (no
		// contract at the address, the estimate API off), not a call that
		// failed — callers branching on ErrContractCallFailed take that for
		// "the real transaction reverts too".
		if code == api.Return_CONTRACT_VALIDATE_ERROR {
			return nil, &ContractValidateError{Code: code, Message: msg}
		}
		return nil, &ContractCallError{Code: code, Message: msg}
	}

	return tx, nil
}
