package core

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestMakeAboutPage(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// findHyperlink returns the hyperlink with the given text in c.
	findHyperlink := func(t *testing.T, c fyne.CanvasObject, text string) *widget.Hyperlink {
		t.Helper()
		for _, o := range test.LaidOutObjects(c) {
			if x, ok := o.(*widget.Hyperlink); ok && x.Text == text {
				return x
			}
		}
		t.Fatalf("hyperlink not found: %s", text)
		return nil
	}

	// isParentVisible reports whether the container holding o is visible.
	isParentVisible := func(t *testing.T, c fyne.CanvasObject, o fyne.CanvasObject) bool {
		t.Helper()
		for _, x := range test.LaidOutObjects(c) {
			if x, ok := x.(*fyne.Container); ok {
				for _, child := range x.Objects {
					if child == o {
						return x.Visible()
					}
				}
			}
		}
		t.Fatal("parent not found")
		return false
	}

	t.Run("should show version", func(t *testing.T) {
		// given
		httpmock.Reset()
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "v0.1.0"})
		u := MakeFakeBaseUI(t, st, fa)
		// when
		c := makeAboutPage(u)
		// then
		var texts []string
		for _, o := range test.LaidOutObjects(c) {
			if x, ok := o.(*widget.Label); ok {
				texts = append(texts, x.Text)
			}
		}
		assert.Contains(t, texts, "0.1.0")
		link := findHyperlink(t, c, "What's new")
		assert.Contains(t, link.URL.String(), "releases/v0.1.0")
	})
	t.Run("should show ? when version is invalid", func(t *testing.T) {
		// given
		httpmock.Reset()
		fa := NewFakeApp(t)
		fa.SetMetadata(fyne.AppMetadata{Version: "invalid"})
		u := MakeFakeBaseUI(t, st, fa)
		// when
		c := makeAboutPage(u)
		// then
		var texts []string
		for _, o := range test.LaidOutObjects(c) {
			if x, ok := o.(*widget.Label); ok {
				texts = append(texts, x.Text)
			}
		}
		assert.Contains(t, texts, "?")
	})
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
		// when
		c := makeAboutPage(u)
		// then
		link := findHyperlink(t, c, "Download")
		assert.True(t, isParentVisible(t, c, link))
		assert.Contains(t, link.URL.String(), "releases/tag/v0.2.0")
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
		// when
		c := makeAboutPage(u)
		// then
		link := findHyperlink(t, c, "Download")
		assert.False(t, isParentVisible(t, c, link))
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
		// when
		c := makeAboutPage(u)
		// then
		link := findHyperlink(t, c, "Download")
		assert.False(t, isParentVisible(t, c, link))
	})
	t.Run("should show debug tools in developer mode", func(t *testing.T) {
		// given
		httpmock.Reset()
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.SetDeveloperMode(true)
		// when
		c := makeAboutPage(u)
		// then
		var button *widget.Button
		var entry *widget.Entry
		for _, o := range test.LaidOutObjects(c) {
			switch x := o.(type) {
			case *widget.Button:
				button = x
			case *widget.Entry:
				entry = x
			}
		}
		require.NotNil(t, button)
		require.NotNil(t, entry)
		xassert.Equal(t, "Show Snackbar (debug)", button.Text)
		assert.True(t, button.Visible())
		assert.True(t, entry.Visible())
	})
	t.Run("should hide debug tools when not in developer mode", func(t *testing.T) {
		// given
		httpmock.Reset()
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// when
		c := makeAboutPage(u)
		// then
		var button *widget.Button
		var entry *widget.Entry
		for _, o := range test.LaidOutObjects(c) {
			switch x := o.(type) {
			case *widget.Button:
				button = x
			case *widget.Entry:
				entry = x
			}
		}
		require.NotNil(t, button)
		require.NotNil(t, entry)
		assert.False(t, button.Visible())
		assert.False(t, entry.Visible())
	})
}
