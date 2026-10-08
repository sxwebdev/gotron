package tests

// End-to-end write paths through the Client, once per transport: every
// transaction is built, signed, broadcast and read back over the transport
// under test alone, so a transport that cannot complete the round trip fails
// its own variant.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// --- TRX transfers ---------------------------------------------------------

func TestLocalTransferTRX_GRPC(t *testing.T) {
	requireLocalNode(t)
	testLocalTransferTRX(t, newLocalGRPCClient(t))
}

func TestLocalTransferTRX_HTTP(t *testing.T) {
	requireLocalNode(t)
	testLocalTransferTRX(t, newLocalHTTPClient(t))
}

func testLocalTransferTRX(t *testing.T, c *client.Client) {
	ctx := localContext(t, 2*time.Minute)
	witness := localWitness(t)
	fresh := newLocalKey(t)

	params, err := c.ChainParams(ctx)
	require.NoError(t, err)

	// A transfer that creates the account it pays.
	estimate, err := c.EstimateTRXTransfer(ctx, witness.Address, fresh.Address, trx(100))
	require.NoError(t, err)
	ext, err := c.CreateTransferTransaction(ctx, witness.Address, fresh.Address, trx(100))
	require.NoError(t, err)
	txid, info := sendAndConfirm(t, c, ext, witness)

	require.Equal(t, mustHex(t, txid), info.GetId())
	require.Positive(t, info.GetBlockNumber())
	// The witness has no staked bandwidth, so it pays both creation fees -
	// this network's, read from the node rather than mainnet's constants.
	assert.Equal(t, params.CreateAccountFee+params.CreateNewAccountFeeInSystemContract, info.GetFee())
	assert.Equal(t, estimate.Fee.Int64(), info.GetFee(), "estimate: %+v", estimate)

	balance, err := c.GetAccountBalance(ctx, fresh.Address)
	require.NoError(t, err)
	require.Equal(t, trx(100), balance)

	tx, err := c.GetTransactionByHash(ctx, txid)
	require.NoError(t, err)
	requireTransferContract(t, tx, witness.Address, fresh.Address, trx(100))
	require.Len(t, tx.GetSignature(), 1)

	ext2, info2, err := c.GetTransactionExtensionByHash(ctx, txid)
	require.NoError(t, err)
	require.Equal(t, mustHex(t, txid), ext2.GetTxid())
	requireProtoEqual(t, info, info2)

	// A transfer to an account that exists now, from the new account itself.
	ext, err = c.CreateTransferTransaction(ctx, fresh.Address, witness.Address, trx(1))
	require.NoError(t, err)
	txid, info = sendAndConfirm(t, c, ext, fresh)
	require.Zero(t, info.GetFee(), "the free bandwidth pays for a plain transfer")

	balance, err = c.GetAccountBalance(ctx, fresh.Address)
	require.NoError(t, err)
	require.Equal(t, trx(99), balance)

	tx, err = c.GetTransactionByHash(ctx, txid)
	require.NoError(t, err)
	requireTransferContract(t, tx, fresh.Address, witness.Address, trx(1))
}

func requireTransferContract(t *testing.T, tx *core.Transaction, from, to string, amount client.SUN) {
	t.Helper()
	contracts := tx.GetRawData().GetContract()
	require.Len(t, contracts, 1)
	transfer := &core.TransferContract{}
	require.NoError(t, contracts[0].GetParameter().UnmarshalTo(transfer))
	require.Equal(t, from, tronutils.EncodeCheck(transfer.GetOwnerAddress()))
	require.Equal(t, to, tronutils.EncodeCheck(transfer.GetToAddress()))
	require.Equal(t, amount.Int64(), transfer.GetAmount())
}

// --- Deployment, contract calls and TRC20 ----------------------------------

func TestLocalDeployContract_GRPC(t *testing.T) {
	requireLocalNode(t)
	testLocalDeployContract(t, newLocalGRPCClient(t))
}

func TestLocalDeployContract_HTTP(t *testing.T) {
	requireLocalNode(t)
	testLocalDeployContract(t, newLocalHTTPClient(t))
}

