package xwidget_test

import (
	"testing"

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

	t.Run("inactive enabled", func(t *testing.T) {
		f.Reset()
		test.AssertRendersToImage(t, "filterchipcompact/neutral_enabled.png", w.Canvas())
	})

	t.Run("active enabled", func(t *testing.T) {
		f.Reset()
		f.SetSelected(map[string]string{"Alpha": "Alpha"})
		test.AssertRendersToImage(t, "filterchipcompact/active_enabled.png", w.Canvas())
	})

	t.Run("inactive disabled", func(t *testing.T) {
		f.Reset()
		f.Disable()
		test.AssertRendersToImage(t, "filterchipcompact/inactive_disabled.png", w.Canvas())
	})

	t.Run("active disabled", func(t *testing.T) {
		f.Reset()
		f.SetSelected(map[string]string{"Alpha": "Alpha"})
		f.Disable()
		test.AssertRendersToImage(t, "filterchipcompact/active_disabled.png", w.Canvas())
	})

}

func TestFilterChipCompact_New(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should report all options as unselected", func(t *testing.T) {
		// when
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionSeparator(),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one"}),
		}, nil)

		// then
		assert.Equal(t, map[string]string{"Alpha": "", "Bravo": ""}, f.Selected())
	})
}

func TestFilterChipCompact_Tap(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show menu when there are options", func(t *testing.T) {
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
		}, nil)
		w := test.NewWindow(f)
		defer w.Close()

		test.Tap(f)

		assert.NotNil(t, w.Canvas().Overlays().Top())
	})
	t.Run("should not show menu when created without options", func(t *testing.T) {
		f := xwidget.NewFilterChipCompact(nil, nil)
		w := test.NewWindow(f)
		defer w.Close()

		test.Tap(f)

		assert.Nil(t, w.Canvas().Overlays().Top())
	})
	t.Run("should not show menu after options were removed", func(t *testing.T) {
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
		}, nil)
		w := test.NewWindow(f)
		defer w.Close()
		f.SetOptions()

		test.Tap(f)

		assert.Nil(t, w.Canvas().Overlays().Top())
	})
}

func TestFilterChipCompact_UndefinedOption(t *testing.T) {
	test.NewTempApp(t)
	t.Run("constructor ignores zero-value option", func(t *testing.T) {
		assert.NotPanics(t, func() {
			xwidget.NewFilterChipCompact([]xwidget.FilterOption{
				xwidget.NewFilterOptionToogle("Alpha"),
				{},
			}, nil)
		})
	})
	t.Run("SetOptions ignores zero-value option", func(t *testing.T) {
		f := xwidget.NewFilterChipCompact(nil, nil)
		assert.NotPanics(t, func() {
			f.SetOptions(xwidget.NewFilterOptionToogle("Alpha"), xwidget.FilterOption{})
		})
	})
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
		assert.False(t, f.IsOn())
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
		assert.True(t, f.IsOn())
	})

	t.Run("should update state", func(t *testing.T) {
		// given
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
		}, nil)
		f.SetSelected(map[string]string{"Alpha": "Alpha", "Bravo": "two"})

		// when
		f.SetOptions(
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two", "three"}),
			xwidget.NewFilterOptionToogle("Delta"),
		)

		// then
		got := f.Selected()
		want := map[string]string{"Bravo": "two", "Delta": ""}
		assert.Equal(t, want, got)
		assert.True(t, f.IsOn())
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
		name         string
		selected     map[string]string
		wantSelected map[string]string
		wantOn       bool
	}{
		{
			"use valid selection",
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
			true,
		},
		{
			"remove invalid options",
			map[string]string{"Alpha": "Alpha", "Bravo": "one", "Charlie": "two"},
			map[string]string{"Alpha": "Alpha", "Bravo": "one"},
			true,
		},
		{
			"reset invalid choices",
			map[string]string{"Alpha": "invalid", "Bravo": "three"},
			map[string]string{"Alpha": "", "Bravo": ""},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			f.SetSelected(tc.selected)

			// then
			assert.Equal(t, tc.wantSelected, f.Selected())
			assert.Equal(t, tc.wantOn, f.IsOn())
		})
	}
}

