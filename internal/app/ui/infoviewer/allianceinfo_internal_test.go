package infoviewer

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestAllianceInfo_update(t *testing.T) {
	t.Run("should show alliance", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		alliance := factory.CreateEveEntityAlliance()
		creator := factory.CreateEveEntityCharacter()
		executor := factory.CreateEveEntityCorporation(app.EveEntity{Name: "Alpha"})
		member := factory.CreateEveEntityCorporation(app.EveEntity{Name: "Bravo"})
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/alliances/%d", alliance.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"creator_corporation_id":  executor.ID,
				"creator_id":              creator.ID,
				"date_founded":            "2016-06-26T21:00:00Z",
				"executor_corporation_id": executor.ID,
				"name":                    "C C P Alliance",
				"ticker":                  "CCP",
			}))
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/alliances/%d/corporations", alliance.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{member.ID, executor.ID}))
		a := newAllianceInfo(New(u), alliance.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "C C P Alliance", a.name.Text)
		assert.Equal(t, []int64{executor.ID, member.ID}, xslices.Map(a.members.items, func(x entityItem) int64 {
			return x.id
		}))
		assert.Subset(t, tabLabelTexts(t, a.tabs, "Attributes"), []string{
			"Executor", executor.Name,
			"Short Name", "CCP",
			"Created By Corporation",
			"Created By", creator.Name,
			"Start Date", "2016.06.26 21:00",
		})
	})
	t.Run("should report error when alliance can not be fetched", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/alliances/\d+`,
			httpmock.NewStringResponder(404, `{"error":"not found"}`))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newAllianceInfo(New(u), 99_000_001)
		// when
		err := a.update(t.Context())
		// then
		assert.Error(t, err)
		assert.Equal(t, "Loading...", a.name.Text)
	})
}
