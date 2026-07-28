package client

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// constantResultClient returns a Client whose constant calls answer with the
// given constant_result and a zero result code, which is what a degraded node
// does: TRC20Call treats code 0 as success no matter what the payload is.
func constantResultClient(constantResult [][]byte) *Client {
	return newTestClient(&fakeTransport{
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
				ConstantResult: constantResult,
			}, nil
		},
	})
}

// An empty constant_result used to be indexed at [0] without a length check.
// Because balances are fetched concurrently and errgroup does not recover
// panics, that took down the whole process instead of failing one call.
func TestTRC20ConstantCallsRejectEmptyConstantResult(t *testing.T) {
	t.Parallel()

	calls := map[string]func(*Client) error{
		"TRC20GetName": func(c *Client) error {
			_, err := c.TRC20GetName(context.Background(), testAddr)
			return err
		},
		"TRC20GetSymbol": func(c *Client) error {
			_, err := c.TRC20GetSymbol(context.Background(), testAddr)
			return err
		},
		"TRC20GetDecimals": func(c *Client) error {
			_, err := c.TRC20GetDecimals(context.Background(), testAddr)
			return err
		},
		"TRC20ContractBalance": func(c *Client) error {
			_, err := c.TRC20ContractBalance(context.Background(), testAddr2, testAddr)
			return err
		},
	}

	constantResults := map[string][][]byte{
		"nil constant_result":   nil,
		"empty constant_result": {},
	}

	for name, call := range calls {
		for resultName, constantResult := range constantResults {
			t.Run(name+"/"+resultName, func(t *testing.T) {
				t.Parallel()

				// require.NotPanics turns the original panic into a failure
				// instead of killing the test binary.
				var err error
				require.NotPanics(t, func() {
					err = call(constantResultClient(constantResult))
				})

				require.ErrorIs(t, err, ErrNilResponse)
				require.ErrorContains(t, err, testAddr, "error must name the contract")
			})
		}
	}
}

// An empty numeric property means the call returned nothing. Reporting it as the
// value zero made TRC20GetDecimals answer 0 for a 6-decimal token, so callers
// displayed and sent amounts off by a factor of a million.
func TestTRC20GetDecimalsRejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	c := constantResultClient([][]byte{{}})

	_, err := c.TRC20GetDecimals(context.Background(), testAddr)
	require.ErrorIs(t, err, ErrNilResponse)
}

func TestTRC20ContractBalanceRejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	c := constantResultClient([][]byte{{}})

	_, err := c.TRC20ContractBalance(context.Background(), testAddr2, testAddr)
	require.ErrorIs(t, err, ErrNilResponse)
	require.ErrorContains(t, err, testAddr)
}

func TestTRC20GetDecimalsValidPayload(t *testing.T) {
	t.Parallel()

	// 6 decimals, ABI-encoded as a 32-byte word.
	word := make([]byte, 32)
	word[31] = 6

	c := constantResultClient([][]byte{word})

	got, err := c.TRC20GetDecimals(context.Background(), testAddr)
	require.NoError(t, err)
	require.Equal(t, int64(6), got.Int64())
}

// captureCalldata returns a client that records the calldata of the single
// contract call it receives.
func captureCalldata(data *[]byte) *Client {
	return newTestClient(&fakeTransport{
		triggerContract: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			*data = ct.GetData()
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}},
			}, nil
		},
	})
}

