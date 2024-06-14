package statuscache_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestStatusCache(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := cache.New()
	sc := statuscache.New(cache)
	ctx := context.Background()
	t.Run("Can init a status cache with character and general sections", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bruce"})
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{ID: ec.ID})
		section1 := model.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section1,
		})
		section2 := model.SectionEveCategories
		y1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section2,
		})
		// when
		err := sc.InitCache(st)
		// then
		if assert.NoError(t, err) {
			x2 := sc.CharacterSectionGet(c.ID, section1)
			assert.Equal(t, x1.CharacterID, x2.CharacterID)
			assert.Equal(t, x1.Section, x2.Section)
			assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
			assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
			assert.Equal(t, x1.StartedAt, x2.StartedAt)
			assert.Equal(t, "Bruce", x2.CharacterName)

			y2 := sc.GeneralSectionGet(section2)
			assert.Equal(t, y1.Section, y2.Section)
			assert.Equal(t, y1.CompletedAt, y2.CompletedAt)
			assert.Equal(t, y1.ErrorMessage, y2.ErrorMessage)
			assert.Equal(t, y1.StartedAt, y2.StartedAt)
		}
	})
	t.Run("Can get and set a character section status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// when
		sc.CharacterSectionSet(x1)
		x2 := sc.CharacterSectionGet(c.ID, section)
		// then
		assert.Equal(t, x1.CharacterID, x2.CharacterID)
		assert.Equal(t, x1.Section, x2.Section)
		assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
		assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
		assert.Equal(t, x1.StartedAt, x2.StartedAt)
	})
	t.Run("Can get and set a general section status", func(t *testing.T) {
		testutil.TruncateTables(db)
		section := model.SectionEveCategories
		x1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section,
		})
		sc.GeneralSectionSet(x1)
		x2 := sc.GeneralSectionGet(section)
		assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
		assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
		assert.Equal(t, x1.StartedAt, x2.StartedAt)
	})
	t.Run("Can update and list characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bruce"})
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{ID: ec.ID})
		// when
		if err := sc.UpdateCharacters(ctx, st); err != nil {
			t.Fatal(err)
		}
		xx := sc.ListCharacters()
		// then
		assert.Len(t, xx, 1)
		assert.Equal(t, c.ID, xx[0].ID)
		assert.Equal(t, "Bruce", xx[0].Name)
	})
}
