package infoviewer

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestRegionInfo_update(t *testing.T) {
	t.Run("should show region", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		region := factory.CreateEveRegion(storage.CreateEveRegionParams{Description: optional.New("My region")})
		constellation := factory.CreateEveConstellation(storage.CreateEveConstellationParams{RegionID: region.ID})
		factory.GetOrCreateEveEntity(*constellation.ToEveEntity())
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/regions/%d", region.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellations": []int64{constellation.ID},
				"name":           region.Name,
				"region_id":      region.ID,
			}))
		a := newRegionInfo(New(u), region.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, region.Name, a.name.Text)
		assert.Equal(t, "My region", a.description.Text)
		assert.Equal(t, []int64{constellation.ID}, xslices.Map(a.constellations.items, func(x entityItem) int64 {
			return x.id
		}))
		assert.NotContains(t, xslices.Map(a.tabs.Items, func(x *container.TabItem) string {
			return x.Text
		}), "Attributes")
	})
	t.Run("should show EVE ID in developer mode", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsDeveloperMode: true, Storage: st})
		region := factory.CreateEveRegion()
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/regions/%d", region.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellations": []int64{},
				"name":           region.Name,
				"region_id":      region.ID,
			}))
		a := newRegionInfo(New(u), region.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Subset(t, tabLabelTexts(t, a.tabs, "Attributes"), []string{"EVE ID", fmt.Sprint(region.ID)})
	})
	t.Run("should report error when constellations can not be fetched", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/universe/regions/\d+`,
			httpmock.NewStringResponder(404, `{"error":"not found"}`))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		region := factory.CreateEveRegion()
		a := newRegionInfo(New(u), region.ID)
		// when
		err := a.update(t.Context())
		// then
		assert.Error(t, err)
		assert.Equal(t, "Loading...", a.name.Text)
	})
}