// A fractional or oversized amount can no longer reach a TRC20 call: the
// TokenAmount constructors reject it. Previously decimal.BigInt truncated 0.5 to
// a transfer of nothing, and common.LeftPadBytes passed an over-32-byte amount
// through unpadded, misaligning the calldata.
func TestUnrepresentableTokenAmountNeverReachesACall(t *testing.T) {
	t.Parallel()

	t.Run("finer than the token", func(t *testing.T) {
		t.Parallel()

		var data []byte
		_ = captureCalldata(&data)

		_, err := units.FromTokenDecimal(decimal.NewFromFloat(0.5), 0)
		require.ErrorContains(t, err, "finer than the token")
		require.Nil(t, data)
	})

	t.Run("wider than uint256", func(t *testing.T) {
		t.Parallel()

		var data []byte
		_ = captureCalldata(&data)

		_, err := units.FromTokenUnits(new(big.Int).Lsh(big.NewInt(1), 256))
		require.ErrorContains(t, err, "needs 33 bytes, an ABI uint256 holds 32")
		require.Nil(t, data)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		_, err := units.FromTokenUnits(big.NewInt(-100))
		require.ErrorContains(t, err, "negative")
	})
}

// A zero amount is a valid TokenAmount - approve(0) is the standard way to
// revoke an allowance - so the per-method positivity rule still has to hold.
func TestTRC20SendRejectsZeroAmount(t *testing.T) {
	t.Parallel()

	var data []byte
	c := captureCalldata(&data)

	zero, err := units.FromTokenUnits(big.NewInt(0))
	require.NoError(t, err)

	_, err = c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, zero, 1_000_000)
	require.ErrorIs(t, err, ErrInvalidAmount)
	require.Nil(t, data)

	_, err = c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, TokenAmount{}, 1_000_000)
	require.ErrorIs(t, err, ErrInvalidAmount)
	require.Nil(t, data)
}

func TestTRC20SendEncodesAmountExactly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		units *big.Int
	}{
		{name: "typical", units: big.NewInt(1_500_000)},
		{name: "one", units: big.NewInt(1)},
		{name: "max uint256", units: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			amount, err := units.FromTokenUnits(tt.units)
			require.NoError(t, err)

			var data []byte
			c := captureCalldata(&data)

			_, err = c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, amount, 1_000_000)
			require.NoError(t, err)
			require.Len(t, data, 4+32+32, "selector plus two ABI words")
			require.Equal(t, 0, new(big.Int).SetBytes(data[36:]).Cmp(tt.units))
		})
	}
}

func TestTRC20TransferFromEncodesAmount(t *testing.T) {
	t.Parallel()

	amount, err := units.FromTokenUnits(big.NewInt(777))
	require.NoError(t, err)

	var data []byte
	c := captureCalldata(&data)

	_, err = c.TRC20TransferFrom(t.Context(), testAddr, testAddr, testAddr2, testAddr, amount, 1_000_000)
	require.NoError(t, err)
	require.Len(t, data, 4+32+32+32, "selector plus three ABI words")
	require.Equal(t, int64(777), new(big.Int).SetBytes(data[68:]).Int64())
}

func TestParseTRC20StringPropertyStillHandlesLongForm(t *testing.T) {
	t.Parallel()

	// Guard against the empty-payload change leaking into the string parser,
	// which feeds ParseTRC20NumericProperty a fixed 64-char slice.
	c := &Client{}
	offset := strings.Repeat("0", 62) + "20"
	length := strings.Repeat("0", 62) + "04"
	body := "54524f4e" + strings.Repeat("0", 56)

	got, err := c.ParseTRC20StringProperty("0x" + offset + length + body)
	require.NoError(t, err)
	require.Equal(t, "TRON", got)
}

