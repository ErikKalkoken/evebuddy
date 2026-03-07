// Package infowindow provides a window for displaying information about Eve objects.
package infowindow

import (
	"context"

	"fmt"

	"log/slog"
	"maps"
	"net/url"

	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"

	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type UIServices interface {
	Character() *characterservice.CharacterService
	EVEImage() *eveimageservice.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	IsOfflineMode() bool
	Janice() *janiceservice.JaniceService
	MainWindow() fyne.Window
	Settings() *settings.Settings
}

// InfoWindow represents a dedicated window for showing information about Eve objects
// similar to the in-game info window.
type InfoWindow struct {
	current       *showParams // parameters for currently shown info window (if any)
	nav           *xwidget.Navigator
	onClosedFuncs []func() // f runs when the window is closed. Useful for cleanup.
	s             UIServices
	sb            *xwidget.Snackbar
	w             fyne.Window
}

const (
	infoWindowHeight    = 600
	infoWindowWidth     = 600
	logoUnitSize        = 64
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	zoomImagePixelSize  = 512
)

// New returns a new InfoWindow.
func New(s UIServices) *InfoWindow {
	iw := &InfoWindow{
		s: s,
		w: s.MainWindow(),
	}
	return iw
}

func (iw *InfoWindow) Show(c app.EveEntityCategory, id int64) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), id)
}

// ShowEntity shows a new info window for an EveEntity.
func (iw *InfoWindow) ShowEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), ee.ID)
}

func (iw *InfoWindow) ShowLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw *InfoWindow) ShowRace(id int64) {
	iw.show(infoRace, id)
}

func (iw *InfoWindow) ShowType(typeID int64) {
	iw.showWithCharacterID(showParams{
		variant:     infoInventoryType,
		entityID:    typeID,
		characterID: 0,
	})
}

func (iw *InfoWindow) ShowTypeWithCharacter(typeID, characterID int64) {
	iw.showWithCharacterID(showParams{
		variant:     infoInventoryType,
		entityID:    typeID,
		characterID: characterID,
	})
}

func (iw *InfoWindow) show(v infoVariant, id int64) {
	iw.showWithCharacterID(showParams{
		variant:     v,
		entityID:    id,
		characterID: id,
	})
}

// infoWidget defines common functionality for all info widgets.
type infoWidget interface {
	fyne.CanvasObject
	update(context.Context) error
	setError(string)
}

type showParams struct {
	variant     infoVariant
	entityID    int64
	characterID int64
}

func (iw *InfoWindow) showWithCharacterID(arg showParams) {
	if iw.s.IsOfflineMode() {
		xdialog.ShowInformation(
			"Offline",
			"Can't show info window when offline",
			iw.w,
		)
		return
	}

	// don't spawn another window if it is already shown.
	if iw.current != nil && iw.w != nil {
		if *iw.current == arg {
			iw.w.Show()
			iw.w.RequestFocus()
			return
		}
	}
	iw.current = &arg

	makeAppBarTitle := func(s string) string {
		if app.IsMobile() {
			return s
		}
		return s + ": Information"
	}

	if arg.variant == infoLocation {
		switch app.LocationVariantFromID(arg.entityID) {
		case app.EveLocationSolarSystem:
			arg.variant = infoSolarSystem
		case app.EveLocationUnknown:
			xdialog.ShowInformation(
				"Unknown location",
				"Can't show info window for an unknown location",
				iw.w,
			)
			return
		}
	}

	var title string
	var page infoWidget
	var ab *xwidget.AppBar
	switch arg.variant {
	case infoAlliance:
		title = "Alliance"
		page = newAllianceInfo(iw, arg.entityID)
	case infoCharacter:
		title = "Character"
		page = newCharacterInfo(iw, arg.entityID)
	case infoConstellation:
		title = "Constellation"
		page = newConstellationInfo(iw, arg.entityID)
	case infoCorporation:
		title = "Corporation"
		page = newCorporationInfo(iw, arg.entityID)
	case infoInventoryType:
		x := newInventoryTypeInfo(iw, arg.entityID, arg.characterID)
		x.setTitle = func(s string) { ab.SetTitle(makeAppBarTitle(s)) }
		page = x
		title = "Item"
	case infoRace:
		title = "Race"
		page = newRaceInfo(iw, arg.entityID)
	case infoRegion:
		title = "Region"
		page = newRegionInfo(iw, arg.entityID)
	case infoSolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, arg.entityID)
	case infoLocation:
		title = "Location"
		page = newLocationInfo(iw, arg.entityID)
	default:
		xdialog.ShowInformation(
			"Warning",
			"Can't show info window for unknown category",
			iw.w,
		)
		return
	}
	ab = xwidget.NewAppBar(makeAppBarTitle(title), page)
	ab.HideBackground = !app.IsMobile()
	if iw.nav == nil {
		w := fyne.CurrentApp().NewWindow(app.MakeWindowTitle("Information"))
		iw.w = w
		iw.sb = xwidget.NewSnackbar(w)
		iw.sb.Start()
		iw.nav = xwidget.NewNavigator(ab)
		w.SetContent(fynetooltip.AddWindowToolTipLayer(iw.nav, w.Canvas()))
		w.Resize(fyne.NewSize(infoWindowWidth, infoWindowHeight))
		w.SetCloseIntercept(func() {
			w.Close()
			fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
		})
		w.SetOnClosed(func() {
			for _, f := range iw.onClosedFuncs {
				f()
			}
			iw.nav = nil
		})
		if fyne.CurrentDevice().IsMobile() {
			w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
				if ev.Name == mobile.KeyBack {
					iw.nav.Pop()
				}
			})
		}
		w.Show()
	} else {
		iw.nav.Push(ab)
		if iw.w != nil {
			iw.w.RequestFocus()
		}
	}
	go func() {
		err := page.update(context.Background())
		if err != nil {
			slog.Error("info widget load", "params", arg, "error", err)
			fyne.Do(func() {
				page.setError("ERROR: " + app.ErrorDisplay(err))
			})
		}
	}()
}

