package statuscacheservice_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestInit(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can init a status cache with character, corporation and general sections", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bruce"})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
		section1 := app.SectionCharacterImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section1,
		})
		section2 := app.SectionEveTypes
		y1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section2,
		})
		er := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{Name: "Alpha"})
		r := factory.CreateCorporation(er.ID)
		section3 := app.SectionCorporationIndustryJobs
		z1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: r.ID,
			Section:       section3,
		})
		// when
		err := sc.InitCache(ctx)
		// then
		if assert.NoError(t, err) {
			x2, ok := sc.CharacterSection(c.ID, section1)
			assert.True(t, ok)
			assert.Equal(t, x1.CharacterID, x2.EntityID)
			assert.Equal(t, string(x1.Section), x2.SectionID)
			assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
			assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
			assert.Equal(t, x1.StartedAt, x2.StartedAt)
			assert.Equal(t, "Bruce", x2.EntityName)
			assert.Equal(t, section1.Timeout(), x2.Timeout)

			y2, ok := sc.GeneralSection(section2)
			assert.True(t, ok)
			assert.Equal(t, int32(app.GeneralSectionEntityID), y2.EntityID)
			assert.Equal(t, string(y1.Section), y2.SectionID)
			assert.Equal(t, y1.CompletedAt, y2.CompletedAt)
			assert.Equal(t, y1.ErrorMessage, y2.ErrorMessage)
			assert.Equal(t, y1.StartedAt, y2.StartedAt)
			assert.Equal(t, section2.Timeout(), y2.Timeout)

			z2, ok := sc.CorporationSection(r.ID, section3)
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
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("should report when all sections are up-to-date", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		for range 2 {
			c := factory.CreateCharacterFull()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
				sc.SetCorporationSection(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.SetGeneralSection(o)
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
			c := factory.CreateCharacterFull()
			characters = append(characters, c.ID)
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
				sc.SetCorporationSection(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.SetGeneralSection(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.CharacterSectionStatus{
			CharacterID:   characters[0],
			Section:       app.SectionCharacterLocation,
			SectionStatus: app.SectionStatus{ErrorMessage: "error"},
		}
		sc.SetCharacterSection(o)
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
			c := factory.CreateCharacterFull()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
				sc.SetCorporationSection(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.SetGeneralSection(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.CorporationSectionStatus{
			CorporationID: corporations[0],
			Section:       app.SectionCorporationIndustryJobs,
			SectionStatus: app.SectionStatus{ErrorMessage: "error"},
		}
		sc.SetCorporationSection(o)
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
			c := factory.CreateCharacterFull()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
				sc.SetCorporationSection(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.SetGeneralSection(o)
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.GeneralSectionStatus{
			Section:       app.SectionEveCharacters,
			SectionStatus: app.SectionStatus{ErrorMessage: "error"},
		}
		sc.SetGeneralSection(o)
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
			c := factory.CreateCharacterFull()
			for _, section := range characterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
				sc.SetCorporationSection(o)
			}
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.SetGeneralSection(o)
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
			c := factory.CreateCharacterFull()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
			sc.SetCorporationSection(o)
		}
		for _, section := range app.GeneralSections {
			o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
				Section:      section,
				ErrorMessage: "",
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			})
			sc.SetGeneralSection(o)
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
			c := factory.CreateCharacterFull()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
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
			sc.SetGeneralSection(o)
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
			c := factory.CreateCharacterFull()
			characters = append(characters, c.ID)
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
			}
			for _, section := range app.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetGeneralSection(o)
			}
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.CharacterSectionStatus{
			CharacterID:   characters[0],
			Section:       app.SectionCharacterLocation,
			SectionStatus: app.SectionStatus{CompletedAt: time.Now().Add(-1 * time.Hour)},
		}
		sc.SetCharacterSection(o)
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
			c := factory.CreateCharacterFull()
			for _, section := range app.CharacterSections {
				o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
					CharacterID:  c.ID,
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetCharacterSection(o)
			}
			for _, section := range app.GeneralSections {
				o := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
					Section:      section,
					ErrorMessage: "",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				})
				sc.SetGeneralSection(o)
			}
		}
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		o := &app.GeneralSectionStatus{
			Section:       app.SectionEveCharacters,
			SectionStatus: app.SectionStatus{CompletedAt: time.Now().Add(-30 * time.Hour)},
		}
		sc.SetGeneralSection(o)
		// when
		ss := sc.Summary()
		// then
		assert.Equal(t, app.StatusWorking, ss.Status())
		assert.Less(t, ss.ProgressP(), float32(1.0))
		assert.Equal(t, 0, ss.Errors)
	})
}

func TestCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("update and list characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacterFull()
		// when
		if err := sc.UpdateCharacters(ctx); err != nil {
			t.Fatal(err)
		}
		// then
		got := sc.ListCharacters()
		assert.Len(t, got, 1)
		assert.Equal(t, c.ID, got[0].ID)
		assert.Equal(t, c.EveCharacter.Name, got[0].Name)
	})
	t.Run("list character IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		if err := sc.UpdateCharacters(ctx); err != nil {
			panic(err)
		}
		// when
		got := sc.ListCharacterIDs()
		// then
		want := set.Of(c1.ID, c2.ID)
		assert.True(t, got.Equal(want))
	})
}

func TestCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("update and list corporations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		ec := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{Name: "Alpha"})
		c := factory.CreateCorporation(ec.ID)
		// when
		if err := sc.UpdateCorporations(ctx); err != nil {
			t.Fatal(err)
		}
		// then
		xx := sc.ListCorporations()
		assert.Len(t, xx, 1)
		assert.Equal(t, c.ID, xx[0].ID)
		assert.Equal(t, "Alpha", xx[0].Name)
	})
}

func TestCharacterSections(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can get and set a character section status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacterFull()
		section := app.SectionCharacterImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// when
		sc.SetCharacterSection(x1)
		x2, ok := sc.CharacterSection(c.ID, section)
		// then
		assert.True(t, ok)
		assert.Equal(t, x1.CharacterID, x2.EntityID)
		assert.Equal(t, string(x1.Section), x2.SectionID)
		assert.Equal(t, x1.CompletedAt, x2.CompletedAt)
		assert.Equal(t, x1.ErrorMessage, x2.ErrorMessage)
		assert.Equal(t, x1.StartedAt, x2.StartedAt)
		assert.Equal(t, section.Timeout(), x2.Timeout)
	})
	t.Run("can report whether a character section exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacterFull()
		section := app.SectionCharacterImplants
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.True(t, sc.HasCharacterSection(c.ID, app.SectionCharacterImplants))
		assert.False(t, sc.HasCharacterSection(99, app.SectionCharacterImplants))
		assert.False(t, sc.HasCharacterSection(c.ID, app.SectionCharacterAssets))
		assert.False(t, sc.HasCharacterSection(0, app.SectionCharacterAssets))
	})
	t.Run("list character sections", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		x := sc.ListCharacterSections(c.ID)
		// then
		m := make(map[app.CharacterSection]app.CacheSectionStatus)
		for _, s := range x {
			m[app.CharacterSection(s.SectionID)] = s
		}
		got := set.Collect(maps.Keys(m))
		want := set.Of(app.CharacterSections...)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		assert.False(t, m[app.SectionCharacterAssets].IsMissing())
		assert.True(t, m[app.SectionCharacterImplants].IsMissing())
	})
	t.Run("list character sections all empty", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCharacterFull()
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		x := sc.ListCharacterSections(c.ID)
		// then
		m := make(map[app.CharacterSection]app.CacheSectionStatus)
		for _, s := range x {
			m[app.CharacterSection(s.SectionID)] = s
		}
		got := set.Collect(maps.Keys(m))
		want := set.Of(app.CharacterSections...)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		assert.True(t, m[app.SectionCharacterImplants].IsMissing())
	})
}

