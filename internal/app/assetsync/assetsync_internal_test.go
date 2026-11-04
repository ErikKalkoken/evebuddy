package assetsync

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestFetchNames(t *testing.T) {
	ids := make([]int64, 1000) // needs two requests
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	names := []ESIAssetName{
		{ItemId: 1, Name: "Alpha"},
		{ItemId: 2, Name: "None"},
	}
	t.Run("returns all IDs when all requests succeed", func(t *testing.T) {
		var requests int
		got, err := fetchNames(context.Background(), ids, func(_ context.Context, chunk []int64) ([]ESIAssetName, error) {
			requests++
			assert.LessOrEqual(t, len(chunk), namesMaxIDs)
			return names, nil
		})
		require.NoError(t, err)
		xassert.Equal(t, 2, requests)
		assert.Len(t, got, 1000)
		xassert.Equal(t, "Alpha", got[1])
		xassert.Equal(t, "", got[2])
		xassert.Equal(t, "", got[1000])
	})
	t.Run("leaves out IDs of failed request and reports error", func(t *testing.T) {
		wantErr := errors.New("failed")
		var requests int
		got, err := fetchNames(context.Background(), ids, func(_ context.Context, _ []int64) ([]ESIAssetName, error) {
			requests++
			if requests == 2 {
				return nil, wantErr
			}
			return names, nil
		})
		require.ErrorIs(t, err, wantErr)
		assert.Len(t, got, 999)
		xassert.Equal(t, "Alpha", got[1])
		assert.NotContains(t, got, int64(1000))
	})
	t.Run("continues after a failed request", func(t *testing.T) {
		var requests int
		got, err := fetchNames(context.Background(), ids, func(_ context.Context, _ []int64) ([]ESIAssetName, error) {
			requests++
			if requests == 1 {
				return nil, errors.New("failed")
			}
			return nil, nil
		})
		require.Error(t, err)
		xassert.Equal(t, 2, requests)
		assert.Len(t, got, 1)
		assert.Contains(t, got, int64(1000))
	})
	t.Run("stops when context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var requests int
		got, err := fetchNames(ctx, ids, func(_ context.Context, _ []int64) ([]ESIAssetName, error) {
			requests++
			cancel()
			return nil, context.Canceled
		})
		require.ErrorIs(t, err, context.Canceled)
		xassert.Equal(t, 1, requests)
		assert.Empty(t, got)
	})
	t.Run("makes no requests without IDs", func(t *testing.T) {
		got, err := fetchNames(context.Background(), nil, func(_ context.Context, _ []int64) ([]ESIAssetName, error) {
			t.Fatal("should not be called")
			return nil, nil
		})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
