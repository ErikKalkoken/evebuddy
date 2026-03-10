package characterui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const labelWith = 45

type SendMail struct {
	widget.BaseWidget

	body      *widget.Entry
	character atomic.Pointer[app.Character]
	from      *eveEntityEntry
	subject   *widget.Entry
	to        *eveEntityEntry
	u         ui
	w         fyne.Window
}

func NewSendMail(u ui, c *app.Character, mode app.SendMailMode, m *app.CharacterMail) *SendMail {
	a := &SendMail{
		u: u,
		w: u.MainWindow(),
	}
	a.character.Store(c)
	a.ExtendBaseWidget(a)

	a.from = newEveEntityEntry(widget.NewLabel("From"), labelWith, awidget.LoadEveEntityIconFunc(u.EVEImage()))
	a.from.ShowInfoWindow = u.InfoWindow().ShowEntity
	a.from.Set([]*app.EveEntity{{
		ID:       c.ID,
		Name:     c.EveCharacter.Name,
		Category: app.EveEntityCharacter,
	}})
	a.from.Disable()

	toButton := widget.NewButton("To", func() {
		if a.u.IsOffline() {
			xdialog.ShowInformation("OFFLINE", "Search not available while offline", a.w)
			return
		}
		showAddDialog(u, c.ID, func(ee *app.EveEntity) {
			a.to.Add(ee)
		}, a.w)
	})
	a.to = newEveEntityEntry(toButton, labelWith, awidget.LoadEveEntityIconFunc(u.EVEImage()))
	a.to.ShowInfoWindow = u.InfoWindow().ShowEntity
	a.to.Placeholder = "Tap To-Button to add recipients..."

	a.subject = widget.NewEntry()
	a.subject.PlaceHolder = "Subject"

	a.body = widget.NewEntry()
	a.body.MultiLine = true
	a.body.Wrapping = fyne.TextWrapWord
	a.body.SetMinRowsVisible(14)
	a.body.PlaceHolder = "Compose message"

	if m != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case app.SendMailReply:
			a.to.Set([]*app.EveEntity{m.From})
			a.subject.SetText(fmt.Sprintf("Re: %s", m.Subject))
			a.body.SetText(sep + m.String())
		case app.SendMailReplyAll:
			oo := slices.Concat([]*app.EveEntity{m.From}, m.Recipients)
			oo = slices.DeleteFunc(oo, func(o *app.EveEntity) bool {
				return o.ID == c.EveCharacter.ID
			})
			a.to.Set(oo)
			a.subject.SetText(fmt.Sprintf("Re: %s", m.Subject))
			a.body.SetText(sep + m.String())
		case app.SendMailForward:
			a.subject.SetText(fmt.Sprintf("Fw: %s", m.Subject))
			a.body.SetText(sep + m.String())
		default:
			panic(fmt.Errorf("undefined mode for create message: %v", mode))
		}
	}
	return a
}

func (a *SendMail) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(a.from, a.to, a.subject),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *SendMail) SetWindow(w fyne.Window) {
	a.w = w
}

// SendAction tries to send the current mail and reports whether it was successful
func (a *SendMail) SendAction() bool {
	showErrorDialog := func(message string) {
		xdialog.ShowInformation("Failed to send mail", message, a.u.MainWindow())
	}
	if a.to.IsEmpty() {
		showErrorDialog("A mail needs to have at least one recipient.")
		return false
	}
	if a.subject.Text == "" {
		showErrorDialog("The subject can not be empty.")
		return false
	}
	if a.body.Text == "" {
		showErrorDialog("The message can not be empty.")
		return false
	}
	ctx := context.Background()
	c := a.character.Load()
	_, err := a.u.Character().SendMail(
		ctx,
		c.ID,
		a.subject.Text,
		a.to.Items(),
		a.body.Text,
	)
	if err != nil {
		showErrorDialog(err.Error())
		return false
	}
	go a.u.Signals().CharacterSectionChanged.Emit(ctx, app.CharacterSectionUpdated{
		CharacterID: c.ID,
		Section:     app.SectionCharacterMailHeaders,
	})
	a.u.ShowSnackbar(fmt.Sprintf("Your mail to %s has been sent.", a.to))
	return true
}

