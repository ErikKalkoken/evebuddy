package widgets

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SubHeading is a label with larger size meant for headings.
type Heading struct {
	label
}

func NewHeading(text string) *Heading {
	w := &Heading{
		label: *newLabel(text, theme.SizeNameHeadingText),
	}
	return w
}

// SubHeading is a label with larger size meant for sub headings.
type SubHeading struct {
	label
}

func NewSubHeading(text string) *SubHeading {
	w := &SubHeading{
		label: *newLabel(text, theme.SizeNameSubHeadingText),
	}
	return w
}

// label is a re-implementation of a Fyne Label. This is necessary for setting a custom size.
type label struct {
	widget.BaseWidget

	Alignment  fyne.TextAlign
	Importance widget.Importance
	Text       string
	TextStyle  fyne.TextStyle
	Truncation fyne.TextTruncation
	Wrapping   fyne.TextWrap

	mu       sync.Mutex
	provider *widget.RichText
}

func newLabel(text string, sizeName fyne.ThemeSizeName) *label {
	w := &label{
		Text: text,
		provider: widget.NewRichText(&widget.TextSegment{
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameForeground,
				SizeName:  sizeName,
			},
			Text: text,
		}),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *label) SetText(text string) {
	w.mu.Lock()
	w.Text = text
	w.mu.Unlock()
	w.Refresh()
}

func (w *label) Refresh() {
	if w.provider == nil { // not created until visible
		return
	}
	w.syncSegments()
	w.provider.Refresh()
	w.BaseWidget.Refresh()
}

func (w *label) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.provider)
}

func (w *label) syncSegments() {
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

	w.mu.Lock()
	defer w.mu.Unlock()
	w.provider.Wrapping = w.Wrapping
	w.provider.Truncation = w.Truncation
	seg := w.provider.Segments[0].(*widget.TextSegment)
	seg.Style = widget.RichTextStyle{
		Alignment: w.Alignment,
		ColorName: color,
		TextStyle: w.TextStyle,
		SizeName:  theme.SizeNameSubHeadingText,
	}
	seg.Text = w.Text
}
