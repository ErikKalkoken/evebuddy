package corporationservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestRemoveSectionDataWhenPermissionLost(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should do nothing when permission exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		const section = app.SectionCorporationIndustryJobs
		factory.CreateCorporationTokenForSection(corporation.ID, section)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       section,
		})
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		if assert.NoError(t, err) {
			j2, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
			if assert.NoError(t, err) {
				xassert.Equal(t, j1.StartDate, j2.StartDate)
			}
			status, err := st.GetCorporationSectionStatus(
				ctx,
				corporation.ID,
				app.SectionCorporationIndustryJobs,
			)
			if assert.NoError(t, err) {
				assert.True(t, status.HasContent())
			}
		}
	})
	t.Run("should delete secion when permission does not exit", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// when
		err := s.RemoveSectionDataWhenPermissionLost(ctx, corporation.ID)
		// then
		if assert.NoError(t, err) {
			_, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
			assert.ErrorIs(t, err, app.ErrNotFound)
			status, err := st.GetCorporationSectionStatus(
				ctx,
				corporation.ID,
				app.SectionCorporationIndustryJobs,
			)
			if assert.NoError(t, err) {
				assert.False(t, status.HasContent())
			}
		}
	})
}

func TestCorporationService_PermittedSections(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCorporationServiceFake(corporationservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should return section when matching token exists", func(t *testing.T) {
		// given
		section := app.SectionCorporationStructures
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(corporation.ID, section)
		// when
		got, err := s.PermittedSections(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		want := set.Of(section)
		xassert.Equal(t, want, got)
	})
	t.Run("should return no sections when permission does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: character.ID,
			Scopes:      app.SectionCharacterAssets.Scopes(),
		})
		// when
		got, err := s.PermittedSections(ctx, corporation.ID)
		// then
		require.NoError(t, err)
		want := set.Of[app.CorporationSection]()
		xassert.Equal(t, want, got)
	})
}
