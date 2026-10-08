package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/stretchr/testify/assert"
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

func TestNewModeChip(t *testing.T) {
	choices := []string{"Beta", "Alpha"}
	chip := newModeChip(choices, "Beta", func(string) {})
	assert.Empty(t, chip.Placeholder)
	assert.Equal(t, "Beta", chip.Selected)
	assert.Equal(t, []string{"Beta", "Alpha"}, chip.Options)
	assert.True(t, chip.ClearDisabled)
	assert.True(t, chip.SortDisabled)
	choices[0] = "Gamma"
	assert.Equal(t, "Beta", chip.Options[0], "choices must be cloned")
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