// A zero TokenAmount is constructible - it is the zero value, and
// FromTokenUnits(0) succeeds - so every write path has to reject it itself. The
// constructors only rule out negative and oversized amounts.
//
// TRC20TransferFrom lost this check when amount encoding moved into the
// TokenAmount constructors: it built calldata with a zero uint256 and returned
// no error, so the caller signed and broadcast a transfer of nothing that still
// burned energy and bandwidth.
func TestTRC20WritesRejectZeroAmount(t *testing.T) {
	t.Parallel()

	zeroFromUnits, err := units.FromTokenUnits(big.NewInt(0))
	require.NoError(t, err, "zero is a valid amount to construct - that is the point")

	amounts := map[string]TokenAmount{
		"zero value":       {},
		"built from units": zeroFromUnits,
	}

	calls := map[string]func(*Client, TokenAmount) error{
		"TRC20Send": func(c *Client, a TokenAmount) error {
			_, err := c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, a, 1_000_000)
			return err
		},
		"TRC20Approve": func(c *Client, a TokenAmount) error {
			_, err := c.TRC20Approve(t.Context(), testAddr, testAddr2, testAddr, a, 1_000_000)
			return err
		},
		"TRC20TransferFrom": func(c *Client, a TokenAmount) error {
			_, err := c.TRC20TransferFrom(t.Context(), testAddr, testAddr, testAddr2, testAddr, a, 1_000_000)
			return err
		},
	}

	for method, call := range calls {
		for amountName, amount := range amounts {
			t.Run(method+"/"+amountName, func(t *testing.T) {
				t.Parallel()

				var data []byte
				c := captureCalldata(&data)

				err := call(c, amount)
				require.ErrorIs(t, err, ErrInvalidAmount)
				require.Nil(t, data, "no transaction may be built for a zero amount")
			})
		}
	}
}

// The string-property layout is [32-byte offset][32-byte length][payload]. The
// length word is contract-supplied, so a hostile or simply broken token can put
// anything in it.
//
// The bound used to be computed as 2*int(l): for l = 2^62+1 that product
// overflows int64 and wraps negative, so the "fits in the payload" check passed,
// while the slice bound 128+2*l stayed at 2^63+130 and panicked the caller.
// Reachable straight through TRC20GetName and TRC20GetSymbol.
func TestParseTRC20StringPropertyRejectsOverflowingLength(t *testing.T) {
	t.Parallel()

	// A 32-byte length word whose LOW 64 bits are the value big.Int.Uint64
	// returns; the high bytes are what makes the value exceed uint64 entirely.
	lengthWord := func(low, high string) string {
		return strings.Repeat("0", 64-len(low)-len(high)) + high + low
	}
	payload := strings.Repeat("41", 32) // "AAAA..."
	offset := strings.Repeat("0", 64)

	tests := []struct {
		name   string
		length string
		want   string
		wantOK bool
	}{
		{
			// 2*int(l) wraps to a negative number, 128+2*l does not.
			name:   "length that overflows the signed bound",
			length: lengthWord("4000000000000001", ""),
		},
		{
			// 2*l overflows uint64 as well, landing back on a small number.
			name:   "length that overflows the unsigned doubling",
			length: lengthWord("8000000000000001", ""),
		},
		{
			name:   "length wider than uint64",
			length: lengthWord("0000000000000004", "ff"),
		},
		{
			name:   "length just past the payload",
			length: lengthWord("0000000000000021", ""),
		},
		{
			name:   "maximum length word",
			length: strings.Repeat("f", 64),
		},
		{
			name:   "honest length still parses",
			length: lengthWord("0000000000000004", ""),
			want:   "AAAA",
			wantOK: true,
		},
		{
			name:   "length exactly filling the payload",
			length: lengthWord("0000000000000020", ""),
			want:   strings.Repeat("A", 32),
			wantOK: true,
		},
		{
			name:   "zero length is the empty string",
			length: lengthWord("0000000000000000", ""),
			want:   "",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Client{}
			got, err := c.ParseTRC20StringProperty(offset + tt.length + payload)

			if !tt.wantOK {
				require.Error(t, err, "an unusable length must be reported, never sliced")
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// The same length word reaching the parser through the public token getters.
func TestTRC20GetNameSurvivesHostileLengthWord(t *testing.T) {
	t.Parallel()

	data := strings.Repeat("0", 64) +
		strings.Repeat("0", 48) + "4000000000000001" +
		strings.Repeat("41", 32)

	c := newTestClient(&fakeTransport{
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			raw, err := hex.DecodeString(data)
			if err != nil {
				return nil, err
			}
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
				ConstantResult: [][]byte{raw},
			}, nil
		},
	})

	_, err := c.TRC20GetName(t.Context(), testAddr)
	require.Error(t, err)
}
