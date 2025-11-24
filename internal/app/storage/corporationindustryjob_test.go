package storage_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
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
		testutil.MustTruncateTables(db)
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
						Name:           optional.New(location.Name),
						SecurityStatus: optional.New(location.SolarSystem.SecurityStatus),
					},
					o.Location,
				)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			Status: app.JobActive,
		})
		// when
		completedCharacter := factory.CreateEveEntityCharacter()
		completedDate := time.Now()
		pauseDate := completedDate.Add(-3 * time.Hour)
		endDate2 := completedDate.Add(20 * time.Hour)
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, storage.UpdateOrCreateCorporationIndustryJobParams{
			ActivityID:           int32(j1.Activity),
			BlueprintID:          j1.BlueprintID,
			BlueprintLocationID:  j1.BlueprintLocationID,
			BlueprintTypeID:      j1.BlueprintType.ID,
			CorporationID:        j1.CorporationID,
			CompletedCharacterID: completedCharacter.ID,
			CompletedDate:        completedDate,
			Duration:             int32(j1.Duration),
			EndDate:              endDate2,
			FacilityID:           j1.FacilityID,
			InstallerID:          j1.Installer.ID,
			JobID:                j1.JobID,
			OutputLocationID:     j1.OutputLocationID,
			PauseDate:            pauseDate,
			Runs:                 int32(j1.Runs),
			StartDate:            j1.StartDate,
			Status:               app.JobDelivered,
			LocationID:           j1.Location.ID,
			SuccessfulRuns:       5,
		})
		// then
		if assert.NoError(t, err) {
			j2, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
			if assert.NoError(t, err) {
				assert.Equal(t, completedCharacter.ID, j2.CompletedCharacter.MustValue().ID)
				assert.True(t, j2.CompletedDate.MustValue().Equal(completedDate), "got %q, wanted %q", j2.CompletedDate.MustValue(), completedDate)
				assert.True(t, j2.EndDate.Equal(endDate2), "got %q, wanted %q", j2.EndDate, endDate2)
				assert.True(t, j2.PauseDate.MustValue().Equal(pauseDate), "got %q, wanted %q", j2.PauseDate.MustValue(), pauseDate)
				assert.Equal(t, app.JobDelivered, j2.Status)
				assert.EqualValues(t, 5, j2.SuccessfulRuns.MustValue())
			}
		}
	})
	t.Run("can list jobs for a corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
		})
		j2 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationIndustryJob()
		// when
		s, err := st.ListCorporationIndustryJobs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(j1.ID, j2.ID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CorporationIndustryJob) int64 {
				return x.ID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list jobs for all corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob()
		j2 := factory.CreateCorporationIndustryJob()
		// when
		s, err := st.ListAllCorporationIndustryJobs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(j1.ID, j2.ID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CorporationIndustryJob) int64 {
				return x.ID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can get jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		testutil.MustTruncateTables(db)
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
	t.Run("can delete all jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
	t.Run("can delete selected jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
		})
		j2 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
		})
		j3 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
		})
		j4 := factory.CreateCorporationIndustryJob()
		// when
		err := st.DeleteCorporationIndustryJobsByID(ctx, c.ID, set.Of(j1.JobID, j2.JobID))
		// then
		if assert.NoError(t, err) {
			oo, err := st.ListAllCorporationIndustryJobs(ctx)
			if assert.NoError(t, err) {
				got := set.Collect(xiter.MapSlice(oo, func(x *app.CorporationIndustryJob) int64 {
					return x.ID
				}))
				want := set.Of(j3.ID, j4.ID)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("can update status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			Status: app.JobActive,
		})
		// when
		err := st.UpdateCorporationIndustryJobStatus(ctx, storage.UpdateCorporationIndustryJobStatusParams{
			CorporationID: j1.CorporationID,
			JobIDs:        set.Of(j1.JobID),
			Status:        app.JobUnknown,
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		j2, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, app.JobUnknown, j2.Status)
	})
}
