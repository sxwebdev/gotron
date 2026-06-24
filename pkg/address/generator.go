package address

import (
	"fmt"

	"github.com/decred/dcrd/hdkeychain/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sxwebdev/go-bip39"
)

// hdNetParams satisfies hdkeychain.NetworkParams. The extended-key version bytes
// only affect xprv/xpub serialization, which is never used for Tron, so the
// concrete values are arbitrary; the standard Bitcoin mainnet prefixes are used.
type hdNetParams struct{}

func (hdNetParams) HDPrivKeyVersion() [4]byte { return [4]byte{0x04, 0x88, 0xad, 0xe4} }
func (hdNetParams) HDPubKeyVersion() [4]byte  { return [4]byte{0x04, 0x88, 0xb2, 0x1e} }

type Generator struct {
	bipPurpose uint32
	coinType   uint32
	account    uint32
	mnemonic   string
	passphrase string
}

// NewGenerator creates a new AddressGenerator with default BIP44 parameters for Tron
func NewGenerator(mnemonic, passphrase string) *Generator {
	return &Generator{
		bipPurpose: bip44Purpose,
		coinType:   tronCoinType,
		account:    defaultAccount,
		mnemonic:   mnemonic,
		passphrase: passphrase,
	}
}

// SetBipPurpose sets a custom BIP purpose
func (ag *Generator) SetBipPurpose(purpose uint32) *Generator {
	ag.bipPurpose = purpose
	return ag
}

// SetCoinType sets a custom coin type
func (ag *Generator) SetCoinType(coinType uint32) *Generator {
	ag.coinType = coinType
	return ag
}

// SetAccount sets a custom account index
func (ag *Generator) SetAccount(account uint32) *Generator {
	ag.account = account
	return ag
}

// Generate generates an address at the specified index
func (ag *Generator) Generate(index uint32) (*Address, error) {
	if ag.mnemonic == "" {
		return nil, ErrInvalidMnemonic
	}

	if !bip39.IsMnemonicValid(ag.mnemonic) {
		return nil, ErrInvalidMnemonic
	}

	seed := bip39.NewSeed(ag.mnemonic, ag.passphrase)

	masterKey, err := hdkeychain.NewMaster(seed, hdNetParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Derive using BIP44 path: m / purpose' / coin_type' / account' / change / address_index
	purpose, err := masterKey.ChildBIP32Std(hdkeychain.HardenedKeyStart + ag.bipPurpose)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose: %w", err)
	}

	coinType, err := purpose.ChildBIP32Std(hdkeychain.HardenedKeyStart + ag.coinType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin type: %w", err)
	}

	account, err := coinType.ChildBIP32Std(hdkeychain.HardenedKeyStart + ag.account)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	change, err := account.ChildBIP32Std(defaultChange)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change: %w", err)
	}

	addressKey, err := change.ChildBIP32Std(index)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	privKeyBytes, err := addressKey.SerializedPrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %w", err)
	}

	return fromECDSA(privKey, ag.mnemonic)
}
