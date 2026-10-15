package screens

import (
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func TestLatestRun(t *testing.T) {
	var l latestRun
	isLatest1 := l.start()
	assert.True(t, isLatest1())
	isLatest2 := l.start()
	assert.False(t, isLatest1())
	assert.True(t, isLatest2())
	l.start()
	assert.False(t, isLatest2())
}

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
