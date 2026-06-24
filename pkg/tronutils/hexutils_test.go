package tronutils_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
)

func TestBytesToHexString(t *testing.T) {
	require.Equal(t, "0x01ab", tronutils.BytesToHexString([]byte{0x01, 0xab}))
	require.Equal(t, "0x", tronutils.BytesToHexString(nil))
}

func TestHexStringToBytes(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := tronutils.HexStringToBytes("")
		require.ErrorIs(t, err, tronutils.EmptyString)
		require.Equal(t, "empty hex string", err.Error())
	})
	t.Run("with 0x prefix", func(t *testing.T) {
		got, err := tronutils.HexStringToBytes("0x01ab")
		require.NoError(t, err)
		require.Equal(t, []byte{0x01, 0xab}, got)
	})
	t.Run("without prefix", func(t *testing.T) {
		got, err := tronutils.HexStringToBytes("01ab")
		require.NoError(t, err)
		require.Equal(t, []byte{0x01, 0xab}, got)
	})
	t.Run("invalid hex", func(t *testing.T) {
		_, err := tronutils.HexStringToBytes("0xzz")
		require.Error(t, err)
	})
}

func TestToHex(t *testing.T) {
	require.Equal(t, "0x0", tronutils.ToHex(nil))
	require.Equal(t, "0x0a", tronutils.ToHex([]byte{0x0a}))
}

func TestToHexArray(t *testing.T) {
	require.Equal(t, []string{"0x01", "0x02"}, tronutils.ToHexArray([][]byte{{0x01}, {0x02}}))
}

func TestFromHex(t *testing.T) {
	t.Run("with 0x prefix", func(t *testing.T) {
		got, err := tronutils.FromHex("0x01ab")
		require.NoError(t, err)
		require.Equal(t, []byte{0x01, 0xab}, got)
	})
	t.Run("odd length is left padded", func(t *testing.T) {
		got, err := tronutils.FromHex("0x1")
		require.NoError(t, err)
		require.Equal(t, []byte{0x01}, got)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := tronutils.FromHex("zz")
		require.Error(t, err)
	})
}

func TestCopyBytes(t *testing.T) {
	require.Nil(t, tronutils.CopyBytes(nil))

	src := []byte{1, 2, 3}
	dst := tronutils.CopyBytes(src)
	require.Equal(t, src, dst)

	// Must be an independent copy, not an alias.
	dst[0] = 9
	require.Equal(t, byte(1), src[0])
}

func TestHas0xPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"0x12", true},
		{"0X12", true},
		{"1x12", false},
		{"0", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tronutils.Has0xPrefix(tt.in); got != tt.want {
			t.Errorf("Has0xPrefix(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"01ab", true},
		{"ABCD", true},
		{"", true}, // even length, no invalid characters
		{"0", false},
		{"0g", false},
	}
	for _, tt := range tests {
		if got := tronutils.IsHex(tt.in); got != tt.want {
			t.Errorf("IsHex(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestBytes2HexAndHex2Bytes(t *testing.T) {
	require.Equal(t, "0aff", tronutils.Bytes2Hex([]byte{0x0a, 0xff}))

	got, err := tronutils.Hex2Bytes("0aff")
	require.NoError(t, err)
	require.Equal(t, []byte{0x0a, 0xff}, got)

	_, err = tronutils.Hex2Bytes("zz")
	require.Error(t, err)
}

func TestHex2BytesFixed(t *testing.T) {
	require.Equal(t, []byte{0x01, 0x02}, tronutils.Hex2BytesFixed("0102", 2))     // exact
	require.Equal(t, []byte{0x02, 0x03}, tronutils.Hex2BytesFixed("010203", 2))   // longer: crop left
	require.Equal(t, []byte{0x00, 0x00, 0x02}, tronutils.Hex2BytesFixed("02", 3)) // shorter: pad left
}

func TestRightPadBytes(t *testing.T) {
	require.Equal(t, []byte{1, 2, 3}, tronutils.RightPadBytes([]byte{1, 2, 3}, 2)) // l <= len: unchanged
	require.Equal(t, []byte{1, 0, 0}, tronutils.RightPadBytes([]byte{1}, 3))
}

func TestLeftPadBytes(t *testing.T) {
	require.Equal(t, []byte{1, 2, 3}, tronutils.LeftPadBytes([]byte{1, 2, 3}, 2)) // l <= len: unchanged
	require.Equal(t, []byte{0, 0, 1}, tronutils.LeftPadBytes([]byte{1}, 3))
}

func TestTrimLeftZeroes(t *testing.T) {
	require.Equal(t, []byte{1, 0, 2}, tronutils.TrimLeftZeroes([]byte{0, 0, 1, 0, 2}))
	require.Equal(t, []byte{}, tronutils.TrimLeftZeroes([]byte{0, 0, 0}))
	require.Equal(t, []byte{1, 2}, tronutils.TrimLeftZeroes([]byte{1, 2}))
}
