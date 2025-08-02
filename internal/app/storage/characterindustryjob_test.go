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
	"github.com/stretchr/testify/assert"
)

func TestCharacterIndustryJob(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
			ActivityID:          int32(app.Manufacturing),
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
		if assert.NoError(t, err) {
			o, err := st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, 42, o.BlueprintID)
				assert.EqualValues(
					t, &app.EveLocationShort{
						ID:             blueprintLocation.ID,
						Name:           optional.New(blueprintLocation.Name),
						SecurityStatus: optional.New(blueprintLocation.SolarSystem.SecurityStatus),
					},
					o.BlueprintLocation,
				)
				assert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
				assert.EqualValues(t, 123, o.Duration)
				assert.Equal(t, endDate, o.EndDate)
				assert.EqualValues(
					t, &app.EveLocationShort{
						ID:             facility.ID,
						Name:           optional.New(facility.Name),
						SecurityStatus: optional.New(facility.SolarSystem.SecurityStatus),
					},
					o.Facility,
				)
				assert.Equal(t, installer, o.Installer)
				assert.EqualValues(
					t, &app.EveLocationShort{
						ID:             outputLocation.ID,
						Name:           optional.New(outputLocation.Name),
						SecurityStatus: optional.New(outputLocation.SolarSystem.SecurityStatus),
					},
					o.OutputLocation,
				)
				assert.EqualValues(t, 7, o.Runs)
				assert.Equal(t, startDate, o.StartDate)
				assert.Equal(t, app.JobActive, o.Status)
				assert.EqualValues(
					t, &app.EveLocationShort{
						ID:             station.ID,
						Name:           optional.New(station.Name),
						SecurityStatus: optional.New(station.SolarSystem.SecurityStatus),
					},
					o.Station,
				)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
			ActivityID:           int32(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  blueprintLocation.ID,
			BlueprintTypeID:      blueprintType.ID,
			CharacterID:          c.ID,
			CompletedCharacterID: completedCharacter.ID,
			CompletedDate:        completedDate,
			Cost:                 123.45,
			Duration:             123,
			EndDate:              endDate,
			FacilityID:           facility.ID,
			InstallerID:          installer.ID,
			LicensedRuns:         3,
			JobID:                1,
			OutputLocationID:     outputLocation.ID,
			Runs:                 7,
			PauseDate:            pauseDate,
			Probability:          0.8,
			ProductTypeID:        productType.ID,
			StartDate:            startDate,
			Status:               app.JobActive,
			StationID:            station.ID,
			SuccessfulRuns:       2,
		}
		// when
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, 42, o.BlueprintID)
				assert.EqualValues(t, blueprintLocation.ID, o.BlueprintLocation.ID)
				assert.Equal(t, blueprintType.ID, o.BlueprintType.ID)
				assert.Equal(t, completedCharacter, o.CompletedCharacter.MustValue())
				assert.Equal(t, completedDate, o.CompletedDate.MustValue())
				assert.EqualValues(t, 123.45, o.Cost.MustValue())
				assert.EqualValues(t, 123, o.Duration)
				assert.Equal(t, endDate, o.EndDate)
				assert.EqualValues(t, facility.ID, o.Facility.ID)
				assert.Equal(t, installer, o.Installer)
				assert.EqualValues(t, 3, o.LicensedRuns.MustValue())
				assert.EqualValues(t, outputLocation.ID, o.OutputLocation.ID)
				assert.Equal(t, float32(0.8), o.Probability.MustValue())
				assert.Equal(t, productType.ID, o.ProductType.MustValue().ID)
				assert.Equal(t, pauseDate, o.PauseDate.MustValue())
				assert.EqualValues(t, 7, o.Runs)
				assert.Equal(t, startDate, o.StartDate)
				assert.Equal(t, app.JobActive, o.Status)
				assert.EqualValues(t, station.ID, o.Station.ID)
				assert.EqualValues(t, 2, o.SuccessfulRuns.MustValue())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
			ActivityID:          int32(app.Manufacturing),
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
		if err := st.UpdateOrCreateCharacterIndustryJob(ctx, arg); err != nil {
			t.Fatal(err)
		}
		completedCharacter := factory.CreateEveEntityCharacter()
		completedDate := now
		pauseDate := now.Add(-3 * time.Hour)
		endDate2 := now.Add(20 * time.Hour)
		arg = storage.UpdateOrCreateCharacterIndustryJobParams{
			ActivityID:           int32(app.Manufacturing),
			BlueprintID:          42,
			BlueprintLocationID:  blueprintLocation.ID,
			BlueprintTypeID:      blueprintType.ID,
			CharacterID:          c.ID,
			CompletedCharacterID: completedCharacter.ID,
			CompletedDate:        completedDate,
			Duration:             123,
			EndDate:              endDate2,
			FacilityID:           facility.ID,
			InstallerID:          installer.ID,
			JobID:                1,
			OutputLocationID:     outputLocation.ID,
			PauseDate:            pauseDate,
			Runs:                 7,
			StartDate:            startDate,
			Status:               app.JobDelivered,
			StationID:            station.ID,
			SuccessfulRuns:       5,
		}
		// when
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
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
	t.Run("can list jobs for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		if assert.NoError(t, err) {
			want := set.Of(j1.JobID, j2.JobID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterIndustryJob) int32 {
				return x.JobID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list jobs for all characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		j1 := factory.CreateCharacterIndustryJob()
		j2 := factory.CreateCharacterIndustryJob()
		// when
		s, err := st.ListAllCharacterIndustryJob(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(j1.JobID, j2.JobID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterIndustryJob) int32 {
				return x.JobID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can get jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		if assert.NoError(t, err) {
			assert.Equal(t, el.ID, x.BlueprintLocation.ID)
			assert.Equal(t, el.ID, x.Facility.ID)
			assert.Equal(t, el.ID, x.OutputLocation.ID)
			assert.Equal(t, el.ID, x.Station.ID)
		}
	})
	t.Run("can list jobs with incomplete locations", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		if assert.NoError(t, err) {
			assert.Len(t, x, 1)
		}
	})
	t.Run("can list jobs activity counts", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character1 := factory.CreateCharacterFull()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobDelivered,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character1.ID,
			ActivityID:  int32(app.Reactions2),
			Status:      app.JobActive,
		})
		character2 := factory.CreateCharacterFull()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character2.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			InstallerID: character1.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobActive,
		})
		// when
		got, err := st.ListAllCharacterIndustryJobActiveCounts(ctx)
		// then
		if assert.NoError(t, err) {
			want := []app.IndustryJobActivityCount{
				{InstallerID: character1.ID, Activity: app.Manufacturing, Status: app.JobActive, Count: 3},
				{InstallerID: character1.ID, Activity: app.Manufacturing, Status: app.JobReady, Count: 1},
				{InstallerID: character1.ID, Activity: app.Reactions2, Status: app.JobActive, Count: 1},
				{InstallerID: character2.ID, Activity: app.Manufacturing, Status: app.JobActive, Count: 1},
			}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("can delete selected jobs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		if assert.NoError(t, err) {
			s, err := st.ListAllCharacterIndustryJob(ctx)
			if err != nil {
				t.Fatal(err)
			}
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterIndustryJob) int64 {
				return x.ID
			}))
			want := set.Of(j3.ID, j4.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
