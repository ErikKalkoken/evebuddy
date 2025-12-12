package corporationservice

import (
	"context"
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateIndustryJobsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new job from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 11,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCorporationIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.Manufacturing, x.Activity)
				assert.EqualValues(t, 1015116533326, x.BlueprintID)
				assert.EqualValues(t, 2047, x.BlueprintType.ID)
				assert.EqualValues(t, 11, x.BlueprintLocationID)
				assert.EqualValues(t, 118.01, x.Cost.MustValue())
				assert.EqualValues(t, 548, x.Duration)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 56, 14, 0, time.UTC), x.EndDate)
				assert.EqualValues(t, 12, x.FacilityID)
				assert.EqualValues(t, 498338451, x.Installer.ID)
				assert.EqualValues(t, 229136101, x.JobID)
				assert.EqualValues(t, 200, x.LicensedRuns.MustValue())
				assert.EqualValues(t, 13, x.OutputLocationID)
				assert.EqualValues(t, 2046, x.ProductType.MustValue().ID)
				assert.EqualValues(t, 1, x.Runs)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
				assert.EqualValues(t, 60006382, x.Location.ID)
				assert.Equal(t, app.JobReady, x.Status)
			}
		}
	})
	t.Run("should update existing job", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		completionDate := time.Now().UTC().Add(-2 * time.Hour)
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
			EndDate:       completionDate,
		})
		completer := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":            j1.Activity,
					"blueprint_id":           j1.BlueprintID,
					"blueprint_location_id":  j1.BlueprintLocationID,
					"blueprint_type_id":      j1.BlueprintType.ID,
					"completed_character_id": completer.ID,
					"completed_date":         completionDate.Format(app.DateTimeFormatESI),
					"cost":                   j1.Cost.ValueOrZero(),
					"duration":               j1.Duration,
					"end_date":               j1.EndDate.Format(app.DateTimeFormatESI),
					"facility_id":            j1.FacilityID,
					"installer_id":           j1.Installer.ID,
					"job_id":                 j1.JobID,
					"licensed_runs":          j1.LicensedRuns.ValueOrZero(),
					"location_id":            j1.Location.ID,
					"output_location_id":     j1.OutputLocationID,
					"runs":                   j1.Runs,
					"start_date":             j1.StartDate.Format(app.DateTimeFormatESI),
					"status":                 "delivered",
					"successful_runs":        42,
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			xx, err := st.ListAllCorporationIndustryJobs(ctx)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
				j2 := xx[0]
				assert.Equal(t, c.ID, j2.CorporationID)
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
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 11,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCorporationIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.JobReady, x.Status)
			}
		}
	})
	t.Run("should fix incorrect status for existing jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
			EndDate:       time.Now().UTC().Add(-2 * time.Hour),
		})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           j1.Activity,
					"blueprint_id":          j1.BlueprintID,
					"blueprint_location_id": j1.BlueprintLocationID,
					"blueprint_type_id":     j1.BlueprintType.ID,
					"cost":                  j1.Cost.ValueOrZero(),
					"duration":              j1.Duration,
					"end_date":              j1.EndDate.Format(app.DateTimeFormatESI),
					"facility_id":           j1.FacilityID,
					"installer_id":          j1.Installer.ID,
					"job_id":                j1.JobID,
					"licensed_runs":         j1.LicensedRuns.ValueOrZero(),
					"location_id":           j1.Location.ID,
					"output_location_id":    j1.OutputLocationID,
					"runs":                  j1.Runs,
					"start_date":            j1.StartDate.Format(app.DateTimeFormatESI),
					"status":                "active",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			xx, err := st.ListAllCorporationIndustryJobs(ctx)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
				j2 := xx[0]
				assert.Equal(t, c.ID, j2.CorporationID)
				assert.Equal(t, app.JobReady, j2.Status)
			}
		}
	})
	t.Run("should not fix correct status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now().Add(+3 * time.Hour)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 11,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              endDate.Format(app.DateTimeFormatESI),
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            startDate.Format(app.DateTimeFormatESI),
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCorporationIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.JobActive, x.Status)
			}
		}
	})
	t.Run("should support all activity IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now().Add(+3 * time.Hour)

		makeObj := func(jobID, activityID int32) map[string]any {
			template := map[string]any{
				"activity_id":           activityID,
				"blueprint_id":          1015116533326,
				"blueprint_location_id": 11,
				"blueprint_type_id":     2047,
				"cost":                  118.01,
				"duration":              548,
				"end_date":              endDate.Format(app.DateTimeFormatESI),
				"facility_id":           12,
				"installer_id":          498338451,
				"job_id":                jobID,
				"licensed_runs":         200,
				"location_id":           60006382,
				"output_location_id":    13,
				"product_type_id":       2046,
				"runs":                  1,
				"start_date":            startDate.Format(app.DateTimeFormatESI),
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
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, objs))

		// when
		_, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			for jobID, activityID := range activities {
				j, err := st.GetCorporationIndustryJob(ctx, c.ID, int32(jobID))
				if assert.NoError(t, err) {
					assert.Equal(t, activityID, int32(j.Activity))
				}
			}
		}
	})
	t.Run("can fetch jobs from multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true&page=1`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 11,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"status":                "active",
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true&page=2`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 11,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                229136102,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"status":                "active",
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		_, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			jobs, err := st.ListAllCorporationIndustryJobs(ctx)
			if assert.NoError(t, err) {
				got := set.Of(xslices.Map(jobs, func(x *app.CorporationIndustryJob) int32 {
					return x.JobID
				})...)
				want := set.Of[int32](229136101, 229136102)
				xassert.EqualSet(t, want, got)
			}
		}
	})
	t.Run("should mark orphaned jobs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		j1 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobDelivered,
		})
		j2 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobCancelled,
		})
		j3 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
		})
		j4 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: c.ID,
			Status:        app.JobActive,
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 11,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                j3.JobID,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"status":                "active",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListAllCorporationIndustryJobs(ctx)
			if assert.NoError(t, err) {
				got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationIndustryJob) (int32, app.IndustryJobStatus) {
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
