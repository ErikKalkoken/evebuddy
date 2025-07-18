package storage_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestCorporationIndustryJob(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		now := time.Now().UTC()
		blueprintType := factory.CreateEveType()
		endDate := now.Add(12 * time.Hour)
		installer := factory.CreateEveEntityCharacter()
		startDate := now.Add(-6 * time.Hour)
		location := factory.CreateEveLocationStructure()
		arg := storage.UpdateOrCreateCorporationIndustryJobParams{
			ActivityID:          int32(app.Manufacturing),
			BlueprintID:         42,
			BlueprintLocationID: 11,
			BlueprintTypeID:     blueprintType.ID,
			CorporationID:       c.ID,
			Duration:            123,
			EndDate:             endDate,
			FacilityID:          12,
			InstallerID:         installer.ID,
			JobID:               1,
			OutputLocationID:    13,
			Runs:                7,
			StartDate:           startDate,
			Status:              app.JobActive,
			LocationID:          location.ID,
		}
		// when
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationIndustryJob(ctx, arg.CorporationID, arg.JobID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, 42, o.BlueprintID)
				assert.EqualValues(t, 11, o.BlueprintLocationID)
				assert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
				assert.EqualValues(t, 123, o.Duration)
				assert.Equal(t, endDate, o.EndDate)
				assert.EqualValues(t, 12, o.FacilityID)
				assert.Equal(t, installer, o.Installer)
				assert.EqualValues(t, 13, o.OutputLocationID)
				assert.EqualValues(t, 7, o.Runs)
				assert.Equal(t, startDate, o.StartDate)
				assert.Equal(t, app.JobActive, o.Status)
				assert.EqualValues(
					t, &app.EveLocationShort{
						ID:             location.ID,
						Name:           optional.From(location.Name),
						SecurityStatus: optional.From(location.SolarSystem.SecurityStatus),
					},
					o.Location,
				)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		now := time.Now().UTC()
		installer := factory.CreateEveEntityCharacter()
		blueprintType := factory.CreateEveType()
		completedCharacter := factory.CreateEveEntityCharacter()
		completedDate := now
		endDate := now.Add(12 * time.Hour)
		pauseDate := now.Add(-3 * time.Hour)
		productType := factory.CreateEveType()
		startDate := now.Add(-6 * time.Hour)
		station := factory.CreateEveLocationStructure()
		arg := storage.UpdateOrCreateCorporationIndustryJobParams{
			ActivityID:           int32(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  11,
			BlueprintTypeID:      blueprintType.ID,
			CorporationID:        c.ID,
			CompletedCharacterID: completedCharacter.ID,
			CompletedDate:        completedDate,
			Cost:                 123.45,
			Duration:             123,
			EndDate:              endDate,
			FacilityID:           12,
			InstallerID:          installer.ID,
			LicensedRuns:         3,
			JobID:                1,
			OutputLocationID:     13,
			Runs:                 7,
			PauseDate:            pauseDate,
			Probability:          0.8,
			ProductTypeID:        productType.ID,
			StartDate:            startDate,
			Status:               app.JobActive,
			LocationID:           station.ID,
			SuccessfulRuns:       2,
		}
		// when
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationIndustryJob(ctx, arg.CorporationID, arg.JobID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, 42, o.BlueprintID)
				assert.EqualValues(t, 11, o.BlueprintLocationID)
				assert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
				assert.Equal(t, completedCharacter, o.CompletedCharacter.MustValue())
				assert.Equal(t, completedDate, o.CompletedDate.MustValue())
				assert.EqualValues(t, 123.45, o.Cost.MustValue())
				assert.EqualValues(t, 123, o.Duration)
				assert.Equal(t, endDate, o.EndDate)
				assert.EqualValues(t, 12, o.FacilityID)
				assert.Equal(t, installer, o.Installer)
				assert.EqualValues(t, 3, o.LicensedRuns.MustValue())
				assert.EqualValues(t, 13, o.OutputLocationID)
				assert.Equal(t, float32(0.8), o.Probability.MustValue())
				assert.Equal(t, productType.ID, o.ProductType.MustValue().ID)
				assert.Equal(t, pauseDate, o.PauseDate.MustValue())
				assert.EqualValues(t, 7, o.Runs)
				assert.Equal(t, startDate, o.StartDate)
				assert.Equal(t, app.JobActive, o.Status)
				assert.EqualValues(t, station.ID, o.Location.ID)
				assert.EqualValues(t, 2, o.SuccessfulRuns.MustValue())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		now := time.Now().UTC()
		blueprintType := factory.CreateEveType()
		endDate := now.Add(12 * time.Hour)
		installer := factory.CreateEveEntityCharacter()
		startDate := now.Add(-6 * time.Hour)
		station := factory.CreateEveLocationStructure()
		arg := storage.UpdateOrCreateCorporationIndustryJobParams{
			ActivityID:          int32(app.Manufacturing),
			BlueprintID:         42,
			BlueprintLocationID: 11,
			BlueprintTypeID:     blueprintType.ID,
			CorporationID:       c.ID,
			Duration:            123,
			EndDate:             endDate,
			FacilityID:          12,
			InstallerID:         installer.ID,
			JobID:               1,
			OutputLocationID:    13,
			Runs:                7,
			StartDate:           startDate,
			Status:              app.JobActive,
			LocationID:          station.ID,
		}
		if err := st.UpdateOrCreateCorporationIndustryJob(ctx, arg); err != nil {
			t.Fatal(err)
		}
		completedCharacter := factory.CreateEveEntityCharacter()
		completedDate := now
		pauseDate := now.Add(-3 * time.Hour)
		endDate2 := now.Add(20 * time.Hour)
		arg = storage.UpdateOrCreateCorporationIndustryJobParams{
			ActivityID:           int32(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  11,
			BlueprintTypeID:      blueprintType.ID,
			CorporationID:        c.ID,
			CompletedCharacterID: completedCharacter.ID,
			CompletedDate:        completedDate,
			Duration:             123,
			EndDate:              endDate2,
			FacilityID:           12,
			InstallerID:          installer.ID,
			JobID:                1,
			OutputLocationID:     13,
			PauseDate:            pauseDate,
			Runs:                 7,
			StartDate:            startDate,
			Status:               app.JobDelivered,
			LocationID:           station.ID,
			SuccessfulRuns:       5,
		}
		// when
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCorporationIndustryJob(ctx, arg.CorporationID, arg.JobID)
			if assert.NoError(t, err) {
				assert.Equal(t, completedCharacter.ID, o.CompletedCharacter.MustValue().ID)
				assert.Equal(t, completedDate, o.CompletedDate.MustValue())
				assert.Equal(t, endDate2, o.EndDate)
				assert.Equal(t, pauseDate, o.PauseDate.MustValue())
				assert.Equal(t, app.JobDelivered, o.Status)
				assert.EqualValues(t, 5, o.SuccessfulRuns.MustValue())
			}
		}
	})
	t.Run("can list jobs for all corporations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob()
		j2 := factory.CreateCorporationIndustryJob()
		// when
		s, err := st.ListAllCorporationIndustryJobs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(j1.JobID, j2.JobID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CorporationIndustryJob) int32 {
				return x.JobID
			}))
			assert.True(t, got.Equal(want))
		}
	})
	t.Run("can get jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		el := factory.CreateEveLocationEmptyStructure()
		j := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			LocationID: el.ID,
		})
		// when
		x, err := st.GetCorporationIndustryJob(ctx, j.CorporationID, j.JobID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, el.ID, x.Location.ID)
		}
	})
	t.Run("can list jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		el := factory.CreateEveLocationEmptyStructure()
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			LocationID: el.ID,
		})
		// when
		x, err := st.ListAllCorporationIndustryJobs(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 1)
		}
	})
	t.Run("can delete jobs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob()
		j2 := factory.CreateCorporationIndustryJob()
		// when
		err := st.DeleteCorporationIndustryJobs(ctx, j1.CorporationID)
		// then
		if assert.NoError(t, err) {
			oo, err := st.ListAllCorporationIndustryJobs(ctx)
			if assert.NoError(t, err) {
				corporationIDs := xslices.Map(oo, func(x *app.CorporationIndustryJob) int32 {
					return x.CorporationID
				})
				assert.NotContains(t, corporationIDs, j1.CorporationID)
				assert.Contains(t, corporationIDs, j2.CorporationID)
			}
		}
	})
}
