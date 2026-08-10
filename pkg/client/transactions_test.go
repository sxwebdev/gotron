package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
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

// signingKeys spans the whole valid scalar range, because the reordering of the
// compact signature and the mod-N handling are exactly what a mid-range key
// would not exercise.
var signingKeys = []struct {
	name string
	hex  string
}{
	{"smallest valid scalar", strings.Repeat("0", 63) + "1"},
	{"arbitrary key", "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"},
	{"largest valid scalar", "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd036413f"},
}

func TestSignTransactionRawMatchesSignTransaction(t *testing.T) {
	c := &Client{}

	for _, k := range signingKeys {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()

			raw, err := hex.DecodeString(k.hex)
			require.NoError(t, err)
			priv, err := ethcrypto.HexToECDSA(k.hex)
			require.NoError(t, err)

			// Both signers must produce the same bytes for the same input, or
			// SignTransactionRaw is not a drop-in replacement. Deterministic
			// because both nonces come from RFC 6979.
			want := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}, Expiration: 42}}
			got := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}, Expiration: 42}}
			require.NoError(t, c.SignTransaction(want, priv))
			require.NoError(t, c.SignTransactionRaw(got, raw))
			require.Equal(t, want.GetSignature(), got.GetSignature())
		})
	}
}

func TestSignTransactionRawRecoversSignerKey(t *testing.T) {
	c := &Client{}

	for _, k := range signingKeys {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()

			raw, err := hex.DecodeString(k.hex)
			require.NoError(t, err)
			priv, err := ethcrypto.HexToECDSA(k.hex)
			require.NoError(t, err)

			tx := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x07}}}
			require.NoError(t, c.SignTransactionRaw(tx, raw))
			require.Len(t, tx.GetSignature(), 1)
			require.Len(t, tx.GetSignature()[0], 65)

			// Recovery is independent of how the signature was produced, so it
			// catches a misplaced recovery id that a comparison against
			// SignTransaction alone would not.
			rawData, err := proto.Marshal(tx.GetRawData())
			require.NoError(t, err)
			hash := sha256.Sum256(rawData)
			pub, err := ethcrypto.SigToPub(hash[:], tx.GetSignature()[0])
			require.NoError(t, err)
			require.Equal(t, ethcrypto.PubkeyToAddress(priv.PublicKey), ethcrypto.PubkeyToAddress(*pub))
		})
	}
}

func TestSignTransactionRawKeepsCallerKeyIntact(t *testing.T) {
	c := &Client{}

	raw, err := hex.DecodeString(signingKeys[1].hex)
	require.NoError(t, err)
	before := slices.Clone(raw)

	tx := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}}}
	require.NoError(t, c.SignTransactionRaw(tx, raw))

	// The key belongs to the caller, who has to be able to wipe it afterwards
	// (and may reuse it for a second signature in between).
	require.Equal(t, before, raw)

	require.NoError(t, c.SignTransactionRaw(tx, raw))
	require.Len(t, tx.GetSignature(), 2)
	require.Equal(t, tx.GetSignature()[0], tx.GetSignature()[1])
}

func TestSignTransactionRawRejects(t *testing.T) {
	c := &Client{}

	valid, err := hex.DecodeString(signingKeys[1].hex)
	require.NoError(t, err)

	// curveOrder is N: the first scalar that is out of range.
	curveOrder, err := hex.DecodeString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141")
	require.NoError(t, err)

	tests := []struct {
		name    string
		tx      *core.Transaction
		key     []byte
		wantErr error
	}{
		{name: "nil tx", tx: nil, key: valid},
		{name: "nil key", tx: &core.Transaction{}, key: nil, wantErr: ErrInvalidPrivateKey},
		{name: "empty key", tx: &core.Transaction{}, key: []byte{}, wantErr: ErrInvalidPrivateKey},
		{name: "short key", tx: &core.Transaction{}, key: valid[:31], wantErr: ErrInvalidPrivateKey},
		// A 33-byte key must not sign with its first 32 bytes: SetByteSlice
		// would have truncated it to those and accepted it silently.
		{name: "long key", tx: &core.Transaction{}, key: append([]byte{0x01}, valid...), wantErr: ErrInvalidPrivateKey},
		{name: "zero key", tx: &core.Transaction{}, key: make([]byte, 32), wantErr: ErrInvalidPrivateKey},
		{name: "key equal to the curve order", tx: &core.Transaction{}, key: curveOrder, wantErr: ErrInvalidPrivateKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := c.SignTransactionRaw(tt.tx, tt.key)
			require.Error(t, err)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			if tt.tx != nil {
				require.Empty(t, tt.tx.GetSignature())
			}
		})
	}
}

func TestValidatePrivateKeyRaw(t *testing.T) {
	valid, err := hex.DecodeString(signingKeys[1].hex)
	require.NoError(t, err)
	require.NoError(t, ValidatePrivateKeyRaw(valid))

	// The validator must not consume or alter what it inspects: callers keep
	// signing with the same slice afterwards.
	before := slices.Clone(valid)
	require.NoError(t, ValidatePrivateKeyRaw(valid))
	require.Equal(t, before, valid)

	require.ErrorIs(t, ValidatePrivateKeyRaw(nil), ErrInvalidPrivateKey)
	require.ErrorIs(t, ValidatePrivateKeyRaw(make([]byte, 32)), ErrInvalidPrivateKey)
}
