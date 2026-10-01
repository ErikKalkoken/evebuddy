package xwidget

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
)

func TestFilterChipCompact_Menu(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should create initial menu", func(t *testing.T) {
		// when
		f := NewFilterChipCompact([]FilterOption{
			NewFilterOptionToogle("Alpha"),
		}, nil)

		// then
		assert.Len(t, f.menu.Items, 3)
		it := f.menu.Items[0]
		assert.Equal(t, "Alpha", it.Label)
		assert.Equal(t, f.blankResource, it.Icon)
	})
	t.Run("should show separators", func(t *testing.T) {
		// when
		f := NewFilterChipCompact([]FilterOption{
			NewFilterOptionToogle("Alpha"),
			NewFilterOptionSeparator(),
			NewFilterOptionMultiChoice("Bravo", nil),
		}, nil)

		// then
		var got []string
		for _, it := range f.menu.Items {
			if it.IsSeparator {
				got = append(got, "---")
			} else {
				got = append(got, it.Label)
			}
		}
		assert.Equal(t, []string{"Alpha", "---", "Bravo (0)", "---", "Clear"}, got)
	})
}

func TestFilterChipCompact_ClearItem(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should reset selection and fire OnChanged once", func(t *testing.T) {
		var got []map[string]string
		f := NewFilterChipCompact([]FilterOption{NewFilterOptionToogle("Alpha")}, func(s map[string]string) {
			got = append(got, s)
		})
		f.SetSelected(map[string]string{"Alpha": "Alpha"})
		got = nil

		f.clearItem.Action()

		assert.Equal(t, []map[string]string{{"Alpha": ""}}, got)
		assert.False(t, f.IsOn())
	})
}

func TestFilterChipCompact_EmptyMenu(t *testing.T) {
	test.NewTempApp(t)
	alpha := NewFilterOptionToogle("Alpha")
	cases := []struct {
		name  string
		build func() *FilterChipCompact
	}{
		{"constructor without options", func() *FilterChipCompact {
			return NewFilterChipCompact(nil, nil)
		}},
		{"SetOptions without options", func() *FilterChipCompact {
			f := NewFilterChipCompact(nil, nil)
			f.SetOptions()
			return f
		}},
		{"SetOptions with separator only", func() *FilterChipCompact {
			f := NewFilterChipCompact(nil, nil)
			f.SetOptions(NewFilterOptionSeparator())
			return f
		}},
		{"SetOptions removes all options", func() *FilterChipCompact {
			f := NewFilterChipCompact([]FilterOption{alpha}, nil)
			f.SetOptions()
			return f
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.build()
			assert.Empty(t, f.menu.Items)
		})
	}
}

func TestFilterChipCompact_DisableWhileInteracting(t *testing.T) {
	test.NewTempApp(t)
	newChip := func(t *testing.T) (*FilterChipCompact, fyne.Window) {
		f := NewFilterChipCompact([]FilterOption{NewFilterOptionToogle("Alpha")}, nil)
		w := test.NewWindow(f)
		t.Cleanup(w.Close)
		return f, w
	}
	t.Run("should not show pointer cursor when disabled while hovered", func(t *testing.T) {
		f, _ := newChip(t)
		f.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.PointerCursor, f.Cursor())

		f.Disable()

		assert.Equal(t, desktop.DefaultCursor, f.Cursor())
	})
	t.Run("should not show focus border when disabled while focused", func(t *testing.T) {
		f, w := newChip(t)
		w.Canvas().Focus(f)
		normalWidth := theme.Size(theme.SizeNameInputBorder)
		assert.Greater(t, f.background.StrokeWidth, normalWidth)

		f.Disable()

		assert.Equal(t, normalWidth, f.background.StrokeWidth)
	})
	t.Run("should restore focus border when enabled again while focused", func(t *testing.T) {
		f, w := newChip(t)
		w.Canvas().Focus(f)
		f.Disable()

		f.Enable()

		assert.Greater(t, f.background.StrokeWidth, theme.Size(theme.SizeNameInputBorder))
	})
}

type filterChipOverrideTheme struct {
	fyne.Theme
}

var filterChipOverrideBorder = color.NRGBA{R: 1, G: 2, B: 3, A: 255}

func (th filterChipOverrideTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameInputBorder {
		return filterChipOverrideBorder
	}
	return th.Theme.Color(n, v)
}

func (th filterChipOverrideTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameButtonRadius:
		return 17
	case theme.SizeNameInputBorder:
		return 7
	case theme.SizeNamePadding:
		return 11
	}
	return th.Theme.Size(n)
}