func TestCorporationSections(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can get and set a corporation section status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCorporation()
		section := app.SectionCorporationIndustryJobs
		x1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			Comment:       "comment",
		})
		// when
		sc.SetCorporationSection(x1)
		x2, ok := sc.CorporationSection(c.ID, section)
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
	t.Run("can report whether a corporation section exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCorporation()
		section := app.SectionCorporationIndustryJobs
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.True(t, sc.HasCorporationSection(c.ID, app.SectionCorporationIndustryJobs))
	})
	t.Run("list corporation sections 1", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		x := sc.ListCorporationSections(c.ID)
		// then
		m := make(map[app.CorporationSection]app.CacheSectionStatus)
		for _, s := range x {
			m[app.CorporationSection(s.SectionID)] = s
		}
		got := set.Collect(maps.Keys(m))
		want := set.Of(app.CorporationSections...)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		assert.False(t, m[app.SectionCorporationIndustryJobs].IsMissing())
	})
	t.Run("list corporation sections all empty", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		c := factory.CreateCorporation()
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		x := sc.ListCorporationSections(c.ID)
		// then
		m := make(map[app.CorporationSection]app.CacheSectionStatus)
		for _, s := range x {
			m[app.CorporationSection(s.SectionID)] = s
		}
		got := set.Collect(maps.Keys(m))
		want := set.Of(app.CorporationSections...)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		assert.True(t, m[app.SectionCorporationIndustryJobs].IsMissing())
	})
}

func TestGeneralSections(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cache := memcache.New()
	sc := statuscacheservice.New(cache, st)
	ctx := context.Background()
	t.Run("Can get and set a general section status", func(t *testing.T) {
		testutil.TruncateTables(db)
		cache.Clear()
		section := app.SectionEveTypes
		x1 := factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: section,
		})
		sc.SetGeneralSection(x1)
		x2, ok := sc.GeneralSection(section)
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
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveTypes,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.True(t, sc.HasGeneralSection(app.SectionEveTypes))
	})
	t.Run("can report whether a general section exists 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when/then
		assert.False(t, sc.HasGeneralSection(app.SectionEveTypes))
	})
	t.Run("list general sections", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cache.Clear()
		factory.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: app.SectionEveTypes,
		})
		if err := sc.InitCache(ctx); err != nil {
			t.Fatal(err)
		}
		// when
		x := sc.ListGeneralSections()
		// then
		m := make(map[app.GeneralSection]app.CacheSectionStatus)
		for _, s := range x {
			m[app.GeneralSection(s.SectionID)] = s
		}
		got := set.Collect(maps.Keys(m))
		want := set.Of(app.GeneralSections...)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		assert.False(t, m[app.SectionEveTypes].IsMissing())
		assert.True(t, m[app.SectionEveCharacters].IsMissing())
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
	sc.SetCharacterSection(&app.CharacterSectionStatus{
		CharacterID: characterID,
		Section:     app.SectionCharacterImplants,
		SectionStatus: app.SectionStatus{
			ErrorMessage: "ERROR",
		},
	})
	sc.SetCharacterSection(&app.CharacterSectionStatus{
		CharacterID: characterID,
		Section:     app.SectionCharacterAssets,
		SectionStatus: app.SectionStatus{
			CompletedAt: time.Now(),
		},
	})
	sc.SetCharacterSection(&app.CharacterSectionStatus{
		CharacterID: characterID,
		Section:     app.SectionCharacterIndustryJobs,
		SectionStatus: app.SectionStatus{
			StartedAt: time.Now().Add(-10 * time.Second),
		},
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
	sc.SetCorporationSection(&app.CorporationSectionStatus{
		CorporationID: corporationID,
		Section:       app.SectionCorporationIndustryJobs,
		SectionStatus: app.SectionStatus{ErrorMessage: "error"},
	})
	// when
	got := sc.CorporationSectionSummary(corporationID)
	// then
	want := app.StatusSummary{
		Current:   0,
		Errors:    1,
		IsRunning: false,
		Total:     len(app.CorporationSections),
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
	sc.SetGeneralSection(&app.GeneralSectionStatus{
		Section:       app.SectionEveTypes,
		SectionStatus: app.SectionStatus{ErrorMessage: "error"},
	})
	sc.SetGeneralSection(&app.GeneralSectionStatus{
		Section:       app.SectionEveCharacters,
		SectionStatus: app.SectionStatus{CompletedAt: time.Now()},
	})
	sc.SetGeneralSection(&app.GeneralSectionStatus{
		Section:       app.SectionEveMarketPrices,
		SectionStatus: app.SectionStatus{StartedAt: time.Now().Add(-10 * time.Second)},
	})
	// when
	got := sc.GeneralSectionSummary()
	// then
	want := app.StatusSummary{
		Current:   1,
		Errors:    1,
		IsRunning: true,
		Total:     5,
	}
	assert.Equal(t, want, got)
}
