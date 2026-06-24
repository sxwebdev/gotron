package tronutils_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
)

// A real mainnet Tron address (21-byte payload, 0x41 prefix).
const validTronAddr = "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"

func TestEncodeDecode(t *testing.T) {
	in := []byte{0x01, 0x02, 0x03, 0xff}
	got, err := tronutils.Decode(tronutils.Encode(in))
	require.NoError(t, err)
	require.Equal(t, in, got)
}

func TestEncodeCheckDecodeCheckRoundTrip(t *testing.T) {
	input := make([]byte, 21)
	input[0] = tronutils.TronBytePrefix
	for i := 1; i < len(input); i++ {
		input[i] = byte(i)
	}

	got, err := tronutils.DecodeCheck(tronutils.EncodeCheck(input))
	require.NoError(t, err)
	require.Equal(t, input, got)
}

func TestDecodeCheckValidAddress(t *testing.T) {
	got, err := tronutils.DecodeCheck(validTronAddr)
	require.NoError(t, err)
	require.Len(t, got, 21)
	require.Equal(t, tronutils.TronBytePrefix, got[0])
}

func TestDecodeCheckErrors(t *testing.T) {
	t.Run("invalid base58", func(t *testing.T) {
		// '0', 'O', 'I', 'l' are not in the Bitcoin base58 alphabet.
		_, err := tronutils.DecodeCheck("0OIl")
		require.Error(t, err)
	})

	t.Run("too short", func(t *testing.T) {
		_, err := tronutils.DecodeCheck(tronutils.Encode([]byte{1, 2}))
		require.EqualError(t, err, "b58 check error")
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := tronutils.DecodeCheck(tronutils.Encode(make([]byte, 10)))
		require.EqualError(t, err, "invalid address length: 10")
	})

	t.Run("wrong prefix", func(t *testing.T) {
		bad := make([]byte, 25)
		bad[0] = 0x42 // not TronBytePrefix
		_, err := tronutils.DecodeCheck(tronutils.Encode(bad))
		require.EqualError(t, err, "invalid prefix")
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		bad := make([]byte, 25)
		bad[0] = tronutils.TronBytePrefix // valid prefix, zero (wrong) checksum
		_, err := tronutils.DecodeCheck(tronutils.Encode(bad))
		require.EqualError(t, err, "b58 check error")
	})
}
