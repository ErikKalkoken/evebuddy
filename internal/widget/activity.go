package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// Activity widget is a variant of the [widget.Activity] that supports tooltips.
type Activity struct {
	widget.Activity
	ttwidget.ToolTipWidgetExtend
}

// NewActivity returns a new [Activity] widget.
func NewActivity() *Activity {
	w := &Activity{}
	w.ExtendBaseWidget(w)
	return w
}

func (w *Activity) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.Activity.ExtendBaseWidget(wid)
}
