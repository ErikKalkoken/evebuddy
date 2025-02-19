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
	Hint   string      // optional hint text
	Label  string      // label
	Getter func() any  // returns the current value for this setting
	Setter func(v any) // sets the value for this setting

	onSelected func(it SettingItem, refresh func()) // action called when selected
	variant    settingVariant                       // the setting variant of this item
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
		Setter: func(v any) {
			onChanged(v.(bool))
		},
		onSelected: func(it SettingItem, refresh func()) {
			it.Setter(!it.Getter().(bool))
			refresh()
		},
		variant: settingSwitch,
	}
}

// NewSettingItemCustom creates a custom setting in a setting list.
func NewSettingItemCustom(
	label, hint string,
	getter func() any,
	onSelected func(it SettingItem, refresh func()),
) SettingItem {
	return SettingItem{
		Label:      label,
		Hint:       hint,
		Getter:     getter,
		onSelected: onSelected,
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
		Setter: func(v any) {
			switch x := v.(type) {
			case float64:
				setter(x)
			case int:
				setter(float64(x))
			default:
				panic("setting item: unsurported type: " + label)
			}
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
		Setter: func(v any) {
			setter(v.(string))
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
