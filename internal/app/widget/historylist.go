package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// MembershipHistoryList represents a membership history in an organization.
type MembershipHistoryList struct {
	widget.BaseWidget

	ShowInfoWindow func(*app.EveEntity)
	IsFoundedShown bool

	items []app.MembershipHistoryItem
}

func NewMembershipHistoryList() *MembershipHistoryList {
	w := &MembershipHistoryList{
		items: make([]app.MembershipHistoryItem, 0),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *MembershipHistoryList) Set(items []app.MembershipHistoryItem) {
	w.items = items
	w.Refresh()
}

func (w *MembershipHistoryList) CreateRenderer() fyne.WidgetRenderer {
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			l := widget.NewRichText()
			l.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, nil, widget.NewIcon(theme.InfoIcon()), l)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			const dateFormat = "2006.01.02 15:04"
			var endDateStr string
			if !it.EndDate.IsZero() {
				endDateStr = it.EndDate.Format(dateFormat)
			} else {
				endDateStr = "this day"
			}
			var closed string
			if it.IsDeleted {
				closed = " (closed)"
			}
			var text string
			if w.IsFoundedShown && it.IsOldest {
				text = fmt.Sprintf("Founded   **%s**", it.StartDate.Format(dateFormat))
			} else {
				text = fmt.Sprintf(
					"%s%s   **%s** to **%s** (%s days)",
					it.OrganizationName(),
					closed,
					it.StartDate.Format(dateFormat),
					endDateStr,
					humanize.Comma(int64(it.Days)),
				)
			}
			border := co.(*fyne.Container).Objects
			border[0].(*widget.RichText).ParseMarkdown(text)
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(w.items) {
			return
		}
		it := w.items[id]
		if w.ShowInfoWindow != nil {
			w.ShowInfoWindow(it.Organization)
		}
	}
	return widget.NewSimpleRenderer(l)
}
