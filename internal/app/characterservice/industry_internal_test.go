package characterservice

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCharacterIndustryJobsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new job from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.Manufacturing, x.Activity)
				assert.EqualValues(t, 1015116533326, x.BlueprintID)
				assert.EqualValues(t, 60006382, x.BlueprintLocation.ID)
				assert.EqualValues(t, 118.01, x.Cost.MustValue())
				assert.EqualValues(t, 548, x.Duration)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 56, 14, 0, time.UTC), x.EndDate)
				assert.EqualValues(t, 60006382, x.Facility.ID)
				assert.EqualValues(t, 498338451, x.Installer.ID)
				assert.EqualValues(t, 229136101, x.JobID)
				assert.EqualValues(t, 200, x.LicensedRuns.MustValue())
				assert.EqualValues(t, 60006382, x.OutputLocation.ID)
				assert.EqualValues(t, 1, x.Runs)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
				assert.EqualValues(t, 60006382, x.Station.ID)
				assert.Equal(t, app.JobReady, x.Status)
			}
		}
	})
	t.Run("should update existing job", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		blueprintType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		installer := factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID:         c.ID,
			BlueprintTypeID:     blueprintType.ID,
			InstallerID:         installer.ID,
			BlueprintLocationID: location.ID,
			OutputLocationID:    location.ID,
			FacilityID:          location.ID,
			StationID:           location.ID,
			Status:              app.JobActive,
			EndDate:             time.Now(),
		})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "delivered",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.Manufacturing, x.Activity)
				assert.EqualValues(t, 1015116533326, x.BlueprintID)
				assert.EqualValues(t, 60006382, x.BlueprintLocation.ID)
				assert.EqualValues(t, 118.01, x.Cost.MustValue())
				assert.EqualValues(t, 548, x.Duration)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 56, 14, 0, time.UTC), x.EndDate)
				assert.EqualValues(t, 60006382, x.Facility.ID)
				assert.EqualValues(t, 498338451, x.Installer.ID)
				assert.EqualValues(t, 229136101, x.JobID)
				assert.EqualValues(t, 200, x.LicensedRuns.MustValue())
				assert.EqualValues(t, 60006382, x.OutputLocation.ID)
				assert.EqualValues(t, 1, x.Runs)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
				assert.EqualValues(t, 60006382, x.Station.ID)
				assert.Equal(t, app.JobDelivered, x.Status)
			}
		}
	})
	t.Run("should fix incorrect status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {

				assert.Equal(t, app.JobReady, x.Status)
			}
		}
	})
	t.Run("should not fix status when correct", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now().Add(+3 * time.Hour)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              endDate.Format("2006-01-02T15:04:05Z"),
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            startDate.Format("2006-01-02T15:04:05Z"),
					"station_id":            60006382,
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.JobActive, x.Status)
			}
		}
	})
	t.Run("should support all activity IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now().Add(+3 * time.Hour)

		makeObj := func(jobID, activityID int32) map[string]any {
			template := map[string]any{
				"activity_id":           activityID,
				"blueprint_id":          1015116533326,
				"blueprint_location_id": 60006382,
				"blueprint_type_id":     2047,
				"cost":                  118.01,
				"duration":              548,
				"end_date":              endDate.Format("2006-01-02T15:04:05Z"),
				"facility_id":           60006382,
				"installer_id":          498338451,
				"job_id":                jobID,
				"licensed_runs":         200,
				"output_location_id":    60006382,
				"runs":                  1,
				"start_date":            startDate.Format("2006-01-02T15:04:05Z"),
				"station_id":            60006382,
				"status":                "active",
			}
			return maps.Clone(template)
		}
		objs := make([]map[string]any, 0)
		activities := []int32{
			int32(app.Manufacturing),
			int32(app.Copying),
			int32(app.Invention),
			int32(app.MaterialEfficiencyResearch),
			int32(app.TimeEfficiencyResearch),
			int32(app.Reactions1),
			int32(app.Reactions2),
		}
		for jobID, activityID := range activities {
			objs = append(objs, makeObj(int32(jobID), activityID))
		}

		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, objs))

		// when
		_, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			for jobID, activityID := range activities {
				j, err := st.GetCharacterIndustryJob(ctx, c.ID, int32(jobID))
				if assert.NoError(t, err) {
					assert.Equal(t, activityID, int32(j.Activity))
				}
			}
		}
	})
}
