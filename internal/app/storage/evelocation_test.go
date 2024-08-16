package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
)

func TestLocation(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		updatedAt := time.Now()
		arg := storage.UpdateOrCreateLocationParams{
			ID:        42,
			Name:      "Alpha",
			UpdatedAt: updatedAt,
		}
		// when
		err := r.UpdateOrCreateEveLocation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveLocation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, updatedAt.UTC(), x.UpdatedAt.UTC())
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		owner := factory.CreateEveEntityCorporation()
		system := factory.CreateEveSolarSystem()
		myType := factory.CreateEveType()
		updatedAt := time.Now()
		arg := storage.UpdateOrCreateLocationParams{
			ID:               42,
			EveSolarSystemID: optional.New(system.ID),
			EveTypeID:        optional.New(myType.ID),
			Name:             "Alpha",
			OwnerID:          optional.New(owner.ID),
			UpdatedAt:        updatedAt,
		}
		// when
		err := r.UpdateOrCreateEveLocation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveLocation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, owner, x.Owner)
				assert.Equal(t, system, x.SolarSystem)
				assert.Equal(t, myType, x.Type)
				assert.Equal(t, updatedAt.UTC(), x.UpdatedAt.UTC())
			}
		}
	})
	t.Run("can get existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 42, Name: "Alpha"})
		// when
		x, err := r.GetEveLocation(ctx, 42)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", x.Name)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 42})
		owner := factory.CreateEveEntityCorporation()
		system := factory.CreateEveSolarSystem()
		myType := factory.CreateEveType()
		updatedAt := time.Now()
		arg := storage.UpdateOrCreateLocationParams{
			ID:               42,
			EveSolarSystemID: optional.New(system.ID),
			EveTypeID:        optional.New(myType.ID),
			Name:             "Alpha",
			OwnerID:          optional.New(owner.ID),
			UpdatedAt:        updatedAt,
		}
		// when
		err := r.UpdateOrCreateEveLocation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveLocation(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, owner, x.Owner)
				assert.Equal(t, system, x.SolarSystem)
				assert.Equal(t, myType, x.Type)
				assert.Equal(t, updatedAt.UTC(), x.UpdatedAt.UTC())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		l1 := factory.CreateLocationStructure()
		l2 := factory.CreateLocationStructure()
		// when
		got, err := r.ListEveLocation(ctx)
		if assert.NoError(t, err) {
			want := []*app.EveLocation{l1, l2}
			assert.Equal(t, want, got)
		}
	})
}
