package gotron

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func initClient() (*Tron, error) {
	// Initialize the Tron client for testing
	cfg := &Config{
		GRPCAddress: "tron-grpc.publicnode.com:443",
		UseTLS:      true,
		Timeout:     10 * time.Second,
	}
	client, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func TestGetNowBlock(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	block, err := c.GetNowBlock(context.Background())
	require.NoError(t, err)
	require.NotNil(t, block)

	fmt.Println(block.BlockHeader.RawData.Number)
}

func TestGetAccount(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	account, err := c.GetAccount(context.Background(), "TDEPJkL1dUGWvrJ4pFkFok2x2zoW3diFMh")
	require.NoError(t, err)
	require.NotNil(t, account)
}

func TestGetAccountBalance(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	balance, err := c.GetAccountBalance(context.Background(), "TDEPJkL1dUGWvrJ4pFkFok2x2zoW3diFMh")
	require.NoError(t, err)
	require.True(t, balance.GreaterThan(decimal.Zero))

	fmt.Println(balance.String())
}
