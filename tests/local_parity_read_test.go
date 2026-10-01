package tests

// Parity of every read method of the Transport interface: the same request
// goes to the same node over gRPC and over HTTP and the two answers must be
// the same message. gRPC is the reference - its wire format is protobuf
// itself, so its answer needs no translation; any difference is an HTTP
// decoding bug unless a comment at the comparison says otherwise.

import (
	"fmt"
	"slices"
	"strings"
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

func mustDecode(t *testing.T, address string) []byte {
	t.Helper()
	b, err := tronutils.DecodeCheck(address)
	require.NoError(t, err)
	return b
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := tronutils.FromHex(s)
	require.NoError(t, err)
	return b
}

// localAddresses names one account of every kind the fixture created, plus
// one that does not exist.
func localAddresses(t *testing.T, chain *localChain) map[string][]byte {
	return map[string][]byte{
		"witness":  mustDecode(t, localWitnessAddress),
		"staker":   mustDecode(t, chain.Staker.Address),
		"receiver": mustDecode(t, chain.Receiver.Address),
		"contract": mustDecode(t, chain.TRC20),
		"unknown":  mustDecode(t, newLocalKey(t).Address),
	}
}

// parity calls both transports and compares their answers. Both must succeed
// or both must fail.
func parity[T proto.Message](t *testing.T, call func(client.Transport) (T, error)) {
	t.Helper()
	g, h := localTransports(t)

	want, gErr := call(g)
	got, hErr := call(h)
	if gErr != nil || hErr != nil {
		require.Equal(t, gErr != nil, hErr != nil, "gRPC error: %v\nHTTP error: %v", gErr, hErr)
		return
	}
	requireProtoEqual(t, want, got)
}

func TestLocalParity_GetAccount(t *testing.T) {
	chain := localFixture(t)
	for name, addr := range localAddresses(t, chain) {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*core.Account, error) {
				return tr.GetAccount(ctx, &core.Account{Address: addr})
			})
		})
	}
}

func TestLocalParity_GetAccountResource(t *testing.T) {
	chain := localFixture(t)
	for name, addr := range localAddresses(t, chain) {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*api.AccountResourceMessage, error) {
				return tr.GetAccountResource(ctx, &core.Account{Address: addr})
			})
			parity(t, func(tr client.Transport) (*api.AccountResourceMessage, error) {
				return tr.GetAccountResourceMessage(ctx, &core.Account{Address: addr})
			})
		})
	}
}

func TestLocalParity_GetBlockByNum(t *testing.T) {
	chain := localFixture(t)
	for _, num := range append(slices.Clone(chain.Blocks), 0, 1<<40) { // genesis, and a height not yet reached
		t.Run(fmt.Sprint(num), func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*api.BlockExtention, error) {
				return tr.GetBlockByNum(ctx, num)
			})
		})
	}
}

func TestLocalParity_GetBlockById(t *testing.T) {
	chain := localFixture(t)
	g, _ := localTransports(t)
	for _, num := range chain.Blocks {
		t.Run(fmt.Sprint(num), func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			block, err := g.GetBlockByNum(ctx, num)
			require.NoError(t, err)
			parity(t, func(tr client.Transport) (*core.Block, error) {
				return tr.GetBlockById(ctx, block.GetBlockid())
			})
		})
	}
	t.Run("unknown", func(t *testing.T) {
		ctx := localContext(t, 10*time.Second)
		parity(t, func(tr client.Transport) (*core.Block, error) {
			return tr.GetBlockById(ctx, make([]byte, 32))
		})
	})
}

// TestLocalParity_GetNowBlock reads the head over HTTP and the same height over
// gRPC: two calls for the head itself would race the next block.
func TestLocalParity_GetNowBlock(t *testing.T) {
	localFixture(t)
	g, h := localTransports(t)
	ctx := localContext(t, 10*time.Second)

	head, err := h.GetNowBlock(ctx)
	require.NoError(t, err)
	want, err := g.GetBlockByNum(ctx, head.GetBlockHeader().GetRawData().GetNumber())
	require.NoError(t, err)
	requireProtoEqual(t, want, head)

	head, err = g.GetNowBlock(ctx)
	require.NoError(t, err)
	got, err := h.GetBlockByNum(ctx, head.GetBlockHeader().GetRawData().GetNumber())
	require.NoError(t, err)
	requireProtoEqual(t, head, got)
}

