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

func TestUpdateEveStatus(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should clear isOffline when server recovers from downtime", func(t *testing.T) {
		bu := MakeFakeBaseUI(st, test.NewTempApp(t), true)

		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/status",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"players":        12345,
				"server_version": "1132976",
				"start_time":     "2017-01-02T12:34:56Z",
			}),
		)

		u := &DesktopUI{baseUI: bu}
		sb := newStatusBar(u)
		u.isOffline.Store(true) // simulate server was previously offline

		sb.updateEveStatus(context.Background())

		assert.False(t, u.isOffline.Load())
	})

	t.Run("should not fetch status when in intentional offline mode", func(t *testing.T) {
		httpmock.Reset()
		// no responder registered — any HTTP call would return a connection error
		bu := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		u := &DesktopUI{baseUI: bu}
		sb := newStatusBar(u)
		u.isOfflineMode = true
		u.isOffline.Store(false)

		sb.updateEveStatus(context.Background())

		assert.Equal(t, 0, httpmock.GetTotalCallCount())
		assert.False(t, u.isOffline.Load()) // unchanged
	})

	t.Run("should set isOffline when server returns error status", func(t *testing.T) {
		bu := MakeFakeBaseUI(st, test.NewTempApp(t), true)

		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/status",
			httpmock.NewJsonResponderOrPanic(503, map[string]any{
				"error": "service unavailable",
			}),
		)

		u := &DesktopUI{baseUI: bu}
		sb := newStatusBar(u)
		u.isOffline.Store(false)

		sb.updateEveStatus(context.Background())

		assert.True(t, u.isOffline.Load())
	})

	t.Run("should show an error for a non-server HTTP error", func(t *testing.T) {
		bu := MakeFakeBaseUI(st, test.NewTempApp(t), true)

		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/status",
			httpmock.NewJsonResponderOrPanic(429, map[string]any{
				"error": "rate limit exceeded",
			}),
		)

		u := &DesktopUI{baseUI: bu}
		sb := newStatusBar(u)
		u.isOffline.Store(false)

		sb.updateEveStatus(context.Background())

		assert.False(t, u.isOffline.Load())
		assert.Equal(t, "ERROR", sb.eveStatus.label.Text)
	})

	t.Run("should show online when ESI responds during scheduled downtime", func(t *testing.T) {
		originalTimeNow := xgoesi.TimeNow
		t.Cleanup(func() {
			xgoesi.TimeNow = originalTimeNow
		})
		xgoesi.TimeNow = func() time.Time {
			return time.Date(2025, 12, 1, 11, 10, 0, 0, time.UTC)
		}

		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/status",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"players":        12345,
				"server_version": "1132976",
				"start_time":     "2017-01-02T12:34:56Z",
			}),
		)

		bu := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		u := &DesktopUI{baseUI: bu}
		sb := newStatusBar(u)
		u.isOffline.Store(true)

		sb.updateEveStatus(context.Background())

		assert.False(t, u.isOffline.Load())
		assert.Equal(t, "12,345 players", sb.eveStatus.label.Text)
	})
}
