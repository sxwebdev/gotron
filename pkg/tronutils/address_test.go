package tronutils_test

import (
	"encoding/base64"
	"math/big"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
)

func TestAddressBytesHex(t *testing.T) {
	a := tronutils.Address([]byte{0x41, 0x02})
	require.Equal(t, []byte{0x41, 0x02}, a.Bytes())
	require.Equal(t, "0x4102", a.Hex())
}

func TestBigToAddress(t *testing.T) {
	a := tronutils.BigToAddress(big.NewInt(0x0102))
	require.Len(t, a, tronutils.AddressLength)
	require.Equal(t, byte(0x01), a[tronutils.AddressLength-2])
	require.Equal(t, byte(0x02), a[tronutils.AddressLength-1])
	require.Equal(t, byte(0x00), a[0])
}

func TestHexToAddress(t *testing.T) {
	require.Equal(t, tronutils.Address([]byte{0x41, 0x02}), tronutils.HexToAddress("0x4102"))
	require.Nil(t, tronutils.HexToAddress("0xzz"))
}

func TestBase58ToAddress(t *testing.T) {
	a, err := tronutils.Base58ToAddress(validTronAddr)
	require.NoError(t, err)
	require.Len(t, a, 21)
	require.Equal(t, tronutils.TronBytePrefix, a[0])

	_, err = tronutils.Base58ToAddress("0OIl")
	require.Error(t, err)
}

func TestBase64ToAddress(t *testing.T) {
	raw := []byte{0x41, 0x02, 0x03}
	a, err := tronutils.Base64ToAddress(base64.StdEncoding.EncodeToString(raw))
	require.NoError(t, err)
	require.Equal(t, tronutils.Address(raw), a)

	_, err = tronutils.Base64ToAddress("!!!not base64!!!")
	require.Error(t, err)
}

func TestAddressString(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Equal(t, "", tronutils.Address(nil).String())
	})

	t.Run("leading zero renders as big integer", func(t *testing.T) {
		require.Equal(t, "10", tronutils.Address([]byte{0x00, 0x0a}).String())
	})

	t.Run("tron address round trips through base58", func(t *testing.T) {
		decoded, err := tronutils.Base58ToAddress(validTronAddr)
		require.NoError(t, err)
		require.Equal(t, validTronAddr, decoded.String())
	})
}

func TestPubkeyToAddress(t *testing.T) {
	// Fixed key whose Tron address is known (cross-checked with pkg/address).
	priv, err := ethcrypto.HexToECDSA("ef80f4f95fe356c6405a0ff976ea8e7ee85caf6a9fd9f4a073ddf46b149733ee")
	require.NoError(t, err)

	a := tronutils.PubkeyToAddress(priv.PublicKey)
	require.Len(t, a, 21)
	require.Equal(t, tronutils.TronBytePrefix, a[0])
	require.Equal(t, "TEeKaYdpN6ujnpVZ1SkohE6Ru6gd9vGC2A", a.String())
}

func TestAddressScan(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		var a tronutils.Address
		src := make([]byte, tronutils.AddressLength)
		src[0] = tronutils.TronBytePrefix
		require.NoError(t, a.Scan(src))
		require.Equal(t, tronutils.Address(src), a)
	})
	t.Run("wrong type", func(t *testing.T) {
		var a tronutils.Address
		require.Error(t, a.Scan("not bytes"))
	})
	t.Run("wrong length", func(t *testing.T) {
		var a tronutils.Address
		require.Error(t, a.Scan(make([]byte, 10)))
	})
}

func TestAddressValue(t *testing.T) {
	a := tronutils.Address([]byte{1, 2, 3})
	v, err := a.Value()
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3}, v)
}
