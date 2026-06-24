package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestRoundRobinDistribution(t *testing.T) {
	calls := make([]int, 3)
	ts := make([]Transport, 3)
	for i := range ts {
		ts[i] = &fakeTransport{
			getChainParameters: func(context.Context) (*core.ChainParameters, error) {
				calls[i]++
				return &core.ChainParameters{}, nil
			},
		}
	}

	rr := NewRoundRobinTransport(ts)
	for range 9 {
		if _, err := rr.GetChainParameters(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	for i, n := range calls {
		if n != 3 {
			t.Errorf("transport %d called %d times, want 3 (even distribution)", i, n)
		}
	}
}

func TestRoundRobinOrderAndWrap(t *testing.T) {
	var order []int
	ts := make([]Transport, 3)
	for i := range ts {
		ts[i] = &fakeTransport{
			getChainParameters: func(context.Context) (*core.ChainParameters, error) {
				order = append(order, i)
				return &core.ChainParameters{}, nil
			},
		}
	}

	rr := NewRoundRobinTransport(ts)
	for range 4 {
		_, _ = rr.GetChainParameters(context.Background())
	}

	require.Equal(t, []int{0, 1, 2, 0}, order)
}

func TestRoundRobinCloseAggregatesError(t *testing.T) {
	boom := errors.New("boom")
	a := &fakeTransport{}
	b := &fakeTransport{closeFn: func() error { return boom }}
	d := &fakeTransport{}

	rr := NewRoundRobinTransport([]Transport{a, b, d})
	err := rr.Close()

	require.ErrorIs(t, err, boom)
	// Every transport must be closed even though one returned an error.
	require.Equal(t, 1, a.closeCalls)
	require.Equal(t, 1, b.closeCalls)
	require.Equal(t, 1, d.closeCalls)
}
