package ui

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
)

type Recipients struct {
	widget.BaseWidget

	Recipients []*app.EveEntity

	bg            *canvas.Rectangle
	main          *fyne.Container
	showAddDialog func(func(*app.EveEntity))
}

func NewRecipients(showAddDialog func(onSelected func(*app.EveEntity))) *Recipients {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w := &Recipients{
		bg:            bg,
		main:          container.NewGridWithColumns(1),
		Recipients:    make([]*app.EveEntity, 0),
		showAddDialog: showAddDialog,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *Recipients) Set(ee []*app.EveEntity) {
	w.Recipients = ee
	w.updateMain()
}

func (w *Recipients) Add(ee *app.EveEntity) {
	for _, o := range w.Recipients {
		if o.ID == ee.ID {
			return
		}
	}
	w.Recipients = append(w.Recipients, ee)
	w.updateMain()
}

func (w *Recipients) remove(id int32) {
	for i, o := range w.Recipients {
		if o.ID == id {
			w.Recipients = slices.Delete(w.Recipients, i, i+1)
			w.updateMain()
			return
		}
	}
}

func (w *Recipients) IsEmpty() bool {
	return len(w.Recipients) == 0
}

func (w *Recipients) updateMain() {
	w.main.RemoveAll()
	for _, r := range w.Recipients {
		w.main.Add(container.NewBorder(
			nil,
			nil,
			nil,
			widgets.NewIconButton(theme.DeleteIcon(), func() {
				w.remove(r.ID)
			}),
			widget.NewLabel(fmt.Sprintf("%s [%s]", r.Name, r.Category))),
		)
	}
	w.main.Add(container.NewHBox(
		widgets.NewIconButton(theme.NewPrimaryThemedResource(theme.ContentAddIcon()), func() {
			w.showAddDialog(func(ee *app.EveEntity) {
				w.Add(ee)
				w.main.Refresh()
			})
		})))
	w.main.Refresh()
}

func (w *Recipients) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.bg.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.BaseWidget.Refresh()
	w.main.Refresh()
	w.bg.Refresh()
}

func (w *Recipients) MinSize() fyne.Size {
	th := w.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	textSize := th.Size(theme.SizeNameText)
	minSize := fyne.MeasureText("M", textSize, fyne.TextStyle{})
	minSize = minSize.Add(fyne.NewSquareSize(innerPadding))
	minSize = minSize.AddWidthHeight(innerPadding*2, innerPadding)
	return minSize.Max(w.BaseWidget.MinSize())
}

func (w *Recipients) CreateRenderer() fyne.WidgetRenderer {
	w.updateMain()
	c := container.NewStack(
		w.bg,
		w.main,
	)
	return widget.NewSimpleRenderer(c)
}
