package xwidget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

func TestDrawerDestination_Tapping(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var tapped int
	d := newDrawerDestination(theme.HomeIcon(), "Home", func() { tapped++ })
	w := test.NewWindow(d)
	defer w.Close()

	test.Tap(d)
	assert.Equal(t, 1, tapped)

	d.Disable()
	test.Tap(d)
	assert.Equal(t, 1, tapped)
}

func TestDrawerDestination_ReflectsState(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	d := newDrawerDestination(theme.HomeIcon(), "Home", nil)
	w := test.NewWindow(d)
	defer w.Close()

	assert.Equal(t, d.iconEnabled, d.icon.Resource)
	assert.Equal(t, widget.MediumImportance, d.title.Importance)
	assert.False(t, d.title.TextStyle.Bold)
	assert.False(t, d.badge.TextStyle.Bold)

	d.setActive(true)
	assert.Equal(t, d.iconSelected, d.icon.Resource)
	assert.Equal(t, widget.HighImportance, d.title.Importance)
	assert.True(t, d.title.TextStyle.Bold)
	assert.True(t, d.badge.TextStyle.Bold)

	d.Disable()
	assert.Equal(t, d.iconDisabled, d.icon.Resource)
	assert.Equal(t, widget.LowImportance, d.title.Importance)
	assert.False(t, d.title.TextStyle.Bold)
	assert.False(t, d.badge.TextStyle.Bold)
}

func TestDrawerDestination_Badge(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	d := newDrawerDestination(theme.HomeIcon(), "Home", nil)
	w := test.NewWindow(d)
	defer w.Close()
	assert.False(t, d.badge.Visible())

	d.setBadge("42")
	assert.True(t, d.badge.Visible())
	assert.Equal(t, "42", d.badge.Text)

	d.setBadge("")
	assert.False(t, d.badge.Visible())
}

func TestDrawerDestination_SetText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	d := newDrawerDestination(theme.HomeIcon(), "Home", nil)
	d.setText("Other")
	assert.Equal(t, "Other", d.title.Text)
}

func TestDrawerDestination_HoverOnlyWhenEnabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	d := newDrawerDestination(theme.HomeIcon(), "Home", nil)
	w := test.NewWindow(d)
	defer w.Close()

	d.MouseIn(&desktop.MouseEvent{})
	_, _, _, a := d.hover.FillColor.RGBA()
	assert.NotZero(t, a)
	assert.Equal(t, desktop.PointerCursor, d.Cursor())

	d.Disable()
	assert.Equal(t, desktop.DefaultCursor, d.Cursor())
	_, _, _, a = d.hover.FillColor.RGBA()
	assert.Zero(t, a)
}

func TestDrawerDestination_SizeIsStableOnHoverAndSelect(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	d := newDrawerDestination(theme.HomeIcon(), "Home", nil)
	w := test.NewWindow(d)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 100))

	want := d.MinSize()
	d.MouseIn(&desktop.MouseEvent{})
	assert.Equal(t, want, d.MinSize())
	d.MouseOut()
	d.setActive(true)
	assert.Equal(t, want, d.MinSize())
}

func TestNavDrawer_TappingItemSelectsIt(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()

	assert.Equal(t, a, nd.Selected())
	assert.True(t, a.content.Visible())
	assert.False(t, b.content.Visible())

	test.Tap(b.dest)
	assert.Equal(t, b, nd.Selected())
	assert.False(t, a.content.Visible())
	assert.True(t, b.content.Visible())
	assert.True(t, b.dest.isActive)
	assert.False(t, a.dest.isActive)

	nd.DisableItem(a)
	test.Tap(a.dest)
	assert.Equal(t, b, nd.Selected())
}

func TestNavDrawer_OnSelectedFiresOnlyOnChange(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selected int
	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	b.OnSelected = func() { selected++ }
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()

	nd.Select(b)
	nd.Select(b)
	assert.Equal(t, 1, selected)
}

func TestNavDrawer_EnableFiresOnSelectedOfFirstItem(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selected int
	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	a.OnSelected = func() { selected++ }
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()
	assert.Equal(t, 1, selected)

	nd.Disable()
	assert.Equal(t, 1, selected)
	nd.Enable()
	assert.Equal(t, 2, selected)

	nd.Select(b)
	nd.Disable()
	nd.Enable()
	assert.Equal(t, 3, selected)
}

func TestNavDrawer_IndicatorInvisibleWhileDisabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	nd := NewNavDrawer(a)
	nd.Disable() // before the window is shown, like at app startup
	w := test.NewWindow(nd)
	defer w.Close()

	_, _, _, alpha := nd.indicator.FillColor.RGBA()
	assert.Zero(t, alpha)

	nd.Enable()
	assert.Equal(t, theme.Color(theme.ColorNamePrimary), nd.indicator.FillColor)

	nd.Disable()
	_, _, _, alpha = nd.indicator.FillColor.RGBA()
	assert.Zero(t, alpha)
}