func (iw *InfoWindow) showZoomWindow(title string, id int64, load func(int64, int, func(fyne.Resource)), w fyne.Window) {
	w2, created := iw.s.GetOrCreateWindow(fmt.Sprintf("zoom-window-%d", id), title)
	if !created {
		w2.Show()
		return
	}
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	image := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(s))
	load(id, zoomImagePixelSize, func(r fyne.Resource) {
		image.Resource = r
		image.Refresh()
	})
	p := theme.Padding()
	w2.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), image))
	w2.Show()
}

func (iw *InfoWindow) openURL(s string) {
	x, err := url.ParseRequestURI(s)
	if err != nil {
		slog.Error("Constructing URL", "url", s, "error", err)
		return
	}
	err = fyne.CurrentApp().OpenURL(x)
	if err != nil {
		slog.Error("Opening URL", "url", x, "error", err)
		return
	}
}

func (iw *InfoWindow) makeZKillboardIcon(id int64, v infoVariant) *xwidget.TappableIcon {
	m := map[infoVariant]string{
		infoAlliance:    "alliance",
		infoCharacter:   "character",
		infoCorporation: "corporation",
		infoRegion:      "region",
		infoSolarSystem: "system",
	}
	var f func()
	var title string
	partial, ok := m[v]
	if ok {
		f = func() {
			iw.openURL(fmt.Sprintf("https://zkillboard.com/%s/%d/", partial, id))
		}
		title = fmt.Sprintf("Show %s on zKillboard.com", v)
	}
	icon := xwidget.NewTappableIcon(icons.ZkillboardPng, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoWindow) makeDotlanIcon(id int64, v infoVariant) *xwidget.TappableIcon {
	m := map[infoVariant]string{
		infoAlliance:    "alliance",
		infoCorporation: "corp",
		infoRegion:      "region",
		infoSolarSystem: "system",
	}
	var f func()
	var title string
	partial, ok := m[v]
	if ok {
		f = func() {
			iw.openURL(fmt.Sprintf("https://evemaps.dotlan.net/%s/%d", partial, id))
		}
		title = fmt.Sprintf("Show %s on evemaps.dotlan.net", v)
	}
	icon := xwidget.NewTappableIcon(icons.DotlanAvatarPng, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoWindow) makeEveWhoIcon(id int64, v infoVariant) *xwidget.TappableIcon {
	m := map[infoVariant]string{
		infoAlliance:    "alliance",
		infoCorporation: "corporation",
		infoCharacter:   "character",
	}
	var f func()
	var title string
	partial, ok := m[v]
	if ok {
		f = func() {
			iw.openURL(fmt.Sprintf("https://evewho.com/%s/%d", partial, id))
		}
		title = fmt.Sprintf("Show %s on evewho.com", v)
	}
	icon := xwidget.NewTappableIcon(icons.Characterplaceholder32Jpeg, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoWindow) renderIconSize() fyne.Size {
	var s float32
	if app.IsMobile() {
		s = logoUnitSize
	} else {
		s = renderIconUnitSize
	}
	return fyne.NewSquareSize(s)
}

type infoVariant uint

const (
	infoNotSupported infoVariant = iota
	infoAlliance
	infoCharacter
	infoConstellation
	infoCorporation
	infoInventoryType
	infoLocation
	infoRegion
	infoRace
	infoSolarSystem
)

func (iv infoVariant) String() string {
	m := map[infoVariant]string{
		infoAlliance:      "alliance",
		infoCharacter:     "character",
		infoConstellation: "constellation",
		infoCorporation:   "corporation",
		infoInventoryType: "type",
		infoLocation:      "location",
		infoRegion:        "region",
		infoRace:          "race",
		infoSolarSystem:   "solar system",
	}
	s, ok := m[iv]
	if !ok {
		return ""
	}
	return s
}

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]infoVariant{
	app.EveEntityAlliance:      infoAlliance,
	app.EveEntityCharacter:     infoCharacter,
	app.EveEntityConstellation: infoConstellation,
	app.EveEntityCorporation:   infoCorporation,
	app.EveEntityRegion:        infoRegion,
	app.EveEntitySolarSystem:   infoSolarSystem,
	app.EveEntityStation:       infoLocation,
	app.EveEntityInventoryType: infoInventoryType,
}

func eveEntity2InfoVariant(ee *app.EveEntity) infoVariant {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return infoNotSupported
	}
	return v

}

// SupportedCategories returns which EveEntity categories are supported.
func SupportedCategories() set.Set[app.EveEntityCategory] {
	return set.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}

// baseInfo represents shared functionality between all info widgets.
type baseInfo struct {
	name *widget.Label
	iw   *InfoWindow
}

func (b *baseInfo) initBase(iw *InfoWindow) {
	b.iw = iw
	b.name = newLabelWithWrapAndSelectable("Loading...")
}

func (b *baseInfo) setError(s string) {
	b.name.Text = s
	b.name.Importance = widget.DangerImportance
	b.name.Refresh()
}
func boolIconResource(ok bool) fyne.Resource {
	if ok {
		return theme.NewSuccessThemedResource(theme.ConfirmIcon())
	}
	return theme.NewErrorThemedResource(theme.CancelIcon())
}

type attributeItem struct {
	Label  string
	Value  any
	Action func(v any)
}

func newAttributeItem(label string, value any) attributeItem {
	return attributeItem{Label: label, Value: value}
}

type attributeList struct {
	widget.BaseWidget

	items   []attributeItem
	iw      *InfoWindow
	openURL func(*url.URL) error
}

func newAttributeList(iw *InfoWindow, items ...attributeItem) *attributeList {
	w := &attributeList{
		items:   items,
		iw:      iw,
		openURL: fyne.CurrentApp().OpenURL,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *attributeList) set(items []attributeItem) {
	w.items = items
	w.Refresh()
}

func (w *attributeList) CreateRenderer() fyne.WidgetRenderer {
	supportedCategories := SupportedCategories()
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			value := widget.NewLabel("Value")
			value.Truncation = fyne.TextTruncateEllipsis
			value.Alignment = fyne.TextAlignTrailing
			label := widget.NewLabel("Label")
			icon := xwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			return container.NewBorder(
				nil,
				nil,
				label,
				container.NewVBox(layout.NewSpacer(), icon, layout.NewSpacer()),
				value,
			)
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
			var s string
			var i widget.Importance
			switch x := it.Value.(type) {
			case *app.EveEntity:
				if x == nil {
					s = "?"
					break
				}
				s = x.Name
			case *app.EveRace:
				if x == nil {
					s = "?"
					break
				}
				s = x.Name
			case *app.EveLocation:
				if x == nil {
					s = "?"
					break
				}
				s = x.DisplayName()
			case *url.URL:
				if x == nil {
					s = "?"
					break
				}
				s = x.String()
				i = widget.HighImportance
			case float32:
				s = fmt.Sprintf("%.1f %%", x*100)
			case time.Time:
				if x.IsZero() {
					s = "-"
				} else {
					s = x.Format(app.DateTimeFormat)
				}
			case int:
				s = humanize.Comma(int64(x))
			case float64:
				s = humanize.Ftoa(x)
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

			var f func()
			switch x := it.Value.(type) {
			case *app.EveEntity:
				if x != nil && supportedCategories.Contains(x.Category) {
					f = func() {
						w.iw.ShowEntity(x)
					}
				}
			case *app.EveLocation:
				if x != nil {
					f = func() {
						w.iw.show(infoLocation, x.ID)
					}
				}
			case *app.EveRace:
				if x != nil {
					f = func() {
						w.iw.show(infoRace, x.ID)
					}
				}
			}
			iconBox := border[2].(*fyne.Container)
			if f != nil {
				iconBox.Objects[1].(*xwidget.TappableIcon).OnTapped = f
				iconBox.Show()
			} else {
				iconBox.Hide()
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(w.items) {
			return
		}
		it := w.items[id]
		x, ok := it.Value.(*url.URL)
		if ok && x != nil {
			err := w.openURL(x)
			if err != nil {
				w.iw.sb.Show(fmt.Sprintf("ERROR: Failed to open URL: %s", app.ErrorDisplay(err)))
			}
			return
		}
		if it.Action != nil {
			it.Action(it.Value)
		}
	}
	return widget.NewSimpleRenderer(l)
}

type entityItem struct {
	id           int64
	category     string
	text         string                   // text in markdown
	textSegments []widget.RichTextSegment // takes precedence over text when not empty
	infoVariant  infoVariant
}

func newEntityItem(id int64, category, text string, v infoVariant) entityItem {
	return entityItem{
		id:          id,
		category:    category,
		text:        text,
		infoVariant: v,
	}
}

func newEntityItemFromEvePlanet(o *app.EvePlanet) entityItem {
	return entityItem{
		id:          o.ID,
		category:    "Planet",
		text:        o.Name,
		infoVariant: infoNotSupported,
	}
}

func newEntityItemFromEveSolarSystem(o *app.EveSolarSystem) entityItem {
	ee := o.EveEntity()
	return entityItem{
		id:           ee.ID,
		category:     ee.CategoryDisplay(),
		textSegments: o.DisplayRichText(),
		infoVariant:  eveEntity2InfoVariant(ee),
	}
}

func newEntityItemFromEveEntity(ee *app.EveEntity) entityItem {
	return newEntityItem(ee.ID, ee.CategoryDisplay(), ee.Name, eveEntity2InfoVariant(ee))
}

func newEntityItemFromEveEntityWithText(ee *app.EveEntity, text string) entityItem {
	if text == "" {
		text = ee.Name
	}
	return newEntityItem(ee.ID, ee.CategoryDisplay(), text, eveEntity2InfoVariant(ee))
}

// entityList is a list widget for showing entities.
type entityList struct {
	widget.BaseWidget

	items    []entityItem
	showInfo func(infoVariant, int64)
}

func entityItemsFromEveEntities(ee []*app.EveEntity) []entityItem {
	items := xslices.Map(ee, func(ee *app.EveEntity) entityItem {
		return newEntityItemFromEveEntityWithText(ee, "")
	})
	return items
}

func newEntityList(show func(infoVariant, int64)) *entityList {
	var items []entityItem
	return newEntityListFromItems(show, items...)
}

func newEntityListFromItems(show func(infoVariant, int64), items ...entityItem) *entityList {
	w := &entityList{
		items:    items,
		showInfo: show,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *entityList) set(items ...entityItem) {
	w.items = items
	w.Refresh()
}

func (w *entityList) CreateRenderer() fyne.WidgetRenderer {
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			category := widget.NewLabel("Category")
			category.SizeName = theme.SizeNameCaptionText
			text := xwidget.NewRichText()
			text.Truncation = fyne.TextTruncateEllipsis
			icon := xwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			p := theme.Padding()
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewVBox(layout.NewSpacer(), icon, layout.NewSpacer()),
				container.New(
					layout.NewCustomPaddedVBoxLayout(0),
					container.New(layout.NewCustomPaddedLayout(0, -1.5*p, 0, 0), category),
					container.New(layout.NewCustomPaddedLayout(-1.5*p, 0, 0, 0), text),
				))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			border1 := co.(*fyne.Container).Objects
			border2 := border1[0].(*fyne.Container).Objects
			icon := border1[1].(*fyne.Container).Objects[1].(*xwidget.TappableIcon)
			category := border2[0].(*fyne.Container).Objects[0].(*widget.Label)
			category.SetText(it.category)
			if it.infoVariant == infoNotSupported {
				icon.Hide()
				icon.OnTapped = nil
			} else {
				icon.OnTapped = func() {
					w.showInfo(it.infoVariant, it.id)
				}
				icon.Show()
			}
			text := border2[1].(*fyne.Container).Objects[0].(*xwidget.RichText)
			if len(it.textSegments) != 0 {
				text.Set(it.textSegments)
			} else {
				text.ParseMarkdown(it.text)
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
	}
	return widget.NewSimpleRenderer(l)
}

func historyItem2EntityItem(hi app.MembershipHistoryItem) entityItem {
	var endDateStr string
	if !hi.EndDate.IsZero() {
		endDateStr = hi.EndDate.Format(app.DateFormat)
	} else {
		endDateStr = "this day"
	}
	var closed string
	if hi.IsDeleted.ValueOrZero() {
		closed = " (closed)"
	}
	text := fmt.Sprintf(
		"%s%s   **%s** to **%s** (%s days)",
		hi.OrganizationName(),
		closed,
		hi.StartDate.Format(app.DateFormat),
		endDateStr,
		humanize.Comma(int64(hi.Days)),
	)
	return newEntityItemFromEveEntityWithText(hi.Organization, text)
}

func makeInfoLogo() *canvas.Image {
	logo := xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(logoUnitSize))
	return logo
}

func newLabelWithWrapAndSelectable(s string) *widget.Label {
	description := widget.NewLabel(s)
	description.Wrapping = fyne.TextWrapWord
	description.Selectable = true
	return description
}
