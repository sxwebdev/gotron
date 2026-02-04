package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TRC10 asset tests

func TestGetAssetIssueById_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// BTT token ID (TRC10)
	assetID := "1002000"
	asset, err := c.GetAssetIssueById(ctx, assetID)
	require.NoError(t, err)
	require.NotNil(t, asset)

	t.Logf("gRPC: Asset name: %s, abbr: %s, total supply: %d",
		string(asset.GetName()), string(asset.GetAbbr()), asset.GetTotalSupply())
}

func TestGetAssetIssueById_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// BTT token ID (TRC10)
	assetID := "1002000"
	asset, err := c.GetAssetIssueById(ctx, assetID)
	require.NoError(t, err)
	require.NotNil(t, asset)

	t.Logf("HTTP: Asset name: %s, abbr: %s, total supply: %d",
		string(asset.GetName()), string(asset.GetAbbr()), asset.GetTotalSupply())
}

func TestGetAssetIssueListByName_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Search for BTT token by name
	assetName := "BTT"
	assets, err := c.GetAssetIssueListByName(ctx, assetName)
	require.NoError(t, err)
	require.NotNil(t, assets)

	t.Logf("gRPC: Found %d assets with name '%s'", len(assets.GetAssetIssue()), assetName)
	for _, asset := range assets.GetAssetIssue() {
		t.Logf("  - ID: %s, name: %s, abbr: %s",
			asset.GetId(), string(asset.GetName()), string(asset.GetAbbr()))
	}
}

func TestGetAssetIssueListByName_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Search for BTT token by name
	assetName := "BTT"
	assets, err := c.GetAssetIssueListByName(ctx, assetName)
	require.NoError(t, err)
	require.NotNil(t, assets)

	t.Logf("HTTP: Found %d assets with name '%s'", len(assets.GetAssetIssue()), assetName)
	for _, asset := range assets.GetAssetIssue() {
		t.Logf("  - ID: %s, name: %s, abbr: %s",
			asset.GetId(), string(asset.GetName()), string(asset.GetAbbr()))
	}
}