func TestNavDrawer_DisablingSelectedItemSwitchesToFirstEnabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	c := NewNavDrawerItem(theme.HomeIcon(), "C", widget.NewLabel("C"))
	nd := NewNavDrawer(a, b, c)
	w := test.NewWindow(nd)
	defer w.Close()

	nd.DisableItem(a)
	nd.Select(c)
	nd.DisableItem(c)
	assert.Equal(t, b, nd.Selected())
	assert.False(t, nd.ItemEnabled(a))
	assert.True(t, nd.ItemEnabled(b))

	nd.EnableItem(a)
	assert.True(t, nd.ItemEnabled(a))
}

func TestNavDrawer_DisableAndEnable(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	c := NewNavDrawerItem(theme.HomeIcon(), "C", widget.NewLabel("C"))
	nd := NewNavDrawer(a, b, c)
	w := test.NewWindow(nd)
	defer w.Close()
	nd.Select(b)
	nd.DisableItem(c)

	nd.Disable()
	assert.Nil(t, nd.Selected())
	for _, it := range []*NavDrawerItem{a, b, c} {
		assert.False(t, nd.ItemEnabled(it))
	}
	nd.Select(a)
	assert.Nil(t, nd.Selected())

	nd.Enable()
	assert.Equal(t, a, nd.Selected(), "switches to first enabled item")
	assert.True(t, nd.ItemEnabled(a))
	assert.True(t, nd.ItemEnabled(b))
	assert.False(t, nd.ItemEnabled(c), "individually disabled item stays disabled")
}

func TestNavDrawer_EnableSwitchesToFirstEnabledWhenSelectionStaysDisabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()
	nd.Select(b)

	nd.Disable()
	nd.DisableItem(b)
	nd.Enable()
	assert.Equal(t, a, nd.Selected())
	assert.True(t, a.content.Visible())
	assert.False(t, b.content.Visible())
}

func TestNavDrawer_Placeholder(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	t.Run("shown while drawer is disabled", func(t *testing.T) {
		a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nd := NewNavDrawer(a, b)
		nd.SetPlaceholder("placeholder")
		p := nd.placeholder
		nd.Disable() // before the window is shown, like at app startup
		w := test.NewWindow(nd)
		defer w.Close()

		assert.True(t, p.Visible())
		assert.False(t, a.content.Visible())
		assert.False(t, b.content.Visible())
		assert.Nil(t, nd.Selected())
		_, _, _, alpha := nd.indicator.FillColor.RGBA()
		assert.Zero(t, alpha)

		nd.Enable()
		assert.False(t, p.Visible())
		assert.True(t, a.content.Visible())
		assert.Equal(t, a, nd.Selected())
		assert.Equal(t, theme.Color(theme.ColorNamePrimary), nd.indicator.FillColor)
	})
	t.Run("shown when all items are disabled", func(t *testing.T) {
		a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
		nd := NewNavDrawer(a, b)
		nd.SetPlaceholder("placeholder")
		p := nd.placeholder
		w := test.NewWindow(nd)
		defer w.Close()
		nd.Select(b)

		nd.DisableItem(a)
		assert.False(t, p.Visible())
		nd.DisableItem(b)
		assert.True(t, p.Visible())
		assert.False(t, b.content.Visible())
		assert.False(t, b.dest.isActive)
		assert.Nil(t, nd.Selected())

		nd.EnableItem(a)
		nd.EnableItem(b)
		assert.False(t, p.Visible())
		assert.True(t, a.content.Visible())
		assert.False(t, b.content.Visible())
		assert.Equal(t, a, nd.Selected(), "does not restore previous selection")
	})
	t.Run("set while empty", func(t *testing.T) {
		a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		nd := NewNavDrawer(a)
		w := test.NewWindow(nd)
		defer w.Close()
		nd.Disable()
		assert.False(t, a.content.Visible())
		assert.True(t, nd.placeholder.Visible())

		nd.SetPlaceholder("placeholder")
		assert.Equal(t, "placeholder", nd.placeholder.text.String())
	})
	t.Run("empty text shows nothing", func(t *testing.T) {
		a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
		nd := NewNavDrawer(a)
		nd.SetPlaceholder("placeholder")

		nd.SetPlaceholder("")
		assert.Empty(t, nd.placeholder.text.String())
	})
}

func TestNavDrawer_ScrollsToTopWhenReenabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var items []*NavDrawerItem
	for range 20 {
		items = append(items, NewNavDrawerItem(theme.HomeIcon(), "X", widget.NewLabel("X")))
	}
	nd := NewNavDrawer(items...)
	w := test.NewWindow(nd)
	defer w.Close()
	w.Resize(fyne.NewSize(400, 200))

	nd.Disable()
	nd.scroll.ScrollToBottom()
	assert.Greater(t, nd.scroll.Offset.Y, float32(0), "drawer scrolled")
	nd.Enable()
	assert.Zero(t, nd.scroll.Offset.Y)
}

func TestNavDrawer_ItemSetters(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	a.SetBadge("7") // works before the item is added to a drawer
	nd := NewNavDrawer(a)
	w := test.NewWindow(nd)
	defer w.Close()
	assert.Equal(t, "7", a.dest.badge.Text)

	a.SetText("X")
	assert.Equal(t, "X", a.dest.title.Text)
}

