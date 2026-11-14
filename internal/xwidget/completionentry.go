package xwidget

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CompletionEntry is an entry that suggests matching options in a popup while typing.
// Options match when they contain the text, ignoring case.
//
// While the popup is shown, Up and Down move the highlight, Enter picks the highlighted option
// and Escape closes the popup. Picking an option sets the entry's text, fires OnChanged and then OnPicked.
// A clear button is shown while the entry has text.
type CompletionEntry struct {
	widget.Entry

	// OnPicked is called with the option after the user picked a suggestion.
	// Typing never triggers it, even when the text matches an option.
	OnPicked func(option string)

	// VisibleSuggestions limits the height of the popup to this many rows.
	// Further suggestions can be reached by scrolling. Zero or less means no limit.
	VisibleSuggestions int

	// SuggestionsAbove shows the suggestions above the entry instead of below.
	SuggestionsAbove bool

	clearButton *TappableIcon
	list        *completionList
	options     []string
	popUp       *widget.PopUp
}

// The disabled state lives in Entry, which also provides Disabled.
var _ fyne.Disableable = (*CompletionEntry)(nil)

// NewCompletionEntry returns a new [CompletionEntry] that suggests from options.
func NewCompletionEntry(options []string) *CompletionEntry {
	w := &CompletionEntry{
		VisibleSuggestions: 10,
		options:            slices.Clone(options),
	}
	w.ExtendBaseWidget(w)
	w.clearButton = NewTappableIcon(theme.CancelIcon(), func() {
		w.SetText("")
		w.HideSuggestions()
	})
	w.clearButton.Hide()
	w.ActionItem = container.New(layout.NewCustomPaddedLayout(0, 0, 0, theme.Padding()), w.clearButton)
	return w
}

// Disable disables the entry including its clear button.
func (w *CompletionEntry) Disable() {
	w.Entry.Disable()
	w.clearButton.Disable()
}

// Enable enables the entry including its clear button.
func (w *CompletionEntry) Enable() {
	w.Entry.Enable()
	w.clearButton.Enable()
}

// SetText sets the text without showing suggestions.
func (w *CompletionEntry) SetText(text string) {
	w.Entry.SetText(text)
	w.updateClearButton()
}

// SetOptions replaces the options. Shown suggestions are updated.
func (w *CompletionEntry) SetOptions(options []string) {
	w.options = slices.Clone(options)
	if w.isShowingSuggestions() {
		w.updateSuggestions()
	}
}

// TypedRune updates the suggestions after the rune was typed.
func (w *CompletionEntry) TypedRune(r rune) {
	w.Entry.TypedRune(r)
	w.updateClearButton()
	w.updateSuggestions()
}

// TypedKey updates the suggestions when the key changed the text.
// Down shows the suggestions when they are hidden.
func (w *CompletionEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyDown && !w.isShowingSuggestions() {
		w.updateSuggestions()
		return
	}
	old := w.Text
	w.Entry.TypedKey(key)
	if w.Text != old {
		w.updateClearButton()
		w.updateSuggestions()
	}
}

// TypedShortcut updates the suggestions when the shortcut changed the text, e.g. paste.
func (w *CompletionEntry) TypedShortcut(s fyne.Shortcut) {
	old := w.Text
	w.Entry.TypedShortcut(s)
	if w.Text != old {
		w.updateClearButton()
		w.updateSuggestions()
	}
}

func (w *CompletionEntry) updateClearButton() {
	if w.Text == "" {
		w.clearButton.Hide()
	} else {
		w.clearButton.Show()
	}
}

// HideSuggestions hides the suggestions popup.
func (w *CompletionEntry) HideSuggestions() {
	if w.popUp != nil {
		w.popUp.Hide()
	}
}

func (w *CompletionEntry) isShowingSuggestions() bool {
	return w.popUp != nil && w.popUp.Visible()
}

// suggestions returns the options matching the current text.
func (w *CompletionEntry) suggestions() []string {
	text := strings.ToLower(strings.TrimSpace(w.Text))
	if text == "" {
		return nil
	}
	var matches []string
	for _, o := range w.options {
		if !strings.Contains(strings.ToLower(o), text) {
			continue
		}
		matches = append(matches, o)
	}
	return matches
}

