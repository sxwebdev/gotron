package client

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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

// decimal.BigInt() truncates towards zero, so a fractional amount used to be
// encoded as a smaller value - 0.5 became a transfer of nothing.
func TestTRC20SendRejectsFractionalAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		amount decimal.Decimal
	}{
		{"below one", decimal.NewFromFloat(0.5)},
		{"fraction above one", decimal.RequireFromString("1.5")},
		{"tiny fraction", decimal.RequireFromString("1000000.000001")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data []byte
			c := captureCalldata(&data)

			_, err := c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, tt.amount, 1_000_000)
			require.ErrorIs(t, err, ErrInvalidAmount)
			require.ErrorContains(t, err, "whole number")
			require.Nil(t, data, "no transaction may be built for a fractional amount")
		})
	}
}

// common.LeftPadBytes returns the input untouched once it is 32 bytes or longer,
// so an oversized amount produced calldata of the wrong length that the contract
// decodes as different arguments entirely.
func TestTRC20SendRejectsOversizedAmount(t *testing.T) {
	t.Parallel()

	// 2^256 needs 33 bytes and cannot be an ABI uint256.
	tooBig := decimal.NewFromBigInt(new(big.Int).Lsh(big.NewInt(1), 256), 0)

	var data []byte
	c := captureCalldata(&data)

	_, err := c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, tooBig, 1_000_000)
	require.ErrorIs(t, err, ErrInvalidAmount)
	require.ErrorContains(t, err, "32 bytes")
	require.Nil(t, data)
}

// The largest valid uint256 must still go through, and the calldata must be
// exactly 4 + 32 + 32 bytes.
func TestTRC20SendAcceptsMaxUint256(t *testing.T) {
	t.Parallel()

	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	var data []byte
	c := captureCalldata(&data)

	_, err := c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, decimal.NewFromBigInt(maxUint256, 0), 1_000_000)
	require.NoError(t, err)
	require.Len(t, data, 4+32+32, "selector plus two ABI words")

	require.Equal(t, maxUint256, new(big.Int).SetBytes(data[36:]))
}

func TestTRC20SendEncodesAmountExactly(t *testing.T) {
	t.Parallel()

	var data []byte
	c := captureCalldata(&data)

	_, err := c.TRC20Send(t.Context(), testAddr, testAddr2, testAddr, decimal.NewFromInt(1_500_000), 1_000_000)
	require.NoError(t, err)
	require.Len(t, data, 4+32+32)
	require.Equal(t, int64(1_500_000), new(big.Int).SetBytes(data[36:]).Int64())
}

func TestTRC20ApproveRejectsFractionalAmount(t *testing.T) {
	t.Parallel()

	var data []byte
	c := captureCalldata(&data)

	_, err := c.TRC20Approve(t.Context(), testAddr, testAddr2, testAddr, decimal.NewFromFloat(0.5), 1_000_000)
	require.ErrorIs(t, err, ErrInvalidAmount)
	require.Nil(t, data)
}

// TRC20TransferFrom validated nothing at all: big.Int.Bytes() drops the sign, so
// a negative amount was encoded as a large positive transfer.
func TestTRC20TransferFromRejectsInvalidAmounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		amount *big.Int
	}{
		{"nil", nil},
		{"zero", big.NewInt(0)},
		{"negative", big.NewInt(-100)},
		{"oversized", new(big.Int).Lsh(big.NewInt(1), 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data []byte
			c := captureCalldata(&data)

			var err error
			require.NotPanics(t, func() {
				_, err = c.TRC20TransferFrom(t.Context(), testAddr, testAddr, testAddr2, testAddr, tt.amount, 1_000_000)
			})

			require.ErrorIs(t, err, ErrInvalidAmount)
			require.Nil(t, data)
		})
	}
}

func TestTRC20TransferFromEncodesAmount(t *testing.T) {
	t.Parallel()

	var data []byte
	c := captureCalldata(&data)

	_, err := c.TRC20TransferFrom(t.Context(), testAddr, testAddr, testAddr2, testAddr, big.NewInt(777), 1_000_000)
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
