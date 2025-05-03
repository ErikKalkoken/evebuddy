package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveCorporation(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.CreateEveCorporationParams{
			ID:   1,
			Name: "Alpha",
		}
		// when
		err := r.CreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, arg.Name, r.Name)
			}
		}
	})
	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCorporation()
		// when
		c2, err := r.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.Name, c2.Name)
		}
	})
	// t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
	// 	// given
	// 	testutil.TruncateTables(db)
	// 	factory.CreateEveCharacter()
	// 	alliance := factory.CreateEveEntityAlliance()
	// 	faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
	// 	arg := storage.CreateEveCharacterParams{AllianceID: alliance.ID, FactionID: faction.ID}
	// 	c1 := factory.CreateEveCharacter(arg)
	// 	// when
	// 	c2, err := r.GetEveCharacter(ctx, c1.ID)
	// 	// then
	// 	if assert.NoError(t, err) {
	// 		assert.Equal(t, alliance, c2.Alliance)
	// 		assert.Equal(t, c1.Birthday.UTC(), c2.Birthday.UTC())
	// 		assert.Equal(t, c1.Corporation, c2.Corporation)
	// 		assert.Equal(t, c1.Description, c2.Description)
	// 		assert.Equal(t, faction, c2.Faction)
	// 		assert.Equal(t, c1.ID, c2.ID)
	// 		assert.Equal(t, c1.Name, c2.Name)
	// 	}
	// })
}
