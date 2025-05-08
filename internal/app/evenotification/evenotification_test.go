package evenotification_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type notification struct {
	NotificationID int       `json:"notification_id"`
	Type           string    `json:"type"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
}

func TestShouldRenderAllNotifications(t *testing.T) {
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
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
	})
	ens := evenotification.New(eus)
	ctx := context.Background()
	solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30002537})
	structureType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 35835})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
		ID:               1000000000001,
		EveSolarSystemID: optional.From(solarSystem.ID),
		EveTypeID:        optional.From(structureType.ID),
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
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46300})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46301})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46302})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46303})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 35894})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 35835})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 32226}) // TCU
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 27})
	factory.CreateEveEntity(app.EveEntity{ID: 60003760, Category: app.EveEntityStation})
	notifTypes := evenotification.SupportedTypes()
	typeTested := make(map[evenotification.Type]bool)
	for _, n := range notifications {
		t2 := evenotification.Type(n.Type)
		if !notifTypes.Contains(t2) {
			continue
		}
		t.Run("should render notification type "+n.Type, func(t *testing.T) {
			typeTested[t2] = true
			title, body, err := ens.RenderESI(ctx, n.Type, n.Text, n.Timestamp)
			if assert.NoError(t, err) {
				assert.NotEqual(t, "", title)
				assert.NotEqual(t, "", body)
				switch n.NotificationID {
				case 1000000515:
					assert.Contains(t, body, "POCO")
				}
			}
		})
	}
	for n := range notifTypes.All() {
		if !typeTested[n] {
			t.Errorf("Failed to test supported notification type: %s", n)
		}
	}
}

func TestEntityIDsWithExample(t *testing.T) {
	en := evenotification.New(nil)
	t.Run("returns entity IDs", func(t *testing.T) {
		// given
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 27
externalID2: 60003760`
		got, err := en.EntityIDs("CorpAllBillMsg", text)
		if assert.NoError(t, err) {
			want := set.Of[int32](1000023, 98267621, 27, 60003760)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestRenderESIErrorHandling(t *testing.T) {
	en := evenotification.New(nil)
	t.Run("return error for unsurported", func(t *testing.T) {
		_, _, err := en.RenderESI(context.Background(), "invalid_type", "", time.Now())
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("return error for empty", func(t *testing.T) {
		_, _, err := en.RenderESI(context.Background(), "", "", time.Now())
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestEntityIDsSupportedNotifications(t *testing.T) {
	data, err := os.ReadFile("testdata/notifications.json")
	if err != nil {
		panic(err)
	}
	notifications := make([]notification, 0)
	if err := json.Unmarshal(data, &notifications); err != nil {
		panic(err)
	}
	notifTypes := evenotification.SupportedTypes()
	en := evenotification.New(nil)
	for _, n := range notifications {
		if t2 := evenotification.Type(n.Type); !notifTypes.Contains(t2) {
			continue
		}
		t.Run("should process notification type "+n.Type, func(t *testing.T) {
			_, err := en.EntityIDs(n.Type, n.Text)
			assert.NoError(t, err)
		})
	}
}

func TestEntityIDErrorHandling(t *testing.T) {
	en := evenotification.New(nil)
	t.Run("return error for unsurported", func(t *testing.T) {
		_, err := en.EntityIDs("invalid_type", "")
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("return error for empty type", func(t *testing.T) {
		_, err := en.EntityIDs("", "")
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
