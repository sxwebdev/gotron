package gotron

import (
	"context"
	"fmt"

	"github.com/sxwebdev/gotron/pb/api"
)

// GetNowBlock return TIP block
func (t *Tron) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	result, err := t.tronClient.GetNowBlock2(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, fmt.Errorf("Get block now: %v", err)
	}

	return result, nil
}
