package corporationservice

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHasSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(st)
	ctx := context.Background()
	t.Run("report true when section has changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		testutil.TruncateTables(db)
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
		testutil.TruncateTables(db)
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
