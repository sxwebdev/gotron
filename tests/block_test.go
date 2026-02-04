package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNowBlock_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	block, err := c.GetLastBlock(ctx)
	require.NoError(t, err)
	require.NotNil(t, block)

	blockNum := block.GetBlockHeader().GetRawData().GetNumber()
	assert.Greater(t, blockNum, int64(0))
	t.Logf("gRPC: Latest block number: %d", blockNum)
}

func TestGetNowBlock_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	block, err := c.GetLastBlock(ctx)
	require.NoError(t, err)
	require.NotNil(t, block)

	blockNum := block.GetBlockHeader().GetRawData().GetNumber()
	assert.Greater(t, blockNum, int64(0))
	t.Logf("HTTP: Latest block number: %d", blockNum)
}

func TestGetBlockByHeight_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get a recent block
	latestHeight, err := c.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	block, err := c.GetBlockByHeight(ctx, latestHeight-10)
	require.NoError(t, err)
	require.NotNil(t, block)

	t.Logf("gRPC: Block %d has %d transactions", latestHeight-10, len(block.GetTransactions()))
}

func TestGetBlockByHeight_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get a recent block
	latestHeight, err := c.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	block, err := c.GetBlockByHeight(ctx, latestHeight-10)
	require.NoError(t, err)
	require.NotNil(t, block)

	t.Logf("HTTP: Block %d has %d transactions", latestHeight-10, len(block.GetTransactions()))
}

func TestGetTransactionInfoByBlockNum_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	txInfoList, err := c.GetTransactionInfoByBlockNum(ctx, testBlockNum)
	require.NoError(t, err)
	require.NotNil(t, txInfoList)

	t.Logf("gRPC: Block %d has %d transaction infos", testBlockNum, len(txInfoList.GetTransactionInfo()))
}

func TestGetTransactionInfoByBlockNum_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	txInfoList, err := c.GetTransactionInfoByBlockNum(ctx, testBlockNum)
	require.NoError(t, err)
	require.NotNil(t, txInfoList)

	t.Logf("HTTP: Block %d has %d transaction infos", testBlockNum, len(txInfoList.GetTransactionInfo()))
}
