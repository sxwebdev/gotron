package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Regression: a transaction with a nil Result (common on success) must not
// panic via tx.Result.Code.
func TestTriggerContractNilResultDoesNotPanic(t *testing.T) {
	c := newTestClient(&fakeTransport{
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Transaction: &core.Transaction{}}, nil // Result == nil
		},
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TriggerContract panicked on nil Result: %v", r)
		}
	}()

	// feeLimit 0 so the FeeLimit/UpdateHash branch is skipped.
	_, err := c.TriggerContract(context.Background(), testAddr, testAddr2, "name()", "", 0, 0, "", 0)
	require.NoError(t, err)
}

func TestTriggerContractResultCodeError(t *testing.T) {
	c := newTestClient(&fakeTransport{
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("reverted")},
			}, nil
		},
	})
	_, err := c.TriggerContract(context.Background(), testAddr, testAddr2, "name()", "", 0, 0, "", 0)
	require.Error(t, err)
	require.ErrorContains(t, err, "reverted")
}

// A reverted constant call is reported with code SUCCESS and result=true; the
// only signal is the message. Reading the code alone - the obvious thing to do,
// and what every other method here does - never fires, and the caller takes the
// energy burned up to the revert for the real cost of the call.
func TestTriggerConstantContractDetectsRevert(t *testing.T) {
	reverted := &api.TransactionExtention{
		Result:         &api.Return{Result: true, Code: api.Return_SUCCESS, Message: []byte("REVERT opcode executed")},
		EnergyUsed:     8624,
		ConstantResult: [][]byte{{}},
	}

	c := newTestClient(&fakeTransport{
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return reverted, nil
		},
	})

	tx, err := c.TriggerConstantContract(context.Background(), &core.TriggerSmartContract{})
	require.ErrorIs(t, err, ErrContractCallFailed)
	require.ErrorContains(t, err, "REVERT opcode executed")

	// The extention comes back with the error: the energy burned before the
	// revert and whatever the VM produced are still the caller's to inspect.
	require.NotNil(t, tx)
	require.Equal(t, int64(8624), tx.GetEnergyUsed())
}

func TestTriggerConstantContractSurfacesValidationCode(t *testing.T) {
	c := newTestClient(&fakeTransport{
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR},
			}, nil
		},
	})

	_, err := c.TriggerConstantContract(context.Background(), &core.TriggerSmartContract{})
	require.ErrorIs(t, err, ErrContractCallFailed)
	require.ErrorContains(t, err, "CONTRACT_VALIDATE_ERROR")
}

func TestTriggerConstantContractSuccess(t *testing.T) {
	cases := []struct {
		name   string
		result *api.Return
	}{
		{"result true, no message", &api.Return{Result: true, Code: api.Return_SUCCESS}},
		// Some nodes omit the result object entirely on a successful read.
		{"no result at all", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(&fakeTransport{
				triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{
						Result:         tc.result,
						ConstantResult: [][]byte{make([]byte, 32)},
						EnergyUsed:     64285,
					}, nil
				},
			})

			tx, err := c.TriggerConstantContract(context.Background(), &core.TriggerSmartContract{})
			require.NoError(t, err)
			require.Equal(t, int64(64285), tx.GetEnergyUsed())
		})
	}
}

func TestGetContract(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getContract: func(context.Context, []byte) (*core.SmartContract, error) { return nil, nil },
		})
		_, err := c.GetContract(context.Background(), testAddr)
		require.ErrorContains(t, err, "contract not found")
	})

	t.Run("invalid address", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetContract(context.Background(), "bad!")
		require.Error(t, err)
	})
}

func TestGetContractABI(t *testing.T) {
	t.Run("no abi", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getContract: func(context.Context, []byte) (*core.SmartContract, error) {
				return &core.SmartContract{}, nil // Abi == nil
			},
		})
		_, err := c.GetContractABI(context.Background(), testAddr)
		require.ErrorContains(t, err, "no ABI")
	})

	t.Run("with abi", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getContract: func(context.Context, []byte) (*core.SmartContract, error) {
				return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
			},
		})
		got, err := c.GetContractABI(context.Background(), testAddr)
		require.NoError(t, err)
		require.NotNil(t, got)
	})
}

// validDeploy is a request the chain would accept, for tests that vary one field.
func validDeploy() DeployContractRequest {
	return DeployContractRequest{
		From:                       testAddr,
		Name:                       "Token",
		Bytecode:                   "6080604052",
		ConsumeUserResourcePercent: 100,
		OriginEnergyLimit:          10_000_000,
	}
}

