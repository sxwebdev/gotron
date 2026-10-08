package tests

// Parity of every transaction-building method: the same contract is built over
// gRPC and over HTTP, and the two transactions must be the same transaction
// but for what the node stamps on each one (timestamp, expiration and the
// reference block, which may advance between the two calls). Nothing here is
// broadcast.
//
// A refusal is compared by outcome and reason, not byte for byte: the two
// transports report it in different shapes by design. gRPC answers with an
// extention whose Result carries the code ("Contract validate error : <reason>"
// in its message); the HTTP transport returns a *client.ContractValidateError
// built from the node's {"Error": "class ...ContractValidateException : <reason>"}.
// Client methods turn both into a *client.ContractValidateError.

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// buildOutcome is what a transaction-building call produced: the transaction,
// or the reason the node refused to build it.
type buildOutcome struct {
	ext    *api.TransactionExtention
	reason string
}

func outcomeOf(t *testing.T, ext *api.TransactionExtention, err error) buildOutcome {
	t.Helper()
	if err != nil {
		var cve *client.ContractValidateError
		require.True(t, errors.As(err, &cve), "not a refusal: %v", err)
		return buildOutcome{reason: refusalReason(cve.Message)}
	}
	if code := ext.GetResult().GetCode(); code != api.Return_SUCCESS {
		return buildOutcome{reason: refusalReason(string(ext.GetResult().GetMessage()))}
	}
	return buildOutcome{ext: ext}
}

// refusalReason strips the transport-specific prefix the node puts in front
// of the actuator's own message.
func refusalReason(msg string) string {
	for _, prefix := range []string{
		"class org.tron.core.exception.ContractValidateException : ",
		"Contract validate error : ",
	} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	return msg
}

// txParity builds through both transports and compares the results.
func txParity(t *testing.T, build func(client.Transport) (*api.TransactionExtention, error)) buildOutcome {
	t.Helper()
	g, h := localTransports(t)

	gExt, gErr := build(g)
	want := outcomeOf(t, gExt, gErr)
	hExt, hErr := build(h)
	got := outcomeOf(t, hExt, hErr)

	if want.ext == nil || got.ext == nil {
		require.Equal(t, want.reason, got.reason, "gRPC and HTTP disagree on the refusal")
		require.Equal(t, want.ext == nil, got.ext == nil)
		return want
	}

	requireTxidMatches(t, want.ext)
	requireTxidMatches(t, got.ext)
	requireProtoEqual(t, normalizeBuilt(want.ext), normalizeBuilt(got.ext))
	return want
}

// requireTxidMatches checks that an extention's txid is its transaction's hash.
func requireTxidMatches(t *testing.T, ext *api.TransactionExtention) {
	t.Helper()
	raw, err := proto.Marshal(ext.GetTransaction().GetRawData())
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	require.Equal(t, sum[:], ext.GetTxid(), "txid is not the hash of raw_data")
}

func normalizeBuilt(ext *api.TransactionExtention) *api.TransactionExtention {
	ext = proto.CloneOf(ext)
	raw := ext.GetTransaction().GetRawData()
	raw.Timestamp, raw.Expiration = 0, 0
	raw.RefBlockBytes, raw.RefBlockHash = nil, nil
	ext.Txid = nil
	return ext
}

// requireBuilt and requireRefused pin which of the two the case is meant to
// exercise, so a fixture drifting into the other branch cannot pass unnoticed.
func requireBuilt(t *testing.T, o buildOutcome) {
	t.Helper()
	require.NotNil(t, o.ext, "expected a transaction, the node refused: %s", o.reason)
}

func requireRefused(t *testing.T, o buildOutcome, reason string) {
	t.Helper()
	require.Nil(t, o.ext, "expected a refusal")
	require.Contains(t, o.reason, reason)
}

func TestLocalParity_CreateAccount(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	witness := mustDecode(t, localWitnessAddress)

	t.Run("new", func(t *testing.T) {
		fresh := mustDecode(t, newLocalKey(t).Address)
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CreateAccount(ctx, &core.AccountCreateContract{OwnerAddress: witness, AccountAddress: fresh})
		}))
	})
	t.Run("exists", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CreateAccount(ctx, &core.AccountCreateContract{OwnerAddress: witness, AccountAddress: mustDecode(t, chain.Staker.Address)})
		}), "Account has existed")
	})
}

