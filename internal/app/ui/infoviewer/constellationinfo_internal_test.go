package infoviewer

import (
	"context"
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestConstellationInfo_update(t *testing.T) {
	t.Run("should show constellation", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		constellation := factory.CreateEveConstellation()
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ConstellationID: constellation.ID})
		httpmock.RegisterResponder("GET", fmt.Sprintf("https://esi.evetech.net/universe/constellations/%d", constellation.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"constellation_id": constellation.ID,
				"name":             constellation.Name,
				"position":         map[string]any{"x": 1, "y": 2, "z": 3},
				"region_id":        constellation.Region.ID,
				"systems":          []int64{system.ID},
			}))
		iw := New(u)
		a := newConstellationInfo(iw, constellation.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, constellation.Name, a.name.Text)
		assert.Equal(t, constellation.Region.Name, a.region.Text)
		assert.Equal(t, []int64{system.ID}, xslices.Map(a.systems.items, func(x entityItem) int64 {
			return x.id
		}))
		// when
		a.region.OnTapped()
		// then
		assert.Equal(t, &showParams{variant: Region, entityID: constellation.Region.ID}, iw.current)
	})
	t.Run("should report error when solar systems can not be fetched", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", `=~^https://esi\.evetech\.net/universe/constellations/\d+`,
			httpmock.NewStringResponder(404, `{"error":"not found"}`))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		constellation := factory.CreateEveConstellation()
		a := newConstellationInfo(New(u), constellation.ID)
		// when
		err := a.update(t.Context())
		// then
		assert.Error(t, err)
	})
}
