package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// The node's own texts, as GreatVoyage-v4.8.2.3 answered them on the local
// network (tests/local_build_refusal_test.go asks for each), except the last
// contract verdict, which is VMActuator's for a call that reaches the VM.
const (
	accountExists      = "Account has existed"
	createFeeShort     = "Validate CreateAccountActuator error, insufficient fee."
	ownerMissing       = "Account[41f594597ec0169e41fa7b290cc4241d2aa990f3a2] not exists"
	constantNoContract = "Smart contract is not exist."
	triggerNoContract  = "No contract or not a valid smart contract"
	vmNoContract       = "No contract or not a smart contract"
	reverted           = "REVERT opcode executed"
	// What tron-rpc.publicnode.com answers an estimate.
	estimateOff = "this node does not support estimate energy"
)

var buildVerdicts = []error{ErrAccountExists, ErrCreateAccountFeeShort, ErrContractNotExist, ErrEstimateEnergyUnsupported, ErrDelegateStakeShort, ErrDelegateBelowMinimum}

func TestContractValidateErrorIsABuildVerdict(t *testing.T) {
	t.Parallel()
	cases := []struct {
		verdict string
		want    error
	}{
		{accountExists, ErrAccountExists},
		{createFeeShort, ErrCreateAccountFeeShort},
		{constantNoContract, ErrContractNotExist},
		{triggerNoContract, ErrContractNotExist},
		{vmNoContract, ErrContractNotExist},
		{estimateOff, ErrEstimateEnergyUnsupported},
	}
	for _, tc := range cases {
		for _, form := range refusalForms {
			t.Run(form.name+"/"+tc.verdict, func(t *testing.T) {
				t.Parallel()
				err := &ContractValidateError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: fmt.Sprintf(form.format, tc.verdict)}
				require.ErrorIs(t, err, tc.want)
				for _, other := range buildVerdicts {
					if other != tc.want {
						require.NotErrorIs(t, err, other)
					}
				}
				require.ErrorIs(t, err, ErrInvalidTransaction)
			})
		}
	}
}

// Refusals that look alike must not read as a verdict.
func TestBuildVerdictNearMisses(t *testing.T) {
	t.Parallel()
	for _, msg := range []string{
		"",
		ownerMissing,
		"Contract validate error : " + ownerMissing,
		"Account has existed already",
		"Validate CreateAccountActuator error, insufficient fee",
		"Validate TransferContract error, balance is not sufficient.",
		"Smart contract is not exist.!",
		"No contract or not a valid smart contract here",
		"Contract validate error : XNo contract or not a smart contract",
		reverted,
	} {
		t.Run(msg, func(t *testing.T) {
			t.Parallel()
			err := &ContractValidateError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: msg}
			for _, v := range buildVerdicts {
				require.NotErrorIs(t, err, v)
			}
			call := &ContractCallError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: msg}
			require.NotErrorIs(t, call, ErrContractNotExist)
		})
	}
}

// CreateAccount's refusal used to come back as bare text over gRPC: nothing
// told a node that would not build the account from one that did not answer.
func TestCreateAccountRefusalIsTyped(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		verdict string
		want    error
	}{
		{createFeeShort, ErrCreateAccountFeeShort},
		{accountExists, ErrAccountExists},
		{ownerMissing, nil},
	} {
		t.Run("grpc/"+tc.verdict, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(&fakeTransport{
				createAccount: func(context.Context, *core.AccountCreateContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{Result: &api.Return{
						Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("Contract validate error : " + tc.verdict),
					}}, nil
				},
			})
			_, err := c.CreateAccount(t.Context(), testAddr, testAddr2, core.AccountType_Normal)
			requireRefusal(t, err, tc.want)
		})
		t.Run("http/"+tc.verdict, func(t *testing.T) {
			t.Parallel()
			tr, _ := newStubTransportAtPath(t, "/wallet/createaccount", http.StatusOK,
				`{"Error":"class org.tron.core.exception.ContractValidateException : `+tc.verdict+`"}`)
			c := &Client{transport: tr}
			_, err := c.CreateAccount(t.Context(), testAddr, testAddr2, core.AccountType_Normal)
			requireRefusal(t, err, tc.want)
		})
	}
}

// requireRefusal: err is a ContractValidateError, and the verdict want (none
// when nil) and no other.
func requireRefusal(t *testing.T, err error, want error) {
	t.Helper()
	_, ok := errors.AsType[*ContractValidateError](err)
	require.True(t, ok, "not a refusal: %v", err)
	require.ErrorIs(t, err, ErrInvalidTransaction)
	for _, v := range buildVerdicts {
		if v == want {
			require.ErrorIs(t, err, v)
		} else {
			require.NotErrorIs(t, err, v)
		}
	}
}

// The other builders that read the result code themselves return the same
// type.
func TestBuildersReturnTheRefusalType(t *testing.T) {
	t.Parallel()
	refused := func() (*api.TransactionExtention, error) {
		return &api.TransactionExtention{Result: &api.Return{
			Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("Contract validate error : Validate TransferContract error, balance is not sufficient."),
		}}, nil
	}
	c := newTestClient(&fakeTransport{
		createTransaction: func(context.Context, *core.TransferContract) (*api.TransactionExtention, error) { return refused() },
		updateEnergyLimit: func(context.Context, *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
			return refused()
		},
		updateSetting: func(context.Context, *core.UpdateSettingContract) (*api.TransactionExtention, error) {
			return refused()
		},
	})
	_, err := c.CreateTransferTransaction(t.Context(), testAddr, testAddr2, 1)
	requireRefusal(t, err, nil)
	_, err = c.UpdateEnergyLimitContract(t.Context(), testAddr, testAddr2, 1)
	requireRefusal(t, err, nil)
	_, err = c.UpdateSettingContract(t.Context(), testAddr, testAddr2, 1)
	requireRefusal(t, err, nil)
}

