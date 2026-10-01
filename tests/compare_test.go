package tests

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// Comparison tests verify that gRPC and HTTP transports return identical data

func TestCompare_BlockHeight(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcHeight, err := grpcClient.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	httpHeight, err := httpClient.GetLastBlockHeight(ctx)
	require.NoError(t, err)

	// Allow for a small difference due to timing
	diff := int64(grpcHeight) - int64(httpHeight)
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, int64(5), "Block height difference should be small")

	t.Logf("gRPC height: %d, HTTP height: %d, diff: %d", grpcHeight, httpHeight, diff)
}

func TestCompare_ChainParams(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcParams, err := grpcClient.ChainParams(ctx)
	require.NoError(t, err)

	httpParams, err := httpClient.ChainParams(ctx)
	require.NoError(t, err)

	assert.Equal(t, grpcParams.EnergyFee, httpParams.EnergyFee)
	assert.Equal(t, grpcParams.TransactionFee, httpParams.TransactionFee)
	assert.Equal(t, grpcParams.CreateAccountFee, httpParams.CreateAccountFee)

	t.Logf("Chain params match: EnergyFee=%d, TransactionFee=%d", grpcParams.EnergyFee, grpcParams.TransactionFee)
}

func TestCompare_AccountBalance(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcBalance, err := grpcClient.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)

	httpBalance, err := httpClient.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)

	// Balances should be very close (may differ slightly due to timing)
	diff := grpcBalance - httpBalance
	if diff < 0 {
		diff = -diff
	}
	assert.Less(t, diff, client.MustFromTRX(decimal.NewFromInt(1)), "Balance difference should be < 1 TRX")

	t.Logf("gRPC balance: %s, HTTP balance: %s", grpcBalance, httpBalance)
}

// The comparisons below are of whole messages. Historical mainnet data does not
// change, so anything short of proto.Equal is a decoding difference - the
// earlier field-by-field versions compared counts and types only, and passed
// for as long as HTTP returned every transaction with an empty contract
// parameter, every receipt with a mis-decoded id, and every signature as noise.

func TestCompare_BlockByNum(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	num := recentBlockNum(t)
	grpcBlock, err := grpcClient.GetBlockByHeight(ctx, num)
	require.NoError(t, err)

	httpBlock, err := httpClient.GetBlockByHeight(ctx, num)
	require.NoError(t, err)

	require.NotEmpty(t, grpcBlock.GetTransactions())
	requireProtoEqual(t, grpcBlock, httpBlock)

	t.Logf("Block %d with %d transactions matches between gRPC and HTTP", num, len(grpcBlock.GetTransactions()))
}

func TestCompare_BlockByHash(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ext, err := grpcClient.GetBlockByHeight(ctx, recentBlockNum(t))
	require.NoError(t, err)

	grpcBlock, err := grpcClient.GetBlockByHash(ctx, ext.GetBlockid())
	require.NoError(t, err)

	httpBlock, err := httpClient.GetBlockByHash(ctx, ext.GetBlockid())
	require.NoError(t, err)

	require.NotEmpty(t, grpcBlock.GetTransactions())
	requireProtoEqual(t, grpcBlock, httpBlock)
}

func TestCompare_TransactionInfoByBlockNum(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	num := recentBlockNum(t)
	grpcTxInfo, err := grpcClient.GetTransactionInfoByBlockNum(ctx, num)
	require.NoError(t, err)

	httpTxInfo, err := httpClient.GetTransactionInfoByBlockNum(ctx, num)
	require.NoError(t, err)

	require.NotEmpty(t, grpcTxInfo.GetTransactionInfo())
	requireProtoEqual(t, grpcTxInfo, httpTxInfo)

	t.Logf("All %d transaction infos match between gRPC and HTTP", len(grpcTxInfo.GetTransactionInfo()))
}

// TestCompare_TransactionByHash reads a sample of the block's transactions
// and receipts one by one - the path a caller watching a payment takes.
func TestCompare_TransactionByHash(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	block, err := grpcClient.GetBlockByHeight(ctx, recentBlockNum(t))
	require.NoError(t, err)
	txs := block.GetTransactions()
	require.NotEmpty(t, txs)

	for i := 0; i < len(txs); i += max(1, len(txs)/10) {
		hash := hex.EncodeToString(txs[i].GetTxid())

		grpcTx, err := grpcClient.GetTransactionByHash(ctx, hash)
		require.NoError(t, err)
		httpTx, err := httpClient.GetTransactionByHash(ctx, hash)
		require.NoError(t, err)
		requireProtoEqual(t, grpcTx, httpTx, "transaction %s", hash)

		grpcInfo, err := grpcClient.GetTransactionInfoByHash(ctx, hash)
		require.NoError(t, err)
		httpInfo, err := httpClient.GetTransactionInfoByHash(ctx, hash)
		require.NoError(t, err)
		requireProtoEqual(t, grpcInfo, httpInfo, "receipt %s", hash)
	}
}

