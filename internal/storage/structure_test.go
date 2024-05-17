package storage_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestStructure(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		owner := factory.CreateEveEntityCorporation()
		system := factory.CreateEveSolarSystem()
		myType := factory.CreateEveType()
		pos := model.Position{X: 1, Y: 2, Z: 3}
		arg := storage.CreateStructureParams{
			ID:               42,
			EveSolarSystemID: system.ID,
			EveTypeID:        sql.NullInt32{Int32: myType.ID, Valid: true},
			Name:             "Alpha",
			OwnerID:          owner.ID,
			Position:         pos,
		}
		// when
		err := r.CreateStructure(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetStructure(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, pos, x.Position)
				assert.Equal(t, owner, x.Owner)
				assert.Equal(t, system, x.SolarSystem)
				assert.Equal(t, myType, x.Type)
			}
		}
	})
	t.Run("can create minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		owner := factory.CreateEveEntityCorporation()
		system := factory.CreateEveSolarSystem()
		pos := model.Position{X: 1, Y: 2, Z: 3}
		arg := storage.CreateStructureParams{
			ID:               42,
			EveSolarSystemID: system.ID,
			Name:             "Alpha",
			OwnerID:          owner.ID,
			Position:         pos,
		}
		// when
		err := r.CreateStructure(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetStructure(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", x.Name)
				assert.Equal(t, pos, x.Position)
				assert.Equal(t, owner, x.Owner)
				assert.Equal(t, system, x.SolarSystem)
			}
		}
	})
}
