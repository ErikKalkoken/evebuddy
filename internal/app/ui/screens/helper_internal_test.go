package screens

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

// renderedTexts returns all texts drawn by a widget.
func renderedTexts(w fyne.Widget) []string {
	var texts []string
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch x := o.(type) {
		case *canvas.Text:
			texts = append(texts, x.Text)
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range test.WidgetRenderer(x).Objects() {
				walk(c)
			}
		}
	}
	walk(w)
	return texts
}

func TestClearSelectsSilent(t *testing.T) {
	test.NewTempApp(t)
	var calls int
	chip := kxwidget.NewFilterChipSelect("Category", []string{"Ship", "Module"}, func(string) {
		calls++
	})
	w := test.NewWindow(chip)
	defer w.Close()
	chip.SetSelected("Ship")
	assert.Contains(t, renderedTexts(chip), "Ship")
	calls = 0

	clearSelectsSilent(chip)

	assert.Empty(t, chip.Selected)
	assert.Zero(t, calls)
	texts := renderedTexts(chip)
	assert.Contains(t, texts, "Category")
	assert.NotContains(t, texts, "Ship")
}

func TestSwapReloadedCharacter(t *testing.T) {
	old := &app.Character{ID: 1}
	reloaded := &app.Character{ID: 1}
	cases := []struct {
		name    string
		current *app.Character // stored when the swap happens
		want    bool
		stored  *app.Character
	}{
		{"unchanged", old, true, reloaded},
		{"replaced by same character", &app.Character{ID: 1}, true, nil}, // keeps the other copy
		{"replaced by other character", &app.Character{ID: 2}, false, nil},
		{"removed", nil, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p atomic.Pointer[app.Character]
			p.Store(tc.current)

			got := swapReloadedCharacter(&p, old, reloaded)

			assert.Equal(t, tc.want, got)
			want := tc.stored
			if want == nil {
				want = tc.current
			}
			assert.Same(t, want, p.Load())
		})
	}
}

func TestCopyRowsToClipboard(t *testing.T) {
	t.Run("copies transformed rows to clipboard", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		fyneApp := test.NewTempApp(t)
		var snackbar []string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 fyneApp,
			Storage:             st,
			DisplaySnackbarFunc: func(text string) { snackbar = append(snackbar, text) },
		})
		rows := []string{"alpha", "bravo"}
		// when
		copyRowsToClipboard(u, "names", rows, func(rows []string) (string, error) {
			return strings.Join(rows, "\n"), nil
		})
		// then
		xassert.Equal(t, "alpha\nbravo", fyneApp.Clipboard().Content())
		xassert.Equal(t, []string{"Copied names to clipboard"}, snackbar)
	})
	t.Run("reports error when rows can not be transformed", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		fyneApp := test.NewTempApp(t)
		var snackbar []string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 fyneApp,
			Storage:             st,
			DisplaySnackbarFunc: func(text string) { snackbar = append(snackbar, text) },
		})
		fyneApp.Clipboard().SetContent("old")
		// when
		copyRowsToClipboard(u, "names", []string{"alpha"}, func([]string) (string, error) {
			return "", errors.New("failed")
		})
		// then
		xassert.Equal(t, "old", fyneApp.Clipboard().Content())
		xassert.Equal(t, []string{"ERROR: Failed to copy names to clipboard"}, snackbar)
	})
	t.Run("copies rows as they were when called", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		fyneApp := test.NewTempApp(t)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     fyneApp,
			Storage: st,
		})
		var pending []func() // run background work later, like the real app
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		rows := []string{"alpha"}
		copyRowsToClipboard(u, "names", rows, func(rows []string) (string, error) {
			return strings.Join(rows, "\n"), nil
		})
		rows[0] = "changed"
		// when
		for _, f := range pending {
			f()
		}
		// then
		xassert.Equal(t, "alpha", fyneApp.Clipboard().Content())
	})
}
