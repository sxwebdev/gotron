package abi

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestSignature(t *testing.T) {
	// transfer(address,uint256) selector is the canonical ERC20/TRC20 value.
	require.Equal(t, "a9059cbb", hex.EncodeToString(Signature("transfer(address,uint256)")))
}

func TestPack(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b, err := Pack("transfer(address,uint256)", []Param{
			{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
			{"uint256": "1000000"},
		})
		require.NoError(t, err)
		require.Len(t, b, 4+64) // selector + two 32-byte words
		require.Equal(t, "a9059cbb", hex.EncodeToString(b[:4]))
	})

	t.Run("propagates param error", func(t *testing.T) {
		_, err := Pack("x()", []Param{{"notatype": "1"}})
		require.Error(t, err)
	})
}

func TestLoadFromJSON(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		got, err := LoadFromJSON("")
		require.NoError(t, err)
		require.Nil(t, got)
	})
	t.Run("invalid json", func(t *testing.T) {
		_, err := LoadFromJSON("{not json")
		require.Error(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		got, err := LoadFromJSON(`[{"uint256":"1"}]`)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})
}

func testABI() *core.SmartContract_ABI {
	return &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name:    "balanceOf",
				Inputs:  []*core.SmartContract_ABI_Entry_Param{{Name: "owner", Type: "address"}},
				Outputs: []*core.SmartContract_ABI_Entry_Param{{Name: "balance", Type: "uint256"}},
			},
			{
				Name:    "broken",
				Inputs:  []*core.SmartContract_ABI_Entry_Param{{Name: "x", Type: "notatype"}},
				Outputs: []*core.SmartContract_ABI_Entry_Param{{Name: "y", Type: "notatype"}},
			},
		},
	}
}

func TestGetParser(t *testing.T) {
	abi := testABI()

	t.Run("found", func(t *testing.T) {
		args, err := GetParser(abi, "balanceOf")
		require.NoError(t, err)
		require.Len(t, args, 1)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := GetParser(abi, "missing")
		require.EqualError(t, err, "not found")
	})
	t.Run("invalid output type", func(t *testing.T) {
		_, err := GetParser(abi, "broken")
		require.Error(t, err)
	})
}

func TestGetInputsParser(t *testing.T) {
	abi := testABI()

	t.Run("found", func(t *testing.T) {
		args, err := GetInputsParser(abi, "balanceOf")
		require.NoError(t, err)
		require.Len(t, args, 1)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := GetInputsParser(abi, "missing")
		require.EqualError(t, err, "not found")
	})
	t.Run("invalid input type", func(t *testing.T) {
		_, err := GetInputsParser(abi, "broken")
		require.Error(t, err)
	})
}

func TestGetPaddedParamErrors(t *testing.T) {
	tests := []struct {
		name  string
		param []Param
	}{
		{"param with multiple keys", []Param{{"a": "1", "b": "2"}}},
		{"unknown type", []Param{{"notatype": "1"}}},
		{"invalid address string", []Param{{"address": "not-an-address"}}},
		{"non-string address", []Param{{"address": 123}}},
		{"address array not an array", []Param{{"address[2]": "notarray"}}},
		{"address array with invalid element", []Param{{"address[2]": []interface{}{"bad", "bad"}}}},
		{"uint array not a string slice", []Param{{"uint256[2]": "notslice"}}},
		{"bytes neither hex nor base64", []Param{{"bytes": "zz"}}},
		{"fixed bytes wrong size", []Param{{"bytes2": "0102030405"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetPaddedParam(tt.param)
			require.Error(t, err)
		})
	}
}

func TestGetPaddedParamIntegerSizes(t *testing.T) {
	// String inputs route through convertToInt for every signed/unsigned width.
	b, err := GetPaddedParam([]Param{
		{"int8": "1"}, {"int16": "1"}, {"int32": "1"}, {"int64": "1"},
		{"uint8": "1"}, {"uint16": "1"}, {"uint32": "1"}, {"uint64": "1"},
		{"int128": "1"}, // > 64 bits routes through the big.Int branch
	})
	require.NoError(t, err)
	require.Len(t, b, 32*9)
}

func TestConvertToBytesVariants(t *testing.T) {
	t.Run("base64 fallback for dynamic bytes", func(t *testing.T) {
		// "++++" is not valid hex but is valid base64.
		b, err := GetPaddedParam([]Param{{"bytes": "++++"}})
		require.NoError(t, err)
		require.NotEmpty(t, b)
	})

	t.Run("fixed byte sizes", func(t *testing.T) {
		b, err := GetPaddedParam([]Param{
			{"bytes1": "01"},
			{"bytes2": "0102"},
			{"bytes8": "0102030405060708"},
			{"bytes16": strings.Repeat("ab", 16)},
		})
		require.NoError(t, err)
		require.Len(t, b, 32*4)
	})

	t.Run("non-string bytes value is passed through", func(t *testing.T) {
		b, err := GetPaddedParam([]Param{{"bytes": []byte{0x01, 0x02, 0x03}}})
		require.NoError(t, err)
		require.NotEmpty(t, b)
	})
}

func TestGetPaddedParamUintArrayHexElements(t *testing.T) {
	// Hex-prefixed elements route through the base-16 branch of the >64-bit array path.
	b, err := GetPaddedParam([]Param{{"uint256[2]": []string{"0x01", "0x02"}}})
	require.NoError(t, err)
	require.Len(t, b, 64)
}

// Regression: unparseable numeric strings must error, never silently encode 0
// (small ints) or panic inside PackValues (big ints).
func TestConvertToIntRejectsBadInput(t *testing.T) {
	tests := []struct {
		name  string
		param []Param
	}{
		{"bad uint64 (was silent 0)", []Param{{"uint64": "abc"}}},
		{"empty int64 (was silent 0)", []Param{{"int64": ""}}},
		{"bad uint256 (was panic)", []Param{{"uint256": "abc"}}},
		{"bad int256 (was panic)", []Param{{"int256": "xyz"}}},
		{"bad hex uint256", []Param{{"uint256": "0xZZ"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetPaddedParam(tt.param)
			require.Error(t, err)
		})
	}
}

// Regression: uint256[] decoded from JSON arrives as []interface{}.
func TestUintArrayFromJSON(t *testing.T) {
	param, err := LoadFromJSON(`[{"uint256[2]": ["1", "2"]}]`)
	require.NoError(t, err)
	b, err := GetPaddedParam(param)
	require.NoError(t, err)
	require.Len(t, b, 64)
}

// Regression: a bad element in a uint256[] must error, not panic.
func TestUintArrayBadElement(t *testing.T) {
	_, err := GetPaddedParam([]Param{{"uint256[2]": []string{"1", "abc"}}})
	require.Error(t, err)
}

// Regression: fixed-bytes types of every size (not just 1/2/8/16/32) must pack.
func TestFixedBytesAllSizes(t *testing.T) {
	tests := []struct {
		typ string
		hex string
	}{
		{"bytes4", "deadbeef"},
		{"bytes20", strings.Repeat("ab", 20)},
		{"bytes31", strings.Repeat("cd", 31)},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			b, err := GetPaddedParam([]Param{{tt.typ: tt.hex}})
			require.NoError(t, err)
			require.Len(t, b, 32) // fixed bytes pad to a single 32-byte word
		})
	}
}
