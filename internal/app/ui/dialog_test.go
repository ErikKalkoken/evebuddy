package ui_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestShowErrorAndLog(t *testing.T) {
	test.NewTempApp(t)

	// labelTexts returns the texts of all labels shown in the top overlay of w.
	labelTexts := func(w fyne.Window) []string {
		var texts []string
		for _, o := range test.LaidOutObjects(w.Canvas().Overlays().Top()) {
			if x, ok := o.(*widget.Label); ok {
				texts = append(texts, x.Text)
			}
		}
		return texts
	}

	t.Run("shows message and error for users", func(t *testing.T) {
		// given
		w := test.NewWindow(nil)
		defer w.Close()
		err := fmt.Errorf("load: %w", app.ErrTokenError)
		// when
		ui.ShowErrorAndLog("Failed to load", err, false, w)
		// then
		require.NotNil(t, w.Canvas().Overlays().Top())
		assert.Contains(t, labelTexts(w), "Failed to load")
		assert.Contains(t, labelTexts(w), "token error")
		assert.NotContains(t, labelTexts(w), err.Error())
	})
	t.Run("shows raw error for developers", func(t *testing.T) {
		// given
		w := test.NewWindow(nil)
		defer w.Close()
		err := fmt.Errorf("load: %w", app.ErrTokenError)
		// when
		ui.ShowErrorAndLog("Failed to load", err, true, w)
		// then
		require.NotNil(t, w.Canvas().Overlays().Top())
		assert.Contains(t, labelTexts(w), err.Error())
	})
	t.Run("shows no dialog for canceled operations", func(t *testing.T) {
		// given
		w := test.NewWindow(nil)
		defer w.Close()
		// when
		ui.ShowErrorAndLog("Failed to load", context.Canceled, false, w)
		// then
		assert.Nil(t, w.Canvas().Overlays().Top())
	})
}

func TestNotifyErrorAndLog(t *testing.T) {
	t.Run("notifies with message", func(t *testing.T) {
		// given
		var notified []string
		// when
		ui.NotifyErrorAndLog("Failed to load", errors.New("boom"), func(s string) {
			notified = append(notified, s)
		})
		// then
		xassert.Equal(t, []string{"Failed to load"}, notified)
	})
	t.Run("does not notify for canceled operations", func(t *testing.T) {
		// given
		var notified []string
		// when
		ui.NotifyErrorAndLog("Failed to load", app.ErrCanceled, func(s string) {
			notified = append(notified, s)
		})
		// then
		assert.Empty(t, notified)
	})
}

func TestShowConfirm(t *testing.T) {
	test.NewTempApp(t)

	// button returns the button with label s shown in the top overlay of w.
	button := func(w fyne.Window, s string) *widget.Button {
		for _, o := range test.LaidOutObjects(w.Canvas().Overlays().Top()) {
			if x, ok := o.(*widget.Button); ok && x.Text == s {
				return x
			}
		}
		return nil
	}

	t.Run("reports confirmation", func(t *testing.T) {
		// given
		w := test.NewWindow(nil)
		defer w.Close()
		var got []bool
		ui.ShowConfirm("Delete", "Delete it?", "Delete", func(b bool) {
			got = append(got, b)
		}, w)
		b := button(w, "Delete")
		require.NotNil(t, b)
		// when
		test.Tap(b)
		// then
		xassert.Equal(t, []bool{true}, got)
		xassert.Equal(t, widget.DangerImportance, b.Importance)
		assert.Nil(t, w.Canvas().Overlays().Top())
	})
	t.Run("reports cancellation", func(t *testing.T) {
		// given
		w := test.NewWindow(nil)
		defer w.Close()
		var got []bool
		ui.ShowConfirm("Delete", "Delete it?", "Delete", func(b bool) {
			got = append(got, b)
		}, w)
		b := button(w, "Cancel")
		require.NotNil(t, b)
		// when
		test.Tap(b)
		// then
		xassert.Equal(t, []bool{false}, got)
		assert.Nil(t, w.Canvas().Overlays().Top())
	})
}
