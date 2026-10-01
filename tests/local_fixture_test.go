package tests

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// localAccount marshals with its key so that a cached fixture can sign again.
func (a localAccount) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"address": a.Address, "key": hex.EncodeToString(crypto.FromECDSA(a.Key))})
}

func (a *localAccount) UnmarshalJSON(b []byte) error {
	var raw map[string]string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	key, err := crypto.HexToECDSA(raw["key"])
	if err != nil {
		return err
	}
	a.Address, a.Key = raw["address"], key
	return nil
}

// loadLocalChain reads the fixture cached in GOTRON_LOCAL_FIXTURE_CACHE, if
// that is set and the chain still has the transactions it names - a restarted
// node starts from genesis again. Building the fixture takes about a minute,
// which the cache saves on every run after the first.
func loadLocalChain(t *testing.T) *localChain {
	t.Helper()
	path := os.Getenv("GOTRON_LOCAL_FIXTURE_CACHE")
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var chain localChain
	if json.Unmarshal(b, &chain) != nil {
		return nil
	}

	c := newLocalGRPCClient(t)
	ctx := localContext(t, 10*time.Second)
	for _, txid := range chain.txids() {
		if _, err := c.GetTransactionInfoByHash(ctx, txid); err != nil {
			return nil
		}
	}
	t.Logf("using the fixture cached in %s", path)
	return &chain
}

func saveLocalChain(t *testing.T, chain *localChain) {
	t.Helper()
	if path := os.Getenv("GOTRON_LOCAL_FIXTURE_CACHE"); path != "" {
		b, err := json.Marshal(chain)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, b, 0o600))
	}
}

// localChain is on-chain state shared by the parity tests: one transaction of
// every kind whose answer carries bytes, so that each read method has
// something non-trivial to decode. It is built once, through gRPC - the
// transport whose wire format is protobuf itself and needs no translation -
// and read back through both.
type localChain struct {
	// Staker is a fresh account that stakes, delegates, votes and issues a
	// TRC10; Receiver is the account it delegates to.
	Staker   localAccount
	Receiver localAccount

	// TRC20 is TetherToken deployed from the witness, with its full ABI.
	TRC20         string
	TRC20DeployTx string
	TRC20Transfer string // witness -> Staker, so the receipt has a log
	TRC20Revert   string // Staker sends more than it has: FAILED, with resMessage

	// Payer is a contract that refunds 1 SUN to its caller, deployed with a
	// call value; PayerCall is a call to it with an internal transaction.
	Payer         string
	PayerDeployTx string
	PayerCall     string

	ActivationTx string // witness -> Staker, a transfer that creates the account
	FreezeTx     string
	DelegateTx   string
	UnfreezeTx   string
	CancelTx     string // CancelAllUnfreezeV2, whose receipt carries a map
	VoteTx       string
	AssetIssueTx string

	AssetID   string
	AssetName string

	// Blocks holds the number of every block one of the above landed in.
	Blocks []int64
}

func (c *localChain) txids() map[string]string {
	return map[string]string{
		"trc20 deploy":   c.TRC20DeployTx,
		"trc20 transfer": c.TRC20Transfer,
		"trc20 revert":   c.TRC20Revert,
		"payer deploy":   c.PayerDeployTx,
		"payer call":     c.PayerCall,
		"activation":     c.ActivationTx,
		"freeze":         c.FreezeTx,
		"delegate":       c.DelegateTx,
		"unfreeze":       c.UnfreezeTx,
		"cancel unstake": c.CancelTx,
		"vote":           c.VoteTx,
		"asset issue":    c.AssetIssueTx,
	}
}

var (
	localChainOnce  sync.Once
	localChainState *localChain
)

// localFixture returns the shared chain state, building it on first use.
func localFixture(t *testing.T) *localChain {
	t.Helper()
	requireLocalNode(t)

	localChainOnce.Do(func() {
		if localChainState = loadLocalChain(t); localChainState == nil {
			localChainState = buildLocalChain(t)
			saveLocalChain(t, localChainState)
		}
	})
	if localChainState == nil {
		t.Fatal("local chain fixture failed to build in an earlier test")
	}
	return localChainState
}

