package characterservice

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestUpdateCharacterIndustryJobsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
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
		changed, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
		require.NoError(t, err)
		xassert.Equal(t, app.Manufacturing, x.Activity)
		xassert.Equal(t, 1015116533326, x.BlueprintID)
		xassert.Equal(t, 60006382, x.BlueprintLocation.ID)
		xassert.EqualOptional(t, 118.01, x.Cost)
		xassert.Equal(t, 548, x.Duration)
		xassert.Equal(t, time.Date(2014, 7, 19, 15, 56, 14, 0, time.UTC), x.EndDate)
		xassert.Equal(t, 60006382, x.Facility.ID)
		xassert.Equal(t, 498338451, x.Installer.ID)
		xassert.Equal(t, 229136101, x.JobID)
		xassert.EqualOptional(t, 200, x.LicensedRuns)
		xassert.Equal(t, 60006382, x.OutputLocation.ID)
		xassert.Equal(t, 1, x.Runs)
		xassert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
		xassert.Equal(t, 60006382, x.Station.ID)
		xassert.Equal(t, app.JobReady, x.Status)
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
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
		changed, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xx, err := st.ListCharacterIndustryJobs(ctx, c.ID)
		require.NoError(t, err)
		assert.Len(t, xx, 1)
		j2 := xx[0]
		xassert.Equal(t, c.ID, j2.CharacterID)
		xassert.Equal(t, app.JobDelivered, j2.Status)
		xassert.EqualOptional(t, 42, j2.SuccessfulRuns)
		assert.WithinDuration(t, completionDate, j2.EndDate, time.Second)
		assert.WithinDuration(t, completionDate, j2.CompletedDate.ValueOrZero(), time.Second)
		xassert.EqualOptional(t, completer, j2.CompletedCharacter)
	})
	t.Run("should add missing facility location", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006381})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006383})
		const facilityID = 1000000000001
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006381,
					"blueprint_type_id":     2047,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           facilityID,
					"installer_id":          498338451,
					"job_id":                229136101,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006383,
					"status":                "delivered",
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/universe/structures/%d", facilityID),
			httpmock.NewJsonResponderOrPanic(http.StatusForbidden, map[string]string{
				"error": "forbidden",
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
		require.NoError(t, err)
		xassert.Equal(t, 60006381, x.BlueprintLocation.ID)
		xassert.Equal(t, facilityID, x.Facility.ID)
		xassert.Equal(t, 60006382, x.OutputLocation.ID)
		xassert.Equal(t, 60006383, x.Station.ID)
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
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
		changed, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xx, err := st.ListCharacterIndustryJobs(ctx, c.ID)
		require.NoError(t, err)
		assert.Len(t, xx, 1)
		j2 := xx[0]
		xassert.Equal(t, c.ID, j2.CharacterID)
		xassert.Equal(t, app.JobReady, j2.Status)
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
		makeObj := func(jobID, activityID int64) map[string]any {
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
		activities := []int64{
			int64(app.Manufacturing),
			int64(app.Copying),
			int64(app.Invention),
			int64(app.MaterialEfficiencyResearch),
			int64(app.TimeEfficiencyResearch),
			int64(app.Reactions1),
			int64(app.Reactions2),
		}
		for jobID, activityID := range activities {
			objs = append(objs, makeObj(int64(jobID), activityID))
		}

		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
			httpmock.NewJsonResponderOrPanic(200, objs))

		// when
		_, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		for jobID, activityID := range activities {
			j, err := st.GetCharacterIndustryJob(ctx, c.ID, int64(jobID))
			require.NoError(t, err)
			xassert.Equal(t, activityID, int64(j.Activity))
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
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
		changed, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		oo, err := st.ListAllCharacterIndustryJob(ctx)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterIndustryJob) (int64, app.IndustryJobStatus) {
			return x.JobID, x.Status
		}))
		want := map[int64]app.IndustryJobStatus{
			j1.JobID: app.JobDelivered,
			j2.JobID: app.JobCancelled,
			j3.JobID: app.JobReady,
			j4.JobID: app.JobUnknown,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("should report no change when nothing changed and no orphans", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: c.ID,
			Status:      app.JobDelivered,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/industry/jobs?include_completed=true", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          j1.BlueprintID,
					"blueprint_location_id": j1.BlueprintLocation.ID,
					"blueprint_type_id":     j1.BlueprintType.ID,
					"duration":              j1.Duration,
					"end_date":              j1.EndDate.Format(time.RFC3339),
					"facility_id":           j1.Facility.ID,
					"installer_id":          j1.Installer.ID,
					"job_id":                j1.JobID,
					"output_location_id":    j1.OutputLocation.ID,
					"runs":                  j1.Runs,
					"start_date":            j1.StartDate.Format(time.RFC3339),
					"station_id":            j1.Station.ID,
					"status":                "delivered",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterIndustryJobs,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.GetCharacterIndustryJob(ctx, c.ID, j1.JobID)
		require.NoError(t, err)
		assert.Equal(t, app.JobDelivered, got.Status)
	})
}

func TestJobStatusFromESIValue_MatchesDomain(t *testing.T) {
	got := set.Collect(maps.Values(jobStatusFromESIValue))
	xassert.Equal(t, app.CharacterIndustryJobStatuses, got)
	assert.Len(t, jobStatusFromESIValue, got.Size())
}

// TestJobStatusFromESIValue_MatchesSpec compares against a stored copy of the ESI spec,
// so it can only fail when that copy is updated.
func TestJobStatusFromESIValue_MatchesSpec(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "genratelimit", "openapi.json"))
	require.NoError(t, err)
	var spec struct {
		Components struct {
			Schemas struct {
				Jobs struct {
					Items struct {
						Properties struct {
							Status struct {
								Enum []string `json:"enum"`
							} `json:"status"`
						} `json:"properties"`
					} `json:"items"`
				} `json:"CharactersCharacterIdIndustryJobsGet"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(data, &spec))
	enum := spec.Components.Schemas.Jobs.Items.Properties.Status.Enum
	require.NotEmpty(t, enum)
	xassert.Equal(t, set.Of(enum...), set.Collect(maps.Keys(jobStatusFromESIValue)))
}