func TestFilterChipCompact_OnChanged(t *testing.T) {
	test.NewTempApp(t)
	newChip := func() (*xwidget.FilterChipCompact, *int) {
		var calls int
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
		}, func(map[string]string) {
			calls++
		})
		return f, &calls
	}
	t.Run("SetSelected fires when selection changes", func(t *testing.T) {
		f, calls := newChip()
		f.SetSelected(map[string]string{"Bravo": "one"})
		assert.Equal(t, 1, *calls)
	})
	t.Run("SetSelected does not fire when selection is unchanged", func(t *testing.T) {
		f, calls := newChip()
		f.SetSelected(map[string]string{"Bravo": "one"})
		f.SetSelected(map[string]string{"Bravo": "one"})
		assert.Equal(t, 1, *calls)
	})
	t.Run("SetSelected does not fire when sanitized selection is unchanged", func(t *testing.T) {
		f, calls := newChip()
		f.SetSelected(map[string]string{"Bravo": "invalid", "Charlie": "x"})
		assert.Equal(t, 0, *calls)
	})
	t.Run("Reset fires when something was selected", func(t *testing.T) {
		f, calls := newChip()
		f.SetSelected(map[string]string{"Alpha": "Alpha"})
		*calls = 0
		f.Reset()
		assert.Equal(t, 1, *calls)
	})
	t.Run("Reset does not fire when nothing was selected", func(t *testing.T) {
		f, calls := newChip()
		f.Reset()
		assert.Equal(t, 0, *calls)
	})
	t.Run("SetOptions does not fire when it drops a selected option", func(t *testing.T) {
		f, calls := newChip()
		f.SetSelected(map[string]string{"Alpha": "Alpha"})
		*calls = 0
		f.SetOptions(xwidget.NewFilterOptionToogle("Bravo"))
		assert.Equal(t, 0, *calls)
	})
}

func TestFilterChipCompact_SetSelectedNil(t *testing.T) {
	test.NewTempApp(t)
	// given
	f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
		xwidget.NewFilterOptionToogle("Alpha"),
	}, nil)

	// when
	f.SetSelected(nil)
	f.SetOptions(
		xwidget.NewFilterOptionToogle("Alpha"),
		xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one"}),
	)

	// then
	assert.Equal(t, map[string]string{"Alpha": "", "Bravo": ""}, f.Selected())
	assert.False(t, f.IsOn())
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
			"empty",
			map[string]string{},
			map[string]string{"Alpha": "", "Bravo": ""},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f.SetSelected(tc.selected)

			// when
			f.Reset()

			// then
			assert.Equal(t, tc.want, f.Selected())
			assert.False(t, f.IsOn())
		})
	}
}

func TestFilterChipCompact_ResetSilent(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should reset selection without calling OnChanged", func(t *testing.T) {
		var calls int
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
			xwidget.NewFilterOptionMultiChoice("Bravo", []string{"one", "two"}),
		}, func(map[string]string) {
			calls++
		})
		f.SetSelected(map[string]string{"Alpha": "Alpha", "Bravo": "one"})
		calls = 0

		f.ResetSilent()

		assert.Equal(t, map[string]string{"Alpha": "", "Bravo": ""}, f.Selected())
		assert.False(t, f.IsOn())
		assert.Zero(t, calls)
	})
	t.Run("should do nothing when nothing is selected", func(t *testing.T) {
		f := xwidget.NewFilterChipCompact([]xwidget.FilterOption{
			xwidget.NewFilterOptionToogle("Alpha"),
		}, nil)

		f.ResetSilent()

		assert.Equal(t, map[string]string{"Alpha": ""}, f.Selected())
		assert.False(t, f.IsOn())
	})
}
