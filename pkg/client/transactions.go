package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	dcrecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

// recoveryIDOffset is the position of the recovery id in a 65-byte
// [r ‖ s ‖ v] Tron signature.
const recoveryIDOffset = 64

// GetTransactionByHash returns transaction details by hash
func (c *Client) GetTransactionByHash(ctx context.Context, hash string) (*core.Transaction, error) {
	hashBytes, err := tronutils.FromHex(hash)
	if err != nil {
		return nil, fmt.Errorf("get transaction by hash error: %v", err)
	}

	tx, err := c.transport.GetTransactionById(ctx, hashBytes)
	if err != nil {
		return nil, err
	}
	if size := proto.Size(tx); size > 0 {
		return tx, nil
	}
	return nil, ErrTransactionNotFound
}

// GetTransactionInfoByHash returns transaction receipt by hash
func (c *Client) GetTransactionInfoByHash(ctx context.Context, hash string) (*core.TransactionInfo, error) {
	hashBytes, err := tronutils.FromHex(hash)
	if err != nil {
		return nil, fmt.Errorf("get transaction by hash error: %v", err)
	}

	txi, err := c.transport.GetTransactionInfoById(ctx, hashBytes)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(txi.GetId(), hashBytes) {
		return txi, nil
	}
	return nil, ErrTransactionInfoNotFound
}

func (c *Client) GetTransactionExtensionByHash(ctx context.Context, hash string) (*api.TransactionExtention, *core.TransactionInfo, error) {
	// Get transaction info
	txi, err := c.GetTransactionInfoByHash(ctx, hash)
	if err != nil {
		return nil, nil, err
	}

	// get block by height
	block, err := c.GetBlockByHeight(ctx, uint64(txi.GetBlockNumber()))
	if err != nil {
		return nil, nil, err
	}

	// find transaction in block
	var tx *api.TransactionExtention
	for _, item := range block.GetTransactions() {
		if bytes.Equal(item.GetTxid(), txi.GetId()) {
			tx = item
			break
		}
	}

	if tx == nil {
		return nil, nil, fmt.Errorf("can not find tx %s in block %d", hash, txi.GetBlockNumber())
	}

	return tx, txi, nil
}

// BroadcastTransaction broadcasts a signed transaction to the network
func (c *Client) BroadcastTransaction(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	result, err := c.transport.BroadcastTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}
	// Both rejection shapes carry the same diagnostic value, so they produce the
	// same typed error: a node may signal failure through Result=false, through
	// a non-SUCCESS Code, or through both.
	if !result.GetResult() || result.GetCode() != api.Return_SUCCESS {
		return result, &BroadcastError{
			Code:    result.GetCode(),
			Message: string(result.GetMessage()),
		}
	}
	return result, nil
}

// SignTransaction signs a raw transaction with the given private key
func (c *Client) SignTransaction(tx *core.Transaction, privateKey *ecdsa.PrivateKey) error {
	if tx == nil {
		return fmt.Errorf("empty tron tx")
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	if _, err := h256h.Write(rawData); err != nil {
		return err
	}

	signature, err := crypto.Sign(h256h.Sum(nil), privateKey)
	if err != nil {
		return err
	}

	tx.Signature = append(tx.Signature, signature)

	return nil
}

// ValidatePrivateKeyRaw reports whether privateKey is usable for signing: it
// must be exactly 32 bytes and a scalar in [1, N-1]. It keeps no copy of the
// key, so a caller can use it to check stored key material without deriving
// anything from it.
//
// The length check is not pedantry: SetByteSlice truncates a longer slice to
// its first 32 bytes and reduces mod N rather than rejecting, so a wrong-sized
// or out-of-range key would otherwise sign as a silently different one.
func ValidatePrivateKeyRaw(privateKey []byte) error {
	if len(privateKey) != secp256k1.PrivKeyBytesLen {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidPrivateKey, secp256k1.PrivKeyBytesLen, len(privateKey))
	}

	var scalar secp256k1.ModNScalar
	defer scalar.Zero()
	if overflow := scalar.SetByteSlice(privateKey); overflow || scalar.IsZero() {
		return fmt.Errorf("%w: key is not a scalar in [1, N-1]", ErrInvalidPrivateKey)
	}
	return nil
}

// AddressFromPrivateKeyRaw derives the base58 Tron address for a raw 32-byte
// secp256k1 private key without creating an ecdsa.PrivateKey. The temporary
// scalar is wiped before return; the caller retains ownership of privateKey.
func AddressFromPrivateKeyRaw(privateKey []byte) (string, error) {
	if err := ValidatePrivateKeyRaw(privateKey); err != nil {
		return "", err
	}

	var priv secp256k1.PrivateKey
	priv.Key.SetByteSlice(privateKey)
	defer priv.Zero()

	pub := priv.PubKey().SerializeUncompressed()
	digest := tronutils.Keccak256(pub[1:])

	return tronutils.EncodeCheck(append([]byte{tronutils.TronBytePrefix}, digest[len(digest)-20:]...)), nil
}

// SignTransactionRaw signs a raw transaction with a raw 32-byte secp256k1
// private key. It produces the same signature as SignTransaction and exists for
// callers to whom the key's lifetime in memory matters.
//
// SignTransaction takes an *ecdsa.PrivateKey, which carries the secret in a
// big.Int whose words cannot be zeroed through the standard library, and
// go-ethereum's crypto.Sign then copies it out again with prv.D.Bytes(). Both
// copies outlive the call and are erased only when the garbage collector
// happens to reuse their memory. Here the key exists in the caller's slice and
// in one secp256k1.PrivateKey that is wiped before returning; nothing else is
// derived from it. The caller owns privateKey and should clear() it once the
// key is no longer needed.
func (c *Client) SignTransactionRaw(tx *core.Transaction, privateKey []byte) error {
	if tx == nil {
		return fmt.Errorf("empty tron tx")
	}
	if err := ValidatePrivateKeyRaw(privateKey); err != nil {
		return err
	}

	var priv secp256k1.PrivateKey
	priv.Key.SetByteSlice(privateKey) // range-checked by ValidatePrivateKeyRaw
	defer priv.Zero()

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return err
	}
	hash := sha256.Sum256(rawData)

	// SignCompact returns [v ‖ r ‖ s] with v offset by 27; Tron expects
	// [r ‖ s ‖ v] with v in {0, 1}, which is what crypto.Sign builds too.
	sig := dcrecdsa.SignCompact(&priv, hash[:], false)
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[recoveryIDOffset] = v

	tx.Signature = append(tx.Signature, sig)

	return nil
}
