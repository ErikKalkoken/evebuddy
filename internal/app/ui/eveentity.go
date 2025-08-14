package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	defaultIconPixelSize = 64
)

// eveEntityEntry represents an entry widget for Eve Entity items.
type eveEntityEntry struct {
	widget.DisableableWidget

	Placeholder    string
	ShowInfoWindow func(*app.EveEntity)

	eis         app.EveImageService
	field       *canvas.Rectangle
	label       fyne.CanvasObject
	labelWidth  float32
	main        *fyne.Container
	mu          sync.Mutex
	placeholder *iwidget.RichText
	s           []*app.EveEntity
}

func newEveEntityEntry(label fyne.CanvasObject, labelWidth float32, eis app.EveImageService) *eveEntityEntry {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	w := &eveEntityEntry{
		field:      bg,
		eis:        eis,
		label:      label,
		labelWidth: labelWidth,
		main:       container.New(layout.NewCustomPaddedVBoxLayout(0)),
		placeholder: iwidget.NewRichText(&widget.TextSegment{
			Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder},
		}),
		s: make([]*app.EveEntity, 0),
	}
	w.ExtendBaseWidget(w)
	return w
}

// Items returns the current list of EveEntities items.
func (w *eveEntityEntry) Items() []*app.EveEntity {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.s
}

// Set replaces the list of items.
func (w *eveEntityEntry) Set(s []*app.EveEntity) {
	w.mu.Lock()
	w.s = s
	w.mu.Unlock()
	w.Refresh()
}

func (w *eveEntityEntry) Add(ee *app.EveEntity) {
	added := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		for _, o := range w.s {
			if o.ID == ee.ID {
				return false
			}
		}
		w.s = append(w.s, ee)
		return true
	}()
	if added {
		w.Refresh()
	}
}

func (w *eveEntityEntry) Remove(id int32) {
	removed := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		for i, o := range w.s {
			if o.ID == id {
				w.s = slices.Delete(w.s, i, i+1)
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
	s := make([]string, len(w.s))
	for i, ee := range w.s {
		s[i] = ee.Name
	}
	return strings.Join(s, ", ")
}

func (w *eveEntityEntry) IsEmpty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.s) == 0
}

func (w *eveEntityEntry) update() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.main.RemoveAll()
	columns := kxlayout.NewColumns(w.labelWidth)
	if len(w.s) == 0 {
		w.placeholder.SetWithText(w.Placeholder)
		w.main.Add(container.New(columns, w.label, w.placeholder))
	} else {
		firstRow := true
		isDisabled := w.Disabled()
		for _, ee := range w.s {
			var label fyne.CanvasObject
			if firstRow {
				label = w.label
				firstRow = false
			} else {
				label = layout.NewSpacer()
			}
			badge := newEveEntityBadge(ee, w.eis, nil)
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
				go func() {
					res, err := fetchEveEntityAvatar(w.eis, ee, icons.Questionmark32Png)
					if err != nil {
						slog.Error("fetch eve entity avatar", "error", err)
						return
					}
					fyne.Do(func() {
						nameItem.Icon = res
						pm.Refresh()
					})
				}()
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
	w.BaseWidget.Refresh()
	w.main.Refresh()
	w.field.Refresh()
	w.placeholder.Refresh()
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
	eis          app.EveImageService
	hovered      bool
}

var _ fyne.Tappable = (*eveEntityBadge)(nil)
var _ desktop.Hoverable = (*eveEntityBadge)(nil)

func newEveEntityBadge(ee *app.EveEntity, eis app.EveImageService, onTapped func()) *eveEntityBadge {
	w := &eveEntityBadge{
		ee:           ee,
		eis:          eis,
		fallbackIcon: icons.Questionmark32Png,
		OnTapped:     onTapped,
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
	icon := iwidget.NewImageFromResource(w.fallbackIcon, fyne.NewSquareSize(th.Size(theme.SizeNameInlineIcon)))
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
	go func() {
		res, err := fetchEveEntityAvatar(w.eis, w.ee, w.fallbackIcon)
		if err != nil {
			slog.Error("fetch eve entity avatar", "error", err)
			res = w.fallbackIcon
		}
		fyne.Do(func() {
			icon.Resource = res
			icon.Refresh()
		})
	}()
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
func (w *eveEntityBadge) MouseIn(e *desktop.MouseEvent) {
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

// fetchEveEntityAvatar fetches an icon for an EveEntity and returns it in avatar style.
func fetchEveEntityAvatar(eis app.EveImageService, ee *app.EveEntity, fallback fyne.Resource) (fyne.Resource, error) {
	if ee == nil {
		return fallback, nil
	}
	if ee.Category == app.EveEntityMailList {
		return theme.MailComposeIcon(), nil
	}
	res, err := EntityIcon(eis, ee, defaultIconPixelSize, fallback)
	if err != nil {
		return nil, err
	}
	return fynetools.MakeAvatar(res)
}

// EntityIcon returns an icon form EveImageService for several entity categories.
// It returns the fallback for unsupported categories.
func EntityIcon(eis app.EveImageService, ee *app.EveEntity, size int, fallback fyne.Resource) (fyne.Resource, error) {
	var r fyne.Resource
	var err error
	switch ee.Category {
	case app.EveEntityAlliance:
		r, err = eis.AllianceLogo(ee.ID, size)
	case app.EveEntityCharacter:
		r, err = eis.CharacterPortrait(ee.ID, size)
	case app.EveEntityCorporation:
		r, err = eis.CorporationLogo(ee.ID, size)
	case app.EveEntityFaction:
		r, err = eis.FactionLogo(ee.ID, size)
	case app.EveEntityInventoryType:
		r, err = eis.InventoryTypeIcon(ee.ID, size)
	default:
		if fallback != nil {
			return fallback, nil
		}
		slog.Warn("unsupported category. Falling back to default", "entity", ee)
		return icons.Questionmark32Png, nil
	}
	if err != nil {
		return nil, fmt.Errorf("entity icon %v %d: %w", ee, size, err)
	}
	return r, nil
}
