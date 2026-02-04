package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

// TestGetBlockRange_LargeResponse_GRPC tests fetching a large range of blocks via gRPC.
// This verifies that MaxCallRecvMsgSize is properly configured for large responses.
func TestGetBlockRange_LargeResponse_GRPC(t *testing.T) {
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolGRPC,
				Address:  grpcAddress,
				UseTLS:   true,
			},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Fetch 100 blocks - response is ~13MB, requires MaxCallRecvMsgSize > 4MB default
	startBlock := int64(79845122)
	endBlock := int64(79845222)

	blocks, err := c.GetBlockByLimitNext2(ctx, uint64(startBlock), uint64(endBlock))
	require.NoError(t, err, "Failed to get block range - check MaxCallRecvMsgSize setting")
	require.NotNil(t, blocks)

	t.Logf("gRPC: Successfully fetched %d blocks from %d to %d", len(blocks.GetBlock()), startBlock, endBlock)
}

// TestGetBlockRange_LargeResponse_HTTP tests that HTTP transport handles large responses.
func TestGetBlockRange_LargeResponse_HTTP(t *testing.T) {
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolHTTP,
				Address:  httpAddress,
			},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	startBlock := int64(79845122)
	endBlock := int64(79845222)

	blocks, err := c.GetBlockByLimitNext2(ctx, uint64(startBlock), uint64(endBlock))
	require.NoError(t, err, "Failed to get block range via HTTP")
	require.NotNil(t, blocks)

	t.Logf("HTTP: Successfully fetched %d blocks from %d to %d", len(blocks.GetBlock()), startBlock, endBlock)
}