// A constant call to an address without a contract matches
// ErrContractNotExist, and keeps matching ErrContractCallFailed as before; a
// revert matches only the latter. An estimate the node would not run (no
// contract, the API off) is a refusal of the request, not a failed call. The
// code the node answered is kept.
func TestContractCallErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		code         api.ReturnResponseCode
		msg          string
		wantNotExist bool
	}{
		{"no contract, grpc", api.Return_CONTRACT_VALIDATE_ERROR, "Contract validate error : " + constantNoContract, true},
		{"no contract, http", api.Return_CONTRACT_VALIDATE_ERROR, constantNoContract, true},
		{"estimate API off", api.Return_CONTRACT_VALIDATE_ERROR, "Contract validate error : " + estimateOff, false},
		{"reverted", api.Return_SUCCESS, reverted, false},
		{"reverted estimate", api.Return_CONTRACT_EXE_ERROR, reverted, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(&fakeTransport{
				triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
					return &api.TransactionExtention{Result: &api.Return{Code: tc.code, Message: []byte(tc.msg)}}, nil
				},
				estimateEnergy: func(context.Context, *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
					return &api.EstimateEnergyMessage{Result: &api.Return{Code: tc.code, Message: []byte(tc.msg)}}, nil
				},
			})

			_, err := c.TriggerConstantContractCustom(t.Context(), testAddr, testAddr2, "name()", "")
			require.ErrorIs(t, err, ErrContractCallFailed)
			require.Equal(t, tc.wantNotExist, errors.Is(err, ErrContractNotExist))
			require.NotErrorIs(t, err, ErrInvalidTransaction)
			cce, ok := errors.AsType[*ContractCallError](err)
			require.True(t, ok)
			require.Equal(t, tc.code, cce.Code)
			require.Equal(t, tc.msg, cce.Message)

			_, err = c.EstimateEnergy(t.Context(), testAddr, testAddr2, "name()", "", 0, "", 0)
			switch tc.code {
			case api.Return_SUCCESS:
				require.NoError(t, err) // an estimate that answers SUCCESS succeeded
			case api.Return_CONTRACT_VALIDATE_ERROR:
				require.NotErrorIs(t, err, ErrContractCallFailed)
				require.Equal(t, tc.wantNotExist, errors.Is(err, ErrContractNotExist))
				require.Equal(t, !tc.wantNotExist, errors.Is(err, ErrEstimateEnergyUnsupported))
				cve, ok := errors.AsType[*ContractValidateError](err)
				require.True(t, ok)
				require.Equal(t, tc.code, cve.Code)
			default:
				require.ErrorIs(t, err, ErrContractCallFailed)
				require.NotErrorIs(t, err, ErrContractNotExist)
				cce, ok := errors.AsType[*ContractCallError](err)
				require.True(t, ok)
				require.Equal(t, tc.code, cce.Code)
			}
		})
	}
	require.Equal(t, "contract call failed: CONTRACT_VALIDATE_ERROR", (&ContractCallError{Code: api.Return_CONTRACT_VALIDATE_ERROR}).Error())
	require.Equal(t, "contract call failed: "+reverted, (&ContractCallError{Message: reverted}).Error())
	require.NotPanics(t, func() { require.NotErrorIs(t, &ContractCallError{}, uncomparableError{"x"}) })
}

// A call to build to an address without a contract is a refusal of the
// request, from either transport.
func TestTriggerContractToNoContract(t *testing.T) {
	t.Parallel()
	c := newTestClient(&fakeTransport{
		triggerContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: &api.Return{
				Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("Contract validate error : " + triggerNoContract),
			}}, nil
		},
	})
	_, err := c.TriggerContract(t.Context(), testAddr, testAddr2, "name()", "", 0, 0, "", 0)
	require.ErrorIs(t, err, ErrContractNotExist)
	require.NotErrorIs(t, err, ErrContractCallFailed)

	tr, _ := newStubTransportAtPath(t, "/wallet/triggersmartcontract", http.StatusOK,
		`{"result":{"code":"CONTRACT_VALIDATE_ERROR","message":"`+fmt.Sprintf("%x", triggerNoContract)+`"}}`)
	_, err = (&Client{transport: tr}).TriggerContract(t.Context(), testAddr, testAddr2, "name()", "", 0, 0, "", 0)
	require.ErrorIs(t, err, ErrContractNotExist)
}

// An estimate of an activation to an account that exists is no account: the
// node's verdict is matched, not its text.
func TestEstimateActivationOfAnExistingAccount(t *testing.T) {
	t.Parallel()
	c := newTestClient(&fakeTransport{
		getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
			// The caller exists; the receiver, as the node first sees it, not.
			if caller, _ := tronutils.DecodeCheck(testAddr); bytes.Equal(a.GetAddress(), caller) {
				return &core.Account{Address: a.GetAddress()}, nil
			}
			return &core.Account{}, nil
		},
		getChainParameters: func(context.Context) (*core.ChainParameters, error) {
			// A fee the estimate would count, were the account not there.
			return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				{Key: "getCreateNewAccountFeeInSystemContract", Value: 1_000_000},
				{Key: "getCreateAccountFee", Value: 100_000},
			}}, nil
		},
		getAccountResource: func(context.Context, *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{}, nil
		},
		getAccountResourceMsg: func(context.Context, *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{}, nil
		},
		createAccount: func(context.Context, *core.AccountCreateContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: &api.Return{
				Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("Contract validate error : " + accountExists),
			}}, nil
		},
	})
	res, err := c.EstimateSystemContractActivation(t.Context(), testAddr, testAddr2)
	require.NoError(t, err)
	require.Zero(t, res.Fee)
}
