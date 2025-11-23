package corporationservice

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestUpdateSectionIfChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(st, Params{CharacterService: &CharacterServiceFake{
		Token: &app.CharacterToken{AccessToken: "accessToken"},
	}})
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		var hasUpdated bool
		arg := app.CorporationSectionUpdateParams{CorporationID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
				return "any", nil
			},
			func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
			x, err := st.GetCorporationSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
				assert.False(t, x.HasError())
			}
		}
	})
	t.Run("should report as changed and run update when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		x1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			ErrorMessage:  "error",
			CompletedAt:   time.Now().Add(-5 * time.Second),
		})
		var hasUpdated bool
		arg := app.CorporationSectionUpdateParams{CorporationID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
				return "any", nil
			},
			func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
			x2, err := st.GetCorporationSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should report as unchanged and not run update when data has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		x1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			Data:          "old",
			CompletedAt:   time.Now().Add(-5 * time.Second),
		})
		hasUpdated := false
		arg := app.CorporationSectionUpdateParams{CorporationID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.False(t, hasUpdated)
			x2, err := st.GetCorporationSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should update when data has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationIndustryJobs
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			Data:          "old",
			CompletedAt:   time.Now().Add(-5 * time.Second),
		})
		var hasUpdated bool
		arg := app.CorporationSectionUpdateParams{CorporationID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
		}
	})
}

func TestHasSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(st)
	ctx := context.Background()
	t.Run("report true when section has changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// when
		got, err := s.hasSectionChanged(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		}, "changed",
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, got)
	})
	t.Run("report true when section does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		got, err := s.hasSectionChanged(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		}, "changed",
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, got)
	})
	t.Run("report false when section has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		status := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// when
		got, err := s.hasSectionChanged(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		}, status.ContentHash,
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, got)
	})
}
