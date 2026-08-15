package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// TODO: Extend tests

func TestFilterChipCompact_CanRender(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
		xwidget.NewFilterOptionToogle("Alpha"),
	}, nil)
	w := test.NewWindow(f)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 500))
}

func TestFilterChipCompact_SetOptions(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can set options", func(t *testing.T) {
		// given
		f := xwidget.NewFilterChipCompact(nil, nil)

		// when
		f.SetOptions(
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{}),
			xwidget.NewFilterOptionSeparator(),
		)

		// then
		got := f.Selected()
		want := map[string]string{"Alpha": "", "Bravo": ""}
		assert.Equal(t, want, got)
	})

	t.Run("should deduplicate options", func(t *testing.T) {
		// given
		f := xwidget.NewFilterChipCompact(nil, nil)

		// when
		f.SetOptions(
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Alpha", []string{}),
		)

		// then
		got := f.Selected()
		want := map[string]string{"Alpha": ""}
		assert.Equal(t, want, got)
	})

	t.Run("should remove obsolete options", func(t *testing.T) {
		// given
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{}),
		}, nil)

		// when
		f.SetOptions(
			xwidget.NewFilterOptionToogle("Alpha"),
		)

		// then
		got := f.Selected()
		want := map[string]string{"Alpha": ""}
		assert.Equal(t, want, got)
	})

	t.Run("should preserve selected state", func(t *testing.T) {
		// given
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
		}, nil)
		f.SetSelected(map[string]string{"Alpha": "Alpha", "Bravo": "two"})

		// when
		f.SetOptions(
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two", "three"}),
			xwidget.NewFilterOptionToogle("Charlie"),
		)

		// then
		got := f.Selected()
		want := map[string]string{"Alpha": "Alpha", "Bravo": "two", "Charlie": ""}
		assert.Equal(t, want, got)
	})
}

func TestFilterChipCompact_SetSelected(t *testing.T) {
	test.NewTempApp(t)
	// given
	f := xwidget.NewFilterChipCompact(nil, nil)
	f.SetOptions(
		xwidget.NewFilterOptionToogle("Alpha"),
		xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
	)

	cases := []struct {
		name     string
		selected map[string]string
		want     map[string]string
	}{
		{
			"use valid selection",
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
		},
		{
			"remove invalid options",
			map[string]string{"Alpha": "Alpha", "Bravo": "one", "Charlie": "two"},
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
		},
		{
			"reset invalid choices",
			map[string]string{"Alpha": "invalid", "Bravo": "three"},
			map[string]string{"Alpha": "", "Bravo": ""},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			f.SetSelected(tc.selected)

			// then
			got := f.Selected()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFilterChipCompact_Reset(t *testing.T) {
	test.NewTempApp(t)
	// given
	f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
		xwidget.NewFilterOptionToogle("Alpha"),
		xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
	}, nil)

	cases := []struct {
		name     string
		selected map[string]string
		want     map[string]string
	}{
		{
			"two selections",
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
			map[string]string{"Alpha": "", "Bravo": ""},
		},
		{
			"emoty",
			map[string]string{},
			map[string]string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f.SetSelected(tc.selected)

			// when
			f.Reset()

			// then
			got := f.Selected()
			assert.Equal(t, tc.want, got)
		})
	}
}
