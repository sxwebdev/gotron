package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

const (
	trc20TransferMethodSignature     = "0xa9059cbb"
	trc20ApproveMethodSignature      = "0x095ea7b3"
	Trc20TransferEventSignature      = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	trc20NameSignature               = "0x06fdde03"
	trc20SymbolSignature             = "0x95d89b41"
	trc20DecimalsSignature           = "0x313ce567"
	trc20BalanceOf                   = "0x70a08231"
	Trc20TransferFromMethodSignature = "0x23b872dd"
)

func (c *Client) TRC20Call(ctx context.Context, from, contractAddress, data string, constant bool, feeLimit int64) (*api.TransactionExtention, error) {
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
	dataBytes, err := tronutils.FromHex(data)
	if err != nil {
		return nil, err
	}
	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc,
		ContractAddress: contractDesc,
		Data:            dataBytes,
	}
	var result *api.TransactionExtention
	if constant {
		result, err = c.triggerConstantContract(ctx, ct)
	} else {
		result, err = c.triggerContract(ctx, ct, feeLimit)
	}
	if err != nil {
		return nil, err
	}
	if result.Result.Code > 0 {
		return result, fmt.Errorf("%s", string(result.Result.Message))
	}
	return result, nil
}

// TRC20GetName get token name
func (c *Client) TRC20GetName(ctx context.Context, contractAddress string) (string, error) {
	result, err := c.TRC20Call(ctx, "", contractAddress, trc20NameSignature, true, 0)
	if err != nil {
		return "", err
	}
	data := tronutils.BytesToHexString(result.GetConstantResult()[0])
	return c.ParseTRC20StringProperty(data)
}

// TRC20GetSymbol get contract symbol
func (c *Client) TRC20GetSymbol(ctx context.Context, contractAddress string) (string, error) {
	result, err := c.TRC20Call(ctx, "", contractAddress, trc20SymbolSignature, true, 0)
	if err != nil {
		return "", err
	}
	data := tronutils.BytesToHexString(result.GetConstantResult()[0])
	return c.ParseTRC20StringProperty(data)
}

// TRC20GetDecimals get contract decimals
func (c *Client) TRC20GetDecimals(ctx context.Context, contractAddress string) (*big.Int, error) {
	result, err := c.TRC20Call(ctx, "", contractAddress, trc20DecimalsSignature, true, 0)
	if err != nil {
		return nil, err
	}
	data := tronutils.BytesToHexString(result.GetConstantResult()[0])
	return c.ParseTRC20NumericProperty(data)
}

// ParseTRC20NumericProperty get number from data
func (c *Client) ParseTRC20NumericProperty(data string) (*big.Int, error) {
	if tronutils.Has0xPrefix(data) {
		data = data[2:]
	}
	if len(data) == 64 {
		var n big.Int
		_, ok := n.SetString(data, 16)
		if ok {
			return &n, nil
		}
	}

	if len(data) == 0 {
		return big.NewInt(0), nil
	}

	return nil, fmt.Errorf("cannot parse %s", data)
}

// ParseTRC20StringProperty get string from data
func (c *Client) ParseTRC20StringProperty(data string) (string, error) {
	if tronutils.Has0xPrefix(data) {
		data = data[2:]
	}
	if len(data) > 128 {
		n, _ := c.ParseTRC20NumericProperty(data[64:128])
		if n != nil {
			l := n.Uint64()
			if 2*int(l) <= len(data)-128 {
				b, err := hex.DecodeString(data[128 : 128+2*l])
				if err == nil {
					return string(b), nil
				}
			}
		}
	} else if len(data) == 64 {
		// allow string properties as 32 bytes of UTF-8 data
		b, err := hex.DecodeString(data)
		if err == nil {
			i := bytes.Index(b, []byte{0})
			if i > 0 {
				b = b[:i]
			}
			if utf8.Valid(b) {
				return string(b), nil
			}
		}
	}
	return "", fmt.Errorf("cannot parse %s,", data)
}