func buildLocalChain(t *testing.T) *localChain {
	t.Helper()

	c := newLocalGRPCClient(t)
	witness := localWitness(t)
	chain := &localChain{Staker: newLocalKey(t), Receiver: newLocalKey(t)}
	blocks := map[int64]bool{}
	confirm := func(ext *api.TransactionExtention, signer localAccount) string {
		txid, info := sendAndConfirm(t, c, ext, signer)
		blocks[info.GetBlockNumber()] = true
		return txid
	}
	ctx := localContext(t, 5*time.Minute)

	// CancelAllUnfreezeV2 is off on a fresh network. The proposal takes a
	// minute or so to apply; the rest of the fixture is built meanwhile.
	cancelReady := enableChainParameter(t, localChainParamAllowCancelAllUnfreezeV2, "getAllowCancelAllUnfreezeV2")

	// A transfer that creates the account it pays.
	ext, err := c.CreateTransferTransaction(ctx, witness.Address, chain.Staker.Address, trx(200_000))
	require.NoError(t, err)
	chain.ActivationTx = confirm(ext, witness)
	fundAccount(t, c, chain.Receiver.Address, trx(10))

	// TetherToken with its whole ABI.
	ext, err = c.DeployContract(ctx, tetherTokenRequest(t, witness.Address))
	require.NoError(t, err)
	chain.TRC20, err = client.DeployedContractAddress(ext.GetTransaction())
	require.NoError(t, err)
	chain.TRC20DeployTx = confirm(ext, witness)

	oneUSDT, err := client.FromTokenUnits(big.NewInt(1_000_000))
	require.NoError(t, err)
	ext, err = c.TRC20Send(ctx, witness.Address, chain.Staker.Address, chain.TRC20, oneUSDT, trx(100))
	require.NoError(t, err)
	chain.TRC20Transfer = confirm(ext, witness)

	// A transfer the contract reverts on chain: mined, charged, FAILED.
	tooMuch, err := client.FromTokenUnits(big.NewInt(5_000_000))
	require.NoError(t, err)
	ext, err = c.TRC20Send(ctx, chain.Staker.Address, witness.Address, chain.TRC20, tooMuch, trx(100))
	require.NoError(t, err)
	chain.TRC20Revert = signAndBroadcast(t, c, ext, chain.Staker)
	revert := waitReceipt(t, c, chain.TRC20Revert)
	require.Equal(t, core.TransactionInfo_FAILED, revert.GetResult())
	require.NotEmpty(t, revert.GetResMessage())
	blocks[revert.GetBlockNumber()] = true

	// The refunding contract, funded at deployment through call_value, which
	// DeployContractRequest has no field for - so the transport is called
	// directly.
	g, _ := localTransports(t)
	owner, err := tronutils.DecodeCheck(witness.Address)
	require.NoError(t, err)
	bytecode, err := hex.DecodeString(localSelfPayingContract)
	require.NoError(t, err)
	ext, err = g.DeployContract(ctx, &core.CreateSmartContract{
		OwnerAddress: owner,
		NewContract: &core.SmartContract{
			OriginAddress:     owner,
			Name:              "Refunder",
			Bytecode:          bytecode,
			CallValue:         trx(1).Int64(),
			OriginEnergyLimit: 1_000_000,
		},
	})
	require.NoError(t, err)
	require.Zero(t, ext.GetResult().GetCode(), "%s", ext.GetResult().GetMessage())
	ext.Transaction.RawData.FeeLimit = trx(100).Int64()
	require.NoError(t, ext.UpdateHash())
	chain.Payer, err = client.DeployedContractAddress(ext.GetTransaction())
	require.NoError(t, err)
	chain.PayerDeployTx = confirm(ext, witness)

	ext, err = c.TriggerContract(ctx, chain.Staker.Address, chain.Payer, "refund()", "[]", trx(10), 0, "", 0)
	require.NoError(t, err)
	chain.PayerCall = confirm(ext, chain.Staker)

	// Stake 2.0: stake both resources, delegate both, vote, then leave one
	// unstake pending after cancelling another.
	ext, err = c.Stake(ctx, chain.Staker.Address, client.ResourceTypeEnergy, trx(50_000))
	require.NoError(t, err)
	chain.FreezeTx = confirm(ext, chain.Staker)
	ext, err = c.Stake(ctx, chain.Staker.Address, client.ResourceTypeBandwidth, trx(50_000))
	require.NoError(t, err)
	confirm(ext, chain.Staker)

	ext, err = c.DelegateResource(ctx, chain.Staker.Address, chain.Receiver.Address, client.ResourceTypeEnergy, trx(1_000), false, 0)
	require.NoError(t, err)
	chain.DelegateTx = confirm(ext, chain.Staker)
	ext, err = c.DelegateResource(ctx, chain.Staker.Address, chain.Receiver.Address, client.ResourceTypeBandwidth, trx(1_000), true, 100)
	require.NoError(t, err)
	confirm(ext, chain.Staker)

	ext, err = c.VoteWitnesses(ctx, chain.Staker.Address, []client.Vote{{WitnessAddress: witness.Address, Count: 10_000}})
	require.NoError(t, err)
	chain.VoteTx = confirm(ext, chain.Staker)

	ext, err = c.Unstake(ctx, chain.Staker.Address, client.ResourceTypeEnergy, trx(10))
	require.NoError(t, err)
	confirm(ext, chain.Staker)
	cancelReady()
	ext, err = c.CancelAllUnstakes(ctx, chain.Staker.Address)
	require.NoError(t, err)
	chain.CancelTx = confirm(ext, chain.Staker)
	ext, err = c.Unstake(ctx, chain.Staker.Address, client.ResourceTypeBandwidth, trx(20))
	require.NoError(t, err)
	chain.UnfreezeTx = confirm(ext, chain.Staker)

	// A TRC10, issued through the raw wallet client because issuing is not
	// part of the Transport interface.
	staker, err := tronutils.DecodeCheck(chain.Staker.Address)
	require.NoError(t, err)
	chain.AssetName = "LocalAsset" + time.Now().Format("150405")
	start := time.Now().Add(time.Minute).UnixMilli()
	ext, err = g.WalletClient().CreateAssetIssue2(ctx, &core.AssetIssueContract{
		OwnerAddress: staker,
		Name:         []byte(chain.AssetName),
		Abbr:         []byte("LA"),
		TotalSupply:  1_000_000_000,
		TrxNum:       1,
		Num:          1,
		Precision:    6,
		StartTime:    start,
		EndTime:      start + int64(24*time.Hour/time.Millisecond),
		Description:  []byte("gotron local parity asset"),
		Url:          []byte("https://example.com/asset"),
		FrozenSupply: []*core.AssetIssueContract_FrozenSupply{{FrozenAmount: 1000, FrozenDays: 1}},
	})
	require.NoError(t, err)
	require.Zero(t, ext.GetResult().GetCode(), "%s", ext.GetResult().GetMessage())
	chain.AssetIssueTx = confirm(ext, chain.Staker)

	acc, err := c.GetAccount(ctx, chain.Staker.Address)
	require.NoError(t, err)
	chain.AssetID = string(acc.GetAssetIssued_ID())
	require.NotEmpty(t, chain.AssetID)

	for b := range blocks {
		chain.Blocks = append(chain.Blocks, b)
	}
	return chain
}

