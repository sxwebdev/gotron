package tests

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Comparison tests verify that gRPC and HTTP transports return identical data

func TestCompare_BlockHeight(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

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
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

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
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcBalance, err := grpcClient.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)

	httpBalance, err := httpClient.GetAccountBalance(ctx, testAddress)
	require.NoError(t, err)

	// Balances should be very close (may differ slightly due to timing)
	diff := grpcBalance.Sub(httpBalance).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromInt(1)), "Balance difference should be < 1 TRX")

	t.Logf("gRPC balance: %s TRX, HTTP balance: %s TRX", grpcBalance.String(), httpBalance.String())
}

func TestCompare_BlockByNum(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get block from both transports
	grpcBlock, err := grpcClient.GetBlockByHeight(ctx, testBlockNum)
	require.NoError(t, err)
	require.NotNil(t, grpcBlock)

	httpBlock, err := httpClient.GetBlockByHeight(ctx, testBlockNum)
	require.NoError(t, err)
	require.NotNil(t, httpBlock)

	grpcTxs := grpcBlock.GetTransactions()
	httpTxs := httpBlock.GetTransactions()

	t.Logf("gRPC block has %d transactions", len(grpcTxs))
	t.Logf("HTTP block has %d transactions", len(httpTxs))

	require.Equal(t, len(grpcTxs), len(httpTxs), "Transaction count mismatch")

	// Compare each transaction
	mismatchCount := 0
	for i := range grpcTxs {
		grpcTx := grpcTxs[i]
		httpTx := httpTxs[i]

		grpcContracts := grpcTx.GetTransaction().GetRawData().GetContract()
		httpContracts := httpTx.GetTransaction().GetRawData().GetContract()

		if len(grpcContracts) != len(httpContracts) {
			mismatchCount++
			if mismatchCount <= 3 { // Only log first 3 mismatches
				t.Logf("TX %d: gRPC contracts=%d, HTTP contracts=%d", i, len(grpcContracts), len(httpContracts))
				t.Logf("  gRPC txid: %x", grpcTx.GetTxid())
				t.Logf("  HTTP txid: %x", httpTx.GetTxid())

				if len(grpcContracts) > 0 {
					t.Logf("  gRPC contract[0] type: %v", grpcContracts[0].GetType())
				}
				if len(httpContracts) > 0 {
					t.Logf("  HTTP contract[0] type: %v", httpContracts[0].GetType())
				}
			}
		}
	}

	if mismatchCount > 0 {
		t.Errorf("Total %d transactions have contract count mismatch", mismatchCount)
	}
}

func TestCompare_BlockTransactionDetails(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get block from both transports
	grpcBlock, err := grpcClient.GetBlockByHeight(ctx, testBlockNum)
	require.NoError(t, err)

	httpBlock, err := httpClient.GetBlockByHeight(ctx, testBlockNum)
	require.NoError(t, err)

	grpcTxs := grpcBlock.GetTransactions()
	httpTxs := httpBlock.GetTransactions()

	require.Equal(t, len(grpcTxs), len(httpTxs), "Transaction count mismatch")

	// Compare transaction details
	for i := range grpcTxs {
		grpcTx := grpcTxs[i]
		httpTx := httpTxs[i]

		// Compare txid
		assert.Equal(t, grpcTx.GetTxid(), httpTx.GetTxid(), "TX %d: txid mismatch", i)

		// Compare contract types
		grpcContracts := grpcTx.GetTransaction().GetRawData().GetContract()
		httpContracts := httpTx.GetTransaction().GetRawData().GetContract()

		require.Equal(t, len(grpcContracts), len(httpContracts), "TX %d: contract count mismatch", i)

		for j := range grpcContracts {
			assert.Equal(t, grpcContracts[j].GetType(), httpContracts[j].GetType(),
				"TX %d, Contract %d: type mismatch", i, j)

			// Compare parameter type_url
			assert.Equal(t,
				grpcContracts[j].GetParameter().GetTypeUrl(),
				httpContracts[j].GetParameter().GetTypeUrl(),
				"TX %d, Contract %d: parameter type_url mismatch", i, j)
		}

		// Compare signatures count
		grpcSigs := grpcTx.GetTransaction().GetSignature()
		httpSigs := httpTx.GetTransaction().GetSignature()
		assert.Equal(t, len(grpcSigs), len(httpSigs), "TX %d: signature count mismatch", i)
	}

	t.Logf("All %d transactions match between gRPC and HTTP", len(grpcTxs))
}

