package address

import (
	"strings"
	"testing"

	"github.com/decred/base58"
	"github.com/stretchr/testify/require"
)

func TestFromPrivateKeyEdgeCases(t *testing.T) {
	t.Run("valid hex but wrong length", func(t *testing.T) {
		_, err := FromPrivateKey("abcd") // decodes to 2 bytes, not 32
		require.Error(t, err)
	})
	t.Run("zero key fails ToECDSA", func(t *testing.T) {
		_, err := FromPrivateKey(strings.Repeat("0", 64))
		require.Error(t, err)
	})
}

func TestFromECDSANil(t *testing.T) {
	_, err := fromECDSA(nil, "")
	require.ErrorIs(t, err, ErrInvalidPrivateKey)
}

func TestPubKeyToAddressNil(t *testing.T) {
	require.Equal(t, "", pubKeyToAddress(nil))
}

func TestValidateLengthAndPrefix(t *testing.T) {
	t.Run("wrong length", func(t *testing.T) {
		// Valid checksum, but a 10-byte payload instead of 21.
		err := Validate(encodeCheck(make([]byte, 10)))
		require.ErrorContains(t, err, "invalid address length")
	})

	t.Run("wrong prefix", func(t *testing.T) {
		payload := make([]byte, addressLength)
		payload[0] = 0x42 // not the mainnet prefix
		err := Validate(encodeCheck(payload))
		require.ErrorContains(t, err, "invalid address prefix")
	})
}

func TestEncodeCheckDoesNotMutateInput(t *testing.T) {
	// Two payloads share one backing array; encoding the first must not corrupt
	// the second (append-aliasing regression).
	buf := make([]byte, 2*addressLength)
	for i := range buf {
		buf[i] = byte(i + 1)
	}
	first := buf[:addressLength]
	second := buf[addressLength:]
	want := append([]byte(nil), second...)

	_ = encodeCheck(first)

	require.Equal(t, want, second, "encodeCheck must not corrupt neighbouring bytes")
}

func TestDecodeCheckErrors(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		_, err := decodeCheck(base58.Encode([]byte{1, 2}))
		require.EqualError(t, err, "invalid encoded data")
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		// 25 raw bytes with a checksum that does not match the payload.
		_, err := decodeCheck(base58.Encode(make([]byte, 25)))
		require.EqualError(t, err, "checksum mismatch")
	})
}
