package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

// publicnode.com restricts /wallet/getnodeinfo (gRPC returns "Method not
// allowed", HTTP returns an empty body), so GetNodeInfo tests use TronGrid
// instead of the shared common_test.go helpers.
const (
	tronGridGRPCAddress = "grpc.trongrid.io:50051"
	tronGridHTTPAddress = "https://api.trongrid.io"
)

func newTronGridGRPCClient(t *testing.T) *client.Client {
	t.Helper()
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{Protocol: client.ProtocolGRPC, Address: tronGridGRPCAddress, UseTLS: false},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	return c
}

func newTronGridHTTPClient(t *testing.T) *client.Client {
	t.Helper()
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{Protocol: client.ProtocolHTTP, Address: tronGridHTTPAddress},
		},
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	return c
}

func TestGetNodeInfo_GRPC(t *testing.T) {
	c := newTronGridGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.GetNodeInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.NotEmpty(t, info.GetBlock())

	t.Logf("gRPC: Block=%s SolidityBlock=%s ActiveConnectCount=%d",
		info.GetBlock(), info.GetSolidityBlock(), info.GetActiveConnectCount())
}

func TestGetNodeInfo_HTTP(t *testing.T) {
	c := newTronGridHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.GetNodeInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.NotEmpty(t, info.GetBlock())

	t.Logf("HTTP: Block=%s SolidityBlock=%s ActiveConnectCount=%d",
		info.GetBlock(), info.GetSolidityBlock(), info.GetActiveConnectCount())
}
