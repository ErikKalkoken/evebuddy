package widget

import (
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ilayout "github.com/ErikKalkoken/evebuddy/internal/layout"
)

// FilterCheckGround represents a group of filter chips.
// Filter chips use tags or descriptive words to filter content.
type FilterChipGroup struct {
	widget.DisableableWidget

	OnChanged func([]string)

	mu       sync.RWMutex
	Selected []string

	options []string
	chips   []*FilterChip
}

// NewFilterChipGroup returns a new [FilterChipGroup].
func NewFilterChipGroup(options []string, changed func([]string)) *FilterChipGroup {
	w := &FilterChipGroup{
		chips:     make([]*FilterChip, 0),
		OnChanged: changed,
		options:   options,
		Selected:  make([]string, 0),
	}
	w.ExtendBaseWidget(w)
	for _, o := range options {
		if o == "" {
			panic("Empty strings are not allowed as options")
		}
		w.chips = append(w.chips, NewFilterChip(o, func(selected bool) {
			w.toggleOption(o, selected)
			if w.OnChanged != nil {
				w.OnChanged(slices.Clone(w.Selected))
			}
		}))
	}
	return w
}

func (w *FilterChipGroup) toggleOption(o string, selected bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if selected {
		if slices.IndexFunc(w.Selected, func(s string) bool {
			return s == o
		}) != 0 {
			w.Selected = append(w.Selected, o)
		}
	} else {
		w.Selected = slices.DeleteFunc(w.Selected, func(s string) bool {
			return s == o
		})
	}
}

// SetSelected updates the selected options.
func (w *FilterChipGroup) SetSelected(s []string) {
	w.mu.Lock()
	w.Selected = slices.Clone(s)
	w.mu.Unlock()
	w.Refresh()

}

// Options returns the options.
func (w *FilterChipGroup) Options() []string {
	return slices.Clone(w.options)
}

func (w *FilterChipGroup) update() {
	optionMap := make(map[string]bool)
	for _, v := range w.options {
		optionMap[v] = true
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, v := range w.Selected {
		if v == "" {
			panic("Empty string in Selected")
		}
		if !optionMap[v] {
			panic("Invalid value in Selected: " + v)
		}
	}
	selected := make(map[string]bool)
	for _, v := range w.Selected {
		selected[v] = true
	}
	for i, v := range w.options {
		w.chips[i].Selected = selected[v]
	}
}

func (w *FilterChipGroup) Refresh() {
	w.update()
	for _, cf := range w.chips {
		cf.Refresh()
	}
	w.BaseWidget.Refresh()
}

func (w *FilterChipGroup) CreateRenderer() fyne.WidgetRenderer {
	w.update()
	p := w.Theme().Size(theme.SizeNamePadding)
	box := container.New(ilayout.NewRowWrapLayoutWithCustomPadding(2*p, 2*p))
	for _, c := range w.chips {
		box.Add(c)
	}
	return widget.NewSimpleRenderer(container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 0, 0), box))
}
