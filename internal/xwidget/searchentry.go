package xwidget

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SearchEntry is a Fyne widget that shows a search entry.
type SearchEntry struct {
	widget.Entry
	clearButton *TappableIcon
}

// NewSearchEntry returns a new SearchEntry widget.
func NewSearchEntry(placeholder string, changed func(string)) *SearchEntry {
	w := &SearchEntry{}
	w.ExtendBaseWidget(w)
	w.PlaceHolder = placeholder
	w.clearButton = NewTappableIcon(theme.CancelIcon(), func() {
		w.SetText("")
	})
	w.clearButton.SetToolTip("Clear search")
	w.clearButton.Hide()
	p := theme.Padding()
	w.ActionItem = container.New(layout.NewCustomPaddedLayout(0, 0, 0, p), w.clearButton)
	w.SetOnChanged(changed)
	return w
}

// ClearSilent clears the entry without calling OnChanged.
func (w *SearchEntry) ClearSilent() {
	w.updateClearButton("")
	w.Text = ""
	w.Refresh()
}

func (w *SearchEntry) SetOnChanged(changed func(s string)) {
	w.OnChanged = func(s string) {
		w.updateClearButton(s)
		if changed != nil {
			changed(s)
		}
	}
}

func (w *SearchEntry) SetText(text string) {
	w.updateClearButton(text)
	w.Entry.SetText(text)
}

func (w *SearchEntry) updateClearButton(s string) {
	if s == "" {
		w.clearButton.Hide()
	} else {
		w.clearButton.Show()
	}
}
