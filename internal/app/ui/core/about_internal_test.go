package core

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestAboutPage(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should show version and release notes", func(t *testing.T) {
		// given
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "v0.1.0"})
		u := MakeFakeBaseUI(t, st, fa)
		u.tasks.Stop() // don't check for updates
		// when
		w := newAboutPage(u)
		// then
		xassert.Equal(t, "0.1.0", w.currentVersionLabel.Text)
		assert.True(t, w.releaseNotesLink.Visible())
		assert.Contains(t, w.releaseNotesLink.URL.String(), "releases/v0.1.0")
	})
	t.Run("should show ? and hide release notes when version is invalid", func(t *testing.T) {
		// given
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "invalid"})
		u := MakeFakeBaseUI(t, st, fa)
		u.tasks.Stop()
		// when
		w := newAboutPage(u)
		// then
		xassert.Equal(t, "?", w.currentVersionLabel.Text)
		assert.False(t, w.releaseNotesLink.Visible())
	})
	t.Run("should show debug tools in developer mode", func(t *testing.T) {
		// given
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.tasks.Stop()
		u.SetDeveloperMode(true)
		// when
		w := newAboutPage(u)
		// then
		assert.True(t, w.techInfoBox.Visible())
		assert.True(t, w.showSnackbarButton.Visible())
		assert.True(t, w.dummyEntry.Visible())
	})
	t.Run("should hide debug tools when not in developer mode", func(t *testing.T) {
		// given
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.tasks.Stop()
		// when
		w := newAboutPage(u)
		// then
		assert.False(t, w.techInfoBox.Visible())
		assert.False(t, w.showSnackbarButton.Visible())
		assert.False(t, w.dummyEntry.Visible())
	})
}

func TestAboutPage_CheckForUpdate(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()

	t.Run("should show update when a newer version is available", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.2.0"}),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeBaseUI(t, st, fa)
		u.tasks.Stop()
		w := newAboutPage(u)
		// when
		w.checkForUpdate(ctx)
		// then
		assert.True(t, w.updateAvailableRow.Visible())
		assert.Contains(t, w.updateAvailableLink.URL.String(), "releases/tag/v0.2.0")
	})
	t.Run("should not show update when there is no newer version", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.1.0"}),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeBaseUI(t, st, fa)
		u.tasks.Stop()
		w := newAboutPage(u)
		// when
		w.checkForUpdate(ctx)
		// then
		assert.False(t, w.updateAvailableRow.Visible())
	})
	t.Run("should not show update when fetching the latest version failed", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewStringResponder(500, "error"),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeBaseUI(t, st, fa)
		u.tasks.Stop()
		w := newAboutPage(u)
		// when
		w.checkForUpdate(ctx)
		// then
		assert.False(t, w.updateAvailableRow.Visible())
	})
	t.Run("should not check for updates in offline mode", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/ErikKalkoken/evebuddy/releases/latest",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{"tag_name": "v0.2.0"}),
		)
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "0.1.0"})
		u := MakeFakeBaseUI(t, st, fa)
		u.isOfflineMode = true
		// when
		w := newAboutPage(u)
		u.tasks.Stop() // waits for a started check
		// then
		xassert.Equal(t, 0, httpmock.GetTotalCallCount())
		assert.False(t, w.updateAvailableRow.Visible())
	})
}