// updateSuggestions shows the suggestions for the current text or hides them when there are none.
func (w *CompletionEntry) updateSuggestions() {
	matches := w.suggestions()
	if len(matches) == 0 || len(matches) == 1 && matches[0] == w.Text {
		w.HideSuggestions()
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(w)
	if c == nil {
		return // not shown yet
	}
	if w.popUp == nil {
		w.list = newCompletionList(w)
		w.popUp = widget.NewPopUp(w.list, c)
	}
	w.list.setItems(matches)
	rows := len(matches)
	if w.VisibleSuggestions > 0 {
		rows = min(rows, w.VisibleSuggestions)
	}
	size := fyne.NewSize(w.Size().Width, w.list.heightFor(rows)+2*theme.Padding())
	w.popUp.Resize(size)
	w.popUp.ShowAtPosition(w.popUpPosition(c, size))
	// Keys go to the focused object of the top overlay, so the list receives them and forwards to the entry.
	c.Focus(w.list)
}

// popUpPosition returns the position of a popup with size below or above the entry.
// Like Fyne's PopUpMenu, the popup is kept inside the canvas' interactive area.
func (w *CompletionEntry) popUpPosition(c fyne.Canvas, size fyne.Size) fyne.Position {
	entryPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(w)
	var pos fyne.Position
	if w.SuggestionsAbove {
		pos = entryPos.SubtractXY(0, size.Height)
	} else {
		pos = entryPos.AddXY(0, w.Size().Height)
	}
	areaPos, areaSize := c.InteractiveArea()
	pos.X = max(min(pos.X, areaPos.X+areaSize.Width-size.Width), areaPos.X)
	pos.Y = max(min(pos.Y, areaPos.Y+areaSize.Height-size.Height), areaPos.Y)
	return pos
}

// pick sets the text to option and hides the suggestions.
func (w *CompletionEntry) pick(option string) {
	w.HideSuggestions()
	w.SetText(option)
	w.CursorColumn = len([]rune(option))
	w.Refresh()
	if w.OnPicked != nil {
		w.OnPicked(option)
	}
}

// completionList shows the suggestions of a [CompletionEntry].
// It has the keyboard focus while shown and forwards typing to the entry.
type completionList struct {
	widget.List

	entry       *CompletionEntry
	highlighted int // -1 means none
	items       []string
}

func newCompletionList(entry *CompletionEntry) *completionList {
	w := &completionList{entry: entry, highlighted: -1}
	w.Length = func() int {
		return len(w.items)
	}
	w.CreateItem = func() fyne.CanvasObject {
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameSelection))
		bg.Hide()
		return container.NewStack(bg, widget.NewLabel(""))
	}
	w.UpdateItem = func(id widget.ListItemID, co fyne.CanvasObject) {
		if id >= len(w.items) {
			return
		}
		c := co.(*fyne.Container)
		bg := c.Objects[0].(*canvas.Rectangle)
		if id == w.highlighted {
			bg.FillColor = theme.Color(theme.ColorNameSelection)
			bg.Show()
		} else {
			bg.Hide()
		}
		c.Objects[1].(*widget.Label).SetText(w.items[id])
	}
	w.OnSelected = func(id widget.ListItemID) {
		w.UnselectAll()
		if id >= 0 && id < len(w.items) {
			w.entry.pick(w.items[id])
		}
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *completionList) setItems(items []string) {
	w.items = items
	w.highlighted = -1
	w.Refresh()
	w.ScrollToTop()
}

// heightFor returns the height needed to show n items.
func (w *completionList) heightFor(n int) float32 {
	if n == 0 {
		return 0
	}
	h := w.CreateItem().MinSize().Height
	return float32(n)*h + float32(n-1)*theme.SeparatorThicknessSize()
}

func (w *completionList) setHighlighted(id int) {
	w.highlighted = id
	w.Refresh()
	w.ScrollTo(id)
}

// FocusGained does nothing. The list shows its own highlight.
func (w *completionList) FocusGained() {}

// FocusLost does nothing. The list shows its own highlight.
func (w *completionList) FocusLost() {}

// TypedKey handles navigation keys and forwards all other keys to the entry.
func (w *completionList) TypedKey(key *fyne.KeyEvent) {
	n := len(w.items)
	switch key.Name {
	case fyne.KeyDown:
		if n > 0 {
			w.setHighlighted((w.highlighted + 1) % n)
		}
	case fyne.KeyUp:
		if n == 0 {
			break
		}
		if w.highlighted <= 0 {
			w.setHighlighted(n - 1)
		} else {
			w.setHighlighted(w.highlighted - 1)
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		if w.highlighted >= 0 && w.highlighted < n {
			w.entry.pick(w.items[w.highlighted])
			return
		}
		w.entry.HideSuggestions()
		w.entry.TypedKey(key)
	case fyne.KeyEscape:
		w.entry.HideSuggestions()
	case fyne.KeyTab:
		w.entry.HideSuggestions()
		if c := fyne.CurrentApp().Driver().CanvasForObject(w.entry); c != nil {
			c.FocusNext()
		}
	default:
		w.entry.TypedKey(key)
	}
}

// TypedRune forwards the rune to the entry.
func (w *completionList) TypedRune(r rune) {
	w.entry.TypedRune(r)
}

// TypedShortcut forwards the shortcut to the entry, e.g. paste.
func (w *completionList) TypedShortcut(s fyne.Shortcut) {
	w.entry.TypedShortcut(s)
}
