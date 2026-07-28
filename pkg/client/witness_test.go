package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestVoteWitnessesValidation(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		votes   []Vote
		wantErr error
	}{
		{"invalid owner", "bad!", []Vote{{testAddr2, 1}}, ErrInvalidAddress},
		{"nil votes", testAddr, nil, ErrInvalidParams},
		{"empty votes", testAddr, []Vote{}, ErrInvalidParams},
		{"invalid witness", testAddr, []Vote{{"bad!", 1}}, ErrInvalidAddress},
		{"invalid witness in second slot", testAddr, []Vote{{testAddr2, 1}, {"bad!", 1}}, ErrInvalidAddress},
		{"zero count", testAddr, []Vote{{testAddr2, 0}}, ErrInvalidAmount},
		{"negative count", testAddr, []Vote{{testAddr2, -3}}, ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			c := newTestClient(&fakeTransport{
				voteWitnessAccount: func(context.Context, *core.VoteWitnessContract) (*api.TransactionExtention, error) {
					calls++
					return okTx(), nil
				},
			})

			_, err := c.VoteWitnesses(t.Context(), tt.owner, tt.votes)
			require.ErrorIs(t, err, tt.wantErr)
			require.Zero(t, calls, "transport must not be called for invalid input")
		})
	}
}

func TestVoteWitnessesBuildsContract(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	wantFirst, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)
	wantSecond, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	var got *core.VoteWitnessContract
	c := newTestClient(&fakeTransport{
		voteWitnessAccount: func(_ context.Context, ct *core.VoteWitnessContract) (*api.TransactionExtention, error) {
			got = ct
			return okTx(), nil
		},
	})

	_, err = c.VoteWitnesses(t.Context(), testAddr, []Vote{
		{WitnessAddress: testAddr2, Count: 10},
		{WitnessAddress: testAddr, Count: 20},
	})
	require.NoError(t, err)

	require.Equal(t, wantOwner, got.GetOwnerAddress())
	require.Len(t, got.GetVotes(), 2)
	// Tron replaces the whole vote set at once, so input order must survive.
	require.Equal(t, wantFirst, got.GetVotes()[0].GetVoteAddress())
	require.EqualValues(t, 10, got.GetVotes()[0].GetVoteCount())
	require.Equal(t, wantSecond, got.GetVotes()[1].GetVoteAddress())
	require.EqualValues(t, 20, got.GetVotes()[1].GetVoteCount())
}

func TestClaimRewards(t *testing.T) {
	wantOwner, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	t.Run("invalid owner", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.ClaimRewards(t.Context(), "bad!")
		require.ErrorIs(t, err, ErrInvalidAddress)
	})

	t.Run("success", func(t *testing.T) {
		var got *core.WithdrawBalanceContract
		c := newTestClient(&fakeTransport{
			withdrawBalance: func(_ context.Context, ct *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
				got = ct
				return okTx(), nil
			},
		})

		_, err := c.ClaimRewards(t.Context(), testAddr)
		require.NoError(t, err)
		require.Equal(t, wantOwner, got.GetOwnerAddress())
	})

	t.Run("empty transaction rejected", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			withdrawBalance: func(context.Context, *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
				return &api.TransactionExtention{}, nil
			},
		})

		_, err := c.ClaimRewards(t.Context(), testAddr)
		require.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("node error surfaced", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			withdrawBalance: func(context.Context, *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
				return &api.TransactionExtention{
					Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
					Result: &api.Return{
						Code:    api.Return_CONTRACT_VALIDATE_ERROR,
						Message: []byte("witnessAccount does not have any reward"),
					},
				}, nil
			},
		})

		_, err := c.ClaimRewards(t.Context(), testAddr)
		require.ErrorContains(t, err, "does not have any reward")
	})
}

func TestGetUnclaimedReward(t *testing.T) {
	wantAddr, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	t.Run("value", func(t *testing.T) {
		var got []byte
		c := newTestClient(&fakeTransport{
			getRewardInfo: func(_ context.Context, address []byte) (*api.NumberMessage, error) {
				got = address
				return &api.NumberMessage{Num: 12_345}, nil
			},
		})

		reward, err := c.GetUnclaimedReward(t.Context(), testAddr)
		require.NoError(t, err)
		require.EqualValues(t, 12_345, reward)
		require.Equal(t, wantAddr, got)
	})

	t.Run("invalid address", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetUnclaimedReward(t.Context(), "bad!")
		require.Error(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		sentinel := errors.New("boom")
		c := newTestClient(&fakeTransport{
			getRewardInfo: func(context.Context, []byte) (*api.NumberMessage, error) { return nil, sentinel },
		})
		_, err := c.GetUnclaimedReward(t.Context(), testAddr)
		require.ErrorIs(t, err, sentinel)
	})
}

func TestGetWitnessBrokerage(t *testing.T) {
	wantAddr, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)

	var got []byte
	c := newTestClient(&fakeTransport{
		getBrokerageInfo: func(_ context.Context, address []byte) (*api.NumberMessage, error) {
			got = address
			return &api.NumberMessage{Num: 20}, nil
		},
	})

	brokerage, err := c.GetWitnessBrokerage(t.Context(), testAddr)
	require.NoError(t, err)
	require.EqualValues(t, 20, brokerage)
	require.Equal(t, wantAddr, got)

	_, err = c.GetWitnessBrokerage(t.Context(), "bad!")
	require.Error(t, err)
}

func TestListWitnesses(t *testing.T) {
	t.Run("passthrough", func(t *testing.T) {
		want := &api.WitnessList{Witnesses: []*core.Witness{{Url: "http://sr.example"}}}
		c := newTestClient(&fakeTransport{
			listWitnesses: func(context.Context) (*api.WitnessList, error) { return want, nil },
		})

		got, err := c.ListWitnesses(t.Context())
		require.NoError(t, err)
		require.Same(t, want, got)
	})

	t.Run("transport error", func(t *testing.T) {
		sentinel := errors.New("boom")
		c := newTestClient(&fakeTransport{
			listWitnesses: func(context.Context) (*api.WitnessList, error) { return nil, sentinel },
		})

		_, err := c.ListWitnesses(t.Context())
		require.ErrorIs(t, err, sentinel)
	})
}