func TestLocalParity_AccountPermissionUpdate(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)

	owner, err := client.NewOwnerPermission("owner", 1, client.PermissionKey{Address: chain.Staker.Address, Weight: 1})
	require.NoError(t, err)
	ops, err := client.ContractOperations(core.Transaction_Contract_TransferContract, core.Transaction_Contract_TriggerSmartContract)
	require.NoError(t, err)
	active, err := client.NewActivePermission("spender", 2, ops,
		client.PermissionKey{Address: chain.Staker.Address, Weight: 1},
		client.PermissionKey{Address: chain.Receiver.Address, Weight: 1})
	require.NoError(t, err)

	requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
		return tr.AccountPermissionUpdate(ctx, &core.AccountPermissionUpdateContract{
			OwnerAddress: mustDecode(t, chain.Staker.Address),
			Owner:        owner,
			Actives:      []*core.Permission{active},
		})
	}))
}

func TestLocalParity_CreateTransaction(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	staker, witness := mustDecode(t, chain.Staker.Address), mustDecode(t, localWitnessAddress)

	t.Run("transfer", func(t *testing.T) {
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CreateTransaction(ctx, &core.TransferContract{OwnerAddress: witness, ToAddress: staker, Amount: 1})
		}))
	})
	t.Run("to a new account", func(t *testing.T) {
		fresh := mustDecode(t, newLocalKey(t).Address)
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CreateTransaction(ctx, &core.TransferContract{OwnerAddress: staker, ToAddress: fresh, Amount: trx(1).Int64()})
		}))
	})
	t.Run("insufficient balance", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CreateTransaction(ctx, &core.TransferContract{OwnerAddress: staker, ToAddress: witness, Amount: trx(10_000_000).Int64()})
		}), "balance is not sufficient")
	})
	t.Run("to itself", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CreateTransaction(ctx, &core.TransferContract{OwnerAddress: staker, ToAddress: staker, Amount: 1})
		}), "Cannot transfer TRX to yourself")
	})
}

func TestLocalParity_TriggerContract(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	staker, usdt := mustDecode(t, chain.Staker.Address), mustDecode(t, chain.TRC20)

	transfer, err := abi.Pack("transfer(address,uint256)", []abi.Param{{"address": chain.Receiver.Address}, {"uint256": "1"}})
	require.NoError(t, err)

	for name, c := range map[string]*core.TriggerSmartContract{
		"trc20 transfer":  {OwnerAddress: staker, ContractAddress: usdt, Data: transfer},
		"with call value": {OwnerAddress: staker, ContractAddress: mustDecode(t, chain.Payer), Data: []byte{1}, CallValue: 7},
		"with trc10": {
			OwnerAddress: staker, ContractAddress: mustDecode(t, chain.Payer), Data: []byte{1},
			CallTokenValue: 3, TokenId: assetID(t, chain),
		},
	} {
		t.Run(name, func(t *testing.T) {
			requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
				return tr.TriggerContract(ctx, c)
			}))
		})
	}
	t.Run("not a contract", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.TriggerContract(ctx, &core.TriggerSmartContract{OwnerAddress: staker, ContractAddress: staker, Data: transfer})
		}), "No contract or not a valid smart contract")
	})
}

func assetID(t *testing.T, chain *localChain) int64 {
	t.Helper()
	var id int64
	for _, c := range chain.AssetID {
		id = id*10 + int64(c-'0')
	}
	return id
}

// TestLocalParity_DeployContract builds deployments with and without an ABI.
//
// Accepted difference: without an ABI the HTTP transaction carries an empty
// "abi {}" where gRPC's has none - DeployContractServlet always sets the
// field. The two mean the same, but the transaction is two bytes longer, so
// its txid, and the contract address derived from it, differ from gRPC's.
// DeployedContractAddress reads the transaction actually returned, so it is
// right for either.
func TestLocalParity_DeployContract(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	witness, staker := mustDecode(t, localWitnessAddress), mustDecode(t, chain.Staker.Address)

	usdt := tetherTokenRequest(t, localWitnessAddress)
	usdtCode := mustHex(t, usdt.Bytecode)
	refunder := mustHex(t, localSelfPayingContract)

	for name, c := range map[string]*core.CreateSmartContract{
		// The full 46-entry ABI is what /wallet/deploycontract used to lose.
		"full abi": {OwnerAddress: witness, NewContract: &core.SmartContract{
			OriginAddress: witness, Name: "TetherToken", Abi: usdt.ABI, Bytecode: usdtCode,
			ConsumeUserResourcePercent: 100, OriginEnergyLimit: 10_000_000,
		}},
		"no abi": {OwnerAddress: witness, NewContract: &core.SmartContract{
			OriginAddress: witness, Name: "Refunder", Bytecode: refunder, OriginEnergyLimit: 1,
		}},
		"call value": {OwnerAddress: witness, NewContract: &core.SmartContract{
			OriginAddress: witness, Name: "Refunder", Bytecode: refunder, OriginEnergyLimit: 1,
			CallValue: 5, ConsumeUserResourcePercent: 30,
		}},
		"trc10": {OwnerAddress: staker, CallTokenValue: 4, TokenId: assetID(t, chain), NewContract: &core.SmartContract{
			OriginAddress: staker, Name: "Refunder", Bytecode: refunder, OriginEnergyLimit: 1,
		}},
	} {
		t.Run(name, func(t *testing.T) {
			requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
				ext, err := tr.DeployContract(ctx, c)
				if err != nil || c.GetNewContract().GetAbi() != nil {
					return ext, err
				}
				return withoutEmptyABI(t, ext), nil
			}))
		})
	}
}

