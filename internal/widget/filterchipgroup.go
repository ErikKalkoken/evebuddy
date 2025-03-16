package widget

import (
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FilterCheckGround represents a group of [FilterChip].
type FilterChipGroup struct {
	widget.DisableableWidget

	OnChanged func([]string)

	mu       sync.RWMutex
	Selected []string

	options   []string
	optionMap map[string]bool
	chips     []*FilterChip
}

func NewFilterChipGroup(options []string, changed func([]string)) *FilterChipGroup {
	w := &FilterChipGroup{
		chips:     make([]*FilterChip, 0),
		OnChanged: changed,
		options:   options,
		Selected:  make([]string, 0),
	}
	w.ExtendBaseWidget(w)
	w.optionMap = make(map[string]bool)
	for _, o := range options {
		if o == "" {
			panic("Empty strings are not allowed as options")
		}
		w.optionMap[o] = true
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

func (w *FilterChipGroup) SetSelected(s []string) {
	w.mu.Lock()
	w.Selected = slices.Clone(s)
	w.mu.Unlock()
	w.Refresh()

}
func (w *FilterChipGroup) Options() []string {
	return slices.Clone(w.options)
}

func (w *FilterChipGroup) update() {
	w.mu.RLock()
	for _, v := range w.Selected {
		if v == "" {
			panic("Empty string in Selected")
		}
		if !w.optionMap[v] {
			panic("Invalid value in Selected: " + v)
		}
	}
	selected := make(map[string]bool)
	for _, v := range w.Selected {
		selected[v] = true
	}
	w.mu.RUnlock()
	for i, v := range w.options {
		w.chips[i].IsSelected = selected[v]
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
	box := container.New(layout.NewCustomPaddedHBoxLayout(3 * p))
	for _, c := range w.chips {
		box.Add(c)
	}
	return widget.NewSimpleRenderer(box)
}
