package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// relative size of dialog window to current window
const (
	dialogWidthScale = 0.8 // except on mobile it is always 100%
	dialogHeightMin  = 100
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
	minV, maxV, defaultV float64,
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
			sl.OnChangeEnded = setter
			w := window()
			d := makeSettingDialog(
				sl,
				it.Label,
				it.Hint,
				func() {
					sl.SetValue(defaultV)
				},
				refresh,
				w,
			)
			d.Show()
		},
		variant: settingCustom,
	}
}

func NewSettingItemSelect(
	label, hint string,
	options []string,
	defaultV string,
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
			sel := widget.NewRadioGroup(options, setter)
			sel.SetSelected(it.Getter().(string))
			w := window()
			d := makeSettingDialog(
				sel,
				it.Label,
				it.Hint,
				func() {
					sel.SetSelected(defaultV)
				},
				refresh,
				w,
			)
			d.Show()
		},
		variant: settingCustom,
	}
}

func makeSettingDialog(
	setting fyne.CanvasObject,
	label, hint string,
	reset, refresh func(),
	w fyne.Window,
) dialog.Dialog {
	var d dialog.Dialog
	buttons := container.NewHBox(
		widget.NewButton("Close", func() {
			d.Hide()
		}),
		layout.NewSpacer(),
		widget.NewButton("Reset", func() {
			reset()
		}),
	)
	c := container.NewBorder(
		nil,
		container.NewVBox(
			NewLabelWithSize(hint, theme.SizeNameCaptionText),
			buttons,
		),
		nil,
		nil,
		setting,
	)
	d = dialog.NewCustomWithoutButtons(label, c, w)
	_, s := w.Canvas().InteractiveArea()
	var width float32
	if fyne.CurrentDevice().IsMobile() {
		width = s.Width
	} else {
		width = s.Width * dialogWidthScale
	}
	d.Resize(fyne.NewSize(width, dialogHeightMin))
	d.SetOnClosed(refresh)
	return d
}