// deployTx is a node's answer to a deployment: a real transaction carrying the
// CreateSmartContract, so DeployedContractAddress has something to read.
func deployTx(t *testing.T, owner string) *api.TransactionExtention {
	t.Helper()

	ownerBytes, err := tronutils.DecodeCheck(owner)
	require.NoError(t, err)

	param, err := anypb.New(&core.CreateSmartContract{
		OwnerAddress: ownerBytes,
		NewContract:  &core.SmartContract{OriginAddress: ownerBytes, Bytecode: []byte{0x60}},
	})
	require.NoError(t, err)

	return &api.TransactionExtention{
		Txid: []byte{0x01},
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Timestamp: 1,
				Contract: []*core.Transaction_Contract{{
					Type:      core.Transaction_Contract_CreateSmartContract,
					Parameter: param,
				}},
			},
		},
	}
}

func TestDeployContractValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*DeployContractRequest)
		wantErr error
	}{
		{"empty from", func(r *DeployContractRequest) { r.From = "" }, ErrEmptyAddress},
		{"invalid from", func(r *DeployContractRequest) { r.From = "bad!" }, ErrInvalidAddress},
		{"empty bytecode", func(r *DeployContractRequest) { r.Bytecode = "" }, ErrInvalidParams},
		{"blank bytecode", func(r *DeployContractRequest) { r.Bytecode = "  " }, ErrInvalidParams},
		{"non-hex bytecode", func(r *DeployContractRequest) { r.Bytecode = "zz" }, ErrInvalidParams},
		{"odd-length hex", func(r *DeployContractRequest) { r.Bytecode = "608" }, ErrInvalidParams},
		// "0x" passes a non-empty string check but decodes to nothing, and a
		// contract with no code is not a contract.
		{"bytecode is just the prefix", func(r *DeployContractRequest) { r.Bytecode = "0x" }, ErrInvalidParams},
		{"percent above 100", func(r *DeployContractRequest) { r.ConsumeUserResourcePercent = 101 }, ErrInvalidParams},
		{"percent negative", func(r *DeployContractRequest) { r.ConsumeUserResourcePercent = -1 }, ErrInvalidParams},
		{"zero origin energy limit", func(r *DeployContractRequest) { r.OriginEnergyLimit = 0 }, ErrInvalidParams},
		{"negative origin energy limit", func(r *DeployContractRequest) { r.OriginEnergyLimit = -1 }, ErrInvalidParams},
		{"negative fee limit", func(r *DeployContractRequest) { r.FeeLimit = -1 }, ErrInvalidAmount},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			c := newTestClient(&fakeTransport{
				deployContract: func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
					called = true
					return nil, nil
				},
			})

			req := validDeploy()
			tc.mutate(&req)

			tx, err := c.DeployContract(context.Background(), req)
			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, tx)
			// A request the SDK can reject must not cost a round-trip.
			require.False(t, called, "transport was called for an invalid request")
		})
	}
}

func TestDeployContractValidationAcceptsTheBoundaries(t *testing.T) {
	// 0 and 100 are both legal percentages, and rejecting either would make a
	// contract that pays all or none of its callers' costs undeployable.
	for _, percent := range []int64{0, 100} {
		req := validDeploy()
		req.ConsumeUserResourcePercent = percent
		require.NoError(t, req.Validate())
	}

	// A zero fee limit is "let the node decide", not an invalid amount.
	req := validDeploy()
	req.FeeLimit = 0
	require.NoError(t, req.Validate())
}

func TestDeployContractBuildsTheContract(t *testing.T) {
	contractABI, err := abi.LoadContractABI(`[{"type":"function","name":"ping","inputs":[],"outputs":[]}]`)
	require.NoError(t, err)

	var got *core.CreateSmartContract
	c := newTestClient(&fakeTransport{
		deployContract: func(_ context.Context, ct *core.CreateSmartContract) (*api.TransactionExtention, error) {
			got = ct
			return deployTx(t, testAddr), nil
		},
	})

	req := validDeploy()
	req.ABI = contractABI
	req.Bytecode = "0x6080604052"
	req.ConsumeUserResourcePercent = 42
	req.OriginEnergyLimit = 7_000_000

	_, err = c.DeployContract(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, got)

	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	require.Equal(t, wantOwner, got.GetOwnerAddress())
	// OriginAddress is what the chain bills for the contract's own resources;
	// leaving it empty deploys an ownerless contract.
	require.Equal(t, wantOwner, got.GetNewContract().GetOriginAddress())

	require.Equal(t, "Token", got.GetNewContract().GetName())
	require.Equal(t, int64(42), got.GetNewContract().GetConsumeUserResourcePercent())
	require.Equal(t, int64(7_000_000), got.GetNewContract().GetOriginEnergyLimit())
	// The 0x prefix must be stripped, not encoded as bytecode.
	require.Equal(t, []byte{0x60, 0x80, 0x60, 0x40, 0x52}, got.GetNewContract().GetBytecode())
	require.Len(t, got.GetNewContract().GetAbi().GetEntrys(), 1)
}

