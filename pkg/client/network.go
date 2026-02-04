package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
)

// ListNodes provides list of network nodes
func (c *Client) ListNodes(ctx context.Context) (*api.NodeList, error) {
	return c.transport.ListNodes(ctx)
}

// GetNextMaintenanceTime get next epoch timestamp
func (c *Client) GetNextMaintenanceTime(ctx context.Context) (*api.NumberMessage, error) {
	return c.transport.GetNextMaintenanceTime(ctx)
}

// TotalTransaction return total transciton in network
func (c *Client) TotalTransaction(ctx context.Context) (*api.NumberMessage, error) {
	return c.transport.TotalTransaction(ctx)
}