func TestCompare_TransactionInfoByBlockNum(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get transaction info from both transports
	grpcTxInfo, err := grpcClient.GetTransactionInfoByBlockNum(ctx, testBlockNum)
	require.NoError(t, err)

	httpTxInfo, err := httpClient.GetTransactionInfoByBlockNum(ctx, testBlockNum)
	require.NoError(t, err)

	grpcInfos := grpcTxInfo.GetTransactionInfo()
	httpInfos := httpTxInfo.GetTransactionInfo()

	require.Equal(t, len(grpcInfos), len(httpInfos), "TransactionInfo count mismatch")

	// Compare transaction info details
	for i := range grpcInfos {
		grpcInfo := grpcInfos[i]
		httpInfo := httpInfos[i]

		// Compare basic fields
		assert.Equal(t, grpcInfo.GetBlockNumber(), httpInfo.GetBlockNumber(),
			"TX %d: blockNumber mismatch", i)
		assert.Equal(t, grpcInfo.GetBlockTimeStamp(), httpInfo.GetBlockTimeStamp(),
			"TX %d: blockTimeStamp mismatch", i)
		assert.Equal(t, grpcInfo.GetFee(), httpInfo.GetFee(),
			"TX %d: fee mismatch", i)

		// Compare transaction ID
		if !bytes.Equal(grpcInfo.GetId(), httpInfo.GetId()) {
			t.Errorf("TX %d: id mismatch: gRPC=%x, HTTP=%x", i, grpcInfo.GetId(), httpInfo.GetId())
		}

		// Compare contract result count
		assert.Equal(t, len(grpcInfo.GetContractResult()), len(httpInfo.GetContractResult()),
			"TX %d: contractResult count mismatch", i)

		// Compare logs count
		assert.Equal(t, len(grpcInfo.GetLog()), len(httpInfo.GetLog()),
			"TX %d: log count mismatch", i)
	}

	t.Logf("All %d transaction infos match between gRPC and HTTP", len(grpcInfos))
}

func TestCompare_Contract(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcContract, err := grpcClient.GetContract(ctx, usdtContract)
	require.NoError(t, err)

	httpContract, err := httpClient.GetContract(ctx, usdtContract)
	require.NoError(t, err)

	assert.Equal(t, grpcContract.GetName(), httpContract.GetName(), "Contract name mismatch")
	assert.Equal(t, grpcContract.GetConsumeUserResourcePercent(), httpContract.GetConsumeUserResourcePercent(),
		"ConsumeUserResourcePercent mismatch")
	assert.Equal(t, grpcContract.GetOriginEnergyLimit(), httpContract.GetOriginEnergyLimit(),
		"OriginEnergyLimit mismatch")

	t.Logf("Contract %s matches between gRPC and HTTP", grpcContract.GetName())
}

func TestCompare_ContractABI(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcABI, err := grpcClient.GetContractABI(ctx, usdtContract)
	require.NoError(t, err)

	httpABI, err := httpClient.GetContractABI(ctx, usdtContract)
	require.NoError(t, err)

	grpcEntries := grpcABI.GetEntrys()
	httpEntries := httpABI.GetEntrys()

	require.Equal(t, len(grpcEntries), len(httpEntries), "ABI entry count mismatch")

	for i := range grpcEntries {
		assert.Equal(t, grpcEntries[i].GetName(), httpEntries[i].GetName(),
			"ABI entry %d: name mismatch", i)
		assert.Equal(t, grpcEntries[i].GetType(), httpEntries[i].GetType(),
			"ABI entry %d: type mismatch", i)
	}

	t.Logf("Contract ABI with %d entries matches between gRPC and HTTP", len(grpcEntries))
}

func TestCompare_AssetIssue(t *testing.T) {
	grpcClient := newGRPCClient(t)
	defer grpcClient.Close()

	httpClient := newHTTPClient(t)
	defer httpClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assetID := "1002000" // BTT

	grpcAsset, err := grpcClient.GetAssetIssueById(ctx, assetID)
	require.NoError(t, err)

	httpAsset, err := httpClient.GetAssetIssueById(ctx, assetID)
	require.NoError(t, err)

	assert.Equal(t, grpcAsset.GetId(), httpAsset.GetId(), "Asset ID mismatch")
	assert.Equal(t, grpcAsset.GetTotalSupply(), httpAsset.GetTotalSupply(), "Asset total supply mismatch")
	assert.Equal(t, grpcAsset.GetPrecision(), httpAsset.GetPrecision(), "Asset precision mismatch")

	// Note: name and abbr fields may have encoding differences between gRPC and HTTP
	// gRPC returns raw bytes, HTTP may return them differently
	t.Logf("gRPC Asset: name=%s, abbr=%s", string(grpcAsset.GetName()), string(grpcAsset.GetAbbr()))
	t.Logf("HTTP Asset: name=%s, abbr=%s", string(httpAsset.GetName()), string(httpAsset.GetAbbr()))
	t.Logf("Asset %s: ID, TotalSupply, Precision match between gRPC and HTTP", grpcAsset.GetId())
}