func TestLocalParity_UpdateContractSettings(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	witness, staker, usdt := mustDecode(t, localWitnessAddress), mustDecode(t, chain.Staker.Address), mustDecode(t, chain.TRC20)

	t.Run("UpdateSetting", func(t *testing.T) {
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UpdateSetting(ctx, &core.UpdateSettingContract{OwnerAddress: witness, ContractAddress: usdt, ConsumeUserResourcePercent: 40})
		}))
	})
	t.Run("UpdateSetting by a stranger", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UpdateSetting(ctx, &core.UpdateSettingContract{OwnerAddress: staker, ContractAddress: usdt, ConsumeUserResourcePercent: 40})
		}), "is not the owner of the contract")
	})
	t.Run("UpdateEnergyLimit", func(t *testing.T) {
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UpdateEnergyLimit(ctx, &core.UpdateEnergyLimitContract{OwnerAddress: witness, ContractAddress: usdt, OriginEnergyLimit: 123})
		}))
	})
	t.Run("UpdateEnergyLimit by a stranger", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UpdateEnergyLimit(ctx, &core.UpdateEnergyLimitContract{OwnerAddress: staker, ContractAddress: usdt, OriginEnergyLimit: 123})
		}), "is not the owner of the contract")
	})
}

func TestLocalParity_Stake(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	staker, receiver := mustDecode(t, chain.Staker.Address), mustDecode(t, chain.Receiver.Address)

	for _, resource := range []core.ResourceCode{core.ResourceCode_BANDWIDTH, core.ResourceCode_ENERGY} {
		t.Run("FreezeBalanceV2/"+resource.String(), func(t *testing.T) {
			requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
				return tr.FreezeBalanceV2(ctx, &core.FreezeBalanceV2Contract{OwnerAddress: staker, FrozenBalance: trx(1).Int64(), Resource: resource})
			}))
		})
		t.Run("UnfreezeBalanceV2/"+resource.String(), func(t *testing.T) {
			requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
				return tr.UnfreezeBalanceV2(ctx, &core.UnfreezeBalanceV2Contract{OwnerAddress: staker, UnfreezeBalance: trx(1).Int64(), Resource: resource})
			}))
		})
		t.Run("DelegateResource/"+resource.String(), func(t *testing.T) {
			requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
				return tr.DelegateResource(ctx, &core.DelegateResourceContract{
					OwnerAddress: staker, ReceiverAddress: receiver, Balance: trx(5).Int64(), Resource: resource,
					Lock: true, LockPeriod: 30,
				})
			}))
		})
	}
	t.Run("FreezeBalanceV2 beyond the balance", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.FreezeBalanceV2(ctx, &core.FreezeBalanceV2Contract{OwnerAddress: staker, FrozenBalance: trx(10_000_000).Int64()})
		}), "frozenBalance must be less than or equal to accountBalance")
	})
	t.Run("UnfreezeBalanceV2 beyond the stake", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UnfreezeBalanceV2(ctx, &core.UnfreezeBalanceV2Contract{OwnerAddress: staker, UnfreezeBalance: trx(10_000_000).Int64()})
		}), "Invalid unfreeze_balance")
	})
	t.Run("UnDelegateResource", func(t *testing.T) {
		// The energy delegation of the fixture is unlocked.
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UnDelegateResource(ctx, &core.UnDelegateResourceContract{OwnerAddress: staker, ReceiverAddress: receiver, Balance: trx(1).Int64(), Resource: core.ResourceCode_ENERGY})
		}))
	})
	t.Run("UnDelegateResource beyond the delegation", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.UnDelegateResource(ctx, &core.UnDelegateResourceContract{OwnerAddress: staker, ReceiverAddress: receiver, Balance: trx(1_000_000).Int64(), Resource: core.ResourceCode_ENERGY})
		}), "insufficient delegateFrozenBalance(Energy)")
	})
	t.Run("DelegateResource beyond the stake", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.DelegateResource(ctx, &core.DelegateResourceContract{OwnerAddress: staker, ReceiverAddress: receiver, Balance: trx(10_000_000).Int64(), Resource: core.ResourceCode_ENERGY})
		}), "delegateBalance must be less than or equal to available FreezeEnergyV2 balance")
	})
	t.Run("CancelAllUnfreezeV2", func(t *testing.T) {
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CancelAllUnfreezeV2(ctx, &core.CancelAllUnfreezeV2Contract{OwnerAddress: staker})
		}))
	})
	t.Run("CancelAllUnfreezeV2 with nothing pending", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.CancelAllUnfreezeV2(ctx, &core.CancelAllUnfreezeV2Contract{OwnerAddress: receiver})
		}), "No unfreezeV2 list to cancel")
	})
	t.Run("WithdrawExpireUnfreeze before the delay", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.WithdrawExpireUnfreeze(ctx, &core.WithdrawExpireUnfreezeContract{OwnerAddress: staker})
		}), "no unFreeze balance to withdraw")
	})
}

