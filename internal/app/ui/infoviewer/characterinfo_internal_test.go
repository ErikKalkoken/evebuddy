package infoviewer

import (
	"context"
	"fmt"
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

func TestCharacterInfo_update(t *testing.T) {
	t.Run("should show character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		corporation := factory.CreateEveEntityCorporation()
		alliance := factory.CreateEveEntityAlliance()
		c := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			AllianceID:       optional.New(alliance.ID),
			CorporationID:    corporation.ID,
			CorporationTitle: optional.New("Director"),
			Description:      optional.New("My bio"),
			SecurityStatus:   optional.New(-2.04),
		})
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/characters/%d/corporationhistory", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"corporation_id": corporation.ID,
					"record_id":      1,
					"start_date":     time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
				},
			}))
		a := newCharacterInfo(New(u), c.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, c.Name, a.name.Text)
		assert.Equal(t, "Security Status: -2.0", a.security.Text)
		assert.Equal(t, corporation.Name, a.corporation.Text)
		assert.Equal(t, alliance.Name, a.alliance.Text)
		assert.True(t, a.alliance.Visible())
		assert.Equal(t, "Title: Director", a.title.Text)
		assert.True(t, a.title.Visible())
		assert.Equal(t, "My bio", a.bio.Text)
		assert.Equal(t, c.Race.Description, a.description.Text)
		assert.Equal(t, "for 2 days", a.membership.Text)
		assert.Equal(t, []int64{corporation.ID}, xslices.Map(a.employeeHistory.items, func(x entityItem) int64 {
			return x.id
		}))
		assert.False(t, a.ownedIcon.Visible())
	})
	t.Run("should hide alliance and title when character has none", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/characters/\d+/corporationhistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateEveCharacter()
		// created directly, because the factory always adds a title
		err := st.UpdateOrCreateEveCharacter(t.Context(), storage.CreateEveCharacterParams{
			ID:            c.ID,
			BloodlineID:   c.Bloodline.MustValue().ID,
			CorporationID: c.Corporation.ID,
			Name:          c.Name,
			RaceID:        c.Race.ID,
		})
		require.NoError(t, err)
		a := newCharacterInfo(New(u), c.ID)
		// when
		err = a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.False(t, a.alliance.Visible())
		assert.False(t, a.title.Visible())
	})
	t.Run("should hide membership when there is no history", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/characters/\d+/corporationhistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateEveCharacter()
		a := newCharacterInfo(New(u), c.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.False(t, a.membership.Visible())
		assert.Empty(t, a.employeeHistory.items)
	})
	t.Run("should mark owned character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/characters/\d+/corporationhistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateCharacter()
		a := newCharacterInfo(New(u), c.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.True(t, a.ownedIcon.Visible())
	})
	t.Run("should show corporation when tapped", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/characters/\d+/corporationhistory`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}))
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		c := factory.CreateEveCharacter()
		iw := New(u)
		a := newCharacterInfo(iw, c.ID)
		err := a.update(t.Context())
		require.NoError(t, err)
		// when
		a.corporation.OnTapped()
		// then
		assert.Equal(t, &showParams{variant: Corporation, entityID: c.Corporation.ID}, iw.current)
	})
}

func TestCharacterInfo_New(t *testing.T) {
	t.Run("should show employment history for player characters", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		// when
		a := newCharacterInfo(New(u), 90_000_001)
		// then
		assert.Equal(t, "Employment History", a.tabs.Items[len(a.tabs.Items)-1].Text)
		assert.Equal(t, "Attributes", a.tabs.Selected().Text)
	})
	t.Run("should not show employment history for NPCs", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		// when
		a := newCharacterInfo(New(u), 3_000_001)
		// then
		assert.Equal(t, "Attributes", a.tabs.Items[len(a.tabs.Items)-1].Text)
	})
}

func TestCharacterInfo_makeAttributes(t *testing.T) {
	t.Run("should show public attributes", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		corporation := &app.EveEntity{ID: 98_000_001, Name: "Alpha Corp", Category: app.EveEntityCorporation}
		alliance := &app.EveEntity{ID: 99_000_001, Name: "Alpha Alliance", Category: app.EveEntityAlliance}
		race := &app.EveRace{ID: 1, Name: "Caldari"}
		o := &app.EveCharacter{
			Alliance:       optional.New(alliance),
			Birthday:       time.Date(2010, 3, 4, 5, 6, 0, 0, time.UTC),
			Bloodline:      optional.New(&app.EntityShort{ID: 2, Name: "Deteis"}),
			Corporation:    corporation,
			ID:             90_000_001,
			Race:           race,
			SecurityStatus: optional.New(1.25),
		}
		a := newCharacterInfo(New(u), o.ID)
		// when
		got, err := a.makeAttributes(t.Context(), o, false)
		// then
		require.NoError(t, err)
		assert.Equal(t, []attributeItem{
			newAttributeItem("Born", "2010.03.04 05:06"),
			newAttributeItem("Race", race),
			newAttributeItem("Bloodline", &eveBloodlineShort{ID: 2, Name: "Deteis"}),
			newAttributeItem("Security Status", "1.2"),
			newAttributeItem("Corporation", corporation),
			newAttributeItem("Alliance", alliance),
			newAttributeItem("NPC", false),
		}, got)
	})
	t.Run("should show last login for owned characters", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		lastLogin := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
		c := factory.CreateCharacter(storage.CreateCharacterParams{LastLoginAt: optional.New(lastLogin)})
		a := newCharacterInfo(New(u), c.ID)
		// when
		got, err := a.makeAttributes(t.Context(), c.EveCharacter, true)
		// then
		require.NoError(t, err)
		assert.Contains(t, got, newAttributeItem("Last Login", lastLogin))
	})
	t.Run("should show EVE ID in developer mode", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsDeveloperMode: true, Storage: st})
		c := factory.CreateEveCharacter()
		a := newCharacterInfo(New(u), c.ID)
		// when
		got, err := a.makeAttributes(t.Context(), c, false)
		// then
		require.NoError(t, err)
		last := got[len(got)-1]
		assert.Equal(t, "EVE ID", last.Label)
		assert.Equal(t, c.ID, last.Value)
	})
}
