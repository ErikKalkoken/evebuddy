package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corp := factory.CreateEveEntityCorporation()
		race := factory.CreateEveRace()
		arg := storage.CreateEveCharacterParams{ID: 1, Name: "Erik", CorporationID: corp.ID, RaceID: race.ID}
		// when
		err := st.UpdateOrCreateEveCharacter(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := st.GetEveCharacter(ctx, arg.ID)
			if assert.NoError(t, err) {
				xassert.Equal(t, arg.Name, r.Name)
			}
		}
	})
	t.Run("can update existing 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCharacter()
		alliance2 := factory.CreateEveEntityAlliance()
		faction2 := factory.CreateEveEntityWithCategory(app.EveEntityFaction)
		// when
		err := st.UpdateOrCreateEveCharacter(ctx, storage.CreateEveCharacterParams{
			ID:             c1.ID,
			AllianceID:     optional.New(alliance2.ID),
			CorporationID:  c1.Corporation.ID,
			Description:    optional.New("new description"),
			FactionID:      optional.New(faction2.ID),
			Gender:         c1.Gender,
			Name:           "Erik",
			RaceID:         c1.Race.ID,
			SecurityStatus: optional.New(-9.9),
			Title:          optional.New("new title"),
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		c2, err := st.GetEveCharacter(ctx, c1.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, alliance2, c2.Alliance)
			xassert.Equal(t, faction2, c2.Faction)
			xassert.Equal(t, "Erik", c2.Name)
			xassert.Equal(t, "new description", c2.Description.ValueOrZero())
			xassert.Equal(t, "new title", c2.Title.ValueOrZero())
			xassert.Equal(t, -9.9, c2.SecurityStatus.ValueOrZero())
		}
	})
	t.Run("can update existing 2", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		c1.Name = "Erik"
		err := st.UpdateEveCharacter(ctx, c1)
		// then
		if assert.NoError(t, err) {
			c2, err := st.GetEveCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				xassert.Equal(t, "Erik", c2.Name)
			}
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveCharacter()
		// when
		err := st.DeleteEveCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := st.GetEveCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetEveCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		c2, err := st.GetEveCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, c1.Birthday.UTC(), c2.Birthday.UTC())
			xassert.Equal(t, c1.Corporation, c2.Corporation)
			xassert.Equal(t, c1.Description, c2.Description)
			xassert.Equal(t, c1.Gender, c2.Gender)
			xassert.Equal(t, c1.ID, c2.ID)
			xassert.Equal(t, c1.Name, c2.Name)
			xassert.Equal(t, c1.Race, c2.Race)
			xassert.Equal(t, c1.SecurityStatus, c2.SecurityStatus)
			xassert.Equal(t, c1.Title, c2.Title)
			assert.False(t, c2.HasAlliance())
			assert.False(t, c2.HasFaction())
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveCharacter()
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		arg := storage.CreateEveCharacterParams{
			AllianceID: optional.New(alliance.ID),
			FactionID:  optional.New(faction.ID),
		}
		c1 := factory.CreateEveCharacter(arg)
		// when
		c2, err := st.GetEveCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, alliance, c2.Alliance)
			xassert.Equal(t, c1.Birthday.UTC(), c2.Birthday.UTC())
			xassert.Equal(t, c1.Corporation, c2.Corporation)
			xassert.Equal(t, c1.Description, c2.Description)
			xassert.Equal(t, faction, c2.Faction)
			xassert.Equal(t, c1.ID, c2.ID)
			xassert.Equal(t, c1.Name, c2.Name)
		}
	})
	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCharacter()
		// when
		err := st.UpdateEveCharacterName(ctx, c1.ID, "Erik")
		// then
		if assert.NoError(t, err) {
			c2, err := st.GetEveCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				xassert.Equal(t, "Erik", c2.Name)
			}
		}
	})
}