// localChainParamAllowCancelAllUnfreezeV2 is the committee parameter id of
// ALLOW_CANCEL_ALL_UNFREEZE_V2.
const localChainParamAllowCancelAllUnfreezeV2 = 77

// enableChainParameter has the witness propose and approve setting committee
// parameter id to 1, unless the chain already has it. The returned function
// blocks until the parameter is in force: a proposal applies at the first
// maintenance after it expires.
func enableChainParameter(t *testing.T, id int64, key string) func() {
	t.Helper()

	c := newLocalGRPCClient(t)
	ctx := localContext(t, 5*time.Minute)
	enabled := func() bool {
		p, err := c.ChainParam(ctx, key)
		require.NoError(t, err)
		return p.GetValue() == 1
	}
	if enabled() {
		return func() {}
	}

	g, _ := localTransports(t)
	witness := localWitness(t)
	owner, err := tronutils.DecodeCheck(witness.Address)
	require.NoError(t, err)

	ext, err := g.WalletClient().ProposalCreate(ctx, &core.ProposalCreateContract{
		OwnerAddress: owner,
		Parameters:   map[int64]int64{id: 1},
	})
	require.NoError(t, err)
	require.Zero(t, ext.GetResult().GetCode(), "%s", ext.GetResult().GetMessage())
	sendAndConfirm(t, c, ext, witness)

	proposals, err := g.WalletClient().ListProposals(ctx, &api.EmptyMessage{})
	require.NoError(t, err)
	var proposalID int64
	for _, p := range proposals.GetProposals() {
		if p.GetParameters()[id] == 1 && p.GetState() == core.Proposal_PENDING {
			proposalID = max(proposalID, p.GetProposalId())
		}
	}
	require.NotZero(t, proposalID, "proposal for parameter %d not found", id)

	ext, err = g.WalletClient().ProposalApprove(ctx, &core.ProposalApproveContract{
		OwnerAddress:  owner,
		ProposalId:    proposalID,
		IsAddApproval: true,
	})
	require.NoError(t, err)
	require.Zero(t, ext.GetResult().GetCode(), "%s", ext.GetResult().GetMessage())
	sendAndConfirm(t, c, ext, witness)

	return func() {
		t.Helper()
		for !enabled() {
			select {
			case <-ctx.Done():
				t.Fatalf("committee parameter %d (%s) not applied in time", id, key)
			case <-time.After(2 * time.Second):
			}
		}
	}
}
