package mailer

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infoviewer"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// eveEntityEntry represents an entry widget for Eve Entity items.
type eveEntityEntry struct {
	widget.BaseWidget

	placeholderText string
	readOnly        bool // items can not be removed
	showInfo        func(*app.EveEntity)

	loadIcon    ui.EveEntityIconLoader
	field       *canvas.Rectangle
	items       []*app.EveEntity
	label       fyne.CanvasObject
	labelWidth  float32
	main        *fyne.Container
	placeholder *xwidget.RichText
}

func newEveEntityEntry(label fyne.CanvasObject, labelWidth float32, loadIcon ui.EveEntityIconLoader) *eveEntityEntry {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	placeholder := xwidget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder},
	})
	w := &eveEntityEntry{
		loadIcon:    loadIcon,
		field:       bg,
		label:       label,
		labelWidth:  labelWidth,
		main:        container.New(layout.NewCustomPaddedVBoxLayout(0)),
		placeholder: placeholder,
	}
	w.ExtendBaseWidget(w)
	return w
}

// Items returns a copy of the current list of EveEntities items.
func (w *eveEntityEntry) Items() []*app.EveEntity {
	return slices.Clone(w.items)
}

// Set replaces the list of items with a copy of s.
func (w *eveEntityEntry) Set(s []*app.EveEntity) {
	w.items = slices.Clone(s)
	w.Refresh()
}

func (w *eveEntityEntry) Add(ee *app.EveEntity) {
	added := func() bool {
		for _, o := range w.items {
			if o.ID == ee.ID {
				return false
			}
		}
		w.items = append(w.items, ee)
		return true
	}()
	if added {
		w.Refresh()
	}
}

func (w *eveEntityEntry) Remove(id int64) {
	removed := func() bool {
		for i, o := range w.items {
			if o.ID == id {
				w.items = slices.Delete(w.items, i, i+1)
				return true
			}
		}
		return false
	}()
	if removed {
		w.Refresh()
	}
}

// String returns a list of all entities as string.
func (w *eveEntityEntry) String() string {
	s := make([]string, len(w.items))
	for i, ee := range w.items {
		s[i] = ee.Name
	}
	return strings.Join(s, ", ")
}

func (w *eveEntityEntry) IsEmpty() bool {
	return len(w.items) == 0
}

func (w *eveEntityEntry) update() {
	w.main.RemoveAll()
	columns := kxlayout.NewColumns(w.labelWidth)
	if len(w.items) == 0 {
		w.placeholder.SetWithText(w.placeholderText)
		w.main.Add(container.New(columns, w.label, w.placeholder))
	} else {
		firstRow := true
		for _, ee := range w.items {
			var label fyne.CanvasObject
			if firstRow {
				label = w.label
				firstRow = false
			} else {
				label = layout.NewSpacer()
			}
			badge := newEveEntityBadge(ee, w.loadIcon)
			badge.OnTapped = func() {
				menu := w.makeBadgeMenu(ee)
				pm := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(badge))
				pm.ShowAtRelativePosition(fyne.Position{}, badge)
			}
			w.main.Add(container.New(columns, label, badge))
		}
	}
}

// makeBadgeMenu returns the context menu for the badge of ee.
// The first item is the name item.
func (w *eveEntityEntry) makeBadgeMenu(ee *app.EveEntity) *fyne.Menu {
	nameItem := fyne.NewMenuItem(fmt.Sprintf("%s (%s)", ee.Name, ee.CategoryDisplay()), nil)
	nameItem.Icon = categoryIcon(ee.Category)
	nameItem.Disabled = true
	var actions []*fyne.MenuItem
	if w.showInfo != nil && infoviewer.SupportedCategories().Contains(ee.Category) {
		infoItem := fyne.NewMenuItem("Show Info", func() {
			w.showInfo(ee)
		})
		infoItem.Icon = theme.InfoIcon()
		actions = append(actions, infoItem)
	}
	if !w.readOnly {
		removeItem := fyne.NewMenuItem("Remove", func() {
			w.Remove(ee.ID)
		})
		removeItem.Icon = theme.DeleteIcon()
		actions = append(actions, removeItem)
	}
	if len(actions) == 0 {
		return fyne.NewMenu("", nameItem)
	}
	return fyne.NewMenu("", slices.Concat([]*fyne.MenuItem{nameItem, fyne.NewMenuItemSeparator()}, actions)...)
}

// categoryIcon returns the icon for the name item of an entity with category c.
func categoryIcon(c app.EveEntityCategory) fyne.Resource {
	switch c {
	case app.EveEntityCharacter:
		return theme.NewThemedResource(icons.AccountCircleSvg)
	case app.EveEntityCorporation, app.EveEntityAlliance:
		return theme.NewThemedResource(icons.StarCircleOutlineSvg)
	case app.EveEntityMailList:
		return theme.MailComposeIcon()
	}
	return theme.QuestionIcon()
}

func (w *eveEntityEntry) Refresh() {
	w.update()
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.field.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.field.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	w.main.Refresh()
	w.field.Refresh()
	w.placeholder.Refresh()
	w.BaseWidget.Refresh()
}

func (w *eveEntityEntry) MinSize() fyne.Size {
	th := w.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	textSize := th.Size(theme.SizeNameText)
	minSize := fyne.MeasureText("M", textSize, fyne.TextStyle{})
	minSize = minSize.Add(fyne.NewSquareSize(innerPadding))
	minSize = minSize.AddWidthHeight(innerPadding*2, innerPadding)
	return minSize.Max(w.BaseWidget.MinSize())
}

func (w *eveEntityEntry) CreateRenderer() fyne.WidgetRenderer {
	w.update()
	c := container.NewStack(w.field, w.main)
	return widget.NewSimpleRenderer(c)
}
