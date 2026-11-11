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
)

func TestCharacterIndustryJob(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		now := time.Now().UTC()
		blueprintLocation := factory.CreateEveLocationStructure()
		blueprintType := factory.CreateEveType()
		endDate := now.Add(12 * time.Hour)
		facility := factory.CreateEveLocationStructure()
		installer := factory.CreateEveEntityCharacter()
		outputLocation := factory.CreateEveLocationStructure()
		startDate := now.Add(-6 * time.Hour)
		station := factory.CreateEveLocationStructure()
		arg := storage.UpdateOrCreateCharacterIndustryJobParams{
			ActivityID:          int64(app.Manufacturing),
			BlueprintID:         42,
			BlueprintLocationID: blueprintLocation.ID,
			BlueprintTypeID:     blueprintType.ID,
			CharacterID:         c.ID,
			Duration:            123,
			EndDate:             endDate,
			FacilityID:          facility.ID,
			InstallerID:         installer.ID,
			JobID:               1,
			OutputLocationID:    outputLocation.ID,
			Runs:                7,
			StartDate:           startDate,
			Status:              app.JobActive,
			StationID:           station.ID,
		}
		// when
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.BlueprintID)
		xassert.Equal(
			t, &app.EveLocationShort{
				ID:             blueprintLocation.ID,
				Name:           optional.New(blueprintLocation.Name),
				SecurityStatus: optional.New(blueprintLocation.SolarSystem.ValueOrZero().SecurityStatus),
			},
			o.BlueprintLocation,
		)
		xassert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
		xassert.Equal(t, 123, o.Duration)
		xassert.Equal(t, endDate, o.EndDate)
		xassert.Equal(
			t, &app.EveLocationShort{
				ID:             facility.ID,
				Name:           optional.New(facility.Name),
				SecurityStatus: optional.New(facility.SolarSystem.ValueOrZero().SecurityStatus),
			},
			o.Facility,
		)
		xassert.Equal(t, installer, o.Installer)
		xassert.Equal(
			t, &app.EveLocationShort{
				ID:             outputLocation.ID,
				Name:           optional.New(outputLocation.Name),
				SecurityStatus: optional.New(outputLocation.SolarSystem.ValueOrZero().SecurityStatus),
			},
			o.OutputLocation,
		)
		xassert.Equal(t, 7, o.Runs)
		xassert.Equal(t, startDate, o.StartDate)
		xassert.Equal(t, app.JobActive, o.Status)
		xassert.Equal(
			t, &app.EveLocationShort{
				ID:             station.ID,
				Name:           optional.New(station.Name),
				SecurityStatus: optional.New(station.SolarSystem.ValueOrZero().SecurityStatus),
			},
			o.Station,
		)
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		now := time.Now().UTC()
		installer := factory.CreateEveEntityCharacter()
		blueprintLocation := factory.CreateEveLocationStructure()
		blueprintType := factory.CreateEveType()
		completedCharacter := factory.CreateEveEntityCharacter()
		completedDate := now
		endDate := now.Add(12 * time.Hour)
		facility := factory.CreateEveLocationStructure()
		outputLocation := factory.CreateEveLocationStructure()
		pauseDate := now.Add(-3 * time.Hour)
		productType := factory.CreateEveType()
		startDate := now.Add(-6 * time.Hour)
		station := factory.CreateEveLocationStructure()
		arg := storage.UpdateOrCreateCharacterIndustryJobParams{
			ActivityID:           int64(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  blueprintLocation.ID,
			BlueprintTypeID:      blueprintType.ID,
			CharacterID:          c.ID,
			CompletedCharacterID: optional.New(completedCharacter.ID),
			CompletedDate:        optional.New(completedDate),
			Cost:                 optional.New(123.45),
			Duration:             123,
			EndDate:              endDate,
			FacilityID:           facility.ID,
			InstallerID:          installer.ID,
			LicensedRuns:         optional.New[int64](3),
			JobID:                1,
			OutputLocationID:     outputLocation.ID,
			Runs:                 7,
			PauseDate:            optional.New(pauseDate),
			Probability:          optional.New(0.8),
			ProductTypeID:        optional.New(productType.ID),
			StartDate:            startDate,
			Status:               app.JobActive,
			StationID:            station.ID,
			SuccessfulRuns:       optional.New[int64](2),
		}
		// when
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.BlueprintID)
		xassert.Equal(t, blueprintLocation.ID, o.BlueprintLocation.ID)
		xassert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
		xassert.EqualOptional(t, completedCharacter, o.CompletedCharacter)
		xassert.EqualOptional(t, completedDate, o.CompletedDate)
		xassert.EqualOptional(t, 123.45, o.Cost)
		xassert.Equal(t, 123, o.Duration)
		xassert.Equal(t, endDate, o.EndDate)
		xassert.Equal(t, facility.ID, o.Facility.ID)
		xassert.Equal(t, installer, o.Installer)
		xassert.EqualOptional(t, 3, o.LicensedRuns)
		xassert.Equal(t, outputLocation.ID, o.OutputLocation.ID)
		xassert.EqualOptional(t, float32(0.8), o.Probability)
		xassert.EqualOptional(t, eveTypeToEntityShort(productType), o.ProductType)
		xassert.EqualOptional(t, pauseDate, o.PauseDate)
		xassert.Equal(t, 7, o.Runs)
		xassert.Equal(t, startDate, o.StartDate)
		xassert.Equal(t, app.JobActive, o.Status)
		xassert.Equal(t, station.ID, o.Station.ID)
		xassert.EqualOptional(t, 2, o.SuccessfulRuns)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			Status: app.JobActive,
		})
		// when
		completedCharacter := factory.CreateEveEntityCharacter()
		completedDate := time.Now().UTC()
		pauseDate := completedDate.Add(-3 * time.Hour)
		endDate2 := completedDate.Add(20 * time.Hour)
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, storage.UpdateOrCreateCharacterIndustryJobParams{
			ActivityID:           int64(j1.Activity),
			BlueprintID:          j1.BlueprintID,
			BlueprintLocationID:  j1.BlueprintLocation.ID,
			BlueprintTypeID:      j1.BlueprintType.ID,
			CharacterID:          j1.CharacterID,
			CompletedCharacterID: optional.New(completedCharacter.ID),
			CompletedDate:        optional.New(completedDate),
			Duration:             int64(j1.Duration),
			EndDate:              endDate2,
			FacilityID:           j1.Facility.ID,
			InstallerID:          j1.Installer.ID,
			JobID:                j1.JobID,
			OutputLocationID:     j1.OutputLocation.ID,
			PauseDate:            optional.New(pauseDate),
			Runs:                 int64(j1.Runs),
			StartDate:            j1.StartDate,
			Status:               app.JobDelivered,
			StationID:            j1.Station.ID,
			SuccessfulRuns:       optional.New[int64](5),
		})
		// then
		require.NoError(t, err)
		j2, err := st.GetCharacterIndustryJob(ctx, j1.CharacterID, j1.JobID)
		require.NoError(t, err)
		xassert.EqualOptional(t, completedCharacter, j2.CompletedCharacter)
		xassert.EqualOptional(t, completedDate, j2.CompletedDate)
		xassert.Equal(t, endDate2, j2.EndDate)
		xassert.EqualOptional(t, pauseDate, j2.PauseDate)
		xassert.Equal(t, app.JobDelivered, j2.Status)
		xassert.EqualOptional(t, 5, j2.SuccessfulRuns)
	})
	t.Run("should return error when facility is missing", func(t *testing.T) {
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, storage.UpdateOrCreateCharacterIndustryJobParams{
			BlueprintLocationID: 1,
			BlueprintTypeID:     2,
			CharacterID:         3,
			InstallerID:         4,
			JobID:               7,
			OutputLocationID:    5,
			StationID:           6,
		})
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should return error when job ID is missing", func(t *testing.T) {
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, storage.UpdateOrCreateCharacterIndustryJobParams{
			BlueprintLocationID: 1,
			BlueprintTypeID:     2,
			CharacterID:         3,
			FacilityID:          7,
			InstallerID:         4,
			OutputLocationID:    5,
			StationID:           6,
		})
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("can list jobs for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		j2 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterIndustryJob()
		// when
		s, err := st.ListCharacterIndustryJobs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(j1.JobID, j2.JobID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterIndustryJob) int64 {
			return x.JobID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can list jobs for all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		j1 := factory.CreateCharacterIndustryJob()
		j2 := factory.CreateCharacterIndustryJob()
		// when
		s, err := st.ListAllCharacterIndustryJob(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(j1.JobID, j2.JobID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterIndustryJob) int64 {
			return x.JobID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can get jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		el := factory.CreateEveLocationEmptyStructure()
		j := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			BlueprintLocationID: el.ID,
			FacilityID:          el.ID,
			OutputLocationID:    el.ID,
			StationID:           el.ID,
		})
		// when
		x, err := st.GetCharacterIndustryJob(ctx, j.CharacterID, j.JobID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, el.ID, x.BlueprintLocation.ID)
		xassert.Equal(t, el.ID, x.Facility.ID)
		xassert.Equal(t, el.ID, x.OutputLocation.ID)
		xassert.Equal(t, el.ID, x.Station.ID)
	})
	t.Run("can list jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		el := factory.CreateEveLocationEmptyStructure()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			BlueprintLocationID: el.ID,
			FacilityID:          el.ID,
			OutputLocationID:    el.ID,
			StationID:           el.ID,
		})
		// when
		x, err := st.ListAllCharacterIndustryJob(ctx)
		// then
		require.NoError(t, err)
		assert.Len(t, x, 1)
	})
	t.Run("can list jobs activity counts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character1 := factory.CreateCharacterFull()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobDelivered,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int64(app.Reactions2),
			Status:      app.JobActive,
		})
		character2 := factory.CreateCharacterFull()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character2.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			InstallerID: character1.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobActive,
		})
		// when
		got, err := st.ListAllCharacterIndustryJobActiveCounts(ctx)
		// then
		require.NoError(t, err)
		want := []app.IndustryJobActivityCount{
			{InstallerID: character1.ID, Activity: app.Manufacturing, Status: app.JobActive, Count: 3},
			{InstallerID: character1.ID, Activity: app.Manufacturing, Status: app.JobReady, Count: 1},
			{InstallerID: character1.ID, Activity: app.Reactions2, Status: app.JobActive, Count: 1},
			{InstallerID: character2.ID, Activity: app.Manufacturing, Status: app.JobActive, Count: 1},
		}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can delete selected jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		j2 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		j3 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		j4 := factory.CreateCharacterIndustryJob()
		// when
		err := st.DeleteCharacterIndustryJobsByID(ctx, c.ID, set.Of(j1.JobID, j2.JobID))
		// then
		require.NoError(t, err)
		s, err := st.ListAllCharacterIndustryJob(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterIndustryJob) int64 {
			return x.ID
		}))
		want := set.Of(j3.ID, j4.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCharacterIndustryJob(t.Context(), 1, 2)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func eveTypeToEntityShort(eb *app.EveType) *app.EntityShort {
	if eb == nil {
		return nil
	}
	o := &app.EntityShort{
		ID:   eb.ID,
		Name: eb.Name,
	}
	return o
}

func TestReplaceCharacterIndustryJobs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		blueprintType := factory.CreateEveType()
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().UTC().Add(12 * time.Hour)
		arg := storage.CharacterIndustryJobParams{
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
			StationID:           station.ID,
		}
		// when
		changed, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterIndustryJob(ctx, c.ID, 1)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.BlueprintID)
		xassert.Equal(t, endDate, o.EndDate)
		xassert.Equal(t, app.JobActive, o.Status)
	})
	t.Run("should report no change when jobs are unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		blueprintType := factory.CreateEveType()
		completedCharacter := factory.CreateEveEntityCharacter()
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().In(time.FixedZone("test", 3600)) // differs from UTC after the DB round trip
		arg := storage.CharacterIndustryJobParams{
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
			StationID:            station.ID,
			SuccessfulRuns:       optional.New[int64](7),
		}
		_, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
		require.NoError(t, err)
		// when
		changed, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should store a job when a changeable column has changed", func(t *testing.T) {
		cases := []struct {
			name   string
			change func(arg *storage.CharacterIndustryJobParams, completedCharacterID int64)
		}{
			{"completed character", func(arg *storage.CharacterIndustryJobParams, completedCharacterID int64) {
				arg.CompletedCharacterID = optional.New(completedCharacterID)
			}},
			{"completed date", func(arg *storage.CharacterIndustryJobParams, _ int64) {
				arg.CompletedDate = optional.New(arg.EndDate.Add(time.Hour))
			}},
			{"end date", func(arg *storage.CharacterIndustryJobParams, _ int64) {
				arg.EndDate = arg.EndDate.Add(time.Hour)
			}},
			{"pause date", func(arg *storage.CharacterIndustryJobParams, _ int64) {
				arg.PauseDate = optional.New(arg.EndDate.Add(-time.Hour))
			}},
			{"status", func(arg *storage.CharacterIndustryJobParams, _ int64) {
				arg.Status = app.JobReady
			}},
			{"successful runs", func(arg *storage.CharacterIndustryJobParams, _ int64) {
				arg.SuccessfulRuns = optional.New[int64](3)
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCharacterFull()
				blueprintType := factory.CreateEveType()
				completedCharacter := factory.CreateEveEntityCharacter()
				installer := factory.CreateEveEntityCharacter()
				station := factory.CreateEveLocationStructure()
				endDate := time.Now().UTC().Add(12 * time.Hour)
				arg := storage.CharacterIndustryJobParams{
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
					StationID:           station.ID,
				}
				_, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
				require.NoError(t, err)
				tc.change(&arg, completedCharacter.ID)
				// when
				changed, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
				// then
				require.NoError(t, err)
				assert.True(t, changed)
				o, err := st.GetCharacterIndustryJob(ctx, c.ID, 1)
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
		c := factory.CreateCharacterFull()
		blueprintType := factory.CreateEveType()
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().UTC().Add(12 * time.Hour)
		arg := storage.CharacterIndustryJobParams{
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
			StationID:           station.ID,
		}
		_, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
		require.NoError(t, err)
		arg.Runs = 8
		// when
		changed, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		o, err := st.GetCharacterIndustryJob(ctx, c.ID, 1)
		require.NoError(t, err)
		xassert.Equal(t, 7, o.Runs)
	})
	t.Run("should mark active jobs missing in args as unknown", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		active := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobActive,
		})
		paused := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobPaused,
		})
		ready := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobReady,
		})
		delivered := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobDelivered,
		})
		// when
		changed, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		jobs, err := st.ListCharacterIndustryJobs(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(jobs, func(j *app.CharacterIndustryJob) (int64, app.IndustryJobStatus) {
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
		c := factory.CreateCharacterFull()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobDelivered,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobUnknown,
		})
		// when
		changed, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should keep jobs unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		active := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobActive,
		})
		installer := factory.CreateEveEntityCharacter()
		station := factory.CreateEveLocationStructure()
		endDate := time.Now().UTC().Add(12 * time.Hour)
		arg := storage.CharacterIndustryJobParams{
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
			StationID:           station.ID,
		}
		// when
		_, err := st.ReplaceCharacterIndustryJobs(ctx, c.ID, []storage.CharacterIndustryJobParams{arg})
		// then
		require.Error(t, err)
		jobs, err := st.ListCharacterIndustryJobs(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(jobs, func(j *app.CharacterIndustryJob) (int64, app.IndustryJobStatus) {
			return j.JobID, j.Status
		}))
		want := map[int64]app.IndustryJobStatus{
			active.JobID: app.JobActive,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("should return error when IDs are invalid", func(t *testing.T) {
		cases := []struct {
			name        string
			characterID int64
			args        []storage.CharacterIndustryJobParams
		}{
			{"zero character ID", 0, nil},
			{"zero job ID", 1, []storage.CharacterIndustryJobParams{{
				BlueprintLocationID: 2,
				BlueprintTypeID:     3,
				FacilityID:          4,
				InstallerID:         5,
				OutputLocationID:    6,
				StationID:           7,
			}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCharacterIndustryJobs(ctx, tc.characterID, tc.args)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
