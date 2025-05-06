package statuscacheservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

func TestInit(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can init a status cache with character, corporation and general sections", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bruce"})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		section1 := app.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section1,
		})
		section2 := app.SectionEveCategories
		y1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section2,
		})
		er := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{Name: "Alpha"})
		r := factory.CreateCorporation(er.ID)
		section3 := app.SectionIndustryJobsCorporation
		z1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: r.ID,
			Section:       section3,
		})
		// when
		err := sc.InitCache(ctx)
		// then
		if assert.NoError(t, err) {
			x2, ok := sc.CharacterSectionGet(c.ID, section1)
			assert.True(t, ok)
			assert.Equal(t, x1.CharacterID, x2.EntityID)
			assert.Equal(t, string(x1.Section), x2.SectionID)
			assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
			assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
			assert.Equal(t, x1.StartedAt, x2.StartedAt)
			assert.Equal(t, "Bruce", x2.EntityName)
			assert.Equal(t, section1.Timeout(), x2.Timeout)

			y2, ok := sc.GeneralSectionGet(section2)
			assert.True(t, ok)
			assert.Equal(t, int32(app.GeneralSectionEntityID), y2.EntityID)
			assert.Equal(t, string(y1.Section), y2.SectionID)
			assert.Equal(t, y1.CompletedAt, y2.CompletedAt)
			assert.Equal(t, y1.ErrorMessage, y2.ErrorMessage)
			assert.Equal(t, y1.StartedAt, y2.StartedAt)
			assert.Equal(t, section2.Timeout(), y2.Timeout)

			z2, ok := sc.CorporationSectionGet(r.ID, section3)
			assert.True(t, ok)
			assert.Equal(t, z1.CorporationID, z2.EntityID)
			assert.Equal(t, string(z1.Section), z2.SectionID)
			assert.Equal(t, z1.CompletedAt, z2.CompletedAt)
			assert.Equal(t, z1.ErrorMessage, z2.ErrorMessage)
			assert.Equal(t, z1.StartedAt, z2.StartedAt)
			assert.Equal(t, "Alpha", z2.EntityName)
			assert.Equal(t, section3.Timeout(), z2.Timeout)
		}
	})
}

