package xwidget

import (
	"testing"

	"fyne.io/fyne/v2/test"
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
