package corporationservice

import (
	"context"
	"fmt"
	"net/http"
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
)

func TestGetStructure(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can return existing structure", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		o := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c.ID})
		got, err := s.GetStructure(ctx, c.ID, o.StructureID)
		require.NoError(t, err)
		xassert.Equal(t, o.StructureID, got.StructureID)
	})
	t.Run("should return error when structure not found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		_, err := s.GetStructure(ctx, c.ID, 42)
		assert.Error(t, err)
	})
}

func TestListStructures(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list structures for a corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		o1 := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c1.ID})
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{CorporationID: c2.ID})
		got, err := s.ListStructures(ctx, c1.ID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		xassert.Equal(t, o1.StructureID, got[0].StructureID)
	})
}

func TestUpdateCorporationStructuresESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should fetch and create full structure from scratch and delete stale structure", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		es := factory.CreateEveSolarSystem()
		et := factory.CreateEveType()
		fuelExpires := factory.RandomTime()
		nextReinforceApply := factory.RandomTime()
		stateTimerEnd := factory.RandomTime()
		stateTimerStart := factory.RandomTime()
		unanchorsAt := factory.RandomTime()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id":       c.ID,
				"fuel_expires":         fuelExpires.Format(app.DateTimeFormatESI),
				"name":                 "Alpha",
				"next_reinforce_apply": nextReinforceApply.Format(app.DateTimeFormatESI),
				"next_reinforce_hour":  8,
				"profile_id":           99,
				"reinforce_hour":       12,
				"services": []map[string]any{{
					"name":  "service1",
					"state": "online",
				}},
				"state":             "anchor_vulnerable",
				"state_timer_end":   stateTimerEnd.Format(app.DateTimeFormatESI),
				"state_timer_start": stateTimerStart.Format(app.DateTimeFormatESI),
				"structure_id":      42,
				"system_id":         es.ID,
				"type_id":           et.ID,
				"unanchors_at":      unanchorsAt.Format(app.DateTimeFormatESI),
			}}),
		)
		// when
		changed, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationStructureIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of[int64](42)
		xassert.Equal(t, want, got)

		x, err := st.GetCorporationStructure(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, c.ID, x.CorporationID)
		xassert.Equal(t, 42, x.StructureID)
		xassert.EqualOptional(t, "Alpha", x.Name)
		xassert.Equal(t, 99, x.ProfileID)
		xassert.Equal(t, es, x.System)
		xassert.Equal(t, et, x.Type)
		xassert.Equal(t, app.StructureStateAnchorVulnerable, x.State)
		assert.WithinDuration(t, fuelExpires, x.FuelExpires.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, nextReinforceApply, x.NextReinforceApply.ValueOrZero(), 1*time.Second)
		xassert.EqualOptional(t, 8, x.NextReinforceHour)
		xassert.EqualOptional(t, 12, x.ReinforceHour)
		assert.WithinDuration(t, stateTimerEnd, x.StateTimerEnd.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, stateTimerStart, x.StateTimerStart.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, unanchorsAt, x.UnanchorsAt.ValueOrZero(), 1*time.Second)
		xassert.Equal(t, "service1", x.Services[0].Name)
		xassert.Equal(t, app.StructureServiceStateOnline, x.Services[0].State)
	})
	t.Run("should fetch and combine all pages from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		es := factory.CreateEveSolarSystem()
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			func(req *http.Request) (*http.Response, error) {
				var structureID int64
				switch req.URL.Query().Get("page") {
				case "", "1":
					structureID = 1
				case "2":
					structureID = 2
				default:
					t.Fatalf("unexpected page requested: %q", req.URL.Query().Get("page"))
				}
				resp, err := httpmock.NewJsonResponse(200, []map[string]any{{
					"corporation_id": c.ID,
					"name":           fmt.Sprintf("Structure %d", structureID),
					"profile_id":     1,
					"state":          "shield_vulnerable",
					"structure_id":   structureID,
					"system_id":      es.ID,
					"type_id":        et.ID,
				}})
				if err != nil {
					return nil, err
				}
				resp.Header.Set("X-Pages", "2")
				return resp, nil
			},
		)
		// when
		changed, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationStructureIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of[int64](1, 2)
		xassert.Equal(t, want, got)
	})
	t.Run("should keep stored structures when adding related objects fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		o := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
		})
		es := factory.CreateEveSolarSystem()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": c.ID,
				"profile_id":     1,
				"state":          "shield_vulnerable",
				"structure_id":   o.StructureID + 1,
				"system_id":      es.ID,
				"type_id":        99999, // not stored and no responder, so fetching it fails
			}}),
		)
		// when
		_, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		// then
		require.Error(t, err)
		got, err := st.ListCorporationStructureIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o.StructureID), got)
	})
	t.Run("should report no change when structures are unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		es := factory.CreateEveSolarSystem()
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": c.ID,
				"fuel_expires":   factory.RandomTime().Format(app.DateTimeFormatESI),
				"name":           "Alpha",
				"profile_id":     1,
				"services": []map[string]any{{
					"name":  "service1",
					"state": "online",
				}},
				"state":        "shield_vulnerable",
				"structure_id": 42,
				"system_id":    es.ID,
				"type_id":      et.ID,
			}}),
		)
		_, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		require.NoError(t, err)
		// when
		changed, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			forceUpdate:   true, // skips the content hash check
			section:       app.SectionCorporationStructures,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("should update profile of existing structure", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		o := factory.CreateCorporationStructure(storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: c.ID,
			ProfileID:     1,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": c.ID,
				"profile_id":     2,
				"state":          "shield_vulnerable",
				"structure_id":   o.StructureID,
				"system_id":      o.System.ID,
				"type_id":        o.Type.ID,
			}}),
		)
		// when
		changed, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		x, err := st.GetCorporationStructure(ctx, c.ID, o.StructureID)
		require.NoError(t, err)
		xassert.Equal(t, 2, x.ProfileID)
	})
	t.Run("should store the last of duplicate structures", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		es := factory.CreateEveSolarSystem()
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"corporation_id": c.ID,
					"name":           "Alpha",
					"profile_id":     1,
					"state":          "shield_vulnerable",
					"structure_id":   42,
					"system_id":      es.ID,
					"type_id":        et.ID,
				},
				{
					"corporation_id": c.ID,
					"name":           "Bravo",
					"profile_id":     1,
					"state":          "shield_vulnerable",
					"structure_id":   42,
					"system_id":      es.ID,
					"type_id":        et.ID,
				},
			}),
		)
		// when
		_, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCorporationStructure(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, "Bravo", x.Name)
	})
	t.Run("should store unknown service state without failing the sync", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		es := factory.CreateEveSolarSystem()
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"corporation_id": c.ID,
				"profile_id":     1,
				"services": []map[string]any{{
					"name":  "service1",
					"state": "new_esi_value",
				}},
				"state":        "shield_vulnerable",
				"structure_id": 42,
				"system_id":    es.ID,
				"type_id":      et.ID,
			}}),
		)
		// when
		_, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationStructures,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCorporationStructure(ctx, c.ID, 42)
		require.NoError(t, err)
		require.Len(t, x.Services, 1)
		xassert.Equal(t, app.StructureServiceStateUnknown, x.Services[0].State)
	})
	t.Run("should store the state matching the ESI value", func(t *testing.T) {
		cases := []struct {
			esiValue string
			want     app.StructureState
		}{
			{"anchor_vulnerable", app.StructureStateAnchorVulnerable},
			{"anchoring", app.StructureStateAnchoring},
			{"armor_reinforce", app.StructureStateArmorReinforce},
			{"armor_vulnerable", app.StructureStateArmorVulnerable},
			{"deploy_vulnerable", app.StructureStateDeployVulnerable},
			{"fitting_invulnerable", app.StructureStateFittingInvulnerable},
			{"hull_reinforce", app.StructureStateHullReinforce},
			{"hull_vulnerable", app.StructureStateHullVulnerable},
			{"online_deprecated", app.StructureStateOnlineDeprecated},
			{"onlining_vulnerable", app.StructureStateOnliningVulnerable},
			{"shield_vulnerable", app.StructureStateShieldVulnerable},
			{"unanchored", app.StructureStateUnanchored},
			{"unknown", app.StructureStateUnknown},
			{"new_esi_value", app.StructureStateUnknown},
		}
		for _, tc := range cases {
			t.Run(tc.esiValue, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				httpmock.Reset()
				s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
				c := factory.CreateCorporation()
				es := factory.CreateEveSolarSystem()
				et := factory.CreateEveType()
				httpmock.RegisterResponder(
					"GET",
					fmt.Sprintf("https://esi.evetech.net/corporations/%d/structures", c.ID),
					httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
						"corporation_id": c.ID,
						"profile_id":     1,
						"state":          tc.esiValue,
						"structure_id":   42,
						"system_id":      es.ID,
						"type_id":        et.ID,
					}}),
				)
				// when
				_, err := s.updateStructuresESI(ctx, corporationSectionUpdateParams{
					corporationID: c.ID,
					section:       app.SectionCorporationStructures,
				})
				// then
				require.NoError(t, err)
				x, err := st.GetCorporationStructure(ctx, c.ID, 42)
				require.NoError(t, err)
				xassert.Equal(t, tc.want, x.State)
			})
		}
	})
}
