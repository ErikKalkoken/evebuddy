package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestEveCharacter(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corp := factory.CreateEveEntityCorporation()
		race := factory.CreateEveRace()
		arg := storage.CreateEveCharacterParams{ID: 1, Name: "Erik", CorporationID: corp.ID, RaceID: race.ID}
		// when
		err := r.CreateEveCharacter(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetEveCharacter(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, arg.Name, r.Name)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		c1.Name = "Erik"
		err := r.UpdateEveCharacter(ctx, c1)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetEveCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "Erik", c2.Name)
			}
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateEveCharacter()
		// when
		err := r.DeleteEveCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetEveCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetEveCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		c2, err := r.GetEveCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, c1.Gender, c2.Gender)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
			assert.Equal(t, c1.Race, c2.Race)
			assert.Equal(t, c1.SecurityStatus, c2.SecurityStatus)
			assert.Equal(t, c1.Title, c2.Title)
			assert.False(t, c2.HasAlliance())
			assert.False(t, c2.HasFaction())
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveCharacter()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityFaction})
		arg := storage.CreateEveCharacterParams{AllianceID: alliance.ID, FactionID: faction.ID}
		c1 := factory.CreateEveCharacter(arg)
		// when
		c2, err := r.GetEveCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, c2.Alliance)
			assert.Equal(t, c1.Birthday.Unix(), c2.Birthday.Unix())
			assert.Equal(t, c1.Corporation, c2.Corporation)
			assert.Equal(t, c1.Description, c2.Description)
			assert.Equal(t, faction, c2.Faction)
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Name, c2.Name)
		}
	})
}