func TestCompare_Contract(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcContract, err := grpcClient.GetContract(ctx, usdtContract)
	require.NoError(t, err)

	httpContract, err := httpClient.GetContract(ctx, usdtContract)
	require.NoError(t, err)

	// Every field, the ABI included. origin_address decides who pays a call's
	// energy, and it really is an address, not merely the same on both sides.
	require.Len(t, grpcContract.GetOriginAddress(), 21)
	require.NotEmpty(t, grpcContract.GetAbi().GetEntrys())
	requireProtoEqual(t, grpcContract, httpContract)

	t.Logf("Contract %s matches between gRPC and HTTP", grpcContract.GetName())
}

func TestCompare_AssetIssue(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assetID := "1002000" // BTT

	grpcAsset, err := grpcClient.GetAssetIssueById(ctx, assetID)
	require.NoError(t, err)

	httpAsset, err := httpClient.GetAssetIssueById(ctx, assetID)
	require.NoError(t, err)

	// The whole record, not a chosen few fields. The byte fields are the ones
	// that used to differ: HTTP sends them hex and protojson read them as
	// base64, so the name came back as "\xe3n\xbd\xef…" and the 21-byte owner as
	// 31 bytes - with no error either side. Comparing only id, supply and
	// precision was what let that stand.
	assert.True(t, proto.Equal(grpcAsset, httpAsset), "gRPC: %v\nHTTP: %v", grpcAsset, httpAsset)
}

// A name lookup returns the same byte fields, one record deeper. It is also the
// only test that can catch the request encoding: HTTP has to hex-encode the
// name, and sent plain it is refused at HTTP 200 - which reaches a one-sided
// test as a believable "no such asset".
func TestCompare_AssetIssueListByName(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcAssets, err := grpcClient.GetAssetIssueListByName(ctx, bttAssetName)
	require.NoError(t, err)
	require.NotEmpty(t, grpcAssets.GetAssetIssue())

	httpAssets, err := httpClient.GetAssetIssueListByName(ctx, bttAssetName)
	require.NoError(t, err)

	// Matched by id rather than by position: the two clients talk to different
	// hosts, anyone may issue another TRC10 under this name between the calls,
	// and the store's iteration order is not part of the API. Every asset gRPC
	// reports must be there and identical - an extra one on the HTTP side is a
	// new issuance, not an encoding bug.
	byID := make(map[string]*core.AssetIssueContract, len(httpAssets.GetAssetIssue()))
	for _, asset := range httpAssets.GetAssetIssue() {
		byID[asset.GetId()] = asset
	}

	for _, want := range grpcAssets.GetAssetIssue() {
		got, ok := byID[want.GetId()]
		if !assert.True(t, ok, "asset %s missing over HTTP", want.GetId()) {
			continue
		}

		assert.True(t, proto.Equal(want, got), "asset %s\ngRPC: %v\nHTTP: %v", want.GetId(), want, got)
	}
}

// The two transports have to agree on the whole account, not on the handful of
// scalars a one-sided test happens to look at. account_resource, the multisig
// permissions and the TRC10 balance maps were parsed out of the HTTP response
// and then never copied into the result, so they came back empty from HTTP and
// populated from gRPC with no error either side.
func TestCompare_AccountDetails(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcAcc, err := grpcClient.GetAccount(ctx, testAddress)
	require.NoError(t, err)

	httpAcc, err := httpClient.GetAccount(ctx, testAddress)
	require.NoError(t, err)

	// The account must actually have the things being compared, or the test
	// passes by both sides returning nothing.
	require.NotNil(t, grpcAcc.GetAccountResource())
	require.NotNil(t, grpcAcc.GetOwnerPermission())
	require.NotEmpty(t, grpcAcc.GetActivePermission())
	require.NotEmpty(t, grpcAcc.GetAssetV2())

	assert.True(t, proto.Equal(grpcAcc.GetAccountResource(), httpAcc.GetAccountResource()),
		"account_resource\ngRPC: %v\nHTTP: %v", grpcAcc.GetAccountResource(), httpAcc.GetAccountResource())
	assert.True(t, proto.Equal(grpcAcc.GetOwnerPermission(), httpAcc.GetOwnerPermission()),
		"owner_permission\ngRPC: %v\nHTTP: %v", grpcAcc.GetOwnerPermission(), httpAcc.GetOwnerPermission())

	require.Len(t, httpAcc.GetActivePermission(), len(grpcAcc.GetActivePermission()))
	for i, want := range grpcAcc.GetActivePermission() {
		assert.True(t, proto.Equal(want, httpAcc.GetActivePermission()[i]),
			"active_permission[%d]\ngRPC: %v\nHTTP: %v", i, want, httpAcc.GetActivePermission()[i])
	}

	assert.Equal(t, grpcAcc.GetAssetV2(), httpAcc.GetAssetV2())
	assert.Equal(t, grpcAcc.GetFreeAssetNetUsageV2(), httpAcc.GetFreeAssetNetUsageV2())
	assert.Equal(t, grpcAcc.GetAssetOptimized(), httpAcc.GetAssetOptimized())
}

