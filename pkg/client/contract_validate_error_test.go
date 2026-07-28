package client

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// A node that refuses to build a transaction is answering correctly about a
// wrong request. Before this was typed, the only way to tell it from a node
// problem was to match on the message - and the message differed by transport.
func TestCheckTransactionReportsAValidateError(t *testing.T) {
	err := checkTransaction(&api.TransactionExtention{
		Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
		Result: &api.Return{
			Code:    api.Return_CONTRACT_VALIDATE_ERROR,
			Message: []byte("frozenBalance must be positive"),
		},
	})

	var cve *ContractValidateError
	require.ErrorAs(t, err, &cve)
	require.Equal(t, api.Return_CONTRACT_VALIDATE_ERROR, cve.Code)
	require.Equal(t, "frozenBalance must be positive", cve.Message)
	require.ErrorContains(t, err, "frozenBalance must be positive")
	require.ErrorContains(t, err, "CONTRACT_VALIDATE_ERROR")
}

func TestCheckTransactionAcceptsWhatTheNodeBuilt(t *testing.T) {
	require.NoError(t, checkTransaction(&api.TransactionExtention{
		Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
		Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
	}))

	// An empty response is a different failure: the node produced nothing at
	// all, so there is no validation verdict to report.
	err := checkTransaction(&api.TransactionExtention{})
	require.ErrorIs(t, err, ErrInvalidTransaction)
	var cve *ContractValidateError
	require.False(t, errors.As(err, &cve))
}

func TestContractValidateErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		err  *ContractValidateError
		want string
	}{
		{"code and message", &ContractValidateError{Code: api.Return_SIGERROR, Message: "bad sig"}, "contract validate error: SIGERROR: bad sig"},
		// HTTP gives no code, so the message must still carry the reason
		// rather than degrading to a bare label.
		{"message only", &ContractValidateError{Message: "no unfreezeV2 list to cancel"}, "contract validate error: no unfreezeV2 list to cancel"},
		{"code only", &ContractValidateError{Code: api.Return_SIGERROR}, "contract validate error: SIGERROR"},
		{"neither", &ContractValidateError{}, "contract validate error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.err.Error())
		})
	}
}

// The point of the type: a refused request must not cost the node its health.
// The same request fails on every node, so evicting them would burn the tier
// and still fail.
func TestValidateErrorsDoNotMarkANodeUnhealthy(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"bare", &ContractValidateError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: "boom"}},
		{
			name: "wrapped in a TransportError, as HTTP returns it",
			err: &TransportError{
				Host: "https://node", Protocol: "http", Method: "/wallet/freezebalancev2",
				Err: &ContractValidateError{Message: "frozenBalance must be positive"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.False(t, isNetworkError(tc.err))
		})
	}
}

// The HTTP transport reports the same refusal through a different field, so it
// must produce the same type - otherwise which errors a caller can match on
// depends on the transport it happens to be using.
func TestHTTPValidateErrorsMatchTheGRPCType(t *testing.T) {
	t.Run("top-level Error field", func(t *testing.T) {
		const body = `{"Error":"class org.tron.core.exception.ContractValidateException : frozenBalance must be positive"}`

		tr, _ := newStubTransport(t, http.StatusOK, body)

		owner, err := tronutils.DecodeCheck(testAddr)
		require.NoError(t, err)

		_, err = tr.FreezeBalanceV2(t.Context(), &core.FreezeBalanceV2Contract{
			OwnerAddress: owner, FrozenBalance: 1, Resource: core.ResourceCode_ENERGY,
		})

		var cve *ContractValidateError
		require.ErrorAs(t, err, &cve)
		require.Contains(t, cve.Message, "frozenBalance must be positive")
		require.False(t, isNetworkError(err))
	})

	t.Run("nested result object", func(t *testing.T) {
		// The hex message is what java-tron sends; it must be decoded, not
		// handed to the caller as hex.
		const body = `{"result":{"code":"CONTRACT_VALIDATE_ERROR","message":"6163636f756e74206465732076616c6964617465206572726f72"}}`

		tr, _ := newStubTransport(t, http.StatusOK, body)

		owner, err := tronutils.DecodeCheck(testAddr)
		require.NoError(t, err)

		_, err = tr.TriggerContract(t.Context(), &core.TriggerSmartContract{
			OwnerAddress: owner, ContractAddress: owner,
		})

		var cve *ContractValidateError
		require.ErrorAs(t, err, &cve)
		require.Equal(t, api.Return_CONTRACT_VALIDATE_ERROR, cve.Code)
		require.Equal(t, "account des validate error", cve.Message)
		require.False(t, isNetworkError(err))
	})
}

// A refusal reaching the client layer through the health-aware transport must
// leave the node in the pool.
func TestValidateErrorLeavesTheNodeInThePool(t *testing.T) {
	c := newTestClient(&fakeTransport{
		freezeBalanceV2: func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
				Result: &api.Return{
					Code:    api.Return_CONTRACT_VALIDATE_ERROR,
					Message: []byte("frozenBalance must be positive"),
				},
			}, nil
		},
	})

	_, err := c.Stake(context.Background(), testAddr, ResourceTypeEnergy, SUN(1))

	var cve *ContractValidateError
	require.ErrorAs(t, err, &cve)
	require.False(t, isNetworkError(err), "a refused request must not count against the node")
}
