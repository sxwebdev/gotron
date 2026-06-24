package crypto_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/sxwebdev/gotron/pkg/crypto"
)

const testPrivHex = "ef80f4f95fe356c6405a0ff976ea8e7ee85caf6a9fd9f4a073ddf46b149733ee"

func TestSignData(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		isErr   error // errors.Is target; nil = any error
		wantErr bool
	}{
		{"empty key", "", crypto.ErrInvalidPrivateKey, true},
		{"invalid hex", "zz", nil, true},
		{"valid hex but wrong length", "abcd", nil, true},
		{"zero key fails ToECDSA", strings.Repeat("0", 64), nil, true},
		{"valid", testPrivHex, nil, false},
		{"valid with 0x prefix", "0x" + testPrivHex, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, err := crypto.SignData([]byte("message"), tt.key)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.isErr != nil && !errors.Is(err, tt.isErr) {
					t.Errorf("error = %v, want errors.Is %v", err, tt.isErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sig) != 65 {
				t.Errorf("signature length = %d, want 65", len(sig))
			}
		})
	}
}

func TestSignRecoverVerifyRoundTrip(t *testing.T) {
	data := []byte("hello tron")

	sig, err := crypto.SignData(data, testPrivHex)
	if err != nil {
		t.Fatalf("SignData() error = %v", err)
	}

	hash := crypto.HashKeccak256(data)

	pub, err := crypto.RecoverPublicKey(hash, sig)
	if err != nil {
		t.Fatalf("RecoverPublicKey() error = %v", err)
	}

	// Recovered key must equal the key derived from the signing private key.
	priv, err := ethcrypto.HexToECDSA(testPrivHex)
	if err != nil {
		t.Fatalf("HexToECDSA() error = %v", err)
	}
	if want, got := ethcrypto.FromECDSAPub(&priv.PublicKey), ethcrypto.FromECDSAPub(pub); !bytes.Equal(want, got) {
		t.Fatalf("recovered pubkey mismatch:\nwant %x\ngot  %x", want, got)
	}

	if !crypto.VerifySignature(pub, hash, sig) {
		t.Error("VerifySignature() = false, want true for a valid signature")
	}

	// Tampering with the signature must break verification.
	bad := bytes.Clone(sig)
	bad[0] ^= 0xff
	if crypto.VerifySignature(pub, hash, bad) {
		t.Error("VerifySignature() = true for a tampered signature, want false")
	}
}

func TestVerifySignatureWrongLength(t *testing.T) {
	priv, err := ethcrypto.HexToECDSA(testPrivHex)
	if err != nil {
		t.Fatalf("HexToECDSA() error = %v", err)
	}
	if crypto.VerifySignature(&priv.PublicKey, make([]byte, 32), make([]byte, 10)) {
		t.Error("VerifySignature() = true for a too-short signature, want false")
	}
}

func TestRecoverPublicKey(t *testing.T) {
	t.Run("wrong length", func(t *testing.T) {
		_, err := crypto.RecoverPublicKey(make([]byte, 32), make([]byte, 10))
		if !errors.Is(err, crypto.ErrInvalidSignature) {
			t.Errorf("error = %v, want %v", err, crypto.ErrInvalidSignature)
		}
	})

	t.Run("undecodable signature", func(t *testing.T) {
		// Correct length (65) but cryptographically invalid: SigToPub must fail.
		_, err := crypto.RecoverPublicKey(make([]byte, 32), make([]byte, 65))
		if err == nil {
			t.Error("expected error for an invalid 65-byte signature")
		}
	})
}

func TestHashSHA256(t *testing.T) {
	got := crypto.HashSHA256([]byte("abc"))
	want := sha256.Sum256([]byte("abc"))
	if !bytes.Equal(got, want[:]) {
		t.Errorf("HashSHA256() = %x, want %x", got, want)
	}
}

func TestHashKeccak256(t *testing.T) {
	// Keccak256 of the empty input is a well-known vector.
	got := hex.EncodeToString(crypto.HashKeccak256([]byte{}))
	want := "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
	if got != want {
		t.Errorf("HashKeccak256() = %s, want %s", got, want)
	}
}