// eveEntityEntry represents an entry widget for Eve Entity items.
type eveEntityEntry struct {
	widget.DisableableWidget

	Placeholder    string
	ShowInfoWindow func(*app.EveEntity)

	loadIcon    awidget.EveEntityIconLoader
	field       *canvas.Rectangle
	items       []*app.EveEntity
	label       fyne.CanvasObject
	labelWidth  float32
	main        *fyne.Container
	placeholder *xwidget.RichText
}

func newEveEntityEntry(label fyne.CanvasObject, labelWidth float32, loadIcon awidget.EveEntityIconLoader) *eveEntityEntry {
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

// Items returns the current list of EveEntities items.
func (w *eveEntityEntry) Items() []*app.EveEntity {
	return w.items
}

// Set replaces the list of items.
func (w *eveEntityEntry) Set(s []*app.EveEntity) {
	w.items = s
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
		w.placeholder.SetWithText(w.Placeholder)
		w.main.Add(container.New(columns, w.label, w.placeholder))
	} else {
		firstRow := true
		isDisabled := w.Disabled()
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
				s := fmt.Sprintf("%s (%s)", ee.Name, ee.CategoryDisplay())
				nameItem := fyne.NewMenuItem(s, nil)
				nameItem.Icon = icons.Questionmark32Png
				if ee.Category == app.EveEntityCharacter && w.ShowInfoWindow != nil {
					nameItem.Action = func() {
						w.ShowInfoWindow(ee)
					}
				}
				removeItem := fyne.NewMenuItem("Remove", func() {
					w.Remove(ee.ID)
				})
				removeItem.Icon = theme.DeleteIcon()
				removeItem.Disabled = isDisabled
				menu := fyne.NewMenu("", nameItem, fyne.NewMenuItemSeparator(), removeItem)
				pm := widget.NewPopUpMenu(menu, fyne.CurrentApp().Driver().CanvasForObject(badge))
				pm.ShowAtRelativePosition(fyne.Position{}, badge)
				w.loadIcon(ee, app.IconPixelSize, func(r fyne.Resource) {
					nameItem.Icon = r
					pm.Refresh()
				})
			}
			w.main.Add(container.New(columns, label, badge))
		}
	}
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

type eveEntityBadge struct {
	widget.DisableableWidget

	OnTapped func()

	ee           *app.EveEntity
	fallbackIcon fyne.Resource
	hovered      bool
	loadIcon     awidget.EveEntityIconLoader
}

var _ fyne.Tappable = (*eveEntityBadge)(nil)
var _ desktop.Hoverable = (*eveEntityBadge)(nil)

func newEveEntityBadge(ee *app.EveEntity, loadIcon awidget.EveEntityIconLoader) *eveEntityBadge {
	w := &eveEntityBadge{
		ee:           ee,
		fallbackIcon: icons.Questionmark32Png,
		loadIcon:     loadIcon,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *eveEntityBadge) CreateRenderer() fyne.WidgetRenderer {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	p := th.Size(theme.SizeNamePadding)

	name := widget.NewLabel(w.ee.Name)
	if w.Disabled() {
		name.Importance = widget.LowImportance
	}
	icon := xwidget.NewImageFromResource(
		w.fallbackIcon,
		fyne.NewSquareSize(th.Size(theme.SizeNameInlineIcon)),
	)
	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	rect.StrokeWidth = 1
	rect.CornerRadius = 10
	c := container.New(layout.NewCustomPaddedLayout(p, p, 0, 0),
		container.NewHBox(
			container.NewStack(
				rect,
				container.New(layout.NewCustomPaddedLayout(-p, -p, 0, 0),
					container.NewHBox(
						container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), icon), name,
					))),
		),
	)
	w.loadIcon(w.ee, app.IconPixelSize, func(r fyne.Resource) {
		icon.Resource = r
		icon.Refresh()
	})
	return widget.NewSimpleRenderer(c)
}

