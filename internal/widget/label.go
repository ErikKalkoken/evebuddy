package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Label is a re-implementation of a Fyne Label, which also allows different sizes but has the same API.
type Label struct {
	widget.BaseWidget

	Alignment  fyne.TextAlign
	Importance widget.Importance
	SizeName   fyne.ThemeSizeName
	Text       string
	TextStyle  fyne.TextStyle
	Truncation fyne.TextTruncation
	Wrapping   fyne.TextWrap

	provider *widget.RichText
}

func NewLabelWithSize(text string, sizeName fyne.ThemeSizeName) *Label {
	w := &Label{
		Text: text,
		provider: widget.NewRichText(&widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameForeground,
				SizeName:  sizeName,
				Inline:    true,
			},
			Text: text,
		}),
		SizeName: sizeName,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *Label) SetText(text string) {
	w.Text = text
	w.Refresh()
}

func (w *Label) Refresh() {
	if w.provider == nil { // not created until visible
		return
	}
	w.syncSegments()
	w.provider.Refresh()
	w.BaseWidget.Refresh()
}

func (w *Label) CreateRenderer() fyne.WidgetRenderer {
	w.syncSegments()
	return widget.NewSimpleRenderer(w.provider)
}

func (w *Label) syncSegments() {
	var color fyne.ThemeColorName
	switch w.Importance {
	case widget.LowImportance:
		color = theme.ColorNameDisabled
	case widget.MediumImportance:
		color = theme.ColorNameForeground
	case widget.HighImportance:
		color = theme.ColorNamePrimary
	case widget.DangerImportance:
		color = theme.ColorNameError
	case widget.WarningImportance:
		color = theme.ColorNameWarning
	case widget.SuccessImportance:
		color = theme.ColorNameSuccess
	default:
		color = theme.ColorNameForeground
	}

	w.provider.Wrapping = w.Wrapping
	w.provider.Truncation = w.Truncation
	seg := w.provider.Segments[0].(*widget.TextSegment)
	seg.Style = widget.RichTextStyle{
		Alignment: w.Alignment,
		ColorName: color,
		TextStyle: w.TextStyle,
		Inline:    true,
		SizeName:  w.SizeName,
	}
	seg.Text = w.Text
}
