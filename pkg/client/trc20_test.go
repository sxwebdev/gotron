package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/pkg/units"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// mustTokenUnits builds a TokenAmount from raw minimal units for tests.
func mustTokenUnits(t *testing.T, v int64) TokenAmount {
	t.Helper()
	a, err := units.FromTokenUnits(big.NewInt(v))
	require.NoError(t, err)
	return a
}

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
		// An empty payload means the call returned nothing; reporting it as the
		// value zero silently corrupts decimals and balances.
		{"empty is an error", "", "", true},
		{"0x only is an error", "0x", "", true},
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

	_, err := c.TRC20Send(context.Background(), testAddr, testAddr2, testAddr, mustTokenUnits(t, 1_000_000), 100*units.SunPerTRX)
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
	_, err := c.TRC20Approve(context.Background(), testAddr, testAddr2, testAddr, mustTokenUnits(t, 1), 0)
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

	_, err := c.TRC20TransferFrom(context.Background(), testAddr, testAddr2, testAddr, testAddr2, mustTokenUnits(t, 5), 1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 4+32+32+32)
	require.Equal(t, "23b872dd", hex.EncodeToString(data[:4]), "transferFrom selector")
}

func TestTRC20SendValidation(t *testing.T) {
	c := &Client{}
	ctx := context.Background()
	const addr = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	tests := []struct {
		name         string
		from, to, ct string
		amount       TokenAmount
		feeLimit     SUN
		expect       string
	}{
		{"empty contract", addr, addr, "", mustTokenUnits(t, 1), 1, "contract address is required"},
		{"empty from", "", addr, addr, mustTokenUnits(t, 1), 1, "from address is required"},
		{"empty to", addr, "", addr, mustTokenUnits(t, 1), 1, "to address is required"},
		{"zero amount", addr, addr, addr, mustTokenUnits(t, 0), 1, "amount must be greater than zero"},
		{"zero fee limit", addr, addr, addr, mustTokenUnits(t, 1), 0, "fee limit must be greater than zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.TRC20Send(ctx, tt.from, tt.to, tt.ct, tt.amount, tt.feeLimit)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.expect)
		})
	}
}