// The node's own refusals of a delegation match the sentinels over both
// transports, each with the prefix it puts in front of the verdict. The client
// refuses a delegation under the minimum without asking a node, so the
// transports are asked directly; gRPC reports through the result, which the
// client turns into the error built here.
func TestLocalDelegateRefusals(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	staker, receiver := mustDecode(t, chain.Staker.Address), mustDecode(t, chain.Receiver.Address)
	g, h := localTransports(t)

	cases := []struct {
		name     string
		resource core.ResourceCode
		balance  client.SUN
		want     error
	}{
		{"under the minimum", core.ResourceCode_ENERGY, client.MinDelegateBalance - 1, client.ErrDelegateBelowMinimum},
		{"beyond the energy stake", core.ResourceCode_ENERGY, trx(10_000_000), client.ErrDelegateStakeShort},
		{"beyond the bandwidth stake", core.ResourceCode_BANDWIDTH, trx(10_000_000), client.ErrDelegateStakeShort},
	}
	for _, tc := range cases {
		for name, tr := range map[string]client.Transport{"grpc": g, "http": h} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				ext, err := tr.DelegateResource(ctx, &core.DelegateResourceContract{
					OwnerAddress: staker, ReceiverAddress: receiver, Balance: tc.balance.Int64(), Resource: tc.resource,
				})
				if err == nil {
					require.NotEqual(t, api.Return_SUCCESS, ext.GetResult().GetCode(), "expected a refusal")
					err = &client.ContractValidateError{Code: ext.GetResult().GetCode(), Message: string(ext.GetResult().GetMessage())}
				}
				require.ErrorIs(t, err, tc.want)
			})
		}
	}
}

func TestLocalParity_Witness_Transactions(t *testing.T) {
	chain := localFixture(t)
	ctx := localContext(t, 20*time.Second)
	staker, witness := mustDecode(t, chain.Staker.Address), mustDecode(t, localWitnessAddress)

	t.Run("VoteWitnessAccount", func(t *testing.T) {
		requireBuilt(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.VoteWitnessAccount(ctx, &core.VoteWitnessContract{
				OwnerAddress: staker,
				Votes:        []*core.VoteWitnessContract_Vote{{VoteAddress: witness, VoteCount: 7}},
			})
		}))
	})
	t.Run("VoteWitnessAccount for a non-witness", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.VoteWitnessAccount(ctx, &core.VoteWitnessContract{
				OwnerAddress: staker,
				Votes:        []*core.VoteWitnessContract_Vote{{VoteAddress: staker, VoteCount: 1}},
			})
		}), "not exists")
	})
	t.Run("WithdrawBalance", func(t *testing.T) {
		// Whether the witness may withdraw right now depends on when it last
		// did; both transports must agree either way.
		txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.WithdrawBalance(ctx, &core.WithdrawBalanceContract{OwnerAddress: witness})
		})
	})
	t.Run("WithdrawBalance with nothing to withdraw", func(t *testing.T) {
		requireRefused(t, txParity(t, func(tr client.Transport) (*api.TransactionExtention, error) {
			return tr.WithdrawBalance(ctx, &core.WithdrawBalanceContract{OwnerAddress: mustDecode(t, chain.Receiver.Address)})
		}), "")
	})
}

// withoutEmptyABI drops the empty ABI the HTTP deployment endpoint adds, and
// re-hashes the transaction.
func withoutEmptyABI(t *testing.T, ext *api.TransactionExtention) *api.TransactionExtention {
	t.Helper()
	if ext.GetResult().GetCode() != api.Return_SUCCESS {
		return ext
	}
	param := ext.GetTransaction().GetRawData().GetContract()[0].GetParameter()
	deploy := &core.CreateSmartContract{}
	require.NoError(t, param.UnmarshalTo(deploy))
	if abi := deploy.GetNewContract().GetAbi(); abi != nil {
		require.Empty(t, abi.GetEntrys())
		deploy.NewContract.Abi = nil
		require.NoError(t, param.MarshalFrom(deploy))
		require.NoError(t, ext.UpdateHash())
	}
	return ext
}
