package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCorporationSectionStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// when
		oo, err := r.ListCorporationSectionStatus(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 1)
		}
	})
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		// when
		error := "error"
		arg := storage.UpdateOrCreateCorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
			ErrorMessage:  &error,
		}
		x1, err := r.UpdateOrCreateCorporationSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			if assert.NoError(t, err) {
				assert.Equal(t, "", x1.ContentHash)
				assert.Equal(t, "error", x1.ErrorMessage)
				assert.True(t, x1.CompletedAt.IsZero())
				assert.False(t, x1.UpdatedAt.IsZero())
			}
			x2, err := r.GetCorporationSectionStatus(ctx, c.ID, app.SectionCorporationIndustryJobs)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		x := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// when
		s := "error"
		arg := storage.UpdateOrCreateCorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       x.Section,
			ErrorMessage:  &s,
		}
		x1, err := r.UpdateOrCreateCorporationSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x.ContentHash, x1.ContentHash)
			assert.Equal(t, "error", x1.ErrorMessage)
			assert.Equal(t, x.CompletedAt, x1.CompletedAt)
			assert.Equal(t, x.StartedAt, x1.StartedAt)
			assert.False(t, x1.UpdatedAt.IsZero())
			x2, err := r.GetCorporationSectionStatus(ctx, c.ID, x.Section)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("can set all fields", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		x := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// when
		e := "error"
		comment := "comment"
		hash := "hash"
		startedAt := optional.From(time.Now())
		completedAt := storage.NewNullTimeFromTime(time.Now())
		arg := storage.UpdateOrCreateCorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       x.Section,
			ErrorMessage:  &e,
			Comment:       &comment,
			CompletedAt:   &completedAt,
			ContentHash:   &hash,
			StartedAt:     &startedAt,
		}
		x1, err := r.UpdateOrCreateCorporationSectionStatus(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, comment, x1.Comment)
			assert.Equal(t, hash, x1.ContentHash)
			assert.Equal(t, e, x1.ErrorMessage)
			assert.True(t, x1.CompletedAt.Equal(completedAt.Time))
			assert.True(t, x1.StartedAt.Equal(startedAt.ValueOrZero()))
			assert.False(t, x1.UpdatedAt.IsZero())
			x2, err := r.GetCorporationSectionStatus(ctx, c.ID, x.Section)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
