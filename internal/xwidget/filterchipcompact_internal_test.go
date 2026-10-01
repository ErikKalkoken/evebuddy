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
