package core

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
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
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
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
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
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
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
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

func TestStatusBar_Start(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	original := xgoesi.TimeNow
	t.Cleanup(func() {
		xgoesi.TimeNow = original
	})
	xgoesi.TimeNow = func() time.Time {
		return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	}
	ctx := context.Background()

	t.Run("should show character count at start", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacter()
		factory.CreateCharacter()
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		// don't start the tickers and only listen with the status bar, not the screens
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		sb := newStatusBar(du)
		// when
		sb.start(ctx)
		// then
		xassert.Equal(t, "2", sb.characterCount.label.Text)
	})
	t.Run("should update character count when a character was added", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		sb := newStatusBar(du)
		sb.start(ctx)
		c := factory.CreateCharacter()
		// when
		du.Signals().CharacterAdded.Emit(ctx, c)
		// then
		xassert.Equal(t, "1", sb.characterCount.label.Text)
	})
	t.Run("should update character count when a character was removed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		sb := newStatusBar(du)
		sb.start(ctx)
		err := st.DeleteCharacter(ctx, c.ID)
		if err != nil {
			t.Fatal(err)
		}
		// when
		du.Signals().CharacterRemoved.Emit(ctx, &app.EntityShort{ID: c.ID})
		// then
		xassert.Equal(t, "0", sb.characterCount.label.Text)
	})
	t.Run("should show update indicator until all updates have stopped", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		sb := newStatusBar(du)
		sb.start(ctx)
		du.Signals().UpdateStarted.Emit(ctx, "alpha")
		du.Signals().UpdateStarted.Emit(ctx, "bravo")
		// when
		du.Signals().UpdateStopped.Emit(ctx, "alpha")
		// then
		assert.True(t, sb.updatingIndicator.Visible())
	})
	t.Run("should hide update indicator when all updates have stopped", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		sb := newStatusBar(du)
		sb.start(ctx)
		du.Signals().UpdateStarted.Emit(ctx, "alpha")
		du.Signals().UpdateStarted.Emit(ctx, "bravo")
		du.Signals().UpdateStopped.Emit(ctx, "alpha")
		// when
		du.Signals().UpdateStopped.Emit(ctx, "bravo")
		// then
		assert.False(t, sb.updatingIndicator.Visible())
	})
	t.Run("should show update status after a section was updated", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		sb := newStatusBar(du)
		sb.start(ctx)
		du.StatusCache().SetCharacterSection(&app.CharacterSectionStatus{
			CharacterID: c.ID,
			SectionStatus: app.SectionStatus{
				ErrorMessage: "error",
				Section:      app.SectionCharacterSkills,
			},
		})
		// when
		du.Signals().CharacterSectionUpdated.Emit(ctx, app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkills,
		})
		// then
		xassert.Equal(t, "1 ERRORS", sb.updateStatus.label.Text)
		xassert.Equal(t, widget.DangerImportance, sb.updateStatus.label.Importance)
	})
	t.Run("should show offline status in offline mode", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop()
		du.signals = testdouble.NewSignalsSync()
		du.isOfflineMode = true
		sb := newStatusBar(du)
		// when
		sb.start(ctx)
		// then
		xassert.Equal(t, "OFFLINE", sb.eveStatus.label.Text)
		xassert.Equal(t, "Offline mode", sb.eveStatusError)
	})
}

func TestStatusBar_UpdateUpdateStatus(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
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
	ctx := context.Background()

	t.Run("should show summary of the status cache", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.SectionCharacterSkills,
			ErrorMessage: "error",
		})
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		sb := newStatusBar(du)
		setNow("2026-09-24T12:00:00Z")
		// when
		sb.updateUpdateStatus(ctx)
		// then
		xassert.Equal(t, "1 ERRORS", sb.updateStatus.label.Text)
		xassert.Equal(t, widget.DangerImportance, sb.updateStatus.label.Importance)
	})
	t.Run("should show OFF when updates are disabled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.isUpdateDisabled.Store(true)
		sb := newStatusBar(du)
		setNow("2026-09-24T12:00:00Z")
		// when
		sb.updateUpdateStatus(ctx)
		// then
		xassert.Equal(t, "OFF", sb.updateStatus.label.Text)
		xassert.Equal(t, widget.MediumImportance, sb.updateStatus.label.Importance)
	})
	t.Run("should show OFF during daily downtime", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		sb := newStatusBar(du)
		setNow("2026-09-24T11:05:00Z")
		// when
		sb.updateUpdateStatus(ctx)
		// then
		xassert.Equal(t, "OFF", sb.updateStatus.label.Text)
	})
}

func TestStatusBar_MissingScope(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should show warning when characters are missing scopes", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		sb := newStatusBar(du)
		// when
		du.onUpdateMissingScope(2)
		// then
		icon := sb.characterCount.trailing.(*ttwidget.Icon)
		assert.True(t, icon.Visible())
		xassert.Equal(t, "2 character(s) missing scope", icon.ToolTip())
	})
	t.Run("should hide warning when no characters are missing scopes", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		sb := newStatusBar(du)
		du.onUpdateMissingScope(2)
		// when
		du.onUpdateMissingScope(0)
		// then
		icon := sb.characterCount.trailing.(*ttwidget.Icon)
		assert.False(t, icon.Visible())
	})
}

