package evenotification_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

type notification struct {
	NotificationID int       `json:"notification_id"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"`
}

// notificationTimestampLayouts are the timestamp formats used across the various
// testdata/notifications*.json fixture files (they were captured/generated independently).
var notificationTimestampLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05-07:00",
}

func (n *notification) UnmarshalJSON(data []byte) error {
	var raw struct {
		NotificationID int    `json:"notification_id"`
		Text           string `json:"text"`
		Timestamp      string `json:"timestamp"`
		Type           string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	n.NotificationID = raw.NotificationID
	n.Text = raw.Text
	n.Type = raw.Type
	var err error
	for _, layout := range notificationTimestampLayouts {
		if n.Timestamp, err = time.Parse(layout, raw.Timestamp); err == nil {
			return nil
		}
	}
	return err
}

// loadNotifications reads and concatenates every testdata/notifications*.json fixture file.
func loadNotifications(t *testing.T) []notification {
	paths, err := filepath.Glob("testdata/notifications*.json")
	require.NoError(t, err)
	notifications := make([]notification, 0)
	for _, p := range paths {
		data, err := os.ReadFile(p)
		require.NoError(t, err)
		batch := make([]notification, 0)
		err = json.Unmarshal(data, &batch)
		require.NoError(t, err)
		notifications = append(notifications, batch...)
	}
	return notifications
}

func TestShouldRenderAllNotifications(t *testing.T) {
	notifications := loadNotifications(t)
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	ens := evenotification.New(eus)
	ctx := context.Background()
	solarSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30002537})
	structureType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 35835})
	structureOwner := factory.CreateEveEntityCorporation(app.EveEntity{ID: 2001})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
		ID:            1000000000001,
		SolarSystemID: optional.New(solarSystem.ID),
		TypeID:        optional.New(structureType.ID),
		OwnerID:       optional.New(structureOwner.ID),
	})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1000134})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 1011})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2002})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2011})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 2021})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 3001})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 3002})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 3011})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 2233})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 16213})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 32458})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 85230})
	factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40161469})
	factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40161465})
	factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40161466})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46300})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46301})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46302})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 46303})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 35894})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 35835})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 32226}) // TCU
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 27})
	factory.CreateEveEntity(app.EveEntity{ID: 60003760, Category: app.EveEntityStation})
	// fixtures referenced by testdata/notifications_2.json and testdata/notifications_3.json
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000006})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60000002})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000002})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000007})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000003})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000009})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000005})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 93000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000014})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000008})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000015})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 840000001})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000009})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000010})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 200000001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000007})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60000003})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 35834})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 56095})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 35832})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 20060})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30003068})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30002686})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30002102})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30000474})
	factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40000001, SolarSystemID: 30002102})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1000000000005, OwnerID: optional.New(structureOwner.ID)})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100002})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100004})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100006})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100009})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100010})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100011})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100012})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000005})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60000001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000005})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000010})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000006})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 98100015})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100001})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100002})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100003})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100004})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 98100016})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100018})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100005})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100006})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100021})
	factory.CreateEveEntityWithCategory(app.EveEntityStation, app.EveEntity{ID: 60100007})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100027})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100029})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100030})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 98100047})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100042})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100048})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100044})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100046})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100048})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100049})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100050})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100052})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 608})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 672})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 90000001})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 35})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 2454})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 34})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100002})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100003})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1000000000002, SolarSystemID: optional.New(int64(30002686)), OwnerID: optional.New(structureOwner.ID)})
	factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1000000000003, SolarSystemID: optional.New(int64(30002686)), OwnerID: optional.New(structureOwner.ID)})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100035})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100036})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100037})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100038})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100039})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100040})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100041})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100042})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100043})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100044})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100045})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100046})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 90000002})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 90000004})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 90000006})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 90000008})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100003})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500003})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500005})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500007})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500009})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500011})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500013})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500015})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500017})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500019})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500021})
	factory.CreateEveEntityFaction(app.EveEntity{ID: 500023})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100033})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100034})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100035})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100004})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100054})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100008})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 16159})
	factory.CreateEveType(storage.CreateEveTypeParams{ID: 34})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100005})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100011})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100060})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100061})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100062})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100005})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100006})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100007})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100053})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100055})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100057})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 98100054})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 98100056})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 98100058})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 672})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 2454})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 35})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 587})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 608})
	factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 670})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100012})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100013})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100014})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100015})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100016})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100017})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100018})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100019})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100020})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100021})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100022})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 710000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000008})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 2100000004})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 94000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100013})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100014})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100014})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100022})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100019})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100020})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100021})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100023})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100022})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100023})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100024})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100024})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100025})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100026})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100032})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 820000001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 94000002})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 96000001})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100055})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100058})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100063})
	factory.CreateEveEntityCharacter(app.EveEntity{ID: 95100064})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000002})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000003})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000004})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000011})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000012})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000002})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98000013})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100001})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100002})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100003})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100004})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100005})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100006})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100007})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100008})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100027})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100028})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100029})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100030})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100031})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100032})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100033})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100034})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100050})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100051})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100061})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100062})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100063})
	factory.CreateEveEntityCorporation(app.EveEntity{ID: 98100064})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000001})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000003})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000004})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000006})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99000007})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100002})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100009})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100010})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100011})
	factory.CreateEveEntityAlliance(app.EveEntity{ID: 99100012})
	factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 30100001})
	notifTypes := app.NotificationTypesSupported()
	typeTested := make(map[app.EveNotificationType]bool)
	for _, n := range notifications {
		nt, found := storage.EveNotificationTypeFromESIString(n.Type)
		if !found || !notifTypes.Contains(nt) {
			continue
		}
		t.Run("should render notification type "+n.Type, func(t *testing.T) {
			typeTested[nt] = true
			title, body, err := ens.RenderESI(ctx, nt, optional.New(n.Text), n.Timestamp)
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
		got, err := en.EntityIDs(app.CorpAllBillMsg, optional.New(text))
		if assert.NoError(t, err) {
			want := set.Of[int64](1000023, 98267621, 27, 60003760)
			xassert.Equal(t, want, got)
		}
	})
}

func TestRenderESIErrorHandling(t *testing.T) {
	en := evenotification.New(nil)
	t.Run("return error for unsurported", func(t *testing.T) {
		_, _, err := en.RenderESI(context.Background(), app.UnknownNotification, optional.New("xxx"), time.Now())
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestEntityIDsSupportedNotifications(t *testing.T) {
	notifications := loadNotifications(t)
	notifTypes := app.NotificationTypesSupported()
	en := evenotification.New(nil)
	for _, n := range notifications {
		nt, found := storage.EveNotificationTypeFromESIString(n.Type)
		if !found || !notifTypes.Contains(nt) {
			continue
		}
		t.Run("should process notification type "+n.Type, func(t *testing.T) {
			_, err := en.EntityIDs(nt, optional.New(n.Text))
			assert.NoError(t, err)
		})
	}
}

func TestEntityIDErrorHandling(t *testing.T) {
	en := evenotification.New(nil)
	t.Run("return error for unsurported", func(t *testing.T) {
		_, err := en.EntityIDs(app.UnknownNotification, optional.New("xxx"))
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
