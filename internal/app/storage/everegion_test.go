package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestEveRegion(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.CreateEveRegionParams{
			ID:          42,
			Description: "description",
			Name:        "name",
		}
		// when
		x1, err := st.CreateEveRegion(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x2, err := st.GetEveRegion(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can list IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		r1 := factory.CreateEveRegion()
		r2 := factory.CreateEveRegion()
		// when
		got, err := st.ListEveRegionIDs(ctx)
		if assert.NoError(t, err) {
			want := set.Of(r1.ID, r2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can return missing IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		r1 := factory.CreateEveRegion(storage.CreateEveRegionParams{ID: 42})
		// when
		got, err := st.MissingEveRegions(ctx, set.Of(r1.ID, 99))
		if assert.NoError(t, err) {
			want := set.Of[int32](99)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