func TestLocalParity_GetBlockByLimitNext(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	first, last := slices.Min(chain.Blocks), slices.Max(chain.Blocks)
	parity(t, func(tr client.Transport) (*api.BlockListExtention, error) {
		return tr.GetBlockByLimitNext(ctx, first, last+1)
	})
	t.Run("beyond the head", func(t *testing.T) {
		parity(t, func(tr client.Transport) (*api.BlockListExtention, error) {
			return tr.GetBlockByLimitNext(ctx, 1<<40, 1<<40+3)
		})
	})
}

// TestLocalParity_GetBlockByLatestNum retries until both calls saw the same
// head, which a block landing between them would otherwise break.
func TestLocalParity_GetBlockByLatestNum(t *testing.T) {
	localFixture(t)
	g, h := localTransports(t)
	ctx := localContext(t, 30*time.Second)

	for {
		want, err := g.GetBlockByLatestNum(ctx, 5)
		require.NoError(t, err)
		got, err := h.GetBlockByLatestNum(ctx, 5)
		require.NoError(t, err)
		if blockNumbers(want) == blockNumbers(got) {
			requireProtoEqual(t, want, got)
			return
		}
		require.NoError(t, ctx.Err(), "the head kept moving between the two calls")
	}
}

func blockNumbers(list *api.BlockListExtention) string {
	var nums []int64
	for _, b := range list.GetBlock() {
		nums = append(nums, b.GetBlockHeader().GetRawData().GetNumber())
	}
	return fmt.Sprint(nums)
}

func TestLocalParity_GetTransactionInfoByBlockNum(t *testing.T) {
	chain := localFixture(t)
	for _, num := range append(slices.Clone(chain.Blocks), 1<<40) {
		t.Run(fmt.Sprint(num), func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*api.TransactionInfoList, error) {
				return tr.GetTransactionInfoByBlockNum(ctx, num)
			})
		})
	}
}

func TestLocalParity_GetTransactionById(t *testing.T) {
	chain := localFixture(t)
	txids := chain.txids()
	txids["unknown"] = "00000000000000000000000000000000000000000000000000000000000000ff"
	for name, txid := range txids {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*core.Transaction, error) {
				return tr.GetTransactionById(ctx, mustHex(t, txid))
			})
		})
	}
}

func TestLocalParity_GetTransactionInfoById(t *testing.T) {
	chain := localFixture(t)
	txids := chain.txids()
	txids["unknown"] = "00000000000000000000000000000000000000000000000000000000000000ff"
	for name, txid := range txids {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*core.TransactionInfo, error) {
				return tr.GetTransactionInfoById(ctx, mustHex(t, txid))
			})
		})
	}
}

func TestLocalParity_GetContract(t *testing.T) {
	chain := localFixture(t)
	for name, addr := range map[string]string{
		"trc20":   chain.TRC20,
		"payer":   chain.Payer,
		"account": chain.Staker.Address,
		"unknown": newLocalKey(t).Address,
	} {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*core.SmartContract, error) {
				return tr.GetContract(ctx, mustDecode(t, addr))
			})
		})
	}
}

func TestLocalParity_Delegation(t *testing.T) {
	chain := localFixture(t)
	staker, receiver := mustDecode(t, chain.Staker.Address), mustDecode(t, chain.Receiver.Address)
	ctx := localContext(t, 30*time.Second)

	for name, msg := range map[string]*api.DelegatedResourceMessage{
		"staker to receiver": {FromAddress: staker, ToAddress: receiver},
		"receiver to staker": {FromAddress: receiver, ToAddress: staker},
	} {
		t.Run("GetDelegatedResource/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*api.DelegatedResourceList, error) {
				return tr.GetDelegatedResource(ctx, msg)
			})
		})
		t.Run("GetDelegatedResourceV2/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*api.DelegatedResourceList, error) {
				return tr.GetDelegatedResourceV2(ctx, msg)
			})
		})
	}

	for name, addr := range localAddresses(t, chain) {
		t.Run("GetDelegatedResourceAccountIndex/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*core.DelegatedResourceAccountIndex, error) {
				return tr.GetDelegatedResourceAccountIndex(ctx, addr)
			})
		})
		t.Run("GetDelegatedResourceAccountIndexV2/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*core.DelegatedResourceAccountIndex, error) {
				return tr.GetDelegatedResourceAccountIndexV2(ctx, addr)
			})
		})
		for _, resource := range []core.ResourceCode{core.ResourceCode_BANDWIDTH, core.ResourceCode_ENERGY} {
			t.Run(fmt.Sprintf("GetCanDelegatedMaxSize/%s/%s", name, resource), func(t *testing.T) {
				parity(t, func(tr client.Transport) (*api.CanDelegatedMaxSizeResponseMessage, error) {
					return tr.GetCanDelegatedMaxSize(ctx, &api.CanDelegatedMaxSizeRequestMessage{
						OwnerAddress: addr,
						Type:         int32(resource),
					})
				})
			})
		}
	}
}