func TestStatusBar_SetEveStatus(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	r := theme.MediaRecordIcon()
	cases := []struct {
		name   string
		status eveStatus
		want   fyne.Resource
	}{
		{"online", eveStatusOnline, theme.NewSuccessThemedResource(r)},
		{"error", eveStatusError, theme.NewErrorThemedResource(r)},
		{"offline", eveStatusOffline, theme.NewWarningThemedResource(r)},
		{"unknown", eveStatusUnknown, theme.NewDisabledResource(r)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
			sb := newStatusBar(du)
			// when
			sb.setEveStatus(tc.status, "title", "message")
			// then
			xassert.Equal(t, tc.want, sb.eveStatus.leadingIcon.Resource)
			xassert.Equal(t, "title", sb.eveStatus.label.Text)
			xassert.Equal(t, "message", sb.eveStatusError)
		})
	}
}

func TestStatusBar_ShowEveStatusDialog(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	// dialogLabel returns the first label shown in the top overlay of w.
	dialogLabel := func(t *testing.T, w fyne.Window) *widget.Label {
		t.Helper()
		top := w.Canvas().Overlays().Top()
		require.NotNil(t, top, "no dialog shown")
		for _, o := range test.LaidOutObjects(top) {
			if x, ok := o.(*widget.Label); ok {
				return x
			}
		}
		t.Fatal("no label found")
		return nil
	}

	t.Run("should show that there is no error", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		sb := newStatusBar(du)
		sb.setEveStatus(eveStatusOnline, "1 players", "")
		// when
		sb.showEveStatusDialog()
		// then
		lb := dialogLabel(t, du.MainWindow())
		xassert.Equal(t, "No error detected", lb.Text)
		xassert.Equal(t, widget.MediumImportance, lb.Importance)
	})
	t.Run("should show error", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		sb := newStatusBar(du)
		sb.setEveStatus(eveStatusError, "ERROR", "some error")
		// when
		sb.showEveStatusDialog()
		// then
		lb := dialogLabel(t, du.MainWindow())
		xassert.Equal(t, "some error", lb.Text)
		xassert.Equal(t, widget.DangerImportance, lb.Importance)
	})
}

func TestStatusBar_ShowClockDialog(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	// closes the DB after the cleanup of MakeFakeBaseUI
	t.Cleanup(func() {
		db.Close()
	})
	// given
	du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
	du.tasks.Stop() // don't start the ticker
	sb := newStatusBar(du)
	// when
	sb.showClockDialog()
	// then
	top := du.MainWindow().Canvas().Overlays().Top()
	require.NotNil(t, top, "no dialog shown")
	var texts []string
	for _, o := range test.LaidOutObjects(top) {
		if x, ok := o.(*widget.Label); ok {
			texts = append(texts, x.Text)
		}
	}
	require.NotEmpty(t, texts)
	assert.Regexp(t, `^\d\d:\d\d:\d\d$`, texts[0])
}

func TestStatusBar_UpdateVersionHint(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()

	t.Run("should show hint when a newer version is available", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.2.0"}),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, fa)}
		sb := newStatusBar(du)
		// when
		sb.updateVersionHint(ctx)
		// then
		assert.True(t, sb.updateHint.Visible())
		xassert.Equal(t, "0.1.0", sb.updateHint.current.Text)
		xassert.Equal(t, "0.2.0", sb.updateHint.latest.Text)
	})
	t.Run("should not show hint when there is no newer version", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.1.0"}),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, fa)}
		sb := newStatusBar(du)
		// when
		sb.updateVersionHint(ctx)
		// then
		assert.False(t, sb.updateHint.Visible())
	})
	t.Run("should not show hint when fetching the latest version failed", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewStringResponder(500, "error"),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, fa)}
		sb := newStatusBar(du)
		// when
		sb.updateVersionHint(ctx)
		// then
		assert.False(t, sb.updateHint.Visible())
	})
}

func TestUpdateHint(t *testing.T) {
	test.NewTempApp(t)
	// given
	w := newUpdateHint(false, test.NewWindow(nil))
	defer w.window.Close()
	w.set(github.VersionInfo{Local: "0.1.0", Latest: "0.2.0"})
	var link *widget.Hyperlink
	for _, o := range test.LaidOutObjects(w) {
		if x, ok := o.(*widget.Hyperlink); ok {
			link = x
		}
	}
	require.NotNil(t, link)
	// when
	link.OnTapped()
	// then
	top := w.window.Canvas().Overlays().Top()
	require.NotNil(t, top, "no dialog shown")
	var texts []string
	for _, o := range test.LaidOutObjects(top) {
		if x, ok := o.(*widget.Label); ok {
			texts = append(texts, x.Text)
		}
	}
	assert.Contains(t, texts, "0.1.0")
	assert.Contains(t, texts, "0.2.0")
}
