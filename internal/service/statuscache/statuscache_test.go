package statuscache_test

import (
	"context"
	"testing"
	"time"

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

func TestStatusCacheSummary(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := cache.New()
	sc := statuscache.New(cache)
	// ctx := context.Background()
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range model.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range model.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		sc.InitCache(st)
		// when
		p, count := sc.Summary()
		// then
		assert.Equal(t, float32(1.0), p)
		assert.Equal(t, 0, count)
	})
	t.Run("should report error when a character section has an error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characters := make([]int32, 0)
		for range 2 {
			c := factory.CreateCharacter()
			characters = append(characters, c.ID)
			for _, section := range model.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range model.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		sc.InitCache(st)
		o := &model.CharacterSectionStatus{
			CharacterID:  characters[0],
			Section:      model.SectionLocation,
			ErrorMessage: "error",
		}
		sc.CharacterSectionSet(o)
		// when
		p, count := sc.Summary()
		// then
		assert.Less(t, p, float32(1.0))
		assert.Equal(t, 1, count)
	})
	t.Run("should report error when a general section has an error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range model.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range model.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		sc.InitCache(st)
		o := &model.GeneralSectionStatus{
			Section:      model.SectionEveCharacters,
			ErrorMessage: "error",
		}
		sc.GeneralSectionSet(o)
		// when
		p, count := sc.Summary()
		// then
		assert.Less(t, p, float32(1.0))
		assert.Equal(t, 1, count)
	})
	t.Run("should report current progress when a character section is stale", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		characters := make([]int32, 0)
		for range 2 {
			c := factory.CreateCharacter()
			characters = append(characters, c.ID)
			for _, section := range model.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range model.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		sc.InitCache(st)
		o := &model.CharacterSectionStatus{
			CharacterID: characters[0],
			Section:     model.SectionLocation,
			CompletedAt: time.Now().Add(-1 * time.Hour),
		}
		sc.CharacterSectionSet(o)
		// when
		p, count := sc.Summary()
		// then
		assert.Less(t, p, float32(1.0))
		assert.Equal(t, 0, count)
	})
	t.Run("should report current progress when a general section is stale", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range model.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range model.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		sc.InitCache(st)
		o := &model.GeneralSectionStatus{
			Section:     model.SectionEveCharacters,
			CompletedAt: time.Now().Add(-30 * time.Hour),
		}
		sc.GeneralSectionSet(o)
		// when
		p, count := sc.Summary()
		// then
		assert.Less(t, p, float32(1.0))
		assert.Equal(t, 0, count)
	})
}
