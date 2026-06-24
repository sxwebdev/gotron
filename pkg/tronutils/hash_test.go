package tronutils_test

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
)

func TestKeccak256(t *testing.T) {
	// Keccak256 of the empty input is a well-known vector.
	got := hex.EncodeToString(tronutils.Keccak256([]byte{}))
	require.Equal(t, "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470", got)
}

func TestBytesToHash(t *testing.T) {
	t.Run("right aligned", func(t *testing.T) {
		h := tronutils.BytesToHash([]byte{0x01, 0x02, 0x03})
		require.Equal(t, byte(0x01), h[tronutils.HashLength-3])
		require.Equal(t, byte(0x03), h[tronutils.HashLength-1])
		require.Equal(t, byte(0x00), h[0])
	})

	t.Run("longer than 32 bytes is cropped from the left", func(t *testing.T) {
		in := make([]byte, 33)
		in[0] = 0xff  // dropped
		in[1] = 0xaa  // becomes h[0]
		in[32] = 0xbb // becomes h[31]
		h := tronutils.BytesToHash(in)
		require.Equal(t, byte(0xaa), h[0])
		require.Equal(t, byte(0xbb), h[tronutils.HashLength-1])
	})
}

func TestBigToHash(t *testing.T) {
	h := tronutils.BigToHash(big.NewInt(0x0102))
	require.Equal(t, byte(0x01), h[tronutils.HashLength-2])
	require.Equal(t, byte(0x02), h[tronutils.HashLength-1])
}

func TestHexToHash(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		h, err := tronutils.HexToHash("0x" + strings.Repeat("00", 31) + "2a")
		require.NoError(t, err)
		require.Equal(t, byte(0x2a), h[tronutils.HashLength-1])
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := tronutils.HexToHash("")
		require.Error(t, err)
	})
}

func TestHashMethods(t *testing.T) {
	h := tronutils.BytesToHash([]byte{0x01})

	require.Len(t, h.Bytes(), tronutils.HashLength)
	require.Equal(t, byte(0x01), h.Bytes()[tronutils.HashLength-1])

	require.Equal(t, big.NewInt(1), h.Big())

	wantHex := "0x" + strings.Repeat("00", 31) + "01"
	require.Equal(t, wantHex, h.Hex())
	require.Equal(t, wantHex, h.String())

	require.Equal(t, "000000…000001", h.TerminalString())
}