func testLocalDeployContract(t *testing.T, c *client.Client) {
	ctx := localContext(t, 2*time.Minute)
	witness := localWitness(t)
	req := tetherTokenRequest(t, witness.Address)

	ext, err := c.DeployContract(ctx, req)
	require.NoError(t, err)

	// The fee limit is not part of the request the node builds from; the
	// client sets it afterwards and must re-hash, or the signature covers a
	// txid nobody will find.
	require.Equal(t, req.FeeLimit.Int64(), ext.GetTransaction().GetRawData().GetFeeLimit())
	raw, err := proto.Marshal(ext.GetTransaction().GetRawData())
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	require.Equal(t, sum[:], ext.GetTxid(), "txid not refreshed after setting the fee limit")

	address, err := client.DeployedContractAddress(ext.GetTransaction())
	require.NoError(t, err)

	txid, info := sendAndConfirm(t, c, ext, witness)
	require.Equal(t, address, tronutils.EncodeCheck(info.GetContractAddress()), "derived address differs from the receipt")
	require.Equal(t, mustHex(t, txid), info.GetId())

	// The ABI on chain, entry by entry, read back over both transports.
	g, h := localTransports(t)
	for name, tr := range map[string]client.Transport{"grpc": g, "http": h} {
		sc, err := tr.GetContract(ctx, mustDecode(t, address))
		require.NoError(t, err, name)
		require.Len(t, sc.GetAbi().GetEntrys(), len(req.ABI.GetEntrys()), name)
		for i, entry := range req.ABI.GetEntrys() {
			requireProtoEqual(t, entry, sc.GetAbi().GetEntrys()[i], "%s: entry %d (%s)", name, i, entry.GetName())
		}
		require.Equal(t, "TetherToken", sc.GetName(), name)
	}

	// The token works through the client under test.
	name, err := c.TRC20GetName(ctx, address)
	require.NoError(t, err)
	require.Equal(t, "Tether USD", name)
	decimals, err := c.TRC20GetDecimals(ctx, address)
	require.NoError(t, err)
	require.Equal(t, int64(6), decimals.Int64())

	supply, err := c.TRC20ContractBalance(ctx, witness.Address, address)
	require.NoError(t, err)
	require.Equal(t, "10000000", supply.String())

	receiver := newLocalKey(t)
	amount, err := client.FromTokenUnits(big.NewInt(1_234_567))
	require.NoError(t, err)

	estimate, err := c.EstimateTRC20Transfer(ctx, witness.Address, receiver.Address, address, amount)
	require.NoError(t, err)
	require.Positive(t, estimate.Usage.Energy.IntPart())

	ext, err = c.TRC20Send(ctx, witness.Address, receiver.Address, address, amount, trx(100))
	require.NoError(t, err)
	_, info = sendAndConfirm(t, c, ext, witness)
	require.Len(t, info.GetLog(), 1, "Transfer event")
	require.Equal(t, mustDecode(t, address)[1:], info.GetLog()[0].GetAddress())
	require.Equal(t, core.Transaction_Result_SUCCESS, info.GetReceipt().GetResult())

	got, err := c.TRC20ContractBalance(ctx, receiver.Address, address)
	require.NoError(t, err)
	require.Equal(t, amount.String(), got.String())

	// A call the contract reverts is reported, with the partial result.
	tooMuch, err := client.FromTokenUnits(big.NewInt(1_000_000_000))
	require.NoError(t, err)
	tx, err := c.TRC20Call(ctx, receiver.Address, address, trc20TransferData(t, witness.Address, tooMuch), true, 0)
	require.ErrorIs(t, err, client.ErrContractCallFailed)
	require.NotNil(t, tx)

	// Changing the deployed contract's settings, as its owner.
	ext, err = c.UpdateSettingContract(ctx, witness.Address, address, 40)
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, witness)
	ext, err = c.UpdateEnergyLimitContract(ctx, witness.Address, address, 5_000_000)
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, witness)

	sc, err := c.GetContract(ctx, address)
	require.NoError(t, err)
	require.Equal(t, int64(40), sc.GetConsumeUserResourcePercent())
	require.Equal(t, int64(5_000_000), sc.GetOriginEnergyLimit())
}

func trc20TransferData(t *testing.T, to string, amount client.TokenAmount) string {
	t.Helper()
	data, err := abi.Pack("transfer(address,uint256)", []abi.Param{{"address": to}, {"uint256": amount.String()}})
	require.NoError(t, err)
	return hex.EncodeToString(data)
}

// TestLocalInternalTransaction_* call the refunding contract, whose receipt
// carries an internal transaction and whose deployment carried a call value.
func TestLocalInternalTransaction_GRPC(t *testing.T) {
	testLocalInternalTransaction(t, client.ProtocolGRPC)
}

func TestLocalInternalTransaction_HTTP(t *testing.T) {
	testLocalInternalTransaction(t, client.ProtocolHTTP)
}