func TestFilterChipCompact_ThemeOverride(t *testing.T) {
	test.NewTempApp(t)
	f := NewFilterChipCompact([]FilterOption{NewFilterOptionToogle("Alpha")}, nil)
	w := test.NewWindow(container.NewThemeOverride(f, filterChipOverrideTheme{test.Theme()}))
	defer w.Close()

	f.Refresh()

	assert.Equal(t, filterChipOverrideBorder, f.background.StrokeColor)
	assert.Equal(t, float32(7), f.background.StrokeWidth)
	assert.Equal(t, float32(17), f.background.CornerRadius)
	assert.Equal(t, f.icon.MinSize().AddWidthHeight(4*11, 4*11), f.MinSize())
}

func TestFilterChipCompact_SetOptionsRestyles(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should clear active styling when selected option is removed", func(t *testing.T) {
		// given
		f := NewFilterChipCompact([]FilterOption{
			NewFilterOptionToogle("Alpha"),
			NewFilterOptionToogle("Bravo"),
		}, nil)
		w := test.NewWindow(f)
		defer w.Close()
		f.SetSelected(map[string]string{"Alpha": "Alpha"})
		assert.NotEqual(t, color.Transparent, f.background.FillColor)

		// when
		f.SetOptions(NewFilterOptionToogle("Bravo"))

		// then
		assert.False(t, f.IsOn())
		assert.Equal(t, color.Transparent, f.background.FillColor)
	})
}

func TestSanitizeSelected(t *testing.T) {
	options := []FilterOption{
		NewFilterOptionToogle("Alpha"),
		NewFilterOptionSeparator(),
		NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
	}
	cases := []struct {
		name     string
		selected map[string]string
		want     map[string]string
	}{
		{
			"should keep valid selection",
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
		},
		{
			"should remove unknown options",
			map[string]string{"Alpha": "Alpha", "Bravo": "one", "Charlie": "two"},
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
		},
		{
			"should reset invalid choices",
			map[string]string{"Alpha": "invalid", "Bravo": "three"},
			map[string]string{"Alpha": "", "Bravo": ""},
		},
		{
			"should keep blank choices",
			map[string]string{"Alpha": "", "Bravo": ""},
			map[string]string{"Alpha": "", "Bravo": ""},
		},
		{
			"should add missing options",
			map[string]string{"Bravo": "two"},
			map[string]string{"Alpha": "", "Bravo": "two"},
		},
		{
			"should add all options when empty",
			map[string]string{},
			map[string]string{"Alpha": "", "Bravo": ""},
		},
		{
			"should add all options when nil",
			nil,
			map[string]string{"Alpha": "", "Bravo": ""},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := sanitizeSelected(options, tc.selected)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeOptions(t *testing.T) {
	alpha := NewFilterOptionToogle("Alpha")
	bravo := NewFilterOptionMultiChoice("Bravo", []string{"one"})
	sep := NewFilterOptionSeparator()
	cases := []struct {
		name    string
		options []FilterOption
		want    []string
	}{
		{"should keep separator between options", []FilterOption{alpha, sep, bravo}, []string{"Alpha", "---", "Bravo"}},
		{"should remove duplicate options", []FilterOption{alpha, bravo, alpha}, []string{"Alpha", "Bravo"}},
		{"should drop leading separator", []FilterOption{sep, alpha, bravo}, []string{"Alpha", "Bravo"}},
		{"should drop trailing separator", []FilterOption{alpha, bravo, sep}, []string{"Alpha", "Bravo"}},
		{"should collapse consecutive separators", []FilterOption{alpha, sep, sep, bravo}, []string{"Alpha", "---", "Bravo"}},
		{"should drop separator left over from removed duplicate", []FilterOption{alpha, sep, alpha}, []string{"Alpha"}},
		{"should collapse separators around removed duplicate", []FilterOption{alpha, sep, alpha, sep, bravo}, []string{"Alpha", "---", "Bravo"}},
		{"should drop undefined options", []FilterOption{alpha, {}, bravo}, []string{"Alpha", "Bravo"}},
		{"should not leave consecutive separators after dropping undefined option", []FilterOption{alpha, sep, {}, sep, bravo}, []string{"Alpha", "---", "Bravo"}},
		{"should return empty for separators only", []FilterOption{sep, sep}, []string{}},
		{"should return empty for nil", nil, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := normalizeOptions(tc.options)

			// then
			names := []string{}
			for _, o := range got {
				if o.kind == optionKindSeparator {
					names = append(names, "---")
				} else {
					names = append(names, o.name)
				}
			}
			assert.Equal(t, tc.want, names)
		})
	}
}
