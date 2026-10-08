package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// java-tron's own texts (DelegateResourceActuator.validate, GreatVoyage-v4.7.0
// to v4.8.2.3). The wording changed in v4.7.3; nodes of both kinds run.
const (
	energyStakeShort       = "delegateBalance must be less than or equal to available FreezeEnergyV2 balance"
	bandwidthStakeShort    = "delegateBalance must be less than or equal to available FreezeBandwidthV2 balance"
	oldEnergyStakeShort    = "delegateBalance must be less than available FreezeEnergyV2 balance"
	oldBandwidthStakeShort = "delegateBalance must be less than available FreezeBandwidthV2 balance"
	belowMinimum           = "delegateBalance must be greater than or equal to 1 TRX"
	oldBelowMinimum        = "delegateBalance must be more than 1TRX"
)

// How the node hands the actuator's message on: gRPC puts Wallet's
// CONTRACT_VALIDATE_ERROR in front, HTTP the exception class (Util.printErrorMsg).
var refusalForms = []struct {
	name   string
	format string
}{
	{"bare", "%s"},
	{"grpc", "Contract validate error : %s"},
	{"http", "class org.tron.core.exception.ContractValidateException : %s"},
	{"trailing newline", "Contract validate error : %s\n"},
}

func TestContractValidateErrorIsADelegateVerdict(t *testing.T) {
	t.Parallel()
	cases := []struct {
		verdict string
		want    error
	}{
		{energyStakeShort, ErrDelegateStakeShort},
		{bandwidthStakeShort, ErrDelegateStakeShort},
		{oldEnergyStakeShort, ErrDelegateStakeShort},
		{oldBandwidthStakeShort, ErrDelegateStakeShort},
		{belowMinimum, ErrDelegateBelowMinimum},
		{oldBelowMinimum, ErrDelegateBelowMinimum},
	}
	for _, tc := range cases {
		for _, form := range refusalForms {
			t.Run(form.name+"/"+tc.verdict, func(t *testing.T) {
				t.Parallel()
				err := &ContractValidateError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: fmt.Sprintf(form.format, tc.verdict)}
				require.ErrorIs(t, err, tc.want)
				// One verdict, one sentinel.
				for _, other := range []error{ErrDelegateStakeShort, ErrDelegateBelowMinimum} {
					if other != tc.want {
						require.NotErrorIs(t, err, other)
					}
				}
				// What callers matched before still matches.
				require.ErrorIs(t, err, ErrInvalidTransaction)
			})
		}
	}
}

// Refusals that look alike must not read as either verdict: other actuators'
// texts about a balance, and a delegateBalance verdict that is not the whole
// tail of the message.
func TestContractValidateErrorNearMisses(t *testing.T) {
	t.Parallel()
	for _, msg := range []string{
		"",
		"unDelegateBalance must be more than 0 TRX",
		"insufficient delegateFrozenBalance(Energy), request=2000000, unlock_balance=1000000",
		"insufficient delegatedFrozenBalance(BANDWIDTH), request=2000000, unlock_balance=1000000",
		"frozenBalance must be greater than or equal to 1 TRX",
		"frozenBalance must be less than or equal to accountBalance",
		"ResourceCode error, valid ResourceCode[BANDWIDTH、ENERGY]",
		"receiverAddress must not be the same as ownerAddress",
		// The made-up wording an earlier test used: no release says it.
		"delegateBalance must be greater than 1 TRX",
		"delegateBalance must be less than or equal to available",
		"delegateBalance must be less than or equal to available FreezeTronPowerV2 balance",
		"Contract validate error : " + energyStakeShort + ", retry later",
		"Contract validate error : X" + energyStakeShort,
		"Contract validate error : X" + belowMinimum,
	} {
		t.Run(msg, func(t *testing.T) {
			t.Parallel()
			err := &ContractValidateError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: msg}
			require.NotErrorIs(t, err, ErrDelegateStakeShort)
			require.NotErrorIs(t, err, ErrDelegateBelowMinimum)
		})
	}
}

// uncomparableError is an error errors.Is cannot compare with ==.
type uncomparableError []string

func (e uncomparableError) Error() string { return strings.Join(e, " ") }

