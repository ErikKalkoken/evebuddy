package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// TappableLabel is a [kxwidget.TappableLabel] with tooltip support.
type TappableLabel struct {
	kxwidget.TappableLabel
	ttwidget.ToolTipWidgetExtend
}

var _ desktop.Hoverable = (*TappableLabel)(nil)

// NewTappableLabelWithClipboardCopy returns a tappable label which copies text to clipboard.
func NewTappableLabelWithClipboardCopy(text string) *TappableLabel {
	x := NewTappableLabel(text, func() {
		fyne.CurrentApp().Clipboard().SetContent(text)
	})
	x.SetToolTip("Click to copy to clipboard")
	return x
}

// NewTappableLabel returns a new TappableLabel instance.
func NewTappableLabel(text string, tapped func()) *TappableLabel {
	w := &TappableLabel{}
	w.ExtendBaseWidget(w)
	w.SetText(text)
	w.OnTapped = tapped
	return w
}

func (w *TappableLabel) ExtendBaseWidget(wid fyne.Widget) {
	w.ExtendToolTipWidget(wid)
	w.TappableLabel.ExtendBaseWidget(wid)
}

func (w *TappableLabel) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseIn(e)
	w.TappableLabel.MouseIn(e)
}

func (w *TappableLabel) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidgetExtend.MouseMoved(e)
	w.TappableLabel.MouseMoved(e)
}

func (w *TappableLabel) MouseOut() {
	w.ToolTipWidgetExtend.MouseOut()
	w.TappableLabel.MouseOut()
}