func TestStatusCacheSummary(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		for range 2 {
			c := factory.CreateCorporation()
			for _, section := range app.CorporationSections {
				o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
					CorporationID: c.ID,
					Section:       section,
					ErrorMessage:  "",
					StartedAt:     time.Now(),
					CompletedAt:   time.Now(),
				})
				sc.CorporationSectionSet(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusOK, ss.Status())
	})
	t.Run("should report when a character section has an error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		characters := make([]int32, 0)
		for range 2 {
			c := factory.CreateCharacter()
			characters = append(characters, c.ID)
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		for range 2 {
			c := factory.CreateCorporation()
			for _, section := range app.CorporationSections {
				o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
					CorporationID: c.ID,
					Section:       section,
					ErrorMessage:  "",
					StartedAt:     time.Now(),
					CompletedAt:   time.Now(),
				})
				sc.CorporationSectionSet(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.CharacterSectionStatus{
			CharacterID:  characters[0],
			Section:      app.SectionLocation,
			ErrorMessage: "error",
		}
		sc.CharacterSectionSet(o)
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusError, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 1, ss.Errors)
	})
	t.Run("corporation section has an error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		corporations := make([]int32, 0)
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		for range 2 {
			c := factory.CreateCorporation()
			corporations = append(corporations, c.ID)
			for _, section := range app.CorporationSections {
				o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
					CorporationID: c.ID,
					Section:       section,
					ErrorMessage:  "",
					StartedAt:     time.Now(),
					CompletedAt:   time.Now(),
				})
				sc.CorporationSectionSet(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.CorporationSectionStatus{
			CorporationID: corporations[0],
			Section:       app.SectionIndustryJobsCorporation,
			ErrorMessage:  "error",
		}
		sc.CorporationSectionSet(o)
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusError, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 1, ss.Errors)
	})
	t.Run("should report when a general section has an error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		for range 2 {
			c := factory.CreateCorporation()
			for _, section := range app.CorporationSections {
				o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
					CorporationID: c.ID,
					Section:       section,
					ErrorMessage:  "",
					StartedAt:     time.Now(),
					CompletedAt:   time.Now(),
				})
				sc.CorporationSectionSet(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.GeneralSectionStatus{
			Section:      app.SectionEveCharacters,
			ErrorMessage: "error",
		}
		sc.GeneralSectionSet(o)
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusError, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 1, ss.Errors)

	})
	t.Run("should report when a character section is missing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		characterSections := app.CharacterSections[:len(app.CharacterSections)-1]
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range characterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		for range 2 {
			c := factory.CreateCorporation()
			for _, section := range app.CorporationSections {
				o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
					CorporationID: c.ID,
					Section:       section,
					ErrorMessage:  "",
					StartedAt:     time.Now(),
					CompletedAt:   time.Now(),
				})
				sc.CorporationSectionSet(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusWorking, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 0, ss.Errors)
	})
	t.Run("should report when a corporation section is missing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		factory.CreateCorporation()
		corporation1 := factory.CreateCorporation()
		for _, section := range app.CorporationSections {
			o := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
				CorporationID: corporation1.ID,
				Section:       section,
				ErrorMessage:  "",
				StartedAt:     time.Now(),
				CompletedAt:   time.Now(),
			})
			sc.CorporationSectionSet(o)
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusWorking, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 0, ss.Errors)
	})
	t.Run("should report when a general section is missing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
		}
		generalSections := app.GeneralSections[:len(app.GeneralSections)-1]
		for _, section := range generalSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.GeneralSectionSet(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusWorking, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 0, ss.Errors)
	})
	t.Run("should report current progress when a character section is stale", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		characters := make([]int32, 0)
		for range 2 {
			c := factory.CreateCharacter()
			characters = append(characters, c.ID)
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range app.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.CharacterSectionStatus{
			CharacterID: characters[0],
			Section:     app.SectionLocation,
			CompletedAt: time.Now().Add(-1 * time.Hour),
		}
		sc.CharacterSectionSet(o)
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusWorking, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 0, ss.Errors)
	})
	t.Run("should report current progress when a general section is stale", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		for range 2 {
			c := factory.CreateCharacter()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.CharacterSectionSet(o)
			}
			for _, section := range app.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.GeneralSectionSet(o)
			}
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.GeneralSectionStatus{
			Section:     app.SectionEveCharacters,
			CompletedAt: time.Now().Add(-30 * time.Hour),
		}
		sc.GeneralSectionSet(o)
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusWorking, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 0, ss.Errors)
	})
}

func TestCharacterSections(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can get and set a character section status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacter()
		section := app.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// when
		sc.CharacterSectionSet(x1)
		x2, ok := sc.CharacterSectionGet(c.ID, section)
		// then
		assert.True(t, ok)
		assert.Equal(t, x1.CharacterID, x2.EntityID)
		assert.Equal(t, string(x1.Section), x2.SectionID)
		assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
		assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
		assert.Equal(t, x1.StartedAt, x2.StartedAt)
		assert.Equal(t, section.Timeout(), x2.Timeout)
	})
	t.Run("Can update and list characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bruce"})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		// when
		if err := sc.UpdateCharacters(ctx); err != nil {
			t.Fatal(err)
		}
		xx := sc.ListCharacters()
		// then
		assert.Len(t, xx, 1)
		assert.Equal(t, c.ID, xx[0].ID)
		assert.Equal(t, "Bruce", xx[0].Name)
	})
	t.Run("can report whether a character section exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacter()
		section := app.SectionImplants
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.True(t, sc.CharacterSectionExists(c.ID, app.SectionImplants))
		assert.False(t, sc.CharacterSectionExists(99, app.SectionImplants))
		assert.False(t, sc.CharacterSectionExists(c.ID, app.SectionAssets))
		assert.False(t, sc.CharacterSectionExists(0, app.SectionAssets))
	})
}

