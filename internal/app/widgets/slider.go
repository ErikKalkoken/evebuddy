package widgets

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// Slider represents a slider widget for Fyne which shows it's current value.
type Slider struct {
	widget.BaseWidget

	OnChangeEnded func(int)

	data   binding.Float
	label  *widget.Label
	slider *widget.Slider
	layout columnsLayout
}

func NewSlider(min, max, start int) *Slider {
	temp := widget.NewLabel(strconv.Itoa(max))
	minW := temp.MinSize().Width
	d := binding.NewFloat()
	w := &Slider{
		label:  widget.NewLabelWithData(binding.FloatToStringWithFormat(d, "%.0f")),
		slider: widget.NewSliderWithData(float64(min), float64(max), d),
		data:   d,
		layout: NewColumnsLayout(minW, 2*minW),
	}
	w.label.Alignment = fyne.TextAlignTrailing
	w.slider.OnChangeEnded = func(v float64) {
		if w.OnChangeEnded == nil {
			return
		}
		w.OnChangeEnded(int(v))
	}
	w.slider.SetValue(float64(start))
	w.ExtendBaseWidget(w)
	return w
}

func (w *Slider) Value() int {
	return int(w.slider.Value)
}

func (w *Slider) SetValue(v int) {
	w.slider.SetValue(float64(v))
}

func (w *Slider) CreateRenderer() fyne.WidgetRenderer {
	c := container.New(w.layout, w.label, w.slider)
	return widget.NewSimpleRenderer(c)
}
