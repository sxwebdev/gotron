package address

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/sxwebdev/go-bip39"
)

type Generator struct {
	bipPurpose uint32
	coinType   uint32
	account    uint32
	mnemonic   string
	passphrase string
	net        *chaincfg.Params
}

// NewGenerator creates a new AddressGenerator with default BIP44 parameters for Tron
func NewGenerator(mnemonic, passphrase string) *Generator {
	return &Generator{
		bipPurpose: bip44Purpose,
		coinType:   tronCoinType,
		account:    defaultAccount,
		mnemonic:   mnemonic,
		passphrase: passphrase,
		net:        &chaincfg.MainNetParams,
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

// SetNetwork sets the network parameters
func (ag *Generator) SetNetwork(net *chaincfg.Params) *Generator {
	ag.net = net
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

	masterKey, err := hdkeychain.NewMaster(seed, ag.net)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Derive using BIP44 path: m / purpose' / coin_type' / account' / change / address_index
	purpose, err := masterKey.Derive(hdkeychain.HardenedKeyStart + ag.bipPurpose)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose: %w", err)
	}

	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + ag.coinType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin type: %w", err)
	}

	account, err := coinType.Derive(hdkeychain.HardenedKeyStart + ag.account)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	change, err := account.Derive(defaultChange)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change: %w", err)
	}

	addressKey, err := change.Derive(index)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	privKey, err := addressKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	return fromECDSA(privKey.ToECDSA(), ag.mnemonic)
}
