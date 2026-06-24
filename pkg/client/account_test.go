package client

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

const testAddr = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

func TestGetAccount(t *testing.T) {
	t.Run("empty address", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetAccount(context.Background(), "")
		require.ErrorIs(t, err, ErrEmptyAddress)
	})

	t.Run("invalid address", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetAccount(context.Background(), "not-base58!")
		require.Error(t, err)
	})

	t.Run("transport error propagates", func(t *testing.T) {
		boom := errors.New("rpc down")
		c := newTestClient(&fakeTransport{
			getAccount: func(context.Context, *core.Account) (*core.Account, error) { return nil, boom },
		})
		_, err := c.GetAccount(context.Background(), testAddr)
		require.ErrorIs(t, err, boom)
	})

	t.Run("address mismatch is not found", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getAccount: func(context.Context, *core.Account) (*core.Account, error) {
				return &core.Account{Address: []byte{0x41, 0x00}}, nil
			},
		})
		_, err := c.GetAccount(context.Background(), testAddr)
		require.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("success echoes account", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
				return &core.Account{Address: a.Address, Balance: 5_000_000}, nil
			},
		})
		acc, err := c.GetAccount(context.Background(), testAddr)
		require.NoError(t, err)
		require.Equal(t, int64(5_000_000), acc.GetBalance())
	})
}

// Regression: a (nil, nil) transport response must not panic on acc.Address.
func TestGetAccountNilResponseDoesNotPanic(t *testing.T) {
	c := newTestClient(&fakeTransport{
		getAccount: func(context.Context, *core.Account) (*core.Account, error) { return nil, nil },
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetAccount panicked on nil response: %v", r)
		}
	}()

	_, err := c.GetAccount(context.Background(), testAddr)
	require.Error(t, err, "nil account should be reported as an error, not a panic")
}

func TestGetAccountBalance(t *testing.T) {
	c := newTestClient(&fakeTransport{
		getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
			return &core.Account{Address: a.Address, Balance: 5_000_000}, nil
		},
	})
	bal, err := c.GetAccountBalance(context.Background(), testAddr)
	require.NoError(t, err)
	// 5_000_000 SUN = 5 TRX
	require.True(t, bal.Equal(decimal.NewFromInt(5)), "got %s", bal)
}

func TestIsAccountActivated(t *testing.T) {
	t.Run("activated", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getAccount: func(_ context.Context, a *core.Account) (*core.Account, error) {
				return &core.Account{Address: a.Address}, nil
			},
		})
		ok, err := c.IsAccountActivated(context.Background(), testAddr)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("not activated (address mismatch)", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getAccount: func(context.Context, *core.Account) (*core.Account, error) {
				return &core.Account{Address: []byte{0x41}}, nil
			},
		})
		ok, err := c.IsAccountActivated(context.Background(), testAddr)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("other error propagates", func(t *testing.T) {
		boom := errors.New("rpc down")
		c := newTestClient(&fakeTransport{
			getAccount: func(context.Context, *core.Account) (*core.Account, error) { return nil, boom },
		})
		_, err := c.IsAccountActivated(context.Background(), testAddr)
		require.ErrorIs(t, err, boom)
	})
}