// The per-TRC10 free-bandwidth maps and the TRON POWER / storage counters were
// missing from the HTTP conversion the same way.
func TestCompare_AccountResourceMaps(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcRes, err := grpcClient.GetAccountResource(ctx, testAddress)
	require.NoError(t, err)

	httpRes, err := httpClient.GetAccountResource(ctx, testAddress)
	require.NoError(t, err)

	require.NotEmpty(t, grpcRes.GetAssetNetLimit(), "account holds no TRC10, nothing to compare")

	assert.Equal(t, grpcRes.GetAssetNetUsed(), httpRes.GetAssetNetUsed())
	assert.Equal(t, grpcRes.GetAssetNetLimit(), httpRes.GetAssetNetLimit())
	assert.Equal(t, grpcRes.GetTotalTronPowerWeight(), httpRes.GetTotalTronPowerWeight())
	assert.Equal(t, grpcRes.GetTronPowerLimit(), httpRes.GetTronPowerLimit())
	assert.Equal(t, grpcRes.GetStorageLimit(), httpRes.GetStorageLimit())
}

// TestCompare_StakeInfo guards the HTTP account mapping of frozenV2/unfrozenV2.
// A missing mapping makes HTTP report an all-zero stake while gRPC reports the
// real one, which a single-transport test would not notice.
func TestCompare_StakeInfo(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcInfo, err := grpcClient.GetStakeInfo(ctx, stakedAddress)
	require.NoError(t, err)

	httpInfo, err := httpClient.GetStakeInfo(ctx, stakedAddress)
	require.NoError(t, err)

	assert.Equal(t, grpcInfo.StakedBandwidth, httpInfo.StakedBandwidth, "Staked bandwidth mismatch")
	assert.Equal(t, grpcInfo.StakedEnergy, httpInfo.StakedEnergy, "Staked energy mismatch")
	assert.Equal(t, grpcInfo.TotalStaked, httpInfo.TotalStaked, "Total staked mismatch")
	assert.Equal(t, grpcInfo.UnstakingTotal, httpInfo.UnstakingTotal, "Unstaking total mismatch")
	assert.Equal(t, len(grpcInfo.PendingUnstakes), len(httpInfo.PendingUnstakes), "Pending unstake count mismatch")

	t.Logf("Stake info matches between gRPC and HTTP: total=%d SUN, unstaking=%d SUN",
		grpcInfo.TotalStaked, grpcInfo.UnstakingTotal)
}

// TestCompare_ListWitnesses compares address sets, not slices: the node returns
// witnesses in an unstable order between consecutive calls.
func TestCompare_ListWitnesses(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcList, err := grpcClient.ListWitnesses(ctx)
	require.NoError(t, err)

	httpList, err := httpClient.ListWitnesses(ctx)
	require.NoError(t, err)

	assert.Equal(t, witnessAddressSet(t, grpcList), witnessAddressSet(t, httpList),
		"witness address sets must match between gRPC and HTTP")

	t.Logf("Witness sets match between gRPC and HTTP: %d witnesses", len(grpcList.GetWitnesses()))
}

// TestCompare_Rewards guards the camelCase /wallet/getReward and
// /wallet/getBrokerage endpoints and their non-standard response field names.
// An HTTP-only test would read the silently-discarded value as 0 and pass.
func TestCompare_Rewards(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer func() { _ = grpcClient.Close() }()

	httpClient := newHTTPClient(t)
	defer func() { _ = httpClient.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcBrokerage, err := grpcClient.GetWitnessBrokerage(ctx, stakedAddress)
	require.NoError(t, err)

	httpBrokerage, err := httpClient.GetWitnessBrokerage(ctx, stakedAddress)
	require.NoError(t, err)

	assert.Equal(t, grpcBrokerage, httpBrokerage, "Brokerage mismatch")
	// A zero on both sides would not prove the HTTP field name is right.
	assert.Greater(t, grpcBrokerage, int64(0), "fixture account is expected to report a non-zero brokerage")

	grpcReward, err := grpcClient.GetUnclaimedReward(ctx, stakedAddress)
	require.NoError(t, err)

	httpReward, err := httpClient.GetUnclaimedReward(ctx, stakedAddress)
	require.NoError(t, err)

	assert.Equal(t, grpcReward, httpReward, "Unclaimed reward mismatch")

	t.Logf("Rewards match between gRPC and HTTP: brokerage=%d%%, reward=%d SUN", grpcBrokerage, grpcReward)
}
