package core

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func TestStatusBar_UpdateEveStatus(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	original := xgoesi.TimeNow
	t.Cleanup(func() {
		xgoesi.TimeNow = original
	})
	setNow := func(s string) {
		x, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		xgoesi.TimeNow = func() time.Time {
			return x
		}
	}
	registerOnline := func() {
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/status",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"players":        12345,
				"server_version": "1132976",
				"start_time":     "2026-09-24T11:15:00Z",
			}),
		)
	}
	ctx := context.Background()

	t.Run("should go back online after daily downtime", func(t *testing.T) {
		// given
		httpmock.Reset()
		du := &DesktopUI{baseUI: MakeFakeBaseUI(st, test.NewTempApp(t), false)}
		sb := newStatusBar(du)
		setNow("2026-09-24T11:05:00Z")
		sb.updateEveStatus(ctx)
		assert.True(t, du.IsOffline())
		assert.Contains(t, sb.eveStatusError, "daily downtime")
		// when
		registerOnline()
		setNow("2026-09-24T11:20:00Z")
		sb.updateEveStatus(ctx)
		// then
		assert.False(t, du.IsOffline())
		assert.Empty(t, sb.eveStatusError)
	})
	t.Run("should go back online after ESI reported an error", func(t *testing.T) {
		// given
		httpmock.Reset()
		du := &DesktopUI{baseUI: MakeFakeBaseUI(st, test.NewTempApp(t), false)}
		sb := newStatusBar(du)
		setNow("2026-09-24T12:00:00Z")
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/status",
			httpmock.NewJsonResponderOrPanic(503, map[string]any{
				"error": "service unavailable",
			}),
		)
		sb.updateEveStatus(ctx)
		assert.True(t, du.IsOffline())
		// when
		httpmock.Reset()
		registerOnline()
		sb.updateEveStatus(ctx)
		// then
		assert.False(t, du.IsOffline())
		assert.Empty(t, sb.eveStatusError)
	})
	t.Run("should stay offline and not call ESI when in offline mode", func(t *testing.T) {
		// given
		httpmock.Reset()
		du := &DesktopUI{baseUI: MakeFakeBaseUI(st, test.NewTempApp(t), false)}
		du.isOfflineMode = true
		sb := newStatusBar(du)
		setNow("2026-09-24T12:00:00Z")
		registerOnline()
		// when
		sb.updateEveStatus(ctx)
		// then
		assert.True(t, du.IsOffline())
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
}
