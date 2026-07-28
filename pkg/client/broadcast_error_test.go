package client

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// A node routinely rejects a transaction with Result=false and an empty Message,
// leaving Code as the only diagnostic. The error text must never degrade to a
// bare "result error:", and the caller must be able to branch on the code
// without parsing strings.
func TestBroadcastTransactionAlwaysReportsCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ret      *api.Return
		wantCode api.ReturnResponseCode
		wantText string
	}{
		{
			name:     "result false with empty message",
			ret:      &api.Return{Result: false, Code: api.Return_DUP_TRANSACTION_ERROR},
			wantCode: api.Return_DUP_TRANSACTION_ERROR,
			wantText: "DUP_TRANSACTION_ERROR",
		},
		{
			name:     "result false with message",
			ret:      &api.Return{Result: false, Code: api.Return_SIGERROR, Message: []byte("bad signature")},
			wantCode: api.Return_SIGERROR,
			wantText: "SIGERROR",
		},
		{
			name:     "result false with bandwidth error",
			ret:      &api.Return{Result: false, Code: api.Return_BANDWITH_ERROR},
			wantCode: api.Return_BANDWITH_ERROR,
			wantText: "BANDWITH_ERROR",
		},
		{
			// Result=true but a non-SUCCESS code is the other rejection shape.
			name:     "non-success code with result true",
			ret:      &api.Return{Result: true, Code: api.Return_TAPOS_ERROR, Message: []byte("tapos")},
			wantCode: api.Return_TAPOS_ERROR,
			wantText: "TAPOS_ERROR",
		},
		{
			// The node can also report failure through Result=false alone,
			// leaving Code at its zero value.
			name:     "result false with zero code",
			ret:      &api.Return{Result: false, Message: []byte("rejected")},
			wantCode: api.Return_SUCCESS,
			wantText: "rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(&fakeTransport{
				broadcastTransaction: func(context.Context, *core.Transaction) (*api.Return, error) {
					return tt.ret, nil
				},
			})

			_, err := c.BroadcastTransaction(t.Context(), &core.Transaction{})
			require.Error(t, err)

			var broadcastErr *BroadcastError
			require.ErrorAs(t, err, &broadcastErr)
			require.Equal(t, tt.wantCode, broadcastErr.Code)
			require.Equal(t, string(tt.ret.GetMessage()), broadcastErr.Message)

			// The rendered text must carry the code even when Message is empty.
			require.ErrorContains(t, err, tt.wantText)
			require.NotEqual(t, "result error: ", err.Error())
		})
	}
}

func TestBroadcastTransactionSuccess(t *testing.T) {
	t.Parallel()

	c := newTestClient(&fakeTransport{
		broadcastTransaction: func(context.Context, *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	})

	res, err := c.BroadcastTransaction(t.Context(), &core.Transaction{})
	require.NoError(t, err)
	require.True(t, res.GetResult())
}

func TestBroadcastErrorMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *BroadcastError
		want string
	}{
		{
			name: "code only",
			err:  &BroadcastError{Code: api.Return_DUP_TRANSACTION_ERROR},
			want: "broadcast rejected: code=DUP_TRANSACTION_ERROR",
		},
		{
			name: "code and message",
			err:  &BroadcastError{Code: api.Return_SIGERROR, Message: "bad signature"},
			want: "broadcast rejected: code=SIGERROR: bad signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.err.Error())
		})
	}
}

// A wrapped BroadcastError must still be reachable, since callers see it through
// their own error chains.
func TestBroadcastErrorSurvivesWrapping(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("broadcast transaction: %w", &BroadcastError{Code: api.Return_TAPOS_ERROR})

	var broadcastErr *BroadcastError
	require.True(t, errors.As(wrapped, &broadcastErr))
	require.Equal(t, api.Return_TAPOS_ERROR, broadcastErr.Code)
}
