package corporationservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestRemoveSectionDataWhenPermissionLost(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	t.Run("should do nothing when permission exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				Token: &app.CharacterToken{AccessToken: "accessToken"},
			},
		})
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
			j2, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
			if assert.NoError(t, err) {
				assert.Equal(t, j1.StartDate, j2.StartDate)
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
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				Error: app.ErrNotFound,
			},
		})
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
