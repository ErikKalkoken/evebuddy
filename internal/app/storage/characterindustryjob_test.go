package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestCharacterIndustryJob(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
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
		c := factory.CreateCharacter()
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
		c := factory.CreateCharacter()
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
		arg = storage.UpdateOrCreateCharacterIndustryJobParams{
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
			Status:              app.JobDelivered,
			StationID:           station.ID,
		}
		// when
		err := st.UpdateOrCreateCharacterIndustryJob(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.JobDelivered, o.Status)
			}
		}
	})
	t.Run("can list jobs for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterIndustryJob()
		// when
		x, err := st.ListCharacterIndustryJob(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 2)
		}
	})
	t.Run("can list jobs for all characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterIndustryJob()
		// when
		x, err := st.ListAllCharacterIndustryJob(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, x, 3)
		}
	})
}