func TestDeployContractAppendsConstructorParams(t *testing.T) {
	// Tron has no field for constructor arguments: they are ABI-encoded and
	// appended to the bytecode. A deployment that drops them leaves the
	// constructor reading zeros.
	var got *core.CreateSmartContract
	c := newTestClient(&fakeTransport{
		deployContract: func(_ context.Context, ct *core.CreateSmartContract) (*api.TransactionExtention, error) {
			got = ct
			return deployTx(t, testAddr), nil
		},
	})

	req := validDeploy()
	req.Bytecode = "6080604052"
	req.ConstructorParams = `[{"uint256":"1000000"},{"address":"` + testAddr2 + `"}]`

	_, err := c.DeployContract(context.Background(), req)
	require.NoError(t, err)

	code := got.GetNewContract().GetBytecode()
	// Two 32-byte words follow the bytecode, and nothing was inserted before it.
	require.Equal(t, 5+64, len(code))
	require.Equal(t, []byte{0x60, 0x80, 0x60, 0x40, 0x52}, code[:5])

	// The arguments must be exactly what the ABI encoder produces, in the
	// order given - swapping them deploys a contract with the wrong owner.
	params, err := abi.LoadFromJSON(req.ConstructorParams)
	require.NoError(t, err)
	want, err := abi.GetPaddedParam(params)
	require.NoError(t, err)
	require.Equal(t, want, code[5:])

	// And the first word really is the number, not the address.
	require.Equal(t, "00000000000000000000000000000000000000000000000000000000000f4240",
		hex.EncodeToString(code[5:37]))
}

func TestDeployContractWithoutConstructorParamsLeavesBytecodeAlone(t *testing.T) {
	var got *core.CreateSmartContract
	c := newTestClient(&fakeTransport{
		deployContract: func(_ context.Context, ct *core.CreateSmartContract) (*api.TransactionExtention, error) {
			got = ct
			return deployTx(t, testAddr), nil
		},
	})

	_, err := c.DeployContract(context.Background(), validDeploy())
	require.NoError(t, err)
	require.Equal(t, []byte{0x60, 0x80, 0x60, 0x40, 0x52}, got.GetNewContract().GetBytecode())
}

