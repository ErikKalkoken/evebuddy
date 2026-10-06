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
	t.Run("should show cleaned up choices", func(t *testing.T) {
		// when
		f := NewFilterChipCompact([]FilterOption{
			NewFilterOptionMultiChoice("Bravo", []string{"b", "a", "", "b", "C"}),
		}, nil)

		// then
		it := f.menu.Items[0]
		assert.Equal(t, "Bravo (3)", it.Label)
		var got []string
		for _, it2 := range it.ChildMenu.Items {
			got = append(got, it2.Label)
		}
		assert.Equal(t, []string{"C", "a", "b"}, got)
	})
}

func TestNewFilterOptionMultiChoice(t *testing.T) {
	cases := []struct {
		name    string
		choices []string
		want    []string
	}{
		{"should drop blanks, deduplicate and sort", []string{"b", "a", "", "b", "C"}, []string{"C", "a", "b"}},
		{"should return empty for nil", nil, nil},
		{"should return empty for empty", []string{}, nil},
		{"should return empty for blanks only", []string{"", ""}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewFilterOptionMultiChoice("Bravo", tc.choices)
			if tc.want == nil {
				assert.Empty(t, o.choices)
			} else {
				assert.Equal(t, tc.want, o.choices)
			}
		})
	}
	t.Run("should not share the caller's slice", func(t *testing.T) {
		choices := []string{"a", "b"}
		o := NewFilterOptionMultiChoice("Bravo", choices)
		choices[0] = "z"
		assert.Equal(t, []string{"a", "b"}, o.choices)
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
	t.Run("should show pointer cursor when enabled while hovered", func(t *testing.T) {
		f, _ := newChip(t)
		f.Disable()
		f.MouseIn(&desktop.MouseEvent{})

		f.Enable()

		assert.Equal(t, desktop.PointerCursor, f.Cursor())
	})
	t.Run("should show default cursor after mouse left", func(t *testing.T) {
		f, _ := newChip(t)
		f.MouseIn(&desktop.MouseEvent{})

		f.MouseOut()

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

func TestFilterChipCompact_Search(t *testing.T) {
	test.NewTempApp(t)
	newChip := func(t *testing.T, choices []string) (*FilterChipCompact, *[]map[string]string) {
		var got []map[string]string
		f := NewFilterChipCompact([]FilterOption{
			NewFilterOptionMultiChoiceWithSearch("Bravo", choices),
		}, func(s map[string]string) {
			got = append(got, s)
		})
		w := test.NewWindow(f)
		w.Resize(fyne.NewSize(800, 600))
		t.Cleanup(w.Close)
		return f, &got
	}
	t.Run("should open search dialog instead of sub menu", func(t *testing.T) {
		f, _ := newChip(t, []string{"a", "b"})
		it := f.menu.Items[0]
		assert.Equal(t, "Bravo (2)", it.Label)
		assert.Equal(t, f.searchResource, it.Icon)
		assert.Nil(t, it.ChildMenu)
		assert.False(t, it.Disabled)

		it.Action()

		c := fyne.CurrentApp().Driver().CanvasForObject(f)
		assert.NotNil(t, c.Overlays().Top())
	})
	t.Run("should disable option without choices", func(t *testing.T) {
		f, _ := newChip(t, nil)
		it := f.menu.Items[0]
		assert.True(t, it.Disabled)
		assert.Nil(t, it.ChildMenu)
	})
	t.Run("should select choice and close dialog", func(t *testing.T) {
		f, got := newChip(t, []string{"a", "b"})
		p := f.showSearch(f.options[0])

		p.list.Select(1)

		assert.False(t, p.popUp.Visible())
		assert.Equal(t, []map[string]string{{"Bravo": "b"}}, *got)
		assert.Equal(t, "Bravo (2): b", f.menu.Items[0].Label)
		assert.True(t, f.IsOn())
	})
	t.Run("should deselect when selected choice is picked again", func(t *testing.T) {
		f, got := newChip(t, []string{"a", "b"})
		f.SetSelected(map[string]string{"Bravo": "b"})
		*got = nil
		p := f.showSearch(f.options[0])

		p.list.Select(1)

		assert.Equal(t, []map[string]string{{"Bravo": ""}}, *got)
		assert.False(t, f.IsOn())
	})
	t.Run("should clear selection", func(t *testing.T) {
		f, got := newChip(t, []string{"a", "b"})
		f.SetSelected(map[string]string{"Bravo": "a"})
		*got = nil
		p := f.showSearch(f.options[0])
		assert.True(t, p.clear.Visible())

		test.Tap(p.clear)

		assert.False(t, p.popUp.Visible())
		assert.Equal(t, []map[string]string{{"Bravo": ""}}, *got)
	})
	t.Run("should hide clear when nothing selected", func(t *testing.T) {
		f, _ := newChip(t, []string{"a", "b"})
		p := f.showSearch(f.options[0])
		assert.False(t, p.clear.Visible())
	})
	t.Run("should close without change on cancel", func(t *testing.T) {
		f, got := newChip(t, []string{"a", "b"})
		p := f.showSearch(f.options[0])

		test.Tap(p.cancel)

		assert.False(t, p.popUp.Visible())
		assert.Empty(t, *got)
	})
	t.Run("should filter choices by search", func(t *testing.T) {
		f, got := newChip(t, []string{"Amarr", "Jita", "Jitaa", "Dodixie"})
		p := f.showSearch(f.options[0])

		test.Type(p.entry, "JIT")

		assert.Equal(t, []string{"Jita", "Jitaa"}, p.filtered)
		assert.False(t, p.noMatch.Visible())
		p.list.Select(1)
		assert.Equal(t, []map[string]string{{"Bravo": "Jitaa"}}, *got)
	})
	t.Run("should show all choices for short search", func(t *testing.T) {
		f, _ := newChip(t, []string{"Amarr", "Jita"})
		p := f.showSearch(f.options[0])

		test.Type(p.entry, "j")

		assert.Equal(t, []string{"Amarr", "Jita"}, p.filtered)
	})
	t.Run("should report no matches", func(t *testing.T) {
		f, _ := newChip(t, []string{"Amarr", "Jita"})
		p := f.showSearch(f.options[0])

		test.Type(p.entry, "xyz")

		assert.Empty(t, p.filtered)
		assert.True(t, p.noMatch.Visible())
	})
	t.Run("should ignore pick when option was removed while open", func(t *testing.T) {
		f, got := newChip(t, []string{"a", "b"})
		p := f.showSearch(f.options[0])
		f.SetOptions(NewFilterOptionToogle("Alpha"))

		p.list.Select(0)

		assert.Empty(t, *got)
		assert.Equal(t, map[string]string{"Alpha": ""}, f.Selected())
	})
}
