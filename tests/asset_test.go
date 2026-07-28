package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// TRC10 asset tests

// The TRC10 asset the assertions below are pinned to. Its fields are immutable
// once issued, so they can be compared against literals rather than logged.
const (
	bttAssetID     = "1002000"
	bttAssetName   = "BitTorrent"
	bttAssetAbbr   = "BTT"
	bttAssetOwner  = "TF5Bn4cJCT6GVeUgyCN4rBhDg42KBrpAjg"
	bttAssetURL    = "www.bittorrent.com"
	bttAssetSupply = int64(990_000_000_000_000_000)
)

// assertBTT checks the fields that carry an encoding: over HTTP every one of
// them arrives hex-encoded, and reading them as base64 turned the name into
// "\xe3n\xbd\xef…" and the 21-byte owner into 31 bytes - with no error, so only
// looking at the values catches it.
func assertBTT(t *testing.T, asset *core.AssetIssueContract) {
	t.Helper()

	require.Equal(t, bttAssetID, asset.GetId())
	require.Equal(t, bttAssetName, string(asset.GetName()))
	require.Equal(t, bttAssetAbbr, string(asset.GetAbbr()))
	require.Equal(t, bttAssetURL, string(asset.GetUrl()))
	require.Equal(t, bttAssetSupply, asset.GetTotalSupply())
	require.Equal(t, int32(6), asset.GetPrecision())

	require.Len(t, asset.GetOwnerAddress(), 21)
	require.Equal(t, bttAssetOwner, tronutils.EncodeCheck(asset.GetOwnerAddress()))
}

func TestGetAssetIssueById_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	asset, err := c.GetAssetIssueById(ctx, bttAssetID)
	require.NoError(t, err)
	assertBTT(t, asset)
}

func TestGetAssetIssueById_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	asset, err := c.GetAssetIssueById(ctx, bttAssetID)
	require.NoError(t, err)
	assertBTT(t, asset)
}

// A name lookup can return several assets - anyone may issue a TRC10 under a
// name already taken - so the assertion is that the one being searched for is
// among them, not how many there are.
func TestGetAssetIssueListByName_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assets, err := c.GetAssetIssueListByName(ctx, bttAssetName)
	require.NoError(t, err)

	assertListContainsBTT(t, assets.GetAssetIssue())
}

func TestGetAssetIssueListByName_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assets, err := c.GetAssetIssueListByName(ctx, bttAssetName)
	require.NoError(t, err)

	assertListContainsBTT(t, assets.GetAssetIssue())
}

func assertListContainsBTT(t *testing.T, assets []*core.AssetIssueContract) {
	t.Helper()

	require.NotEmpty(t, assets, "no asset issued under %q", bttAssetName)

	var found *core.AssetIssueContract
	for _, asset := range assets {
		// Every match must carry the searched name, or the request was not the
		// one that was meant - a mangled name would also fail here.
		require.Equal(t, bttAssetName, string(asset.GetName()))

		if asset.GetId() == bttAssetID {
			found = asset
		}
	}

	require.NotNil(t, found, "asset %s missing from the results for %q", bttAssetID, bttAssetName)
	assertBTT(t, found)
}

// A name nobody issued under is an empty answer, not an error - and not the
// whole list either.
func TestGetAssetIssueListByNameUnknown_GRPC(t *testing.T) {
	c := newGRPCClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assets, err := c.GetAssetIssueListByName(ctx, unknownAssetName)
	require.NoError(t, err)
	require.Empty(t, assets.GetAssetIssue())
}

func TestGetAssetIssueListByNameUnknown_HTTP(t *testing.T) {
	c := newHTTPClient(t)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assets, err := c.GetAssetIssueListByName(ctx, unknownAssetName)
	require.NoError(t, err)
	require.Empty(t, assets.GetAssetIssue())
}

const unknownAssetName = "no such trc10 asset exists"
