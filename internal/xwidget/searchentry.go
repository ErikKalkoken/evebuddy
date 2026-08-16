package xwidget

import (
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// SearchEntry is a Fyne widget that shows a search entry.
type SearchEntry struct {
	widget.Entry
	clearButton *kxwidget.IconButton
}

// NewSearchEntry returns a new SearchEntry widget.
func NewSearchEntry(placeholder string, changed func(string)) *SearchEntry {
	w := &SearchEntry{}
	w.ExtendBaseWidget(w)
	w.PlaceHolder = placeholder
	w.clearButton = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		w.SetText("")
	})
	w.clearButton.Hide()
	w.ActionItem = w.clearButton
	w.SetOnChanged(changed)
	return w
}

func (w *SearchEntry) SetOnChanged(changed func(s string)) {
	w.OnChanged = func(s string) {
		w.updateClearButton(s)
		changed(s)
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