// TRC20ContractBalance get Address balance
func (c *Client) TRC20ContractBalance(ctx context.Context, addr, contractAddress string) (*big.Int, error) {
	addrB, err := tronutils.DecodeCheck(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %v", addr, addr)
	}
	req := trc20BalanceOf + "0000000000000000000000000000000000000000000000000000000000000000"[len(tronutils.BytesToHexString(addrB[:]))-2:] + tronutils.BytesToHexString(addrB[:])[2:]
	result, err := c.TRC20Call(ctx, "", contractAddress, req, true, 0)
	if err != nil {
		return nil, err
	}
	data := tronutils.BytesToHexString(result.GetConstantResult()[0])
	r, err := c.ParseTRC20NumericProperty(data)
	if err != nil {
		return nil, fmt.Errorf("contract address %s: %v", contractAddress, err)
	}
	if r == nil {
		return nil, fmt.Errorf("contract address %s: invalid balance of %s", contractAddress, addr)
	}
	return r, nil
}

func (c *Client) TRC20Send(ctx context.Context, from, to, contract string, amount decimal.Decimal, feeLimit int64) (*api.TransactionExtention, error) {
	if contract == "" {
		return nil, fmt.Errorf("%w: contract address is required", ErrInvalidAddress)
	}

	if from == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if to == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	if feeLimit <= 0 {
		return nil, fmt.Errorf("%w: fee limit must be greater than zero", ErrInvalidParams)
	}

	addrB, err := tronutils.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.BigInt().Bytes(), 32)
	req := trc20TransferMethodSignature + "0000000000000000000000000000000000000000000000000000000000000000"[len(tronutils.BytesToHexString(addrB[:]))-4:] + tronutils.BytesToHexString(addrB[:])[4:]
	req += common.Bytes2Hex(ab)
	return c.TRC20Call(ctx, from, contract, req, false, feeLimit)
}

func (c *Client) TRC20TransferFrom(ctx context.Context, owner, from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	addrA, err := tronutils.DecodeCheck(from)
	if err != nil {
		return nil, err
	}
	addrB, err := tronutils.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.Bytes(), 32)
	req := "0x23b872dd" +
		"0000000000000000000000000000000000000000000000000000000000000000"[len(tronutils.BytesToHexString(addrA[:]))-4:] + tronutils.BytesToHexString(addrA[:])[4:] +
		"0000000000000000000000000000000000000000000000000000000000000000"[len(tronutils.BytesToHexString(addrB[:]))-4:] + tronutils.BytesToHexString(addrB[:])[4:]
	req += common.Bytes2Hex(ab)
	return c.TRC20Call(ctx, owner, contract, req, false, feeLimit)
}

// TRC20Approve approve token to address
func (c *Client) TRC20Approve(ctx context.Context, from, to, contract string, amount decimal.Decimal, feeLimit int64) (*api.TransactionExtention, error) {
	if contract == "" {
		return nil, fmt.Errorf("%w: contract address is required", ErrInvalidAddress)
	}

	if from == "" {
		return nil, fmt.Errorf("%w: from address is required", ErrInvalidAddress)
	}

	if to == "" {
		return nil, fmt.Errorf("%w: to address is required", ErrInvalidAddress)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: amount must be greater than zero", ErrInvalidAmount)
	}

	if feeLimit < 0 {
		return nil, fmt.Errorf("%w: fee limit must be greater than zero", ErrInvalidParams)
	}

	addrB, err := tronutils.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.BigInt().Bytes(), 32)
	req := trc20ApproveMethodSignature + "0000000000000000000000000000000000000000000000000000000000000000"[len(tronutils.BytesToHexString(addrB[:]))-4:] + tronutils.BytesToHexString(addrB[:])[4:]
	req += common.Bytes2Hex(ab)
	return c.TRC20Call(ctx, from, contract, req, false, feeLimit)
}
