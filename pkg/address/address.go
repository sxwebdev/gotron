// Package address provides functionality for generating and managing Tron addresses.
package address

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"github.com/sxwebdev/go-bip39"
)

const (
	// BIP44 constants
	bip44Purpose   = 44
	tronCoinType   = 195
	defaultAccount = 0
	defaultChange  = 0

	// Address constants
	addressLength = 21
	prefixByte    = 0x41 // Mainnet prefix
)

var (
	// ErrInvalidMnemonic is returned when the mnemonic is invalid
	ErrInvalidMnemonic = errors.New("invalid mnemonic")
	// ErrInvalidPrivateKey is returned when the private key is invalid
	ErrInvalidPrivateKey = errors.New("invalid private key")
	// ErrInvalidAddress is returned when the address is invalid
	ErrInvalidAddress = errors.New("invalid address")
)

// Address represents a Tron address with its keys
type Address struct {
	PrivateKeyECDSA *ecdsa.PrivateKey
	PublicKeyECDSA  *ecdsa.PublicKey
	PrivateKey      string
	PublicKey       string
	Address         string
	Mnemonic        string
}

// GenerateMnemonic generates a new BIP39 mnemonic phrase
// strength should be 128, 160, 192, 224, or 256 bits
func GenerateMnemonic(strength int) (string, error) {
	if strength%32 != 0 || strength < 128 || strength > 256 {
		return "", errors.New("invalid strength: must be 128, 160, 192, 224, or 256")
	}

	entropy, err := bip39.NewEntropy(strength)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return mnemonic, nil
}

// FromMnemonic creates an address from a BIP39 mnemonic and optional passphrase
func FromMnemonic(mnemonic, passphrase string, index uint32) (*Address, error) {
	return NewGenerator(mnemonic, passphrase).Generate(index)
}

// Generate creates a new random Tron address
func Generate() (*Address, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return fromECDSA(privateKey, "")
}

// FromPrivateKey imports an address from a hex-encoded private key
func FromPrivateKey(privateKeyHex string) (*Address, error) {
	if privateKeyHex == "" {
		return nil, ErrInvalidPrivateKey
	}

	// Remove 0x prefix if present
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	pkBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(pkBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(pkBytes))
	}

	privateKey, err := crypto.ToECDSA(pkBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to ECDSA: %w", err)
	}

	return fromECDSA(privateKey, "")
}

// fromECDSA creates an Address from an ECDSA private key
func fromECDSA(privateKey *ecdsa.PrivateKey, mnemonic string) (*Address, error) {
	if privateKey == nil {
		return nil, ErrInvalidPrivateKey
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to cast public key to ECDSA")
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Use compressed public key format
	publicKeyBytes := crypto.CompressPubkey(publicKeyECDSA)
	publicKeyHex := hex.EncodeToString(publicKeyBytes)

	// Generate Tron address
	address := pubKeyToAddress(publicKeyECDSA)

	return &Address{
		PrivateKeyECDSA: privateKey,
		PublicKeyECDSA:  publicKeyECDSA,
		PrivateKey:      privateKeyHex,
		PublicKey:       publicKeyHex,
		Address:         address,
		Mnemonic:        mnemonic,
	}, nil
}

// pubKeyToAddress converts a public key to a Tron address
func pubKeyToAddress(publicKey *ecdsa.PublicKey) string {
	if publicKey == nil {
		return ""
	}

	address := crypto.PubkeyToAddress(*publicKey)

	addressTron := make([]byte, 0)
	addressTron = append(addressTron, prefixByte)
	addressTron = append(addressTron, address.Bytes()...)
	return encodeCheck(addressTron)
}

// Validate checks if a Tron address is valid
func Validate(address string) error {
	if address == "" {
		return ErrInvalidAddress
	}

	decoded, err := decodeCheck(address)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	if len(decoded) != addressLength {
		return fmt.Errorf("invalid address length: expected %d, got %d", addressLength, len(decoded))
	}

	if decoded[0] != prefixByte {
		return fmt.Errorf("invalid address prefix: expected 0x%02x, got 0x%02x", prefixByte, decoded[0])
	}

	return nil
}

// encodeCheck encodes a byte slice to base58 with checksum
func encodeCheck(input []byte) string {
	hash := doubleSHA256(input)
	checksum := hash[:4]
	return base58.Encode(append(input, checksum...))
}

// decodeCheck decodes a base58 string and verifies checksum
func decodeCheck(input string) ([]byte, error) {
	decoded, err := base58.Decode(input)
	if err != nil {
		return nil, err
	}

	if len(decoded) < 4 {
		return nil, errors.New("invalid encoded data")
	}

	data := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]

	hash := doubleSHA256(data)
	for i := range 4 {
		if hash[i] != checksum[i] {
			return nil, errors.New("checksum mismatch")
		}
	}

	return data, nil
}

// doubleSHA256 performs SHA256 twice
func doubleSHA256(data []byte) []byte {
	hash1 := sha256.Sum256(data)
	hash2 := sha256.Sum256(hash1[:])
	return hash2[:]
}
