package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestSegmentedButton_CanRender(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	t.Run("first selected", func(t *testing.T) {
		b := xwidget.NewSegmentedButton([]string{"Day", "Week", "Month"}, nil)
		w := test.NewWindow(b)
		defer w.Close()
		test.AssertImageMatches(t, "segmentedbutton/first_selected.png", w.Canvas().Capture())
	})

	t.Run("middle selected", func(t *testing.T) {
		b := xwidget.NewSegmentedButton([]string{"Day", "Week", "Month"}, nil)
		b.SetSelected(1)
		w := test.NewWindow(b)
		defer w.Close()
		test.AssertImageMatches(t, "segmentedbutton/middle_selected.png", w.Canvas().Capture())
	})

	t.Run("disabled", func(t *testing.T) {
		b := xwidget.NewSegmentedButton([]string{"Day", "Week", "Month"}, nil)
		b.Disable()
		w := test.NewWindow(b)
		defer w.Close()
		test.AssertImageMatches(t, "segmentedbutton/disabled.png", w.Canvas().Capture())
	})

	t.Run("focused", func(t *testing.T) {
		b := xwidget.NewSegmentedButton([]string{"Day", "Week", "Month"}, nil)
		w := test.NewWindow(b)
		defer w.Close()
		w.Canvas().Focus(b)
		test.AssertImageMatches(t, "segmentedbutton/focused.png", w.Canvas().Capture())
	})
}

func TestSegmentedButton_New(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should select first segment by default", func(t *testing.T) {
		b := xwidget.NewSegmentedButton([]string{"A", "B"}, nil)
		assert.Equal(t, 0, b.Selected)
	})
	t.Run("should panic when labels are empty", func(t *testing.T) {
		assert.Panics(t, func() {
			xwidget.NewSegmentedButton(nil, nil)
		})
	})
}

func TestSegmentedButton_SetSelected(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should select and call OnChanged", func(t *testing.T) {
		var got []int
		b := xwidget.NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			got = append(got, index)
		})
		b.SetSelected(2)
		assert.Equal(t, 2, b.Selected)
		assert.Equal(t, []int{2}, got)
	})
	t.Run("should not call OnChanged when already selected", func(t *testing.T) {
		var called bool
		b := xwidget.NewSegmentedButton([]string{"A", "B"}, func(index int) {
			called = true
		})
		b.SetSelected(0)
		assert.False(t, called)
	})
	t.Run("should ignore out-of-range index", func(t *testing.T) {
		var called bool
		b := xwidget.NewSegmentedButton([]string{"A", "B"}, func(index int) {
			called = true
		})
		b.SetSelected(-1)
		b.SetSelected(2)
		assert.Equal(t, 0, b.Selected)
		assert.False(t, called)
	})
}

func TestSegmentedButton_Tap(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should select tapped segment", func(t *testing.T) {
		var got []int
		b := xwidget.NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			got = append(got, index)
		})
		b.Resize(fyne.NewSize(300, 40))
		test.TapAt(b, fyne.NewPos(250, 20))
		assert.Equal(t, 2, b.Selected)
		assert.Equal(t, []int{2}, got)
	})
	t.Run("should not call OnChanged when tapping selected segment", func(t *testing.T) {
		var called bool
		b := xwidget.NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			called = true
		})
		b.Resize(fyne.NewSize(300, 40))
		test.TapAt(b, fyne.NewPos(50, 20))
		assert.False(t, called)
	})
	t.Run("should ignore taps when disabled", func(t *testing.T) {
		var called bool
		b := xwidget.NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			called = true
		})
		b.Resize(fyne.NewSize(300, 40))
		b.Disable()
		test.TapAt(b, fyne.NewPos(250, 20))
		assert.Equal(t, 0, b.Selected)
		assert.False(t, called)
	})
}
