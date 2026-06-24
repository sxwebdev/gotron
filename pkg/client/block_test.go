package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestGetLastBlockHeight(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getNowBlock: func(context.Context) (*api.BlockExtention, error) { return nil, nil },
		})
		_, err := c.GetLastBlockHeight(context.Background())
		require.ErrorIs(t, err, ErrNilResponse)
	})

	t.Run("missing header", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getNowBlock: func(context.Context) (*api.BlockExtention, error) {
				return &api.BlockExtention{}, nil
			},
		})
		_, err := c.GetLastBlockHeight(context.Background())
		require.ErrorIs(t, err, ErrNilResponse)
	})

	t.Run("success", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getNowBlock: func(context.Context) (*api.BlockExtention, error) {
				return &api.BlockExtention{
					BlockHeader: &core.BlockHeader{RawData: &core.BlockHeaderRaw{Number: 42}},
				}, nil
			},
		})
		h, err := c.GetLastBlockHeight(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint64(42), h)
	})
}

func TestGetBlockByHeight(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		c := newTestClient(&fakeTransport{
			getBlockByNum: func(context.Context, int64) (*api.BlockExtention, error) { return nil, nil },
		})
		_, err := c.GetBlockByHeight(context.Background(), 1)
		require.ErrorIs(t, err, ErrNilResponse)
	})

	t.Run("success passes height through", func(t *testing.T) {
		var gotNum int64
		c := newTestClient(&fakeTransport{
			getBlockByNum: func(_ context.Context, num int64) (*api.BlockExtention, error) {
				gotNum = num
				return &api.BlockExtention{}, nil
			},
		})
		_, err := c.GetBlockByHeight(context.Background(), 12345)
		require.NoError(t, err)
		require.Equal(t, int64(12345), gotNum)
	})
}
