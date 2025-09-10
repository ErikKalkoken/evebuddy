package corporationservice

import (
	"context"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestUpdateCorporationStructuresESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should fetch and create full structure from scratch and delete stale structure", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
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
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/structures/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"corporation_id":       c.ID,
					"fuel_expires":         fuelExpires.Format(app.DateTimeFormatESI),
					"name":                 "Alpha",
					"next_reinforce_apply": nextReinforceApply.Format(app.DateTimeFormatESI),
					"next_reinforce_hour":  8,
					"profile_id":           99,
					"reinforce_hour":       12,
					"services": []map[string]any{
						{
							"name":  "service1",
							"state": "online",
						},
					},
					"state":             "anchor_vulnerable",
					"state_timer_end":   stateTimerEnd.Format(app.DateTimeFormatESI),
					"state_timer_start": stateTimerStart.Format(app.DateTimeFormatESI),
					"structure_id":      42,
					"system_id":         es.ID,
					"type_id":           et.ID,
					"unanchors_at":      unanchorsAt.Format(app.DateTimeFormatESI),
				},
			}),
		)
		// when
		changed, err := s.updateStructuresESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationStructures,
		})
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		got, err := st.ListCorporationStructureIDs(ctx, c.ID)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of[int64](42)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)

		x, err := st.GetCorporationStructure(ctx, c.ID, 42)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.EqualValues(t, c.ID, x.CorporationID)
		assert.EqualValues(t, 42, x.StructureID)
		assert.EqualValues(t, "Alpha", x.Name)
		assert.EqualValues(t, 99, x.ProfileID)
		assert.EqualValues(t, es, x.System)
		assert.EqualValues(t, et, x.Type)
		assert.Equal(t, app.StructureStateAnchorVulnerable, x.State)
		assert.WithinDuration(t, fuelExpires, x.FuelExpires.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, nextReinforceApply, x.NextReinforceApply.ValueOrZero(), 1*time.Second)
		assert.EqualValues(t, 8, x.NextReinforceHour.ValueOrZero())
		assert.EqualValues(t, 12, x.ReinforceHour.ValueOrZero())
		assert.WithinDuration(t, stateTimerEnd, x.StateTimerEnd.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, stateTimerStart, x.StateTimerStart.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, unanchorsAt, x.UnanchorsAt.ValueOrZero(), 1*time.Second)
	})
}