func TestLocalParity_Unstaking(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 30*time.Second)
	farFuture := time.Now().Add(30 * 24 * time.Hour).UnixMilli()

	for name, addr := range localAddresses(t, chain) {
		t.Run("GetAvailableUnfreezeCount/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
				return tr.GetAvailableUnfreezeCount(ctx, &api.GetAvailableUnfreezeCountRequestMessage{OwnerAddress: addr})
			})
		})
		for when, ts := range map[string]int64{"now": time.Now().UnixMilli(), "after delay": farFuture} {
			t.Run("GetCanWithdrawUnfreezeAmount/"+name+"/"+when, func(t *testing.T) {
				parity(t, func(tr client.Transport) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
					return tr.GetCanWithdrawUnfreezeAmount(ctx, &api.CanWithdrawUnfreezeAmountRequestMessage{
						OwnerAddress: addr,
						Timestamp:    ts,
					})
				})
			})
		}
	}
}

func TestLocalParity_Witness(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 30*time.Second)

	t.Run("ListWitnesses", func(t *testing.T) {
		parity(t, func(tr client.Transport) (*api.WitnessList, error) {
			return tr.ListWitnesses(ctx)
		})
	})
	for name, addr := range localAddresses(t, chain) {
		t.Run("GetRewardInfo/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*api.NumberMessage, error) {
				return tr.GetRewardInfo(ctx, addr)
			})
		})
		t.Run("GetBrokerageInfo/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*api.NumberMessage, error) {
				return tr.GetBrokerageInfo(ctx, addr)
			})
		})
	}
}

func TestLocalParity_Asset(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 30*time.Second)

	for name, id := range map[string]string{"issued": chain.AssetID, "unknown": "1999999"} {
		t.Run("GetAssetIssueById/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*core.AssetIssueContract, error) {
				return tr.GetAssetIssueById(ctx, []byte(id))
			})
		})
	}
	for name, assetName := range map[string]string{"issued": chain.AssetName, "unknown": "NoSuchAsset"} {
		t.Run("GetAssetIssueListByName/"+name, func(t *testing.T) {
			parity(t, func(tr client.Transport) (*api.AssetIssueList, error) {
				return tr.GetAssetIssueListByName(ctx, []byte(assetName))
			})
		})
	}
}

func TestLocalParity_Network(t *testing.T) {
	localFixture(t)
	ctx := localContext(t, 30*time.Second)

	t.Run("ListNodes", func(t *testing.T) {
		parity(t, func(tr client.Transport) (*api.NodeList, error) {
			return tr.ListNodes(ctx)
		})
	})
	t.Run("GetChainParameters", func(t *testing.T) {
		parity(t, func(tr client.Transport) (*core.ChainParameters, error) {
			return tr.GetChainParameters(ctx)
		})
	})
	t.Run("GetNextMaintenanceTime", func(t *testing.T) {
		stableParity(t, func(tr client.Transport) (*api.NumberMessage, error) {
			return tr.GetNextMaintenanceTime(ctx)
		})
	})
	t.Run("TotalTransaction", func(t *testing.T) {
		stableParity(t, func(tr client.Transport) (*api.NumberMessage, error) {
			return tr.TotalTransaction(ctx)
		})
	})
	t.Run("GetNodeInfo", func(t *testing.T) {
		g, h := localTransports(t)
		want, err := g.GetNodeInfo(ctx)
		require.NoError(t, err)
		got, err := h.GetNodeInfo(ctx)
		require.NoError(t, err)

		// Only what does not move from one call to the next is compared: the
		// head block, peer counts, memory and CPU figures all do.
		requireProtoEqual(t, want.GetConfigNodeInfo(), got.GetConfigNodeInfo())
		assert.Equal(t, want.GetMachineInfo().GetJavaVersion(), got.GetMachineInfo().GetJavaVersion())
		assert.Equal(t, want.GetMachineInfo().GetCpuCount(), got.GetMachineInfo().GetCpuCount())
		assert.Equal(t, want.GetSolidityBlock() != "", got.GetSolidityBlock() != "")
	})
}

