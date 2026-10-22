package infoviewer

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationInfo_update(t *testing.T) {
	t.Run("should show corporation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		alliance := factory.CreateEveEntityAlliance()
		c := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			AllianceID:  optional.New(alliance.ID),
			DateFounded: optional.New(time.Date(2010, 3, 4, 0, 0, 0, 0, time.UTC)),
			Description: "My corp",
		})
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/corporations/%d/alliancehistory", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": alliance.ID,
					"record_id":   1,
					"start_date":  "2020-01-02T00:00:00Z",
				},
			}))
		a := newCorporationInfo(New(u), c.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, c.Name, a.name.Text)
		assert.Equal(t, "My corp", a.description.Text)
		assert.Equal(t, alliance.Name, a.alliance.Text)
		assert.True(t, a.allianceBox.Visible())
		assert.Equal(t, "Headquarters: "+c.HomeStation.MustValue().Name, a.hq.Text)
		assert.True(t, a.hq.Visible())
		require.Len(t, a.allianceHistory.items, 2)
		assert.Equal(t, alliance.ID, a.allianceHistory.items[0].id)
		assert.Equal(t, Alliance, a.allianceHistory.items[0].infoVariant)
		assert.Equal(t, "Corporation Founded: **2010.03.04**", entityItemTexts(a.allianceHistory)[1])
	})
	t.Run("should hide alliance when corporation has none", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/corporations/\d+/alliancehistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateEveCorporation()
		a := newCorporationInfo(New(u), c.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.False(t, a.allianceBox.Visible())
	})
	t.Run("should show unknown founding date", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/corporations/\d+/alliancehistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateEveCorporation()
		// created directly, because the factory always adds a founding date
		err := st.UpdateOrCreateEveCorporation(t.Context(), storage.UpdateOrCreateEveCorporationParams{
			ID:            c.ID,
			HomeStationID: c.HomeStation.MustValue().ID,
			Name:          c.Name,
		})
		require.NoError(t, err)
		a := newCorporationInfo(New(u), c.ID)
		// when
		err = a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"Corporation Founded: ?"}, entityItemTexts(a.allianceHistory))
	})
	t.Run("should show alliance and headquarters when tapped", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/corporations/\d+/alliancehistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		alliance := factory.CreateEveEntityAlliance()
		station := factory.CreateEveEntity(app.EveEntity{ID: 60_000_001, Category: app.EveEntityStation})
		c := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.New(alliance.ID),
			HomeStationID: station.ID,
		})
		iw := New(u)
		a := newCorporationInfo(iw, c.ID)
		err := a.update(t.Context())
		require.NoError(t, err)
		// when
		a.alliance.OnTapped()
		// then
		assert.Equal(t, &showParams{variant: Alliance, entityID: alliance.ID}, iw.current)
		// when
		a.hq.OnTapped()
		// then
		assert.Equal(t, &showParams{variant: Location, entityID: station.ID}, iw.current)
	})
}

func TestCorporationInfo_New(t *testing.T) {
	t.Run("should show alliance history for player corporations", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		// when
		a := newCorporationInfo(New(u), 98_000_001)
		// then
		assert.Equal(t, "Alliance History", a.tabs.Items[len(a.tabs.Items)-1].Text)
		assert.Equal(t, "Attributes", a.tabs.Selected().Text)
	})
	t.Run("should not show alliance history for NPC corporations", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		// when
		a := newCorporationInfo(New(u), 1_000_035)
		// then
		assert.Equal(t, "Attributes", a.tabs.Items[len(a.tabs.Items)-1].Text)
	})
}

func TestCorporationInfo_makeAttributes(t *testing.T) {
	t.Run("should show attributes", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		ceo := &app.EveEntity{ID: 90_000_001, Name: "Bruce", Category: app.EveEntityCharacter}
		creator := &app.EveEntity{ID: 90_000_002, Name: "Clark", Category: app.EveEntityCharacter}
		alliance := &app.EveEntity{ID: 99_000_001, Name: "Alpha Alliance", Category: app.EveEntityAlliance}
		o := &app.EveCorporation{
			Alliance:    optional.New(alliance),
			Ceo:         optional.New(ceo),
			Creator:     optional.New(creator),
			ID:          98_000_001,
			MemberCount: 42,
			Shares:      optional.New[int64](1000),
			TaxRate:     0.1,
			Ticker:      "ALPHA",
			URL:         optional.New("https://www.example.com"),
		}
		a := newCorporationInfo(New(u), o.ID)
		// when
		got := a.makeAttributes(o)
		// then
		myURL, _ := url.ParseRequestURI("https://www.example.com")
		assert.Equal(t, []string{
			"CEO", "Founder", "Alliance", "Ticker Name", "NPC", "Shares", "Member Count", "ISK Tax Rate", "War Eligibility", "URL",
		}, xslices.Map(got, func(x attributeItem) string {
			return x.Label
		}))
		assert.Contains(t, got, newAttributeItem("CEO", ceo))
		assert.Contains(t, got, newAttributeItem("Ticker Name", "ALPHA"))
		assert.Contains(t, got, newAttributeItem("NPC", false))
		assert.Contains(t, got, newAttributeItem("Member Count", int64(42)))
		assert.Contains(t, got, newAttributeItem("ISK Tax Rate", "10.0 %"))
		assert.Contains(t, got, newAttributeItem("URL", myURL))
	})
	t.Run("should not show invalid URL", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		o := &app.EveCorporation{ID: 98_000_001, URL: optional.New("www.example.com")}
		a := newCorporationInfo(New(u), o.ID)
		// when
		got := a.makeAttributes(o)
		// then
		assert.NotContains(t, xslices.Map(got, func(x attributeItem) string {
			return x.Label
		}), "URL")
	})
	t.Run("should show EVE ID in developer mode", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsDeveloperMode: true, Storage: st})
		o := &app.EveCorporation{ID: 98_000_001}
		a := newCorporationInfo(New(u), o.ID)
		// when
		got := a.makeAttributes(o)
		// then
		last := got[len(got)-1]
		assert.Equal(t, "EVE ID", last.Label)
		assert.Equal(t, o.ID, last.Value)
	})
}
