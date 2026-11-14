package xwidget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCompletionEntry(t *testing.T) {
	options := []string{"Navigation", "Afterburner", "Advanced Navigation", "Evasive Maneuvering"}
	newEntry := func(t *testing.T) (*CompletionEntry, fyne.Window) {
		test.NewTempApp(t)
		e := NewCompletionEntry(options)
		w := test.NewWindow(container.NewVBox(e))
		t.Cleanup(w.Close)
		w.Resize(fyne.NewSize(300, 400))
		w.Canvas().Focus(e)
		return e, w
	}
	// typeText sends runes to the focused object, like the keyboard does.
	typeText := func(w fyne.Window, text string) {
		for _, r := range text {
			w.Canvas().Focused().TypedRune(r)
		}
	}
	typeKey := func(w fyne.Window, name fyne.KeyName) {
		w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: name})
	}
	t.Run("should show options containing the text ignoring case in option order", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		// when
		typeText(w, "NAV")
		// then
		require.True(t, e.isShowingSuggestions())
		xassert.Equal(t, []string{"Navigation", "Advanced Navigation"}, e.list.items)
		assert.Same(t, w.Canvas().Focused(), fyne.Focusable(e.list))
	})
	t.Run("should show all matches and limit the popup height to VisibleSuggestions", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		e.VisibleSuggestions = 1
		// when
		typeText(w, "nav")
		// then
		xassert.Equal(t, []string{"Navigation", "Advanced Navigation"}, e.list.items)
		xassert.Equal(t, e.list.heightFor(1)+2*theme.Padding(), e.popUp.Content.Size().Height)
	})
	t.Run("can pick a suggestion beyond the visible rows with keyboard", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		e.VisibleSuggestions = 1
		typeText(w, "nav")
		// when
		typeKey(w, fyne.KeyDown)
		typeKey(w, fyne.KeyDown)
		typeKey(w, fyne.KeyReturn)
		// then
		xassert.Equal(t, "Advanced Navigation", e.Text)
	})
	t.Run("should forward typing to the entry while suggestions are shown", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		// when
		typeText(w, "ig")
		typeKey(w, fyne.KeyBackspace)
		// then
		xassert.Equal(t, "navi", e.Text)
		xassert.Equal(t, []string{"Navigation", "Advanced Navigation"}, e.list.items)
	})
	t.Run("should forward shortcuts to the entry while suggestions are shown", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "a")
		fyne.CurrentApp().Clipboard().SetContent("fter")
		// when
		w.Canvas().Focused().(fyne.Shortcutable).TypedShortcut(&fyne.ShortcutPaste{
			Clipboard: fyne.CurrentApp().Clipboard(),
		})
		// then
		xassert.Equal(t, "after", e.Text)
		xassert.Equal(t, []string{"Afterburner"}, e.list.items)
	})
	t.Run("should hide suggestions when nothing matches", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		// when
		typeText(w, "x")
		// then
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should hide suggestions when text becomes empty", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "n")
		require.True(t, e.isShowingSuggestions())
		// when
		typeKey(w, fyne.KeyBackspace)
		// then
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should not show suggestions when the only match equals the text", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		// when
		typeText(w, "Afterburner")
		// then
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should not call OnPicked when typing matches an option", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		var called bool
		e.OnPicked = func(string) {
			called = true
		}
		// when
		typeText(w, "Afterburner")
		// then
		xassert.Equal(t, "Afterburner", e.Text)
		assert.False(t, called)
	})
	t.Run("can pick highlighted suggestion with keyboard", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		var changed, picked string
		e.OnChanged = func(s string) {
			changed = s
		}
		e.OnPicked = func(s string) {
			picked = s
		}
		typeText(w, "nav")
		// when
		typeKey(w, fyne.KeyDown)
		typeKey(w, fyne.KeyDown)
		typeKey(w, fyne.KeyReturn)
		// then
		xassert.Equal(t, "Advanced Navigation", e.Text)
		xassert.Equal(t, "Advanced Navigation", changed)
		xassert.Equal(t, "Advanced Navigation", picked)
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should highlight last suggestion when moving up without highlight", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		// when
		typeKey(w, fyne.KeyUp)
		typeKey(w, fyne.KeyReturn)
		// then
		xassert.Equal(t, "Advanced Navigation", e.Text)
	})
	t.Run("can pick suggestion by tapping", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		var picked string
		e.OnPicked = func(s string) {
			picked = s
		}
		typeText(w, "nav")
		// when
		e.list.Select(0) // same as tapping the first row
		// then
		xassert.Equal(t, "Navigation", e.Text)
		xassert.Equal(t, "Navigation", picked)
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should hide suggestions on escape and keep text", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		// when
		typeKey(w, fyne.KeyEscape)
		// then
		assert.False(t, e.isShowingSuggestions())
		xassert.Equal(t, "nav", e.Text)
		assert.Same(t, w.Canvas().Focused(), fyne.Focusable(e))
	})
	t.Run("should submit when enter is pressed without highlight", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		var submitted string
		e.OnSubmitted = func(s string) {
			submitted = s
		}
		typeText(w, "nav")
		// when
		typeKey(w, fyne.KeyReturn)
		// then
		xassert.Equal(t, "nav", submitted)
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should submit when enter is pressed while suggestions are hidden", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		var submitted string
		e.OnSubmitted = func(s string) {
			submitted = s
		}
		typeText(w, "Afterburner")
		require.False(t, e.isShowingSuggestions())
		// when
		typeKey(w, fyne.KeyReturn)
		// then
		xassert.Equal(t, "Afterburner", submitted)
	})
	t.Run("should show suggestions again with down key", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		typeKey(w, fyne.KeyEscape)
		// when
		typeKey(w, fyne.KeyDown)
		// then
		assert.True(t, e.isShowingSuggestions())
	})
	t.Run("should position suggestions below the entry by default", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		// when
		got := e.popUpPosition(w.Canvas(), fyne.NewSize(100, 100))
		// then
		entryPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(e)
		xassert.Equal(t, entryPos.AddXY(0, e.Size().Height), got)
	})
	t.Run("should position suggestions above the entry when configured", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		e := NewCompletionEntry(options)
		e.SuggestionsAbove = true
		w := test.NewWindow(container.NewBorder(nil, e, nil, nil))
		defer w.Close()
		w.Resize(fyne.NewSize(300, 400))
		// when
		got := e.popUpPosition(w.Canvas(), fyne.NewSize(100, 100))
		// then
		entryPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(e)
		xassert.Equal(t, entryPos.SubtractXY(0, 100), got)
	})
	t.Run("should keep suggestions inside the window when they do not fit below", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		e := NewCompletionEntry(options)
		w := test.NewWindow(container.NewBorder(nil, e, nil, nil))
		defer w.Close()
		w.Resize(fyne.NewSize(300, 400))
		size := fyne.NewSize(100, 100)
		// when
		got := e.popUpPosition(w.Canvas(), size)
		// then
		xassert.Equal(t, w.Canvas().Size().Height-size.Height, got.Y)
	})
	t.Run("should keep suggestions inside the window when they do not fit above", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		e.SuggestionsAbove = true
		// when
		got := e.popUpPosition(w.Canvas(), fyne.NewSize(100, 100))
		// then
		xassert.Equal(t, float32(0), got.Y)
	})
	t.Run("should keep suggestions inside the window when they do not fit to the right", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		size := fyne.NewSize(400, 100)
		// when
		got := e.popUpPosition(w.Canvas(), size)
		// then
		xassert.Equal(t, float32(0), got.X)
	})
	t.Run("should not show suggestions when text is set programmatically", func(t *testing.T) {
		// given
		e, _ := newEntry(t)
		// when
		e.SetText("nav")
		// then
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should update shown suggestions when options change", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		// when
		e.SetOptions([]string{"Navigation", "Warp Drive Operation"})
		// then
		xassert.Equal(t, []string{"Navigation"}, e.list.items)
	})
	t.Run("should hide clear button when empty", func(t *testing.T) {
		// when
		e, _ := newEntry(t)
		// then
		assert.False(t, e.clearButton.Visible())
	})
	t.Run("should show clear button after typing", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		// when
		typeText(w, "n")
		// then
		assert.True(t, e.clearButton.Visible())
	})
	t.Run("can clear text with clear button", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		var changed string
		e.OnChanged = func(s string) {
			changed = s
		}
		// when
		test.Tap(e.clearButton)
		// then
		xassert.Equal(t, "", e.Text)
		xassert.Equal(t, "", changed)
		assert.False(t, e.clearButton.Visible())
		assert.False(t, e.isShowingSuggestions())
	})
	t.Run("should show clear button when text is set", func(t *testing.T) {
		// given
		e, _ := newEntry(t)
		// when
		e.SetText("nav")
		// then
		assert.True(t, e.clearButton.Visible())
	})
	t.Run("should hide clear button when text is set to empty", func(t *testing.T) {
		// given
		e, _ := newEntry(t)
		e.SetText("nav")
		// when
		e.SetText("")
		// then
		assert.False(t, e.clearButton.Visible())
	})
	t.Run("should show clear button after picking a suggestion", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		e.clearButton.Hide() // so only the pick can show it
		// when
		e.list.Select(0)
		// then
		assert.True(t, e.clearButton.Visible())
	})
	t.Run("should disable clear button with entry", func(t *testing.T) {
		// given
		e, w := newEntry(t)
		typeText(w, "nav")
		// when
		e.Disable()
		// then
		assert.True(t, e.clearButton.Disabled())
	})
}