// stableParity is parity for values that change over time: it repeats the
// pair of calls until the gRPC answer is the same before and after the HTTP
// one, so a difference is a decoding bug and not a block landing in between.
func stableParity[T proto.Message](t *testing.T, call func(client.Transport) (T, error)) {
	t.Helper()
	g, h := localTransports(t)

	for range 10 {
		before, err := call(g)
		require.NoError(t, err)
		got, err := call(h)
		require.NoError(t, err)
		after, err := call(g)
		require.NoError(t, err)
		if proto.Equal(before, after) {
			requireProtoEqual(t, before, got)
			return
		}
	}
	t.Fatal("value kept changing between calls")
}

// constantCalls are the calls TriggerConstantContract and EstimateEnergy are
// compared on: a read, a state-changing call, one that reverts, one that
// fails validation and a deployment (no contract address).
func constantCalls(t *testing.T, chain *localChain) map[string]*core.TriggerSmartContract {
	t.Helper()
	witness := mustDecode(t, localWitnessAddress)
	staker := mustDecode(t, chain.Staker.Address)
	usdt := mustDecode(t, chain.TRC20)

	balanceOf, err := abi.Pack("balanceOf(address)", []abi.Param{{"address": chain.Staker.Address}})
	require.NoError(t, err)
	transfer, err := abi.Pack("transfer(address,uint256)", []abi.Param{{"address": chain.Receiver.Address}, {"uint256": "1"}})
	require.NoError(t, err)
	tooMuch, err := abi.Pack("transfer(address,uint256)", []abi.Param{{"address": chain.Receiver.Address}, {"uint256": "999999999999"}})
	require.NoError(t, err)

	return map[string]*core.TriggerSmartContract{
		"read":            {OwnerAddress: staker, ContractAddress: usdt, Data: balanceOf},
		"write":           {OwnerAddress: witness, ContractAddress: usdt, Data: transfer},
		"revert":          {OwnerAddress: staker, ContractAddress: usdt, Data: tooMuch},
		"internal call":   {OwnerAddress: staker, ContractAddress: mustDecode(t, chain.Payer), Data: []byte{1}},
		"not a contract":  {OwnerAddress: staker, ContractAddress: staker, Data: balanceOf},
		"deploy":          {OwnerAddress: witness, Data: mustHex(t, localSelfPayingContract)},
		"with call value": {OwnerAddress: witness, ContractAddress: mustDecode(t, chain.Payer), Data: []byte{1}, CallValue: 5},
	}
}

// TestLocalParity_TriggerConstantContract compares everything but what the
// node stamps on each call afresh: the transaction's timestamp, and the txid
// and internal transaction hashes derived from it. Nothing checks the txid
// against the transaction: for a constant deployment gRPC itself answers with
// one that is not the hash of the transaction it returns (HTTP's is), and a
// constant call's transaction is never broadcast.
//
// Accepted difference: a call refused before execution carries the node's
// message worded per transport - "Contract validate error : <reason>" over
// gRPC, "<reason>" over HTTP. java-tron builds the two in different places;
// the code is the same.
func TestLocalParity_TriggerConstantContract(t *testing.T) {
	chain := localFixture(t)
	for name, call := range constantCalls(t, chain) {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
				ext, err := tr.TriggerConstantContract(ctx, call)
				if err != nil {
					return nil, err
				}
				return normalizeConstantCall(t, ext), nil
			})
		})
	}
}

func normalizeConstantCall(t *testing.T, ext *api.TransactionExtention) *api.TransactionExtention {
	t.Helper()
	ext = proto.CloneOf(ext)
	if raw := ext.GetTransaction().GetRawData(); raw != nil {
		raw.Timestamp = 0
	}
	ext.Txid = nil
	for _, internal := range ext.GetInternalTransactions() {
		internal.Hash = nil
	}
	if r := ext.GetResult(); r != nil {
		r.Message = []byte(strings.TrimPrefix(string(r.GetMessage()), "Contract validate error : "))
	}
	return ext
}

func TestLocalParity_EstimateEnergy(t *testing.T) {
	chain := localFixture(t)
	for name, call := range constantCalls(t, chain) {
		t.Run(name, func(t *testing.T) {
			ctx := localContext(t, 10*time.Second)
			parity(t, func(tr client.Transport) (*api.EstimateEnergyMessage, error) {
				msg, err := tr.EstimateEnergy(ctx, call)
				if r := msg.GetResult(); r != nil {
					// The same per-transport wording as TriggerConstantContract.
					r.Message = []byte(strings.TrimPrefix(string(r.GetMessage()), "Contract validate error : "))
				}
				return msg, err
			})
		})
	}
}
