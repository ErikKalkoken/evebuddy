package evenotification_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
	"github.com/ErikKalkoken/evebuddy/pkg/set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestRenderCharacterNotification(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eu := eveuniverse.New(st, nil)
	en := evenotification.New()
	en.EveUniverseService = eu
	ctx := context.Background()
	t.Run("should render notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		creditor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000023})
		debtor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 98267621})
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 27
externalID2: 60002599`
		title, body, err := en.RenderESI(ctx, "CorpAllBillMsg", text, time.Now())
		if assert.NoError(t, err) {
			assert.Equal(t, "Bill issued", title.ValueOrZero())
			assert.Contains(t, body.ValueOrZero(), creditor.Name)
			assert.Contains(t, body.ValueOrZero(), debtor.Name)
		}
	})
}

type notification struct {
	NotificationID int       `json:"notification_id"`
	Type           string    `json:"type"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
}

func TestRenderCharacterNotification2(t *testing.T) {
	data, err := os.ReadFile("testdata/notifications.json")
	if err != nil {
		panic(err)
	}
	notifications := make([]notification, 0)
	if err := json.Unmarshal(data, &notifications); err != nil {
		panic(err)
	}
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eu := eveuniverse.New(st, nil)
	en := evenotification.New()
	en.EveUniverseService = eu
	ctx := context.Background()
	solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30002537})
	structureType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 35835})
	factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{
		ID:               1000000000001,
		EveSolarSystemID: optional.New(solarSystem.ID),
		EveTypeID:        optional.New(structureType.ID),
	})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1000134})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1011})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2002})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2011})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2021})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 3001})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 3002})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 3011})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 2233})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 32458})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 16213})
	factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40161469})
	factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40161465})
	factory.CreateEveEntityInventoryType(app.EveEntity{ID: 46300})
	factory.CreateEveEntityInventoryType(app.EveEntity{ID: 46301})
	factory.CreateEveEntityInventoryType(app.EveEntity{ID: 46302})
	factory.CreateEveEntityInventoryType(app.EveEntity{ID: 46303})
	factory.CreateEveEntityInventoryType(app.EveEntity{ID: 35894})
	factory.CreateEveEntityInventoryType(app.EveEntity{ID: 35835})
	notifTypes := set.NewFromSlice(evenotification.NotificationTypesSupported())
	typeTested := make(map[string]bool)
	for _, n := range notifications {
		t.Run("should render notification type "+n.Type, func(t *testing.T) {
			if notifTypes.Has(n.Type) {
				typeTested[n.Type] = true
				title, body, err := en.RenderESI(ctx, n.Type, n.Text, n.Timestamp)
				if assert.NoError(t, err) {
					assert.False(t, title.IsEmpty())
					assert.False(t, body.IsEmpty())
					switch n.NotificationID {
					case 1000000515:
						assert.Contains(t, body.ValueOrZero(), "POCO")
					}
				}
			}
		})
	}
	for n := range notifTypes.Iter() {
		if !typeTested[n] {
			t.Errorf("Failed to test supported notification type: %s", n)
		}
	}
}