// Only the node's validation verdict is matched: the same text in any other
// error is not it, and a target the method does not know answers false.
func TestDelegateVerdictsMatchOnlyARefusal(t *testing.T) {
	t.Parallel()
	require.NotErrorIs(t, errors.New(energyStakeShort), ErrDelegateStakeShort)
	require.NotErrorIs(t, &BroadcastError{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: energyStakeShort}, ErrDelegateStakeShort)
	require.NotErrorIs(t, &TransportError{Protocol: "http", Err: errors.New(belowMinimum)}, ErrDelegateBelowMinimum)

	cve := &ContractValidateError{Message: energyStakeShort}
	require.NotErrorIs(t, cve, ErrInvalidAmount)
	require.NotErrorIs(t, cve, ErrNodeRefusedRequest)
	require.NotPanics(t, func() { require.NotErrorIs(t, cve, uncomparableError{"not", "a", "verdict"}) })

	// Wrapped as callers and transports wrap it, it still matches.
	wrapped := fmt.Errorf("build delegate: %w", &TransportError{Protocol: "http", Err: cve})
	require.ErrorIs(t, wrapped, ErrDelegateStakeShort)
	got, ok := errors.AsType[*ContractValidateError](wrapped)
	require.True(t, ok)
	require.Same(t, cve, got)
}

// The verdict arrives at DelegateResource's caller from either transport.
func TestDelegateResourceReportsStakeShort(t *testing.T) {
	t.Parallel()
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		c := newTestClient(&fakeTransport{
			delegateResource: func(context.Context, *core.DelegateResourceContract) (*api.TransactionExtention, error) {
				return &api.TransactionExtention{Result: &api.Return{
					Code:    api.Return_CONTRACT_VALIDATE_ERROR,
					Message: []byte("Contract validate error : " + energyStakeShort),
				}}, nil
			},
		})
		_, err := c.DelegateResource(t.Context(), testAddr, testAddr2, ResourceTypeEnergy, 5*MinDelegateBalance, false, 0)
		require.ErrorIs(t, err, ErrDelegateStakeShort)
		require.NotErrorIs(t, err, ErrDelegateBelowMinimum)
	})
	t.Run("http", func(t *testing.T) {
		t.Parallel()
		tr, _ := newStubTransportAtPath(t, "/wallet/delegateresource", http.StatusOK,
			`{"Error":"class org.tron.core.exception.ContractValidateException : `+bandwidthStakeShort+`"}`)
		c := &Client{transport: tr}
		_, err := c.DelegateResource(t.Context(), testAddr, testAddr2, ResourceTypeBandwidth, 5*MinDelegateBalance, false, 0)
		require.ErrorIs(t, err, ErrDelegateStakeShort)
		require.NotErrorIs(t, err, ErrDelegateBelowMinimum)
	})
}

// Under the minimum no node builds a delegation, so the client does not ask
// one, and answers as a node would: the same type, code and verdict.
func TestDelegateResourceRefusesUnderTheMinimum(t *testing.T) {
	t.Parallel()
	var sent []int64
	c := newTestClient(&fakeTransport{
		delegateResource: func(_ context.Context, ct *core.DelegateResourceContract) (*api.TransactionExtention, error) {
			sent = append(sent, ct.GetBalance())
			return okTx(), nil
		},
	})

	for _, balance := range []SUN{1, MinDelegateBalance - 1} {
		_, err := c.DelegateResource(t.Context(), testAddr, testAddr2, ResourceTypeEnergy, balance, false, 0)
		require.ErrorIs(t, err, ErrDelegateBelowMinimum)
		require.ErrorIs(t, err, ErrInvalidTransaction)
		require.NotErrorIs(t, err, ErrDelegateStakeShort)
		cve, ok := errors.AsType[*ContractValidateError](err)
		require.True(t, ok)
		require.Equal(t, api.Return_CONTRACT_VALIDATE_ERROR, cve.Code)
		require.Equal(t, belowMinimum, cve.Message)
	}
	require.Empty(t, sent, "a delegation under the minimum reached the node")

	_, err := c.DelegateResource(t.Context(), testAddr, testAddr2, ResourceTypeEnergy, MinDelegateBalance, false, 0)
	require.NoError(t, err)
	require.Equal(t, []int64{int64(MinDelegateBalance)}, sent)
}
