package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

const (
	confirmText = "OK"
	dismissText = "Cancel"
)

type settingVariant uint

const (
	settingUndefined settingVariant = iota
	settingCustom
	settingHeading
	settingSeperator
	settingSwitch
)

// SettingItem represents an item in a setting list.
type SettingItem struct {
	Hint   string     // optional hint text
	Label  string     // label
	Getter func() any // returns the current value for this setting

	onSelected      func(it SettingItem, refresh func()) // action called when selected
	onSwitchChanged func(on bool)                        // action called when switch changes
	variant         settingVariant                       // the setting variant of this item
}

// NewSettingItemHeading creates a heading in a setting list.
func NewSettingItemHeading(label string) SettingItem {
	return SettingItem{Label: label, variant: settingHeading}
}

// NewSettingItemSeperator creates a seperator in a setting list.
func NewSettingItemSeperator() SettingItem {
	return SettingItem{variant: settingSeperator}
}

// NewSettingItemSwitch creates a switch setting in a setting list.
func NewSettingItemSwitch(
	label, hint string,
	getter func() bool,
	onChanged func(bool),
) SettingItem {
	return SettingItem{
		Label: label,
		Hint:  hint,
		Getter: func() any {
			return getter()
		},
		onSwitchChanged: onChanged,
		onSelected: func(it SettingItem, refresh func()) {
			it.onSwitchChanged(!it.Getter().(bool))
			refresh()
		},
		variant: settingSwitch,
	}
}

// NewSettingItemCustom creates a custom setting in a setting list.
func NewSettingItemCustom(
	label, hint string,
	getter func() any,
	onselected func(it SettingItem, refresh func()),
) SettingItem {
	return SettingItem{
		Label:      label,
		Hint:       hint,
		Getter:     getter,
		onSelected: onselected,
		variant:    settingCustom,
	}
}

func NewSettingItemSlider(
	label, hint string,
	minV, maxV float64,
	getter func() float64,
	setter func(v float64),
	window func() fyne.Window,
) SettingItem {
	return SettingItem{
		Label: label,
		Hint:  hint,
		Getter: func() any {
			return getter()
		},
		onSelected: func(it SettingItem, refresh func()) {
			sl := kxwidget.NewSlider(minV, maxV)
			sl.SetValue(float64(getter()))
			w := window()
			c := container.NewBorder(
				nil,
				NewLabelWithSize(it.Hint, theme.SizeNameCaptionText),
				nil,
				nil,
				sl,
			)
			d := dialog.NewCustomConfirm(it.Label, confirmText, dismissText, c, func(confirmed bool) {
				if !confirmed {
					return
				}
				setter(sl.Value())
				refresh()
			}, w)
			d.Show()
			d.Resize(fyne.NewSize(w.Canvas().Size().Width, 100))
		},
		variant: settingCustom,
	}
}

func NewSettingItemSelect(
	label, hint string,
	options []string,
	getter func() string,
	setter func(v string),
	window func() fyne.Window,
) SettingItem {
	return SettingItem{
		Label: label,
		Hint:  hint,
		Getter: func() any {
			return getter()
		},
		onSelected: func(it SettingItem, refresh func()) {
			sel := widget.NewRadioGroup(options, nil)
			sel.SetSelected(it.Getter().(string))
			w := window()
			c := container.NewBorder(
				nil,
				NewLabelWithSize(it.Hint, theme.SizeNameCaptionText),
				nil,
				nil,
				sel,
			)
			d := dialog.NewCustomConfirm(it.Label, confirmText, dismissText, c, func(confirmed bool) {
				if !confirmed {
					return
				}
				setter(sel.Selected)
				refresh()
			}, w)
			d.Show()
			d.Resize(fyne.NewSize(w.Canvas().Size().Width, 100))
		},
		variant: settingCustom,
	}
}
