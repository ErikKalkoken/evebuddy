package storage_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
			ActivityID:          int64(app.Manufacturing),
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
		require.NoError(t, err)
		o, err := st.GetCorporationIndustryJob(ctx, arg.CorporationID, arg.JobID)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.BlueprintID)
		xassert.Equal(t, 11, o.BlueprintLocationID)
		xassert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
		xassert.Equal(t, 123, o.Duration)
		xassert.Equal(t, endDate, o.EndDate)
		xassert.Equal(t, 12, o.FacilityID)
		xassert.Equal(t, installer, o.Installer)
		xassert.Equal(t, 13, o.OutputLocationID)
		xassert.Equal(t, 7, o.Runs)
		xassert.Equal(t, startDate, o.StartDate)
		xassert.Equal(t, app.JobActive, o.Status)
		xassert.Equal(
			t, &app.EveLocationShort{
				ID:             location.ID,
				Name:           optional.New(location.Name),
				SecurityStatus: optional.New(location.SolarSystem.ValueOrZero().SecurityStatus),
			},
			o.Location,
		)
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
			ActivityID:           int64(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  11,
			BlueprintTypeID:      blueprintType.ID,
			CorporationID:        c.ID,
			CompletedCharacterID: optional.New(completedCharacter.ID),
			CompletedDate:        optional.New(completedDate),
			Cost:                 optional.New(123.45),
			Duration:             123,
			EndDate:              endDate,
			FacilityID:           12,
			InstallerID:          installer.ID,
			LicensedRuns:         optional.New[int64](3),
			JobID:                1,
			OutputLocationID:     13,
			Runs:                 7,
			PauseDate:            optional.New(pauseDate),
			Probability:          optional.New(0.8),
			ProductTypeID:        optional.New(productType.ID),
			StartDate:            startDate,
			Status:               app.JobActive,
			LocationID:           station.ID,
			SuccessfulRuns:       optional.New[int64](2),
		}
		// when
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCorporationIndustryJob(ctx, arg.CorporationID, arg.JobID)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.BlueprintID)
		xassert.Equal(t, 11, o.BlueprintLocationID)
		xassert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
		xassert.EqualOptional(t, completedCharacter, o.CompletedCharacter)
		xassert.EqualOptional(t, completedDate, o.CompletedDate)
		xassert.EqualOptional(t, 123.45, o.Cost)
		xassert.Equal(t, 123, o.Duration)
		xassert.Equal(t, endDate, o.EndDate)
		xassert.Equal(t, 12, o.FacilityID)
		xassert.Equal(t, installer, o.Installer)
		xassert.EqualOptional(t, 3, o.LicensedRuns)
		xassert.Equal(t, 13, o.OutputLocationID)
		xassert.EqualOptional(t, float32(0.8), o.Probability)
		xassert.EqualOptional(t, eveTypeToEntityShort(productType), o.ProductType)
		xassert.EqualOptional(t, pauseDate, o.PauseDate)
		xassert.Equal(t, 7, o.Runs)
		xassert.Equal(t, startDate, o.StartDate)
		xassert.Equal(t, app.JobActive, o.Status)
		xassert.Equal(t, station.ID, o.Location.ID)
		xassert.EqualOptional(t, 2, o.SuccessfulRuns)
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
			ActivityID:           int64(j1.Activity),
			BlueprintID:          j1.BlueprintID,
			BlueprintLocationID:  j1.BlueprintLocationID,
			BlueprintTypeID:      j1.BlueprintType.ID,
			CorporationID:        j1.CorporationID,
			CompletedCharacterID: optional.New(completedCharacter.ID),
			CompletedDate:        optional.New(completedDate),
			Duration:             int64(j1.Duration),
			EndDate:              endDate2,
			FacilityID:           j1.FacilityID,
			InstallerID:          j1.Installer.ID,
			JobID:                j1.JobID,
			OutputLocationID:     j1.OutputLocationID,
			PauseDate:            optional.New(pauseDate),
			Runs:                 int64(j1.Runs),
			StartDate:            j1.StartDate,
			Status:               app.JobDelivered,
			LocationID:           j1.Location.ID,
			SuccessfulRuns:       optional.New[int64](5),
		})
		// then
		require.NoError(t, err)
		j2, err := st.GetCorporationIndustryJob(ctx, j1.CorporationID, j1.JobID)
		require.NoError(t, err)
		xassert.EqualOptional(t, completedCharacter, j2.CompletedCharacter)
		assert.True(t, j2.CompletedDate.MustValue().Equal(completedDate), "got %q, wanted %q", j2.CompletedDate.MustValue(), completedDate)
		assert.True(t, j2.EndDate.Equal(endDate2), "got %q, wanted %q", j2.EndDate, endDate2)
		assert.True(t, j2.PauseDate.MustValue().Equal(pauseDate), "got %q, wanted %q", j2.PauseDate.MustValue(), pauseDate)
		xassert.Equal(t, app.JobDelivered, j2.Status)
		xassert.EqualOptional(t, 5, j2.SuccessfulRuns)
	})
	t.Run("should return error when facility is missing", func(t *testing.T) {
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, storage.UpdateOrCreateCorporationIndustryJobParams{
			BlueprintLocationID: 1,
			BlueprintTypeID:     2,
			CorporationID:       3,
			InstallerID:         4,
			JobID:               7,
			LocationID:          6,
			OutputLocationID:    5,
		})
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should return error when job ID is missing", func(t *testing.T) {
		err := st.UpdateOrCreateCorporationIndustryJob(ctx, storage.UpdateOrCreateCorporationIndustryJobParams{
			BlueprintLocationID: 1,
			BlueprintTypeID:     2,
			CorporationID:       3,
			FacilityID:          7,
			InstallerID:         4,
			LocationID:          6,
			OutputLocationID:    5,
		})
		assert.ErrorIs(t, err, app.ErrInvalid)
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
		require.NoError(t, err)
		want := set.Of(j1.ID, j2.ID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CorporationIndustryJob) int64 {
			return x.ID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can list jobs for all corporations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob()
		j2 := factory.CreateCorporationIndustryJob()
		// when
		s, err := st.ListAllCorporationIndustryJobs(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(j1.ID, j2.ID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CorporationIndustryJob) int64 {
			return x.ID
		}))
		xassert.Equal(t, want, got)
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
		require.NoError(t, err)
		xassert.Equal(t, el.ID, x.Location.ID)
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
		require.NoError(t, err)
		assert.Len(t, x, 1)
	})
	t.Run("can delete all jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCorporationIndustryJob()
		j2 := factory.CreateCorporationIndustryJob()
		// when
		err := st.DeleteCorporationIndustryJobs(ctx, j1.CorporationID)
		// then
		require.NoError(t, err)
		oo, err := st.ListAllCorporationIndustryJobs(ctx)
		require.NoError(t, err)
		corporationIDs := xslices.Map(oo, func(x *app.CorporationIndustryJob) int64 {
			return x.CorporationID
		})
		assert.NotContains(t, corporationIDs, j1.CorporationID)
		assert.Contains(t, corporationIDs, j2.CorporationID)
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
		require.NoError(t, err)
		oo, err := st.ListAllCorporationIndustryJobs(ctx)
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CorporationIndustryJob) int64 {
			return x.ID
		}))
		want := set.Of(j3.ID, j4.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can count jobs with status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
		})
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobReady,
		})
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobDelivered,
		})
		// job for a different corporation with a matching status must not be counted
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			Status: app.JobActive,
		})
		// when
		got, err := st.CountCorporationIndustryJobsWithStatus(ctx, c.ID, set.Of(app.JobActive, app.JobReady))
		// then
		require.NoError(t, err)
		assert.Equal(t, int64(2), got)
	})
	t.Run("count returns error for empty statuses", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		_, err := st.CountCorporationIndustryJobsWithStatus(ctx, c.ID, set.Set[app.IndustryJobStatus]{})
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})

	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCorporationIndustryJob(t.Context(), 1, 2)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestReplaceCorporationIndustryJobs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		blueprintType := factory.CreateEveType()
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().UTC().Add(12 * time.Hour)
		arg := storage.CorporationIndustryJobParams{
			ActivityID:          int64(app.Manufacturing),
			BlueprintID:         42,
			BlueprintLocationID: station.ID,
			BlueprintTypeID:     blueprintType.ID,
			Duration:            123,
			EndDate:             endDate,
			FacilityID:          station.ID,
			InstallerID:         installer.ID,
			JobID:               1,
			OutputLocationID:    station.ID,
			Runs:                7,
			StartDate:           endDate.Add(-123 * time.Second),
			Status:              app.JobActive,
			LocationID:          station.ID,
		}
		// when
		changed, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCorporationIndustryJob(ctx, c.ID, 1)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.BlueprintID)
		xassert.Equal(t, endDate, o.EndDate)
		xassert.Equal(t, app.JobActive, o.Status)
	})
	t.Run("should report no change when jobs are unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		blueprintType := factory.CreateEveType()
		completedCharacter := factory.CreateEveEntityCharacter()
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().In(time.FixedZone("test", 3600)) // differs from UTC after the DB round trip
		arg := storage.CorporationIndustryJobParams{
			ActivityID:           int64(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  station.ID,
			BlueprintTypeID:      blueprintType.ID,
			CompletedCharacterID: optional.New(completedCharacter.ID),
			CompletedDate:        optional.New(endDate.Add(time.Hour)),
			Duration:             123,
			EndDate:              endDate,
			FacilityID:           station.ID,
			InstallerID:          installer.ID,
			JobID:                1,
			OutputLocationID:     station.ID,
			PauseDate:            optional.New(endDate.Add(-time.Hour)),
			Runs:                 7,
			StartDate:            endDate.Add(-123 * time.Second),
			Status:               app.JobDelivered,
			LocationID:           station.ID,
			SuccessfulRuns:       optional.New[int64](7),
		}
		_, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
		require.NoError(t, err)
		// when
		changed, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should store a job when a changeable column has changed", func(t *testing.T) {
		cases := []struct {
			name   string
			change func(arg *storage.CorporationIndustryJobParams, completedCharacterID int64)
		}{
			{"completed character", func(arg *storage.CorporationIndustryJobParams, completedCharacterID int64) {
				arg.CompletedCharacterID = optional.New(completedCharacterID)
			}},
			{"completed date", func(arg *storage.CorporationIndustryJobParams, _ int64) {
				arg.CompletedDate = optional.New(arg.EndDate.Add(time.Hour))
			}},
			{"end date", func(arg *storage.CorporationIndustryJobParams, _ int64) {
				arg.EndDate = arg.EndDate.Add(time.Hour)
			}},
			{"pause date", func(arg *storage.CorporationIndustryJobParams, _ int64) {
				arg.PauseDate = optional.New(arg.EndDate.Add(-time.Hour))
			}},
			{"status", func(arg *storage.CorporationIndustryJobParams, _ int64) {
				arg.Status = app.JobReady
			}},
			{"successful runs", func(arg *storage.CorporationIndustryJobParams, _ int64) {
				arg.SuccessfulRuns = optional.New[int64](3)
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCorporation()
				blueprintType := factory.CreateEveType()
				completedCharacter := factory.CreateEveEntityCharacter()
				installer := factory.CreateEveEntityCharacter()
				station := factory.CreateEveLocationStructure()
				endDate := time.Now().UTC().Add(12 * time.Hour)
				arg := storage.CorporationIndustryJobParams{
					ActivityID:          int64(app.Manufacturing),
					BlueprintID:         42,
					BlueprintLocationID: station.ID,
					BlueprintTypeID:     blueprintType.ID,
					Duration:            123,
					EndDate:             endDate,
					FacilityID:          station.ID,
					InstallerID:         installer.ID,
					JobID:               1,
					OutputLocationID:    station.ID,
					Runs:                7,
					StartDate:           endDate.Add(-123 * time.Second),
					Status:              app.JobActive,
					LocationID:          station.ID,
				}
				_, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
				require.NoError(t, err)
				tc.change(&arg, completedCharacter.ID)
				// when
				changed, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
				// then
				require.NoError(t, err)
				assert.True(t, changed)
				o, err := st.GetCorporationIndustryJob(ctx, c.ID, 1)
				require.NoError(t, err)
				xassert.Equal(t, arg.CompletedCharacterID.ValueOrZero(), optional.Map(o.CompletedCharacter, 0, func(x *app.EveEntity) int64 {
					return x.ID
				}))
				xassert.Equal(t, arg.CompletedDate, o.CompletedDate)
				xassert.Equal(t, arg.EndDate, o.EndDate)
				xassert.Equal(t, arg.PauseDate, o.PauseDate)
				xassert.Equal(t, arg.Status, o.Status)
				xassert.Equal(t, arg.SuccessfulRuns, o.SuccessfulRuns)
			})
		}
	})
	t.Run("should ignore changes to columns which can not change", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		blueprintType := factory.CreateEveType()
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().UTC().Add(12 * time.Hour)
		arg := storage.CorporationIndustryJobParams{
			ActivityID:          int64(app.Manufacturing),
			BlueprintID:         42,
			BlueprintLocationID: station.ID,
			BlueprintTypeID:     blueprintType.ID,
			Duration:            123,
			EndDate:             endDate,
			FacilityID:          station.ID,
			InstallerID:         installer.ID,
			JobID:               1,
			OutputLocationID:    station.ID,
			Runs:                7,
			StartDate:           endDate.Add(-123 * time.Second),
			Status:              app.JobActive,
			LocationID:          station.ID,
		}
		_, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
		require.NoError(t, err)
		arg.Runs = 8
		// when
		changed, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		o, err := st.GetCorporationIndustryJob(ctx, c.ID, 1)
		require.NoError(t, err)
		xassert.Equal(t, 7, o.Runs)
	})
	t.Run("should mark active jobs missing in args as unknown", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		active := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
		})
		paused := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobPaused,
		})
		ready := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobReady,
		})
		delivered := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobDelivered,
		})
		// when
		changed, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		jobs, err := st.ListCorporationIndustryJobs(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(jobs, func(j *app.CorporationIndustryJob) (int64, app.IndustryJobStatus) {
			return j.JobID, j.Status
		}))
		want := map[int64]app.IndustryJobStatus{
			active.JobID:    app.JobUnknown,
			paused.JobID:    app.JobUnknown,
			ready.JobID:     app.JobUnknown,
			delivered.JobID: app.JobDelivered,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("should report no change when only inactive jobs are missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobDelivered,
		})
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobUnknown,
		})
		// when
		changed, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should keep jobs unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		active := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
		})
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().UTC().Add(12 * time.Hour)
		arg := storage.CorporationIndustryJobParams{
			ActivityID:          int64(app.Manufacturing),
			BlueprintID:         42,
			BlueprintLocationID: station.ID,
			BlueprintTypeID:     42, // has no eve type
			Duration:            123,
			EndDate:             endDate,
			FacilityID:          station.ID,
			InstallerID:         installer.ID,
			JobID:               active.JobID + 1,
			OutputLocationID:    station.ID,
			Runs:                7,
			StartDate:           endDate.Add(-123 * time.Second),
			Status:              app.JobActive,
			LocationID:          station.ID,
		}
		// when
		_, err := st.ReplaceCorporationIndustryJobs(ctx, c.ID, []storage.CorporationIndustryJobParams{arg})
		// then
		require.Error(t, err)
		jobs, err := st.ListCorporationIndustryJobs(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(jobs, func(j *app.CorporationIndustryJob) (int64, app.IndustryJobStatus) {
			return j.JobID, j.Status
		}))
		want := map[int64]app.IndustryJobStatus{
			active.JobID: app.JobActive,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("should return error when IDs are invalid", func(t *testing.T) {
		cases := []struct {
			name          string
			corporationID int64
			args          []storage.CorporationIndustryJobParams
		}{
			{"zero corporation ID", 0, nil},
			{"zero job ID", 1, []storage.CorporationIndustryJobParams{{
				BlueprintLocationID: 2,
				BlueprintTypeID:     3,
				FacilityID:          4,
				InstallerID:         5,
				LocationID:          6,
				OutputLocationID:    7,
			}}},
			{"zero facility ID", 1, []storage.CorporationIndustryJobParams{{
				BlueprintLocationID: 2,
				BlueprintTypeID:     3,
				InstallerID:         5,
				JobID:               8,
				LocationID:          6,
				OutputLocationID:    7,
			}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCorporationIndustryJobs(ctx, tc.corporationID, tc.args)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