func TestCorporationSections(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can get and set a corporation section status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCorporation()
		section := app.SectionIndustryJobsCorporation
		x1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			Comment:       "comment",
		})
		// when
		sc.CorporationSectionSet(x1)
		x2, ok := sc.CorporationSectionGet(c.ID, section)
		// then
		assert.True(t, ok)
		assert.Equal(t, x1.CorporationID, x2.EntityID)
		assert.Equal(t, string(x1.Section), x2.SectionID)
		assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
		assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
		assert.Equal(t, x1.StartedAt, x2.StartedAt)
		assert.Equal(t, section.Timeout(), x2.Timeout)
		assert.Equal(t, x1.Comment, x2.Comment)
	})
	t.Run("Can update and list corporations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		ec := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{Name: "Alpha"})
		c := factory.CreateCorporation(ec.ID)
		// when
		if err := sc.UpdateCorporations(ctx); err != nil {
			t.Fatal(err)
		}
		xx := sc.ListCorporations()
		// then
		assert.Len(t, xx, 1)
		assert.Equal(t, c.ID, xx[0].ID)
		assert.Equal(t, "Alpha", xx[0].Name)
	})
	t.Run("can report whether a corporation section exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCorporation()
		section := app.SectionIndustryJobsCorporation
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.True(t, sc.CorporationSectionExists(c.ID, app.SectionIndustryJobsCorporation))
	})
}

func TestGeneralSections(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can get and set a general section status", func(t *testing.T) {
		testutil.TruncateTables(db)
		cache.Clear()
		section := app.SectionEveCategories
		x1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section,
		})
		sc.GeneralSectionSet(x1)
		x2, ok := sc.GeneralSectionGet(section)
		assert.True(t, ok)
		assert.Equal(t, int32(app.GeneralSectionEntityID), x2.EntityID)
		assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
		assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
		assert.Equal(t, x1.StartedAt, x2.StartedAt)
		assert.Equal(t, section.Timeout(), x2.Timeout)
	})
	t.Run("can report whether a general section exists 1", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		section := app.SectionEveCategories
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.True(t, sc.GeneralSectionExists(app.SectionEveCategories))
	})
	t.Run("can report whether a general section exists 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.False(t, sc.GeneralSectionExists(app.SectionEveCategories))
	})
}

func TestCharacterSectionSummary(t *testing.T) {
	cache := memcache.New()
	sc := statuscacheservice.New(cache, nil)
	// given
	const (
		characterID = 42
	)
	cache.Clear()
	sc.CharacterSectionSet(&app.CharacterSectionStatus{
		CharacterID:  characterID,
		Section:      app.SectionImplants,
		ErrorMessage: "ERROR",
	})
	sc.CharacterSectionSet(&app.CharacterSectionStatus{
		CharacterID: characterID,
		Section:     app.SectionAssets,
		CompletedAt: time.Now(),
	})
	sc.CharacterSectionSet(&app.CharacterSectionStatus{
		CharacterID: characterID,
		Section:     app.SectionIndustryJobs,
		StartedAt:   time.Now().Add(-10 * time.Second),
	})
	// when
	got := sc.CharacterSectionSummary(characterID)
	// then
	want := app.StatusSummary{
		Current:   1,
		Errors:    1,
		IsRunning: true,
		Total:     20,
	}
	assert.Equal(t, want, got)
}

func TestCorporationSectionSummary(t *testing.T) {
	cache := memcache.New()
	sc := statuscacheservice.New(cache, nil)
	// given
	const (
		corporationID = 42
	)
	sc.CorporationSectionSet(&app.CorporationSectionStatus{
		CorporationID: corporationID,
		Section:       app.SectionIndustryJobsCorporation,
		ErrorMessage:  "ERROR",
	})
	// when
	got := sc.CorporationSectionSummary(corporationID)
	// then
	want := app.StatusSummary{
		Current:   0,
		Errors:    1,
		IsRunning: false,
		Total:     1,
	}
	assert.Equal(t, want, got)
}
func TestGeneralSectionSummary(t *testing.T) {
	cache := memcache.New()
	sc := statuscacheservice.New(cache, nil)
	// given
	const (
		characterID = 42
	)
	cache.Clear()
	sc.GeneralSectionSet(&app.GeneralSectionStatus{
		Section:      app.SectionEveCategories,
		ErrorMessage: "ERROR",
	})
	sc.GeneralSectionSet(&app.GeneralSectionStatus{
		Section:     app.SectionEveCharacters,
		CompletedAt: time.Now(),
	})
	sc.GeneralSectionSet(&app.GeneralSectionStatus{
		Section: app.SectionEveMarketPrices,

		StartedAt: time.Now().Add(-10 * time.Second),
	})
	// when
	got := sc.GeneralSectionSummary()
	// then
	want := app.StatusSummary{
		Current:   1,
		Errors:    1,
		IsRunning: true,
		Total:     4,
	}
	assert.Equal(t, want, got)
}
