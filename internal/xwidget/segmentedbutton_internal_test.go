package xwidget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
)

func TestSegmentedButton_CopiesLabels(t *testing.T) {
	test.NewTempApp(t)
	labels := []string{"A", "B"}
	w := NewSegmentedButton(labels, nil)
	labels[0] = "X"
	assert.Equal(t, []string{"A", "B"}, w.labels)
}

func TestSegmentedButton_SegmentAt(t *testing.T) {
	test.NewTempApp(t)
	w := NewSegmentedButton([]string{"A", "B", "C"}, nil)
	w.Resize(fyne.NewSize(300, 40))
	cases := []struct {
		x    float32
		want int
	}{
		{-5, 0},
		{0, 0},
		{99, 0},
		{100, 1},
		{199, 1},
		{200, 2},
		{299, 2},
		{300, 2},
		{500, 2},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, w.segmentAt(tc.x), "x=%v", tc.x)
	}
}

func TestSegmentedButton_Keyboard(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should move selection with arrow keys", func(t *testing.T) {
		var got []int
		w := NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			got = append(got, index)
		})
		w.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		w.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		w.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight}) // stops at last
		w.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
		assert.Equal(t, 1, w.Selected)
		assert.Equal(t, []int{1, 2, 1}, got)
	})
	t.Run("should stop at first segment", func(t *testing.T) {
		var called bool
		w := NewSegmentedButton([]string{"A", "B"}, func(index int) {
			called = true
		})
		w.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
		assert.Equal(t, 0, w.Selected)
		assert.False(t, called)
	})
	t.Run("should ignore keys when disabled", func(t *testing.T) {
		w := NewSegmentedButton([]string{"A", "B"}, nil)
		w.Disable()
		w.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		assert.Equal(t, 0, w.Selected)
	})
}

func TestSegmentedButton_Focus(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should track focus", func(t *testing.T) {
		w := NewSegmentedButton([]string{"A", "B"}, nil)
		w.FocusGained()
		assert.True(t, w.focused)
		w.FocusLost()
		assert.False(t, w.focused)
	})
	t.Run("should not gain focus when disabled", func(t *testing.T) {
		w := NewSegmentedButton([]string{"A", "B"}, nil)
		w.Disable()
		w.FocusGained()
		assert.False(t, w.focused)
	})
}

func TestSegmentedButton_Hover(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should show pointer cursor when hovered", func(t *testing.T) {
		w := NewSegmentedButton([]string{"A", "B"}, nil)
		assert.Equal(t, desktop.DefaultCursor, w.Cursor())
		w.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.PointerCursor, w.Cursor())
		w.MouseOut()
		assert.Equal(t, desktop.DefaultCursor, w.Cursor())
	})
	t.Run("should show default cursor when disabled", func(t *testing.T) {
		w := NewSegmentedButton([]string{"A", "B"}, nil)
		w.Disable()
		w.MouseIn(&desktop.MouseEvent{})
		assert.Equal(t, desktop.DefaultCursor, w.Cursor())
	})
}

func TestSegmentedButton_ContentFitsAtMinSize(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, theme.DefaultTheme())
	labels := []string{"On", "Off"}
	w := NewSegmentedButton(labels, nil)
	r := test.WidgetRenderer(w).(*segmentedButtonRenderer)
	w.Resize(w.MinSize())
	p := w.Theme().Size(theme.SizeNameInnerPadding)
	for i := range labels {
		w.SetSelected(i)
		s := r.segments[i]
		left := s.bg.Position().X + p
		right := s.bg.Position().X + s.bg.Size().Width - p
		assert.GreaterOrEqual(t, s.icon.Position().X, left, "selected=%d", i)
		assert.LessOrEqual(t, s.label.Position().X+s.label.Size().Width, right, "selected=%d", i)
	}
}

func TestSegmentedButton_SetSelectedField(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should update selection without calling OnChanged", func(t *testing.T) {
		var called bool
		w := NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			called = true
		})
		r := test.WidgetRenderer(w).(*segmentedButtonRenderer)
		w.Selected = 2
		w.Refresh()
		assert.False(t, called)
		assert.False(t, r.segments[0].icon.Visible())
		assert.True(t, r.segments[2].icon.Visible())
	})
	t.Run("should keep out-of-range value and show no segment as selected", func(t *testing.T) {
		w := NewSegmentedButton([]string{"A", "B", "C"}, nil)
		r := test.WidgetRenderer(w).(*segmentedButtonRenderer)
		for _, v := range []int{-1, 3} {
			w.Selected = v
			w.Refresh()
			assert.Equal(t, v, w.Selected)
			for i, s := range r.segments {
				assert.False(t, s.icon.Visible(), "value=%d segment=%d", v, i)
			}
		}
	})
	t.Run("should select tapped segment when out of range", func(t *testing.T) {
		var got []int
		w := NewSegmentedButton([]string{"A", "B", "C"}, func(index int) {
			got = append(got, index)
		})
		w.Resize(fyne.NewSize(300, 40))
		w.Selected = 7
		w.Refresh()
		test.TapAt(w, fyne.NewPos(50, 20))
		assert.Equal(t, 0, w.Selected)
		assert.Equal(t, []int{0}, got)
	})
}
