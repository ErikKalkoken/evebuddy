package infowindow

import (
	"fmt"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type AttributeItem struct {
	Label  string
	Value  any
	Action func(v any)
}

func NewAtributeItem(label string, value any) AttributeItem {
	return AttributeItem{Label: label, Value: value}
}

type AttributeList struct {
	widget.BaseWidget

	iw      *InfoWindow
	items   []AttributeItem
	openURL func(*url.URL) error
}

func NewAttributeList(iw *InfoWindow, items ...AttributeItem) *AttributeList {
	w := &AttributeList{
		iw:      iw,
		items:   items,
		openURL: fyne.CurrentApp().OpenURL,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *AttributeList) Set(items []AttributeItem) {
	w.items = items
	w.Refresh()
}

func (w *AttributeList) CreateRenderer() fyne.WidgetRenderer {
	supportedCategories := SupportedEveEntities()
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			value := widget.NewLabel("Value")
			value.Truncation = fyne.TextTruncateEllipsis
			value.Alignment = fyne.TextAlignTrailing
			icon := widget.NewIcon(theme.InfoIcon())
			label := widget.NewLabel("Label")
			return container.NewBorder(nil, nil, label, icon, value)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			border := co.(*fyne.Container).Objects
			label := border[1].(*widget.Label)
			label.SetText(it.Label)
			value := border[0].(*widget.Label)
			icon := border[2]
			icon.Hide()
			var s string
			var i widget.Importance
			switch x := it.Value.(type) {
			case *app.EveEntity:
				s = x.Name
				if supportedCategories.Contains(x.Category) {
					icon.Show()
				}
			case *app.EveRace:
				s = x.Name
				icon.Show()
			case *url.URL:
				s = x.String()
				i = widget.HighImportance
			case float32:
				s = fmt.Sprintf("%.1f %%", x*100)
			case time.Time:
				s = x.Format(app.DateTimeFormat)
			case int:
				s = humanize.Comma(int64(x))
			case bool:
				if x {
					s = "yes"
					i = widget.SuccessImportance
				} else {
					s = "no"
					i = widget.DangerImportance
				}
			default:
				s = fmt.Sprint(x)
			}
			value.Text = s
			value.Importance = i
			value.Refresh()
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(w.items) {
			return
		}
		it := w.items[id]
		switch x := it.Value.(type) {
		case *app.EveEntity:
			if supportedCategories.Contains(x.Category) {
				w.iw.ShowEveEntity(x)
			}
		case *app.EveRace:
			w.iw.ShowRace(x.ID)
		case *url.URL:
			w.openURL(x)
			// TODO
			// if err != nil {
			// 	a.iw.u.ShowSnackbar(fmt.Sprintf("ERROR: Failed to open URL: %s", a.iw.u.ErrorDisplay(err)))
			// }
		}
		if it.Action != nil {
			it.Action(it.Value)
		}
	}
	return widget.NewSimpleRenderer(l)
}
