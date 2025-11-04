// Package crypto provides cryptographic functions for Tron transactions.
package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// ErrInvalidPrivateKey is returned when the private key is invalid
	ErrInvalidPrivateKey = errors.New("invalid private key")
	// ErrInvalidSignature is returned when the signature is invalid
	ErrInvalidSignature = errors.New("invalid signature")
)

// SignData signs data with the given private key using ECDSA
func SignData(data []byte, privateKeyHex string) ([]byte, error) {
	if privateKeyHex == "" {
		return nil, ErrInvalidPrivateKey
	}

	// Remove 0x prefix if present
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKeyBytes))
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to ECDSA: %w", err)
	}

	// Hash the data
	hash := crypto.Keccak256(data)

	// Sign the hash
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	return signature, nil
}

// HashSHA256 performs SHA256 hashing
func HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashKeccak256 performs Keccak256 hashing
func HashKeccak256(data []byte) []byte {
	return crypto.Keccak256(data)
}

// VerifySignature verifies an ECDSA signature
func VerifySignature(publicKey *ecdsa.PublicKey, hash, signature []byte) bool {
	if len(signature) != 65 {
		return false
	}

	// Remove recovery ID
	return crypto.VerifySignature(
		crypto.FromECDSAPub(publicKey),
		hash,
		signature[:64],
	)
}

// RecoverPublicKey recovers the public key from a signature
func RecoverPublicKey(hash, signature []byte) (*ecdsa.PublicKey, error) {
	if len(signature) != 65 {
		return nil, ErrInvalidSignature
	}

	publicKey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to recover public key: %w", err)
	}

	return publicKey, nil
}
