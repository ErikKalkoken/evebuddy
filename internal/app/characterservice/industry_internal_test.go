package characterservice

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCharacterIndustryJobsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new job from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
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
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
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
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		completionDate := time.Now().UTC().Add(-2 * time.Hour)
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobActive,
			EndDate:     completionDate,
		})
		completer := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":            j1.Activity,
					"blueprint_id":           j1.BlueprintID,
					"blueprint_location_id":  j1.BlueprintLocation.ID,
					"blueprint_type_id":      j1.BlueprintType.ID,
					"completed_character_id": completer.ID,
					"completed_date":         completionDate.Format(app.DateTimeFormatESI),
					"cost":                   j1.Cost.ValueOrZero(),
					"duration":               j1.Duration,
					"end_date":               j1.EndDate.Format(app.DateTimeFormatESI),
					"facility_id":            j1.Facility.ID,
					"installer_id":           j1.Installer.ID,
					"job_id":                 j1.JobID,
					"licensed_runs":          j1.LicensedRuns.ValueOrZero(),
					"output_location_id":     j1.OutputLocation.ID,
					"runs":                   j1.Runs,
					"start_date":             j1.StartDate.Format(app.DateTimeFormatESI),
					"station_id":             j1.Station.ID,
					"status":                 "delivered",
					"successful_runs":        42,
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			xx, err := st.ListCharacterIndustryJobs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
				j2 := xx[0]
				assert.Equal(t, c.ID, j2.CharacterID)
				assert.Equal(t, app.JobDelivered, j2.Status)
				assert.EqualValues(t, 42, j2.SuccessfulRuns.MustValue())
				assert.WithinDuration(t, completionDate, j2.EndDate, time.Second)
				assert.WithinDuration(t, completionDate, j2.CompletedDate.ValueOrZero(), time.Second)
				assert.EqualValues(t, completer, j2.CompletedCharacter.MustValue())
			}
		}
	})
	t.Run("should fix incorrect status for new jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
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
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
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
	t.Run("should fix incorrect status for existing jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobActive,
			EndDate:     time.Now().UTC().Add(-2 * time.Hour),
		})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           j1.Activity,
					"blueprint_id":          j1.BlueprintID,
					"blueprint_location_id": j1.BlueprintLocation.ID,
					"blueprint_type_id":     j1.BlueprintType.ID,
					"cost":                  j1.Cost.ValueOrZero(),
					"duration":              j1.Duration,
					"end_date":              j1.EndDate.Format(app.DateTimeFormatESI),
					"facility_id":           j1.Facility.ID,
					"installer_id":          j1.Installer.ID,
					"job_id":                j1.JobID,
					"licensed_runs":         j1.LicensedRuns.ValueOrZero(),
					"output_location_id":    j1.OutputLocation.ID,
					"runs":                  j1.Runs,
					"start_date":            j1.StartDate.Format(app.DateTimeFormatESI),
					"station_id":            j1.Station.ID,
					"status":                "active",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			xx, err := st.ListCharacterIndustryJobs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
				j2 := xx[0]
				assert.Equal(t, c.ID, j2.CharacterID)
				assert.Equal(t, app.JobReady, j2.Status)
			}
		}
	})
	t.Run("should not fix status when correct", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
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
					"end_date":              endDate.Format(app.DateTimeFormatESI),
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            startDate.Format(app.DateTimeFormatESI),
					"station_id":            60006382,
					"status":                "active",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
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
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
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
				"end_date":              endDate.Format(app.DateTimeFormatESI),
				"facility_id":           60006382,
				"installer_id":          498338451,
				"job_id":                jobID,
				"licensed_runs":         200,
				"output_location_id":    60006382,
				"runs":                  1,
				"start_date":            startDate.Format(app.DateTimeFormatESI),
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
		_, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
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
	t.Run("should mark orphaned jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobDelivered,
		})
		j2 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobCancelled,
		})
		j3 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobActive,
		})
		j4 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobActive,
		})
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
					"job_id":                j3.JobID,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "active",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListAllCharacterIndustryJob(ctx)
			if assert.NoError(t, err) {
				got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterIndustryJob) (int32, app.IndustryJobStatus) {
					return x.JobID, x.Status
				}))
				want := map[int32]app.IndustryJobStatus{
					j1.JobID: app.JobDelivered,
					j2.JobID: app.JobCancelled,
					j3.JobID: app.JobReady,
					j4.JobID: app.JobUnknown,
				}
				assert.Equal(t, want, got)
			}
		}
	})
}