func testLocalInternalTransaction(t *testing.T, protocol client.Protocol) {
	chain := localFixture(t)
	c := newLocalClient(t, protocol)
	ctx := localContext(t, time.Minute)

	ext, err := c.TriggerContract(ctx, chain.Staker.Address, chain.Payer, "refund()", "[]", trx(10), 0, "", 0)
	require.NoError(t, err)
	_, info := sendAndConfirm(t, c, ext, chain.Staker)

	require.Len(t, info.GetInternalTransactions(), 1)
	internal := info.GetInternalTransactions()[0]
	require.Equal(t, chain.Payer, tronutils.EncodeCheck(internal.GetCallerAddress()))
	require.Equal(t, chain.Staker.Address, tronutils.EncodeCheck(internal.GetTransferToAddress()))
	require.Equal(t, int64(1), internal.GetCallValueInfo()[0].GetCallValue())
	require.Equal(t, "call", string(internal.GetNote()))
}

// --- Stake 2.0 and delegation ----------------------------------------------

func TestLocalStakeAndDelegate_GRPC(t *testing.T) {
	requireLocalNode(t)
	testLocalStakeAndDelegate(t, newLocalGRPCClient(t))
}

func TestLocalStakeAndDelegate_HTTP(t *testing.T) {
	requireLocalNode(t)
	testLocalStakeAndDelegate(t, newLocalHTTPClient(t))
}

func testLocalStakeAndDelegate(t *testing.T, c *client.Client) {
	ctx := localContext(t, 3*time.Minute)
	owner, receiver := newLocalKey(t), newLocalKey(t)
	fundAccount(t, c, owner.Address, trx(10_000))
	fundAccount(t, c, receiver.Address, trx(1))

	ext, err := c.Stake(ctx, owner.Address, client.ResourceTypeEnergy, trx(5_000))
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, owner)
	ext, err = c.Stake(ctx, owner.Address, client.ResourceTypeBandwidth, trx(1_000))
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, owner)

	info, err := c.GetStakeInfo(ctx, owner.Address)
	require.NoError(t, err)
	require.Equal(t, trx(5_000), info.StakedEnergy)
	require.Equal(t, trx(1_000), info.StakedBandwidth)

	maxEnergy, err := c.GetCanDelegatedMaxSize(ctx, owner.Address, client.ResourceTypeEnergy)
	require.NoError(t, err)
	require.Equal(t, trx(5_000), maxEnergy)

	// More than the stake: the node's verdict, through this transport, is the
	// sentinel for either resource.
	_, err = c.DelegateResource(ctx, owner.Address, receiver.Address, client.ResourceTypeEnergy, trx(5_001), false, 0)
	require.ErrorIs(t, err, client.ErrDelegateStakeShort)
	_, err = c.DelegateResource(ctx, owner.Address, receiver.Address, client.ResourceTypeBandwidth, trx(1_001), false, 0)
	require.ErrorIs(t, err, client.ErrDelegateStakeShort)

	ext, err = c.DelegateResource(ctx, owner.Address, receiver.Address, client.ResourceTypeEnergy, trx(2_000), false, 0)
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, owner)

	lent, err := c.GetDelegatedResourcesV2(ctx, owner.Address)
	require.NoError(t, err)
	require.Len(t, lent, 1)
	require.Equal(t, receiver.Address, lent[0].To)
	require.Equal(t, trx(2_000), lent[0].Energy)

	res, err := c.GetAccountResource(ctx, receiver.Address)
	require.NoError(t, err)
	require.Positive(t, res.GetEnergyLimit(), "the receiver has the delegated energy")

	ext, err = c.ReclaimResource(ctx, owner.Address, receiver.Address, client.ResourceTypeEnergy, trx(2_000))
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, owner)

	lent, err = c.GetDelegatedResourcesV2(ctx, owner.Address)
	require.NoError(t, err)
	require.Empty(t, lent)

	ext, err = c.Unstake(ctx, owner.Address, client.ResourceTypeEnergy, trx(100))
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, owner)

	count, err := c.GetAvailableUnstakeCount(ctx, owner.Address)
	require.NoError(t, err)
	require.Equal(t, int64(31), count)

	acc, err := c.GetAccount(ctx, owner.Address)
	require.NoError(t, err)
	require.Len(t, acc.GetUnfrozenV2(), 1)
	require.Equal(t, trx(100).Int64(), acc.GetUnfrozenV2()[0].GetUnfreezeAmount())

	// The unstake delay is days long, so there is nothing to withdraw yet.
	withdrawable, err := c.GetWithdrawableUnstaked(ctx, owner.Address)
	require.NoError(t, err)
	require.Zero(t, withdrawable)
	_, err = c.WithdrawUnstaked(ctx, owner.Address)
	var cve *client.ContractValidateError
	require.ErrorAs(t, err, &cve)

	if allowCancel, err := c.ChainParam(ctx, "getAllowCancelAllUnfreezeV2"); err == nil && allowCancel.GetValue() == 1 {
		ext, err = c.CancelAllUnstakes(ctx, owner.Address)
		require.NoError(t, err)
		_, receipt := sendAndConfirm(t, c, ext, owner)
		// Every resource is listed, the ones with nothing cancelled at zero.
		require.Equal(t, map[string]int64{"BANDWIDTH": 0, "ENERGY": trx(100).Int64(), "TRON_POWER": 0}, receipt.GetCancelUnfreezeV2Amount())
	}

	ext, err = c.VoteWitnesses(ctx, owner.Address, []client.Vote{{WitnessAddress: localWitnessAddress, Count: 10}})
	require.NoError(t, err)
	sendAndConfirm(t, c, ext, owner)

	acc, err = c.GetAccount(ctx, owner.Address)
	require.NoError(t, err)
	require.Len(t, acc.GetVotes(), 1)
	require.Equal(t, int64(10), acc.GetVotes()[0].GetVoteCount())
}

