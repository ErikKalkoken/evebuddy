package widget

import (
	"slices"

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

	OnChanged func(selected []string)

	Options  []string // readonly TODO: Enable setting options
	Selected []string // readonly after first render

	chips    []*FilterChip
	options  []string
	selected []string
}

// NewFilterChipGroup returns a new [FilterChipGroup].
func NewFilterChipGroup(options []string, changed func([]string)) *FilterChipGroup {
	optionsCleaned := slices.DeleteFunc(deduplicateSlice(options), func(v string) bool {
		return v == ""
	})
	w := &FilterChipGroup{
		chips:     make([]*FilterChip, 0),
		OnChanged: changed,
		options:   optionsCleaned,
		Options:   slices.Clone(optionsCleaned),
		Selected:  make([]string, 0),
	}
	w.ExtendBaseWidget(w)
	isSelected := make(map[string]bool)
	for _, v := range w.options {
		w.chips = append(w.chips, NewFilterChip(v, func(on bool) {
			if on {
				isSelected[v] = true
			} else {
				isSelected[v] = false
			}
			w.selected = make([]string, 0)
			for _, o := range w.options {
				if isSelected[o] {
					w.selected = append(w.selected, o)
				}
			}
			w.Selected = slices.Clone(w.selected)
			if w.OnChanged != nil {
				w.OnChanged(w.Selected)
			}
		}))
	}
	return w
}

func (w *FilterChipGroup) CreateRenderer() fyne.WidgetRenderer {
	w.SetSelected(w.Selected)
	p := w.Theme().Size(theme.SizeNamePadding)
	box := container.New(ilayout.NewRowWrapLayoutWithCustomPadding(2*p, 2*p))
	for _, c := range w.chips {
		box.Add(c)
	}
	return widget.NewSimpleRenderer(container.New(layout.NewCustomPaddedLayout(2*p, 2*p, 0, 0), box))
}

// SetSelected updates the selected options.
// Invalid elements including empty strings will be ignored.
func (w *FilterChipGroup) SetSelected(s []string) {
	isValid := make(map[string]bool)
	for _, v := range w.options {
		isValid[v] = true
	}
	isSelected := make(map[string]bool)
	for _, v := range s {
		if !isValid[v] {
			continue
		}
		isSelected[v] = true
	}
	for i, v := range w.options {
		w.chips[i].SetState(isSelected[v])
	}
}

func deduplicateSlice[S ~[]E, E comparable](s S) []E {
	seen := make(map[E]bool)
	s2 := make([]E, 0)
	for _, v := range s {
		if seen[v] {
			continue
		}
		s2 = append(s2, v)
		seen[v] = true
	}
	return s2
}
