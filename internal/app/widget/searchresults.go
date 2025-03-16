package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type MyListItemWidget struct {
	widget.BaseWidget
	Title   *widget.Label
	Comment *widget.Label
}

func NewMyListItemWidget(title, comment string) *MyListItemWidget {
	w := &MyListItemWidget{
		Title:   widget.NewLabel(title),
		Comment: widget.NewLabel(comment),
	}
	w.Title.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)

	return w
}

func (w *MyListItemWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, nil, w.Comment, w.Title)
	return widget.NewSimpleRenderer(c)
}
