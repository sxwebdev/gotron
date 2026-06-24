package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestParseTRC20NumericProperty(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name    string
		data    string
		want    string // decimal string of big.Int
		wantErr bool
	}{
		{"value with 0x prefix", "0x" + fmt.Sprintf("%064x", 10), "10", false},
		{"value without prefix", fmt.Sprintf("%064x", 255), "255", false},
		{"empty is zero", "", "0", false},
		{"0x only is zero", "0x", "0", false},
		{"wrong length", "abcd", "", true},
		{"64 chars but not hex", strings.Repeat("g", 64), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.ParseTRC20NumericProperty(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
		})
	}
}

func TestParseTRC20StringProperty(t *testing.T) {
	c := &Client{}

	// 32-byte form: "USDT" followed by zero padding.
	short := "0x" + "55534454" + strings.Repeat("0", 64-8)

	// Long ABI form: [offset][length][data].
	long := "0x" + fmt.Sprintf("%064x", 32) + fmt.Sprintf("%064x", 10) + "54657468657220555344"

	tests := []struct {
		name    string
		data    string
		want    string
		wantErr bool
	}{
		{"32-byte utf8", short, "USDT", false},
		{"long abi string", long, "Tether USD", false},
		{"too short", "0xabcd", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.ParseTRC20StringProperty(tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// Regression: an invalid address must surface the underlying decode error,
// not repeat the address string.
func TestTRC20ContractBalanceInvalidAddressError(t *testing.T) {
	c := &Client{}

	const bad = "not-a-valid-address"
	_, decodeErr := tronutils.DecodeCheck(bad)
	require.Error(t, decodeErr)

	_, err := c.TRC20ContractBalance(context.Background(), bad, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.Error(t, err)
	require.ErrorContains(t, err, decodeErr.Error())
}

// Regression: a constant-call result with a nil Result must not panic.
func TestTRC20CallNilResultDoesNotPanic(t *testing.T) {
	c := newTestClient(&fakeTransport{
		triggerConstantContract: func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil // Result == nil
		},
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TRC20Call panicked on nil Result: %v", r)
		}
	}()

	res, err := c.TRC20Call(context.Background(), "", testAddr, trc20BalanceOf, true, 0)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestTRC20SendBuildsTransferData(t *testing.T) {
	var data []byte
	c := newTestClient(&fakeTransport{
		triggerContract: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			data = ct.Data
			return &api.TransactionExtention{Transaction: &core.Transaction{RawData: &core.TransactionRaw{}}}, nil
		},
	})

	_, err := c.TRC20Send(context.Background(), testAddr, testAddr2, testAddr, decimal.NewFromInt(1_000_000), 100*1e6)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 4+32+32)
	require.Equal(t, "a9059cbb", hex.EncodeToString(data[:4]), "transfer selector")
}

func TestTRC20ApproveAllowsZeroFeeLimit(t *testing.T) {
	var data []byte
	c := newTestClient(&fakeTransport{
		triggerContract: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			data = ct.Data
			return &api.TransactionExtention{}, nil
		},
	})

	// Approve permits feeLimit == 0 (unlike Send, which requires > 0).
	_, err := c.TRC20Approve(context.Background(), testAddr, testAddr2, testAddr, decimal.NewFromInt(1), 0)
	require.NoError(t, err)
	require.Equal(t, "095ea7b3", hex.EncodeToString(data[:4]), "approve selector")
}

func TestTRC20TransferFromBuildsData(t *testing.T) {
	var data []byte
	c := newTestClient(&fakeTransport{
		triggerContract: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			data = ct.Data
			return &api.TransactionExtention{Transaction: &core.Transaction{RawData: &core.TransactionRaw{}}}, nil
		},
	})

	_, err := c.TRC20TransferFrom(context.Background(), testAddr, testAddr2, testAddr, testAddr2, big.NewInt(5), 1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 4+32+32+32)
	require.Equal(t, "23b872dd", hex.EncodeToString(data[:4]), "transferFrom selector")
}

func TestTRC20SendValidation(t *testing.T) {
	c := &Client{}
	ctx := context.Background()
	const addr = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	tests := []struct {
		name              string
		from, to, ct      string
		amount            decimal.Decimal
		feeLimit          int64
		expect            string
	}{
		{"empty contract", addr, addr, "", decimal.NewFromInt(1), 1, "contract address is required"},
		{"empty from", "", addr, addr, decimal.NewFromInt(1), 1, "from address is required"},
		{"empty to", addr, "", addr, decimal.NewFromInt(1), 1, "to address is required"},
		{"zero amount", addr, addr, addr, decimal.Zero, 1, "amount must be greater than zero"},
		{"negative amount", addr, addr, addr, decimal.NewFromInt(-1), 1, "amount must be greater than zero"},
		{"zero fee limit", addr, addr, addr, decimal.NewFromInt(1), 0, "fee limit must be greater than zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.TRC20Send(ctx, tt.from, tt.to, tt.ct, tt.amount, tt.feeLimit)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.expect)
		})
	}
}
