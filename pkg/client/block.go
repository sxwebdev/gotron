package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/grpc"
)

var defaultMaxSizeOption = grpc.MaxCallRecvMsgSize(32 * 10e6)

// GetLastBlockHeight return last block number from the blockchain.
func (c *Client) GetLastBlockHeight(ctx context.Context) (uint64, error) {
	result, err := c.walletClient.GetNowBlock2(ctx, new(api.EmptyMessage))
	if err != nil {
		return 0, err
	}

	if result == nil || result.GetBlockHeader() == nil || result.GetBlockHeader().GetRawData() == nil {
		return 0, ErrNilResponse
	}

	return uint64(result.GetBlockHeader().GetRawData().GetNumber()), nil
}

// GetLastBlock return last block from the blockchain.
func (c *Client) GetLastBlock(ctx context.Context) (*api.BlockExtention, error) {
	result, err := c.walletClient.GetNowBlock2(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}

// GetBlockByHeight returns block by its number.
func (c *Client) GetBlockByHeight(ctx context.Context, height uint64) (*api.BlockExtention, error) {
	req := &api.NumberMessage{
		Num: int64(height),
	}

	result, err := c.walletClient.GetBlockByNum2(ctx, req, defaultMaxSizeOption)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}

// GetBlockByHash returns block by its hash.
func (c *Client) GetBlockByHash(ctx context.Context, hash []byte) (*core.Block, error) {
	req := &api.BytesMessage{
		Value: hash,
	}

	result, err := c.walletClient.GetBlockById(ctx, req, defaultMaxSizeOption)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}

// GetTransactionInfoByBlockNum returns transaction info list by block number.
func (c *Client) GetTransactionInfoByBlockNum(ctx context.Context, number uint64) (*api.TransactionInfoList, error) {
	req := &api.NumberMessage{
		Num: int64(number),
	}

	result, err := c.walletClient.GetTransactionInfoByBlockNum(ctx, req, defaultMaxSizeOption)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}

// GetBlockByLimitNext returns blocks in the range [start, start+limit).
func (c *Client) GetBlockByLimitNext2(ctx context.Context, start uint64, end uint64) (*api.BlockListExtention, error) {
	req := &api.BlockLimit{
		StartNum: int64(start),
		EndNum:   int64(end),
	}

	result, err := c.walletClient.GetBlockByLimitNext2(ctx, req, defaultMaxSizeOption)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}

// GetBlockByLatestNum returns the latest 'num' blocks.
func (c *Client) GetBlockByLatestNum2(ctx context.Context, height uint64) (*api.BlockListExtention, error) {
	req := &api.NumberMessage{
		Num: int64(height),
	}

	result, err := c.walletClient.GetBlockByLatestNum2(ctx, req, defaultMaxSizeOption)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}