func TestNavDrawer_PanicsOnInvalidItems(t *testing.T) {
	test.NewTempApp(t)

	assert.Panics(t, func() { NewNavDrawer() })
	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	NewNavDrawer(a)
	assert.Panics(t, func() { NewNavDrawer(a) })
}

func TestNavDrawer_IndicatorFollowsSelection(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	c := NewNavDrawerItem(theme.HomeIcon(), "C", widget.NewLabel("C"))
	nd := NewNavDrawer(a, b, c)
	w := test.NewWindow(nd)
	defer w.Close()
	w.Resize(fyne.NewSize(400, 400))

	assertDrawerIndicatorAt(t, nd, a)

	nd.Select(c)
	assertDrawerIndicatorAt(t, nd, c)

	nd.DisableItem(c)
	assertDrawerIndicatorAt(t, nd, a)

	w.Resize(fyne.NewSize(600, 500))
	assertDrawerIndicatorAt(t, nd, a)
}

func TestNavDrawer_IndicatorFollowsWidthChange(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 400))
	nd.Select(b)
	x := nd.separator.Position().X

	b.SetBadge("a much wider badge than before")
	nd.Refresh() // real drivers re-layout on min size changes, the test driver does not
	assert.Greater(t, nd.separator.Position().X, x, "strip got wider")
	assertDrawerIndicatorAt(t, nd, b)
}

func TestNavDrawer_IndicatorStaysWithItemWhenScrolled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var items []*NavDrawerItem
	for range 20 {
		items = append(items, NewNavDrawerItem(theme.HomeIcon(), "X", widget.NewLabel("X")))
	}
	nd := NewNavDrawer(items...)
	w := test.NewWindow(nd)
	defer w.Close()
	w.Resize(fyne.NewSize(400, 200))
	last := items[len(items)-1]
	nd.Select(last)

	nd.scroll.ScrollToBottom()
	assert.Greater(t, nd.scroll.Offset.Y, float32(0), "drawer scrolled")
	d := fyne.CurrentApp().Driver()
	assert.Equal(t, d.AbsolutePositionForObject(last.dest).Y, d.AbsolutePositionForObject(nd.indicator).Y)
	assert.Less(t, d.AbsolutePositionForObject(nd.indicator).Y, float32(200), "indicator is in view")
}

func TestNavDrawer_MinWidth(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	nd := NewNavDrawer(a)
	w := test.NewWindow(nd)
	defer w.Close()
	natural := nd.column.MinSize().Width + nd.separator.MinSize().Width
	assert.Equal(t, natural, nd.scroll.MinSize().Width)

	nd.MinWidth = natural + 100
	assert.Equal(t, natural+100, nd.scroll.MinSize().Width)
}

func TestNavDrawer_EnableItemWhileDrawerDisabledKeepsItDisabled(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()
	nd.DisableItem(b)
	nd.Disable()

	nd.EnableItem(b)
	assert.False(t, nd.ItemEnabled(b))

	nd.Enable()
	assert.True(t, nd.ItemEnabled(b))
}

func TestNavDrawer_TappingWhileDisabledDoesNothing(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	b := NewNavDrawerItem(theme.HomeIcon(), "B", widget.NewLabel("B"))
	nd := NewNavDrawer(a, b)
	w := test.NewWindow(nd)
	defer w.Close()
	nd.Disable()

	test.Tap(b.dest)
	assert.Nil(t, nd.Selected())
	assert.False(t, b.content.Visible())
}

func TestNavDrawer_IgnoresForeignItems(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	nd := NewNavDrawer(a)
	x := NewNavDrawerItem(theme.HomeIcon(), "X", widget.NewLabel("X"))
	other := NewNavDrawer(x)
	w := test.NewWindow(nd)
	defer w.Close()

	for _, it := range []*NavDrawerItem{x, nil} {
		nd.Select(it)
		nd.DisableItem(it)
		nd.EnableItem(it)
		assert.False(t, nd.ItemEnabled(it))
	}
	assert.Equal(t, a, nd.Selected())
	assert.True(t, other.ItemEnabled(x), "foreign item not affected")
}

func assertDrawerIndicatorAt(t *testing.T, nd *NavDrawer, it *NavDrawerItem) {
	t.Helper()
	dest := it.dest
	assert.Equal(t, nd.separator.Position().X, nd.indicator.Position().X, "x")
	assert.Equal(t, nd.column.Position().Y+dest.Position().Y, nd.indicator.Position().Y, "y")
	assert.Equal(t, dest.Size().Height, nd.indicator.Size().Height, "height")
	assert.Equal(t, nd.separator.Position().X, dest.Position().X+dest.Size().Width, "touches item")
}

func TestNavDrawer_RefreshReachesItems(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := NewNavDrawerItem(theme.HomeIcon(), "A", widget.NewLabel("A"))
	nd := NewNavDrawer(a)
	w := test.NewWindow(nd)
	defer w.Close()

	a.dest.badge.Text = "5" // change state without refreshing the item
	nd.Refresh()
	assert.True(t, a.dest.badge.Visible())
}
