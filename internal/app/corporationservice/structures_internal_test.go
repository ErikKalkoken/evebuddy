package corporationservice

import (
	"context"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateCorporationStructuresESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should fetch and create full structure from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
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
		got, err := st.GetCorporationStructure(ctx, c.ID, 42)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.EqualValues(t, c.ID, got.CorporationID)
		assert.EqualValues(t, 42, got.StructureID)
		assert.EqualValues(t, "Alpha", got.Name)
		assert.EqualValues(t, 99, got.ProfileID)
		assert.EqualValues(t, es, got.System)
		assert.EqualValues(t, et, got.Type)
		assert.Equal(t, app.StructureStateAnchorVulnerable, got.State)
		assert.WithinDuration(t, fuelExpires, got.FuelExpires.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, nextReinforceApply, got.NextReinforceApply.ValueOrZero(), 1*time.Second)
		assert.EqualValues(t, 8, got.NextReinforceHour.ValueOrZero())
		assert.EqualValues(t, 12, got.ReinforceHour.ValueOrZero())
		assert.WithinDuration(t, stateTimerEnd, got.StateTimerEnd.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, stateTimerStart, got.StateTimerStart.ValueOrZero(), 1*time.Second)
		assert.WithinDuration(t, unanchorsAt, got.UnanchorsAt.ValueOrZero(), 1*time.Second)
	})
}
