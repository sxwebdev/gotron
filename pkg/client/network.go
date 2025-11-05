package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
)

// ListNodes provides list of network nodes
func (c *Client) ListNodes(ctx context.Context) (*api.NodeList, error) {
	return c.walletClient.ListNodes(ctx, new(api.EmptyMessage))
}

// GetNextMaintenanceTime get next epoch timestamp
func (c *Client) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	return c.walletClient.GetNextMaintenanceTime(ctx, new(api.EmptyMessage))
}

// TotalTransaction return total transciton in network
func (c *Client) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	return c.walletClient.TotalTransaction(ctx, new(api.EmptyMessage))
}
