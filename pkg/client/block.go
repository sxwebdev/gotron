package client

import (
	"context"

	"github.com/sxwebdev/gotron/schema/pb/api"
)

// GetLastBlock return TIP block
func (t *Client) GetLastBlock(ctx context.Context) (*api.BlockExtention, error) {
	result, err := t.walletClient.GetNowBlock2(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, err
	}

	return result, nil
}
