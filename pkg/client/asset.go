package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// GetAssetIssueById returns TRC10 asset information by its ID
func (c *Client) GetAssetIssueById(ctx context.Context, id string) (*core.AssetIssueContract, error) {
	return c.transport.GetAssetIssueById(ctx, []byte(id))
}

// GetAssetIssueListByName returns list of TRC10 assets matching the given name
func (c *Client) GetAssetIssueListByName(ctx context.Context, name string) (*api.AssetIssueList, error) {
	return c.transport.GetAssetIssueListByName(ctx, []byte(name))
}
