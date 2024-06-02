package storage_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
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
			EveSolarSystemID: sql.NullInt32{Int32: system.ID, Valid: true},
			EveTypeID:        sql.NullInt32{Int32: myType.ID, Valid: true},
			Name:             "Alpha",
			OwnerID:          sql.NullInt32{Int32: owner.ID, Valid: true},
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
			EveSolarSystemID: sql.NullInt32{Int32: system.ID, Valid: true},
			EveTypeID:        sql.NullInt32{Int32: myType.ID, Valid: true},
			Name:             "Alpha",
			OwnerID:          sql.NullInt32{Int32: owner.ID, Valid: true},
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
}
