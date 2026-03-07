package awidget

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
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type EveEntityEIS interface {
	AllianceLogo(int64, int) (fyne.Resource, error)
	CharacterPortrait(int64, int) (fyne.Resource, error)
	CorporationLogo(int64, int) (fyne.Resource, error)
	FactionLogo(int64, int) (fyne.Resource, error)
	InventoryTypeIcon(int64, int) (fyne.Resource, error)
}

// EveEntityEntry represents an entry widget for Eve Entity items.
type EveEntityEntry struct {
	widget.DisableableWidget

	Placeholder    string
	ShowInfoWindow func(*app.EveEntity)

	eis         EveEntityEIS
	field       *canvas.Rectangle
	items       []*app.EveEntity
	label       fyne.CanvasObject
	labelWidth  float32
	main        *fyne.Container
	mu          sync.Mutex
	placeholder *xwidget.RichText
}

func NewEveEntityEntry(label fyne.CanvasObject, labelWidth float32, eis EveEntityEIS) *EveEntityEntry {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	bg.StrokeWidth = theme.Size(theme.SizeNameInputBorder)
	bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	placeholder := xwidget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{ColorName: theme.ColorNamePlaceHolder},
	})
	w := &EveEntityEntry{
		eis:         eis,
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
func (w *EveEntityEntry) Items() []*app.EveEntity {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.items
}

// Set replaces the list of items.
func (w *EveEntityEntry) Set(s []*app.EveEntity) {
	w.mu.Lock()
	w.items = s
	w.mu.Unlock()
	w.Refresh()
}

func (w *EveEntityEntry) Add(ee *app.EveEntity) {
	added := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
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

func (w *EveEntityEntry) Remove(id int64) {
	removed := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
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
func (w *EveEntityEntry) String() string {
	s := make([]string, len(w.items))
	for i, ee := range w.items {
		s[i] = ee.Name
	}
	return strings.Join(s, ", ")
}

func (w *EveEntityEntry) IsEmpty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.items) == 0
}

func (w *EveEntityEntry) update() {
	w.mu.Lock()
	defer w.mu.Unlock()
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
				LoadEveEntityIconAsync(w.eis, ee, func(r fyne.Resource) {
					nameItem.Icon = r
					pm.Refresh()
				})
			}
			w.main.Add(container.New(columns, label, badge))
		}
	}
}

func (w *EveEntityEntry) Refresh() {
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

func (w *EveEntityEntry) MinSize() fyne.Size {
	th := w.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	textSize := th.Size(theme.SizeNameText)
	minSize := fyne.MeasureText("M", textSize, fyne.TextStyle{})
	minSize = minSize.Add(fyne.NewSquareSize(innerPadding))
	minSize = minSize.AddWidthHeight(innerPadding*2, innerPadding)
	return minSize.Max(w.BaseWidget.MinSize())
}

func (w *EveEntityEntry) CreateRenderer() fyne.WidgetRenderer {
	w.update()
	c := container.NewStack(w.field, w.main)
	return widget.NewSimpleRenderer(c)
}

type eveEntityBadge struct {
	widget.DisableableWidget

	OnTapped func()

	ee           *app.EveEntity
	fallbackIcon fyne.Resource
	eis          EveEntityEIS
	hovered      bool
}

var _ fyne.Tappable = (*eveEntityBadge)(nil)
var _ desktop.Hoverable = (*eveEntityBadge)(nil)

func newEveEntityBadge(ee *app.EveEntity, eis EveEntityEIS, onTapped func()) *eveEntityBadge {
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
	LoadEveEntityIconAsync(w.eis, w.ee, func(r fyne.Resource) {
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

var eveEntityResourceCache xsync.Map[int64, fyne.Resource]

// LoadEveEntityIconAsync fetches an icon for an EveEntity and returns it in avatar style.
func LoadEveEntityIconAsync(eis EveEntityEIS, ee *app.EveEntity, setter func(r fyne.Resource)) {
	if ee == nil {
		setter(theme.BrokenImageIcon())
		return
	}
	if ee.Category == app.EveEntityMailList {
		setter(theme.MailComposeIcon())
		return
	}
	xwidget.LoadResourceAsyncWithCache(
		icons.BlankSvg,
		func() (fyne.Resource, bool) {
			return eveEntityResourceCache.Load(ee.ID)
		},
		func(r fyne.Resource) {
			setter(r)
		},
		func() (fyne.Resource, error) {
			return EntityIcon(eis, ee, app.IconPixelSize, theme.NewThemedResource(icons.QuestionmarkSvg))
		},
		func(r fyne.Resource) {
			eveEntityResourceCache.Store(ee.ID, r)
		},
	)
}

// EntityIcon returns an icon form EveImageService for several entity categories.
// It returns the fallback for unsupported categories.
func EntityIcon(eis EveEntityEIS, ee *app.EveEntity, size int, fallback fyne.Resource) (fyne.Resource, error) {
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

type MakeEveEntityColumnParams[T any] struct {
	ColumnID  int
	EIS       EveEntityEIS
	GetEntity func(r T) *app.EveEntity
	IsAvatar  bool
	Label     string
	Width     int
}

// MakeEveEntityColumn returns a new data column for showing an entity.
func MakeEveEntityColumn[T any](arg MakeEveEntityColumnParams[T]) xwidget.DataColumn[T] {
	// set defaults
	if arg.Width == 0 {
		arg.Width = 220
	}
	if arg.GetEntity == nil {
		panic("must define entity getter")
	}
	if arg.EIS == nil {
		panic("must define eis")
	}
	c := xwidget.DataColumn[T]{
		ID:    arg.ColumnID,
		Label: arg.Label,
		Width: float32(arg.Width),
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			if arg.IsAvatar {
				icon.CornerRadius = app.IconUnitSize / 2
			}
			name := widget.NewLabel(arg.Label)
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r T, co fyne.CanvasObject) {
			ee := arg.GetEntity(r)
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(ee.Name)
			x := border[1].(*canvas.Image)
			LoadEveEntityIconAsync(arg.EIS, ee, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
		Sort: func(a, b T) int {
			return xstrings.CompareIgnoreCase(arg.GetEntity(a).Name, arg.GetEntity(b).Name)
		},
	}
	return c
}
