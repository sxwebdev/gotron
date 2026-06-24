package client

import (
	"context"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

const testHash = "abababababababababababababababababababababababababababababababab"

func TestGetTransactionByHash(t *testing.T) {
	t.Run("invalid hex", func(t *testing.T) {
		c := newTestClient(&fakeTransport{})
		_, err := c.GetTransactionByHash(context.Background(), "zz")
		require.Error(t, err)
	})

	t.Run("not found (empty tx)", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getTransactionById: func(context.Context, []byte) (*core.Transaction, error) { return nil, nil },
		})
		_, err := c.GetTransactionByHash(context.Background(), testHash)
		require.ErrorIs(t, err, ErrTransactionNotFound)
	})

	t.Run("found", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getTransactionById: func(context.Context, []byte) (*core.Transaction, error) {
				return &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}}}, nil
			},
		})
		tx, err := c.GetTransactionByHash(context.Background(), testHash)
		require.NoError(t, err)
		require.NotNil(t, tx)
	})
}

func TestGetTransactionInfoByHash(t *testing.T) {
	// Regression: a nil info response must not panic on txi.Id.
	t.Run("nil response does not panic", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getTransactionInfoById: func(context.Context, []byte) (*core.TransactionInfo, error) { return nil, nil },
		})
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panicked on nil info: %v", r)
			}
		}()
		_, err := c.GetTransactionInfoByHash(context.Background(), testHash)
		require.ErrorIs(t, err, ErrTransactionInfoNotFound)
	})

	t.Run("id mismatch", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getTransactionInfoById: func(context.Context, []byte) (*core.TransactionInfo, error) {
				return &core.TransactionInfo{Id: []byte{0x99}}, nil
			},
		})
		_, err := c.GetTransactionInfoByHash(context.Background(), testHash)
		require.ErrorIs(t, err, ErrTransactionInfoNotFound)
	})

	t.Run("found", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getTransactionInfoById: func(_ context.Context, id []byte) (*core.TransactionInfo, error) {
				return &core.TransactionInfo{Id: id}, nil
			},
		})
		txi, err := c.GetTransactionInfoByHash(context.Background(), testHash)
		require.NoError(t, err)
		require.NotNil(t, txi)
	})
}

func TestBroadcastTransaction(t *testing.T) {
	t.Run("result false", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			broadcastTransaction: func(context.Context, *core.Transaction) (*api.Return, error) {
				return &api.Return{Result: false, Message: []byte("rejected")}, nil
			},
		})
		_, err := c.BroadcastTransaction(context.Background(), &core.Transaction{})
		require.ErrorContains(t, err, "rejected")
	})

	t.Run("non-success code", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			broadcastTransaction: func(context.Context, *core.Transaction) (*api.Return, error) {
				return &api.Return{Result: true, Code: api.Return_TAPOS_ERROR, Message: []byte("tapos")}, nil
			},
		})
		_, err := c.BroadcastTransaction(context.Background(), &core.Transaction{})
		require.ErrorContains(t, err, "tapos")
	})

	t.Run("success", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			broadcastTransaction: func(context.Context, *core.Transaction) (*api.Return, error) {
				return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
			},
		})
		res, err := c.BroadcastTransaction(context.Background(), &core.Transaction{})
		require.NoError(t, err)
		require.True(t, res.GetResult())
	})
}

func TestSignTransaction(t *testing.T) {
	c := &Client{}

	t.Run("nil tx", func(t *testing.T) {
		require.Error(t, c.SignTransaction(nil, nil))
	})

	t.Run("appends signature", func(t *testing.T) {
		priv, err := ethcrypto.HexToECDSA(strings.Repeat("0", 63) + "1")
		require.NoError(t, err)
		tx := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}}}
		require.NoError(t, c.SignTransaction(tx, priv))
		require.Len(t, tx.Signature, 1)
		require.Len(t, tx.Signature[0], 65)
	})
}