// --- Refused broadcasts ----------------------------------------------------

// refusedBroadcasts returns signed transactions the node must reject, one per
// rejection the tests know how to provoke, each with the code expected.
func refusedBroadcasts(t *testing.T, c *client.Client) map[string]struct {
	tx   *core.Transaction
	code api.ReturnResponseCode
} {
	t.Helper()
	ctx := localContext(t, 30*time.Second)
	witness := localWitness(t)

	build := func() *api.TransactionExtention {
		ext, err := c.CreateTransferTransaction(ctx, witness.Address, newLocalKey(t).Address, 1)
		require.NoError(t, err)
		return ext
	}

	unsigned := build()

	wrongKey := build()
	require.NoError(t, c.SignTransaction(wrongKey.GetTransaction(), newLocalKey(t).Key))

	expired := build()
	expired.Transaction.RawData.Expiration = time.Now().Add(-time.Hour).UnixMilli()
	require.NoError(t, expired.UpdateHash())
	require.NoError(t, c.SignTransaction(expired.GetTransaction(), witness.Key))

	badRef := build()
	badRef.Transaction.RawData.RefBlockHash = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	require.NoError(t, badRef.UpdateHash())
	require.NoError(t, c.SignTransaction(badRef.GetTransaction(), witness.Key))

	duplicate := build()
	signAndBroadcast(t, c, duplicate, witness)

	return map[string]struct {
		tx   *core.Transaction
		code api.ReturnResponseCode
	}{
		"unsigned":       {unsigned.GetTransaction(), api.Return_SIGERROR},
		"wrong key":      {wrongKey.GetTransaction(), api.Return_SIGERROR},
		"expired":        {expired.GetTransaction(), api.Return_TRANSACTION_EXPIRATION_ERROR},
		"bad ref block":  {badRef.GetTransaction(), api.Return_TAPOS_ERROR},
		"already posted": {duplicate.GetTransaction(), api.Return_DUP_TRANSACTION_ERROR},
	}
}

func TestLocalBroadcastRefused_GRPC(t *testing.T) {
	requireLocalNode(t)
	testLocalBroadcastRefused(t, newLocalGRPCClient(t))
}

func TestLocalBroadcastRefused_HTTP(t *testing.T) {
	requireLocalNode(t)
	testLocalBroadcastRefused(t, newLocalHTTPClient(t))
}

func testLocalBroadcastRefused(t *testing.T, c *client.Client) {
	for name, tc := range refusedBroadcasts(t, c) {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			ret, err := c.BroadcastTransaction(ctx, tc.tx)
			var be *client.BroadcastError
			require.True(t, errors.As(err, &be), "want a BroadcastError, got %v", err)
			require.Equal(t, tc.code, be.Code, "message: %s", be.Message)
			require.NotEmpty(t, be.Message)
			require.False(t, ret.GetResult())
		})
	}
}

// TestLocalParity_BroadcastTransaction sends each refused transaction over
// both transports and compares the answers in full, and then one accepted
// transaction over each.
func TestLocalParity_BroadcastTransaction(t *testing.T) {
	requireLocalNode(t)
	c := newLocalGRPCClient(t)
	t.Run("accepted", func(t *testing.T) {
		ctx := localContext(t, 10*time.Second)
		witness := localWitness(t)
		parity(t, func(tr client.Transport) (*api.Return, error) {
			ext, err := c.CreateTransferTransaction(ctx, witness.Address, newLocalKey(t).Address, 1)
			require.NoError(t, err)
			require.NoError(t, c.SignTransaction(ext.GetTransaction(), witness.Key))
			return tr.BroadcastTransaction(ctx, ext.GetTransaction())
		})
	})
	for name, tc := range refusedBroadcasts(t, c) {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*api.Return, error) {
				return tr.BroadcastTransaction(ctx, tc.tx)
			})
		})
	}
}
