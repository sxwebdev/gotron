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

// EstimateBandwidth calculates the estimated bandwidth.
func (c *Client) EstimateBandwidth(tx *core.Transaction) (decimal.Decimal, error) {
	if err := fillFakeTX(tx); err != nil {
		return decimal.Decimal{}, err
	}

	return decimal.NewFromInt(int64(proto.Size(tx))).Add(decimal.NewFromInt(64)), nil
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

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}
