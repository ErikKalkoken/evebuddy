package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestEveType(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		g := factory.CreateEveGroup()
		arg := storage.CreateEveTypeParams{
			ID:             42,
			Capacity:       3,
			Description:    "description",
			GraphicID:      4,
			GroupID:        g.ID,
			IconID:         5,
			IsPublished:    true,
			MarketGroupID:  6,
			Mass:           7,
			Name:           "name",
			PackagedVolume: 8,
			Radius:         9,
			Volume:         10,
		}
		// when
		err := r.CreateEveType(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveType(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), x.ID)
				assert.Equal(t, float32(3), x.Capacity)
				assert.Equal(t, "description", x.Description)
				assert.Equal(t, int32(4), x.GraphicID)
				assert.Equal(t, int32(5), x.IconID)
				assert.Equal(t, true, x.IsPublished)
				assert.Equal(t, int32(6), x.MarketGroupID)
				assert.Equal(t, float32(7), x.Mass)
				assert.Equal(t, "name", x.Name)
				assert.Equal(t, float32(8), x.PackagedVolume)
				assert.Equal(t, float32(9), x.Radius)
				assert.Equal(t, float32(10), x.Volume)
				assert.Equal(t, g, x.Group)
			}
		}
	})
	t.Run("can identify missing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 7})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 8})
		// when
		x, err := r.MissingEveTypes(ctx, []int32{7, 9})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []int32{9}, x)
		}
	})
}