func (w *eveEntityBadge) Tapped(_ *fyne.PointEvent) {
	if w.Disabled() {
		return
	}
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *eveEntityBadge) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *eveEntityBadge) Cursor() desktop.Cursor {
	if !w.Disabled() && w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *eveEntityBadge) MouseIn(_ *desktop.MouseEvent) {
	if w.Disabled() {
		return
	}
	w.hovered = true
}

func (w *eveEntityBadge) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *eveEntityBadge) MouseOut() {
	if w.Disabled() {
		return
	}
	w.hovered = false
}

func showAddDialog(u ui, characterID int64, onSelected func(ee *app.EveEntity), w fyne.Window) {
	var modal *widget.PopUp
	var results []*app.EveEntity
	list := widget.NewList(
		func() int {
			return len(results)
		},
		func() fyne.CanvasObject {
			return newEntityItem(awidget.LoadEveEntityIconFunc(u.EVEImage()))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(results) {
				return
			}
			co.(*entityItem).set(results[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(results) {
			list.UnselectAll()
			return
		}
		onSelected(results[id])
		modal.Hide()
	}
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		xdialog.ShowErrorAndLog("Something went wrong", err, u.IsDeveloperMode(), w)
	}
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		entry.SetText("")
	})
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			results = results[:0]
			list.Refresh()
			return
		}
		ctx := context.Background()
		go func() {
			r, err := u.EVEUniverse().ListEntitiesByPartialName(ctx, search)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			fyne.Do(func() {
				results = r
				list.Refresh()
			})
		}()
		go func() {
			missingIDs, err := u.Character().AddEveEntitiesFromSearchESI(
				ctx,
				characterID,
				search,
			)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			if missingIDs.Size() == 0 {
				return // no need to update when not changed
			}
			r, err := u.EVEUniverse().ListEntitiesByPartialName(ctx, search)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			fyne.Do(func() {
				results = r
				list.Refresh()
			})
		}()
	}
	c := container.NewBorder(
		container.NewBorder(
			widget.NewLabel("Add Recipient"),
			nil,
			nil,
			widget.NewButton("Cancel", func() {
				modal.Hide()
			}),
			entry,
		),
		nil,
		nil,
		nil,
		list,
	)
	modal = widget.NewModalPopUp(c, w.Canvas())
	_, s := w.Canvas().InteractiveArea()
	modal.Resize(fyne.NewSize(s.Width, s.Height))
	modal.Show()
	w.Canvas().Focus(entry)
}

type entityItem struct {
	widget.BaseWidget

	category *widget.Label
	icon     *canvas.Image
	name     *widget.Label
	loadIcon awidget.EveEntityIconLoader
}

func newEntityItem(loadIcon awidget.EveEntityIconLoader) *entityItem {
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateClip
	category := widget.NewLabel("")
	category.SizeName = theme.SizeNameCaptionText
	icon := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
	w := &entityItem{
		category: category,
		loadIcon: loadIcon,
		icon:     icon,
		name:     name,
	}
	w.ExtendBaseWidget(w)

	return w
}

func (w *entityItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(layout.NewCustomPaddedLayout(p, p, p, -p), w.icon),
			layout.NewSpacer(),
		),
		w.category,
		container.NewVBox(
			layout.NewSpacer(),
			w.name,
			layout.NewSpacer(),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *entityItem) set(o *app.EveEntity) {
	w.name.SetText(o.Name)
	w.category.SetText(o.CategoryDisplay())
	w.loadIcon(o, app.IconPixelSize, func(r fyne.Resource) {
		w.icon.Resource = r
		w.icon.Refresh()
	})
}
