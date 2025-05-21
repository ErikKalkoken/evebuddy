package corporationservice

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type CharacterServiceFake struct {
	Token *app.CharacterToken
	Error error
}

func (s CharacterServiceFake) ValidCharacterTokenForCorporation(ctx context.Context, corporationID int32, role app.Role) (*app.CharacterToken, error) {
	return s.Token, s.Error
}

func NewFake(st *storage.Storage, args ...Params) *CorporationService {
	scs := statuscacheservice.New(memcache.New(), st)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		StatusCacheService: scs,
		Storage:            st,
	})
	arg := Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	}
	if len(args) > 0 {
		arg.CharacterService = args[0].CharacterService
	}
	s := New(arg)
	return s
}

func TestUpdateIndustryJobsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t.TempDir())
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new job from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
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
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationUpdateSectionParams{
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
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		blueprintType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		productType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 2046})
		installer := factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			BlueprintLocationID: location.ID,
			BlueprintTypeID:     blueprintType.ID,
			CorporationID:       c.ID,
			EndDate:             time.Now(),
			FacilityID:          location.ID,
			InstallerID:         installer.ID,
			LocationID:          location.ID,
			OutputLocationID:    location.ID,
			ProductTypeID:       productType.ID,
			Status:              app.JobActive,
		})
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
					"status":                "delivered",
				},
			}),
		)
		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationUpdateSectionParams{
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
				assert.EqualValues(t, 1, x.Runs)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
				assert.EqualValues(t, 60006382, x.Location.ID)
				assert.Equal(t, app.JobDelivered, x.Status)

			}
		}
	})
	t.Run("should fix incorrect status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
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
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationUpdateSectionParams{
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
	t.Run("should not fix correct status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
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
					"end_date":              endDate.Format("2006-01-02T15:04:05Z"),
					"facility_id":           12,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"location_id":           60006382,
					"output_location_id":    13,
					"product_type_id":       2046,
					"runs":                  1,
					"start_date":            startDate.Format("2006-01-02T15:04:05Z"),
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CorporationUpdateSectionParams{
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
}
