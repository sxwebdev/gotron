package gotron_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/address"
	"github.com/sxwebdev/gotron/pkg/tronutils"
)

// documentedAddressAPI lists every pkg/address identifier the docs are allowed
// to reference. Each entry is a compile-time reference to the real symbol, so a
// rename in the package breaks this file, and a doc example naming something
// that is not here fails TestDocsReferenceRealAPI.
//
// This exists because the docs previously advertised address.PrivateKeyFromHex
// and address.NewAddressGenerator, neither of which was ever defined - anyone
// copying those examples got a compile error.
var documentedAddressAPI = map[string]any{
	"Address":          address.Address{},
	"FromMnemonic":     address.FromMnemonic,
	"FromPrivateKey":   address.FromPrivateKey,
	"Generate":         address.Generate,
	"GenerateMnemonic": address.GenerateMnemonic,
	"NewGenerator":     address.NewGenerator,
	"Validate":         address.Validate,
}

// documentedTronutilsAPI plays the same role for pkg/tronutils. Deprecated
// symbols (ToHex, HexStringToBytes) are deliberately absent so that documenting
// one fails this test instead of steering readers towards them.
var documentedTronutilsAPI = map[string]any{
	"Base58ToAddress":  tronutils.Base58ToAddress,
	"Base64ToAddress":  tronutils.Base64ToAddress,
	"BigToAddress":     tronutils.BigToAddress,
	"Bytes2Hex":        tronutils.Bytes2Hex,
	"BytesToHexString": tronutils.BytesToHexString,
	"DecodeCheck":      tronutils.DecodeCheck,
	"EncodeCheck":      tronutils.EncodeCheck,
	"FromHex":          tronutils.FromHex,
	"Has0xPrefix":      tronutils.Has0xPrefix,
	"Hex2Bytes":        tronutils.Hex2Bytes,
	"HexToAddress":     tronutils.HexToAddress,
	"IsHex":            tronutils.IsHex,
	"Keccak256":        tronutils.Keccak256,
	"LeftPadBytes":     tronutils.LeftPadBytes,
	"RightPadBytes":    tronutils.RightPadBytes,
}

func TestDocsReferenceRealAPI(t *testing.T) {
	t.Parallel()

	docFiles := []string{"README.md", "doc.go", "skills/gotron/references/api-surface.md"}

	pkgs := []struct {
		name  string
		known map[string]any
	}{
		{"address", documentedAddressAPI},
		{"tronutils", documentedTronutilsAPI},
	}

	for _, p := range pkgs {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			re := regexp.MustCompile(`\b` + p.name + `\.([A-Z]\w*)`)

			for _, file := range docFiles {
				content, err := os.ReadFile(file)
				require.NoError(t, err)

				for _, m := range re.FindAllStringSubmatch(string(content), -1) {
					require.Contains(t, p.known, m[1],
						"%s references %s.%s, which is not part of the package's public API",
						file, p.name, m[1])
				}
			}
		})
	}
}

// TestDocumentedSigningFlowCompiles exercises the exact call sequence the README
// and doc.go show for signing: FromPrivateKey followed by SignTransaction on the
// returned *ecdsa.PrivateKey field.
func TestDocumentedSigningFlowCompiles(t *testing.T) {
	t.Parallel()

	generated, err := address.Generate()
	require.NoError(t, err)

	signer, err := address.FromPrivateKey(generated.PrivateKey)
	require.NoError(t, err)
	require.NotNil(t, signer.PrivateKeyECDSA)
	require.Equal(t, generated.Address, signer.Address)
}
