package address

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"
)

// test mnemonic generated from https://iancoleman.io/bip39/
const testMnemonic = "recipe need harsh web order laptop seek filter among federal glory balcony video fault shed myself crush orient figure crack beach weather find match"

func TestGenerateMnemonic(t *testing.T) {
	tests := []struct {
		name     string
		strength int
		wantErr  bool
	}{
		{"12 words", 128, false},
		{"15 words", 160, false},
		{"18 words", 192, false},
		{"21 words", 224, false},
		{"24 words", 256, false},
		{"invalid strength", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mnemonic, err := GenerateMnemonic(tt.strength)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMnemonic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(mnemonic) == 0 {
				t.Error("GenerateMnemonic() returned empty mnemonic")
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	addr, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Logf("generated address / private key: %s / %s", addr.Address, addr.PrivateKey)

	if len(addr.PrivateKey) == 0 {
		t.Error("Generate() returned empty private key")
	}
	if len(addr.Address) == 0 {
		t.Error("Generate() returned empty address")
	}
	if addr.Address[0] != 'T' {
		t.Errorf("Generate() address should start with 'T', got %c", addr.Address[0])
	}
}

func TestFromMnemonic(t *testing.T) {
	// test mnemonic generated from https://iancoleman.io/bip39/
	mnemonic := testMnemonic

	tests := []struct {
		name               string
		mnemonic           string
		passphrase         string
		index              uint32
		expectedAddress    string
		expectedPrivateKey string
		expectedPublicKey  string
		wantErr            bool
	}{
		{
			name:               "valid mnemonic index 0",
			mnemonic:           mnemonic,
			passphrase:         "",
			index:              0,
			expectedAddress:    "TEeKaYdpN6ujnpVZ1SkohE6Ru6gd9vGC2A",
			expectedPrivateKey: "ef80f4f95fe356c6405a0ff976ea8e7ee85caf6a9fd9f4a073ddf46b149733ee",
			expectedPublicKey:  "02d43d57680133e79c72e00db89d431509d2df1a741f426b9647a4ee4fbf4a265d",
			wantErr:            false,
		},
		{
			name:               "valid mnemonic index 1",
			mnemonic:           mnemonic,
			passphrase:         "",
			index:              1,
			expectedAddress:    "TUwbUgKvC1RsT3qShxmZcfMpvMdbE6JPST",
			expectedPrivateKey: "103fe88a78c52f2f3e00d57fee72b713d3999b4c377581c127e2887ebd0e1665",
			expectedPublicKey:  "029ffe37a0934bbc1f52c8842c3704d1b3a8758eaf3233f6dbe674ff7030a5033e",
			wantErr:            false,
		},
		{
			name:               "with passphrase",
			mnemonic:           mnemonic,
			passphrase:         "test",
			index:              0,
			expectedAddress:    "TLutkfK9N2BaBEzUngAuaNKTC9SZu3ER1K",
			expectedPrivateKey: "45d4d746b72d62b06fa02f83b119d8cfd7a8a17cf25854cead2cb7f7f09e8e0e",
			expectedPublicKey:  "03a784723062a3371976bd68fe07e1ee10229c359cc8c1d985df5c4f54d54043c8",
			wantErr:            false,
		},
		{
			name:            "empty mnemonic",
			mnemonic:        "",
			passphrase:      "",
			index:           0,
			expectedAddress: "",
			wantErr:         true,
		},
		{
			name:            "invalid mnemonic",
			mnemonic:        "invalid words",
			passphrase:      "",
			index:           0,
			expectedAddress: "",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := FromMnemonic(tt.mnemonic, tt.passphrase, tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromMnemonic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && addr.Mnemonic != tt.mnemonic {
				t.Error("FromMnemonic() mnemonic not stored")
			}

			if addr != nil {
				require.Equal(t, tt.expectedAddress, addr.Address)
				require.Equal(t, tt.expectedPrivateKey, addr.PrivateKey)
				require.Equal(t, tt.expectedPublicKey, addr.PublicKey)
			}
		})
	}
}

func TestFromPrivateKey(t *testing.T) {
	addr, err := Generate()
	if err != nil {
		t.Fatalf("Failed to generate address: %v", err)
	}

	tests := []struct {
		name       string
		privateKey string
		wantErr    bool
	}{
		{"valid key", addr.PrivateKey, false},
		{"with 0x prefix", "0x" + addr.PrivateKey, false},
		{"empty key", "", true},
		{"invalid hex", "zzz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FromPrivateKey(tt.privateKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.Address != addr.Address {
				t.Errorf("FromPrivateKey() address = %v, want %v", result.Address, addr.Address)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	validAddr, err := Generate()
	if err != nil {
		t.Fatalf("Failed to generate address: %v", err)
	}

	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid address", validAddr.Address, false},
		{"empty address", "", true},
		{"invalid format", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddressGenerator_Generate(t *testing.T) {
	// test mnemonic generated from https://iancoleman.io/bip39/
	mnemonic := testMnemonic

	t.Run("valid generation with default parameters", func(t *testing.T) {
		generator := NewGenerator(mnemonic, "")

		addr, err := generator.Generate(0)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if addr == nil {
			t.Fatal("Generate() returned nil address")
		}

		if len(addr.PrivateKey) == 0 {
			t.Error("Generate() returned empty private key")
		}

		if len(addr.Address) == 0 {
			t.Error("Generate() returned empty address")
		}

		if addr.Address[0] != 'T' {
			t.Errorf("Generate() address should start with 'T', got %c", addr.Address[0])
		}

		if addr.Mnemonic != mnemonic {
			t.Error("Generate() mnemonic not stored correctly")
		}

		t.Logf("Generated address at index 0: %s", addr.Address)
		t.Logf("Private key: %s", addr.PrivateKey)
	})

	t.Run("generate multiple addresses with different indices", func(t *testing.T) {
		generator := NewGenerator(mnemonic, "")

		addresses := make(map[string]bool)
		for i := uint32(0); i < 5; i++ {
			addr, err := generator.Generate(i)
			if err != nil {
				t.Fatalf("Generate(%d) error = %v", i, err)
			}

			// Check that each address is unique
			if addresses[addr.Address] {
				t.Errorf("Duplicate address generated at index %d: %s", i, addr.Address)
			}
			addresses[addr.Address] = true

			// Validate the address
			if err := Validate(addr.Address); err != nil {
				t.Errorf("Generated invalid address at index %d: %v", i, err)
			}

			t.Logf("Index %d: %s", i, addr.Address)
		}

		if len(addresses) != 5 {
			t.Errorf("Expected 5 unique addresses, got %d", len(addresses))
		}
	})

	t.Run("generate with passphrase", func(t *testing.T) {
		generator := NewGenerator(mnemonic, "mypassphrase")

		addr1, err := generator.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with passphrase error = %v", err)
		}

		// Same index but no passphrase should produce different address
		generatorNoPass := NewGenerator(mnemonic, "")
		addr2, err := generatorNoPass.Generate(0)
		if err != nil {
			t.Fatalf("Generate() without passphrase error = %v", err)
		}

		if addr1.Address == addr2.Address {
			t.Error("Passphrase should produce different address")
		}

		t.Logf("With passphrase: %s", addr1.Address)
		t.Logf("Without passphrase: %s", addr2.Address)
	})

	t.Run("deterministic generation", func(t *testing.T) {
		generator1 := NewGenerator(mnemonic, "")
		generator2 := NewGenerator(mnemonic, "")

		addr1, err := generator1.Generate(0)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		addr2, err := generator2.Generate(0)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if addr1.Address != addr2.Address {
			t.Errorf("Same mnemonic and index should produce same address: %s != %s",
				addr1.Address, addr2.Address)
		}

		if addr1.PrivateKey != addr2.PrivateKey {
			t.Error("Same mnemonic and index should produce same private key")
		}
	})

	t.Run("empty mnemonic", func(t *testing.T) {
		generator := NewGenerator("", "")

		_, err := generator.Generate(0)
		if err != ErrInvalidMnemonic {
			t.Errorf("Generate() with empty mnemonic should return ErrInvalidMnemonic, got %v", err)
		}
	})

	t.Run("invalid mnemonic", func(t *testing.T) {
		generator := NewGenerator("invalid mnemonic words test", "")

		_, err := generator.Generate(0)
		if err != ErrInvalidMnemonic {
			t.Errorf("Generate() with invalid mnemonic should return ErrInvalidMnemonic, got %v", err)
		}
	})

	t.Run("custom BIP parameters", func(t *testing.T) {
		generator := NewGenerator(mnemonic, "").
			SetBipPurpose(44).
			SetCoinType(195).
			SetAccount(0)

		addr, err := generator.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with custom BIP parameters error = %v", err)
		}

		if err := Validate(addr.Address); err != nil {
			t.Errorf("Generated address with custom parameters is invalid: %v", err)
		}

		t.Logf("Custom BIP params address: %s", addr.Address)
	})

	t.Run("different coin type", func(t *testing.T) {
		// Using Bitcoin coin type (0) should produce different addresses
		generatorTron := NewGenerator(mnemonic, "").SetCoinType(195)
		generatorOther := NewGenerator(mnemonic, "").SetCoinType(0)

		addrTron, err := generatorTron.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with Tron coin type error = %v", err)
		}

		addrOther, err := generatorOther.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with other coin type error = %v", err)
		}

		if addrTron.Address == addrOther.Address {
			t.Error("Different coin types should produce different addresses")
		}

		t.Logf("Tron coin type (195): %s", addrTron.Address)
		t.Logf("Other coin type (0): %s", addrOther.Address)
	})

	t.Run("different account", func(t *testing.T) {
		generator1 := NewGenerator(mnemonic, "").SetAccount(0)
		generator2 := NewGenerator(mnemonic, "").SetAccount(1)

		addr1, err := generator1.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with account 0 error = %v", err)
		}

		addr2, err := generator2.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with account 1 error = %v", err)
		}

		if addr1.Address == addr2.Address {
			t.Error("Different accounts should produce different addresses")
		}

		t.Logf("Account 0: %s", addr1.Address)
		t.Logf("Account 1: %s", addr2.Address)
	})

	t.Run("testnet network", func(t *testing.T) {
		generatorMainnet := NewGenerator(mnemonic, "").
			SetNetwork(&chaincfg.MainNetParams)

		generatorTestnet := NewGenerator(mnemonic, "").
			SetNetwork(&chaincfg.TestNet3Params)

		addrMainnet, err := generatorMainnet.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with mainnet error = %v", err)
		}

		addrTestnet, err := generatorTestnet.Generate(0)
		if err != nil {
			t.Fatalf("Generate() with testnet error = %v", err)
		}

		// Private keys should be the same regardless of network
		if addrMainnet.PrivateKey != addrTestnet.PrivateKey {
			t.Error("Same mnemonic should produce same private key on different networks")
		}

		// Addresses should also be the same (Tron uses same address format for mainnet/testnet)
		// The difference is in which network the address is used on
		if addrMainnet.Address != addrTestnet.Address {
			t.Logf("Mainnet: %s", addrMainnet.Address)
			t.Logf("Testnet: %s", addrTestnet.Address)
			// Note: In Tron, mainnet and testnet addresses have the same format
			// The network parameter is used for key derivation context
		}

		// Validate both addresses
		if err := Validate(addrMainnet.Address); err != nil {
			t.Errorf("Mainnet address is invalid: %v", err)
		}

		if err := Validate(addrTestnet.Address); err != nil {
			t.Errorf("Testnet address is invalid: %v", err)
		}

		t.Logf("Mainnet address: %s", addrMainnet.Address)
		t.Logf("Testnet address: %s", addrTestnet.Address)
		t.Logf("Private key (same for both): %s", addrMainnet.PrivateKey)
	})
}
