package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// SettingList is a custom list widget for settings.
type SettingList struct {
	widget.List
}

// NewSettingList returns a new SettingList widget.
func NewSettingList(items []SettingItem) *SettingList {
	w := &SettingList{}
	w.Length = func() int {
		return len(items)
	}
	w.CreateItem = func() fyne.CanvasObject {
		// p := theme.Padding()
		label := widget.NewLabel("Template")
		label.Truncation = fyne.TextTruncateClip
		hint := NewLabelWithSize("", theme.SizeNameCaptionText)
		hint.Truncation = fyne.TextTruncateClip
		c := container.NewPadded(container.NewBorder(
			nil,
			container.New(layout.NewCustomPaddedLayout(0, 0, 0, 0), widget.NewSeparator()),
			nil,
			container.NewVBox(layout.NewSpacer(), container.NewStack(kxwidget.NewSwitch(nil), widget.NewLabel("")), layout.NewSpacer()),
			container.New(layout.NewCustomPaddedVBoxLayout(0), layout.NewSpacer(), label, hint, layout.NewSpacer()),
		))
		return c
	}
	w.UpdateItem = func(id widget.ListItemID, co fyne.CanvasObject) {
		if id >= len(items) {
			return
		}
		it := items[id]
		border := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects
		right := border[2].(*fyne.Container).Objects[1].(*fyne.Container).Objects
		sw := right[0].(*kxwidget.Switch)
		value := right[1].(*widget.Label)
		main := border[0].(*fyne.Container).Objects
		hint := main[2].(*Label)
		if it.Hint != "" {
			hint.SetText(it.Hint)
			hint.Show()
		} else {
			hint.Hide()
		}
		label := main[1].(*widget.Label)
		label.Text = it.Label
		label.TextStyle.Bold = false
		switch it.variant {
		case settingHeading:
			label.TextStyle.Bold = true
			value.Hide()
			sw.Hide()
		case settingSwitch:
			value.Hide()
			sw.OnChanged = func(v bool) {
				it.Setter(v)
			}
			sw.On = it.Getter().(bool)
			sw.Show()
			sw.Refresh()
		case settingCustom:
			value.SetText(fmt.Sprint(it.Getter()))
			value.Show()
			sw.Hide()
		}
		sep := border[1].(*fyne.Container)
		if it.variant == settingSeperator {
			sep.Show()
			value.Hide()
			sw.Hide()
			label.Hide()
		} else {
			sep.Hide()
			label.Show()
			label.Refresh()
		}
		w.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
	}
	w.OnSelected = func(id widget.ListItemID) {
		defer w.UnselectAll()
		if id >= len(items) {
			return
		}
		it := items[id]
		if it.onSelected != nil {
			it.onSelected(it, func() {
				w.RefreshItem(id)
			})
		}
	}
	w.HideSeparators = true
	w.ExtendBaseWidget(w)
	return w
}
