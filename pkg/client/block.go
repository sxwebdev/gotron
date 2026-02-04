package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// GetLastBlockHeight return last block number from the blockchain.
func (c *Client) GetLastBlockHeight(ctx context.Context) (uint64, error) {
	result, err := c.transport.GetNowBlock(ctx)
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
	result, err := c.transport.GetNowBlock(ctx)
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
	result, err := c.transport.GetBlockByNum(ctx, int64(height))
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
	result, err := c.transport.GetBlockById(ctx, hash)
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
	result, err := c.transport.GetTransactionInfoByBlockNum(ctx, int64(number))
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
	result, err := c.transport.GetBlockByLimitNext(ctx, int64(start), int64(end))
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
	result, err := c.transport.GetBlockByLatestNum(ctx, int64(height))
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrNilResponse
	}

	return result, nil
}
