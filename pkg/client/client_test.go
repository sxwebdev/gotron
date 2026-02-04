package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client"
)

func initClient() (*client.Client, error) {
	// Initialize the Tron client for testing
	cfg := client.Config{
		Nodes: []client.NodeConfig{
			{
				Protocol: client.ProtocolGRPC,
				Address:  "tron-grpc.publicnode.com:443",
				UseTLS:   true,
			},
		},
	}

	client, err := client.New(cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func TestGetNowBlock(t *testing.T) {
	c, err := initClient()
	require.NoError(t, err)

	number, err := c.GetLastBlockHeight(context.Background())
	require.NoError(t, err)

	fmt.Println(number)
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