func TestDeployContractRejectsBadConstructorParams(t *testing.T) {
	cases := []struct{ name, params string }{
		{"malformed json", `[{"uint256":`},
		{"unknown solidity type", `[{"notatype":"1"}]`},
		{"number instead of string", `[{"uint256":1}]`},
		{"invalid address", `[{"address":"nope"}]`},
		{"two types in one argument", `[{"uint256":"1","address":"` + testAddr + `"}]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			c := newTestClient(&fakeTransport{
				deployContract: func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
					called = true
					return deployTx(t, testAddr), nil
				},
			})

			req := validDeploy()
			req.ConstructorParams = tc.params

			_, err := c.DeployContract(context.Background(), req)
			require.ErrorIs(t, err, ErrInvalidParams)
			require.False(t, called, "a deployment with unencodable arguments was sent anyway")
		})
	}
}

func TestDeployContractRejectsWhatTheNodeRefused(t *testing.T) {
	cases := []struct {
		name string
		tx   *api.TransactionExtention
	}{
		{
			// Over gRPC a refusal arrives like this, with a nil error - so
			// without the check the caller signs and broadcasts nothing.
			name: "validation error",
			tx: &api.TransactionExtention{
				Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("validate error")},
			},
		},
		{"empty response", &api.TransactionExtention{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(&fakeTransport{
				deployContract: func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
					return tc.tx, nil
				},
			})

			tx, err := c.DeployContract(context.Background(), validDeploy())
			require.Error(t, err)
			require.Nil(t, tx)
		})
	}
}

func TestDeployContractFeeLimit(t *testing.T) {
	newClient := func() *Client {
		return newTestClient(&fakeTransport{
			deployContract: func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) {
				return deployTx(t, testAddr), nil
			},
		})
	}

	t.Run("set and hashed", func(t *testing.T) {
		req := validDeploy()
		req.FeeLimit = MustFromTRX(decimal.NewFromInt(1000))

		tx, err := newClient().DeployContract(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, int64(1000)*1_000_000, tx.GetTransaction().GetRawData().GetFeeLimit())

		// The fee limit is inside raw_data, so leaving the txID stale would
		// hand back a hash that no longer identifies the transaction - and the
		// contract address is derived from that hash.
		raw, err := proto.Marshal(tx.GetTransaction().GetRawData())
		require.NoError(t, err)
		want := sha256.Sum256(raw)
		require.Equal(t, want[:], tx.GetTxid())
	})

	t.Run("zero leaves it unset", func(t *testing.T) {
		tx, err := newClient().DeployContract(context.Background(), validDeploy())
		require.NoError(t, err)
		require.Zero(t, tx.GetTransaction().GetRawData().GetFeeLimit())
	})
}

// The derivation itself, pinned to mainnet. Both contracts are live and their
// creation transactions are public, so these are the chain's own answers, not
// this package's.
func TestContractAddressFromTxID(t *testing.T) {
	cases := []struct {
		name, txID, owner, want string
	}{
		{
			name:  "USDT",
			txID:  "3b27180746e68744e5e2e981ae6fa54d502f2aa6e18b8a98824fd1a69069d55a",
			owner: "THPvaUhoh2Qn2y9THCZML3H815hhFhn5YC",
			want:  "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		},
		{
			name:  "account activator",
			txID:  "1402ac1360d8f6c539f7ab4dc0feb5d30fc8716ab40bc2be8fe1f5091d55c9ab",
			owner: "TG3bVVPCouQzQNwp2uhNqcFWi19UrntBQt",
			want:  "TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			txID, err := hex.DecodeString(tc.txID)
			require.NoError(t, err)
			owner, err := tronutils.DecodeCheck(tc.owner)
			require.NoError(t, err)

			require.Equal(t, tc.want, contractAddressFromTxID(txID, owner))
		})
	}
}

func TestDeployedContractAddress(t *testing.T) {
	tx := deployTx(t, testAddr).GetTransaction()

	got, err := DeployedContractAddress(tx)
	require.NoError(t, err)

	// Derived from this transaction's own hash and owner - not from the Txid
	// field, which a caller may have left stale.
	raw, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)
	txID := sha256.Sum256(raw)
	owner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	require.Equal(t, contractAddressFromTxID(txID[:], owner), got)

	// And it is a real Tron address, not a hex blob: decodable, 21 bytes, with
	// the network prefix the chain expects.
	decoded, err := tronutils.DecodeCheck(got)
	require.NoError(t, err)
	require.Len(t, decoded, 21)
	require.Equal(t, tronutils.TronBytePrefix, decoded[0])
}

func TestDeployedContractAddressDependsOnTheWholeTransaction(t *testing.T) {
	base := deployTx(t, testAddr).GetTransaction()
	first, err := DeployedContractAddress(base)
	require.NoError(t, err)

	t.Run("different owner", func(t *testing.T) {
		other, err := DeployedContractAddress(deployTx(t, testAddr2).GetTransaction())
		require.NoError(t, err)
		require.NotEqual(t, first, other)
	})

	t.Run("fee limit moves it", func(t *testing.T) {
		// Documents the ordering hazard: read the address after the last edit
		// to raw_data, never before.
		edited := proto.Clone(base).(*core.Transaction)
		edited.RawData.FeeLimit = 1_000_000

		moved, err := DeployedContractAddress(edited)
		require.NoError(t, err)
		require.NotEqual(t, first, moved)
	})
}

func TestDeployedContractAddressRejects(t *testing.T) {
	transferParam, err := anypb.New(&core.TransferContract{})
	require.NoError(t, err)

	cases := []struct {
		name string
		tx   *core.Transaction
	}{
		{"nil transaction", nil},
		{"no contracts", &core.Transaction{RawData: &core.TransactionRaw{}}},
		{
			name: "wrong contract type",
			tx: &core.Transaction{RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{
					Type:      core.Transaction_Contract_TransferContract,
					Parameter: transferParam,
				}},
			}},
		},
		{
			name: "type says deployment but payload does not",
			tx: &core.Transaction{RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{
					Type:      core.Transaction_Contract_CreateSmartContract,
					Parameter: transferParam,
				}},
			}},
		},
		{
			name: "deployment without an owner",
			tx: &core.Transaction{RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{
					Type:      core.Transaction_Contract_CreateSmartContract,
					Parameter: mustAny(t, &core.CreateSmartContract{}),
				}},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DeployedContractAddress(tc.tx)
			require.ErrorIs(t, err, ErrInvalidTransaction)
			require.Empty(t, got)
		})
	}
}

func mustAny(t *testing.T, m proto.Message) *anypb.Any {
	t.Helper()
	a, err := anypb.New(m)
	require.NoError(t, err)
	return a
}
