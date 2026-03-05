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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"

	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type UIService interface {
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	HumanizeError(err error) string
	IsDeveloperMode() bool
	IsOffline() bool
	MakeWindowTitle(parts ...string) string
	ShowInformationDialog(title, message string, parent fyne.Window)
}

type CS interface {
	GetCharacter(ctx context.Context, id int64) (*app.Character, error)
	GetSkill(ctx context.Context, characterID int64, typeID int64) (*app.CharacterSkill, error)
}

type SCS interface {
	ListCharacterIDs() set.Set[int64]
}

type EUS interface {
	FetchAlliance(ctx context.Context, allianceID int64) (*app.EveAlliance, error)
	FetchAllianceCorporations(ctx context.Context, allianceID int64) ([]*app.EveEntity, error)
	FetchCharacterCorporationHistory(ctx context.Context, characterID int64) ([]app.MembershipHistoryItem, error)
	FetchCorporationAllianceHistory(ctx context.Context, corporationID int64) ([]app.MembershipHistoryItem, error)
	FormatDogmaValue(ctx context.Context, value float64, unitID app.EveUnitID) (string, int64)
	GetCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, error)
	GetConstellationSolarSystemsESI(ctx context.Context, id int64) ([]*app.EveSolarSystem, error)
	GetOrCreateCharacterESI(ctx context.Context, characterID int64) (*app.EveCharacter, bool, error)
	GetOrCreateConstellationESI(ctx context.Context, id int64) (*app.EveConstellation, error)
	GetOrCreateCorporationESI(ctx context.Context, id int64) (*app.EveCorporation, error)
	GetOrCreateEntityESI(ctx context.Context, id int64) (*app.EveEntity, error)
	GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error)
	GetOrCreateRaceESI(ctx context.Context, id int64) (*app.EveRace, error)
	GetOrCreateRegionESI(ctx context.Context, id int64) (*app.EveRegion, error)
	GetOrCreateSolarSystemESI(ctx context.Context, id int64) (*app.EveSolarSystem, error)
	GetOrCreateTypeESI(ctx context.Context, id int64) (*app.EveType, error)
	GetRegionConstellationsESI(ctx context.Context, id int64) ([]*app.EveEntity, error)
	GetSolarSystemInfoESI(ctx context.Context, solarSystemID int64) (starID optional.Optional[int64], planets []app.EveSolarSystemPlanet, stargateIDs []int64, stations []*app.EveEntity, structures []*app.EveLocation, err error)
	GetSolarSystemPlanets(ctx context.Context, planets []app.EveSolarSystemPlanet) ([]*app.EvePlanet, error)
	GetStargatesSolarSystemsESI(ctx context.Context, stargateIDs []int64) ([]*app.EveSolarSystem, error)
	GetStarTypeID(ctx context.Context, id int64) (int64, error)
	GetStationServicesESI(ctx context.Context, id int64) ([]string, error)
	GetType(ctx context.Context, id int64) (*app.EveType, error)
	ListTypeDogmaAttributesForType(ctx context.Context, typeID int64) ([]*app.EveTypeDogmaAttribute, error)
	MarketPrice(ctx context.Context, typeID int64) (optional.Optional[float64], error)
}

type EIS interface {
	AllianceLogoAsync(id int64, size int, setter func(r fyne.Resource))
	CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource))
	CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource))
	FactionLogoAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeRenderAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeBPOAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeBPCAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeSKINAsync(id int64, size int, setter func(r fyne.Resource))
}

type Settings interface {
	PreferMarketTab() bool
}

// InfoWindow represents a dedicated window for showing information about Eve objects
// similar to the in-game info window.
type InfoWindow struct {
	cs            CS
	eis           EIS
	eus           EUS
	isMobile      bool
	js            *janiceservice.JaniceService
	nav           *iwidget.Navigator
	onClosedFuncs []func() // f runs when the window is closed. Useful for cleanup.
	sb            *iwidget.Snackbar
	scs           SCS
	settings      Settings
	u             UIService
	w             fyne.Window
}

type Params struct {
	CharacterService   CS
	EveImageService    EIS
	EveUniverseService EUS
	IsMobile           bool
	JaniceService      *janiceservice.JaniceService
	StatusCacheService SCS
	Settings           Settings
	UIService          UIService
	Window             fyne.Window
}

const (
	infoWindowHeight    = 600
	infoWindowWidth     = 600
	logoUnitSize        = 64
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	zoomImagePixelSize  = 512
)

// New returns a configured InfoWindow.
func New(arg Params) *InfoWindow {
	iw := &InfoWindow{
		cs:       arg.CharacterService,
		eis:      arg.EveImageService,
		eus:      arg.EveUniverseService,
		isMobile: arg.IsMobile,
		js:       arg.JaniceService,
		scs:      arg.StatusCacheService,
		settings: arg.Settings,
		u:        arg.UIService,
		w:        arg.Window,
	}
	if iw.cs == nil ||
		iw.eis == nil ||
		iw.eus == nil ||
		iw.js == nil ||
		iw.scs == nil ||
		iw.settings == nil ||
		iw.u == nil ||
		iw.w == nil {
		panic(app.ErrInvalid)
	}
	return iw
}

// ShowEveEntity shows a new info window for an EveEntity.
func (iw *InfoWindow) ShowEveEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), ee.ID)
}

// Show shows a new info window for an EveEntity.
func (iw *InfoWindow) Show(c app.EveEntityCategory, id int64) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), id)
}

func (iw *InfoWindow) ShowLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw *InfoWindow) ShowRace(id int64) {
	iw.show(infoRace, id)
}

func (iw *InfoWindow) ShowTypeWithCharacter(typeID, characterID int64) {
	iw.showWithCharacterID(infoInventoryType, typeID, characterID)
}

// infoWidget defines common functionality for all info widgets.
type infoWidget interface {
	fyne.CanvasObject
	update(context.Context) error
	setError(string)
}

func (iw *InfoWindow) show(v infoVariant, id int64) {
	iw.showWithCharacterID(v, id, 0)
}

func (iw *InfoWindow) showWithCharacterID(v infoVariant, entityID int64, characterID int64) {
	if iw.u.IsOffline() {
		iw.u.ShowInformationDialog(
			"Offline",
			"Can't show info window when offline",
			iw.w,
		)
		return
	}

	makeAppBarTitle := func(s string) string {
		if iw.isMobile {
			return s
		}
		return s + ": Information"
	}

	if v == infoLocation {
		switch app.LocationVariantFromID(entityID) {
		case app.EveLocationSolarSystem:
			v = infoSolarSystem
		case app.EveLocationUnknown:
			iw.u.ShowInformationDialog(
				"Unknown location",
				"Can't show info window for an unknown location",
				iw.w,
			)
			return
		}
	}

	var title string
	var page infoWidget
	var ab *iwidget.AppBar
	switch v {
	case infoAlliance:
		title = "Alliance"
		page = newAllianceInfo(iw, entityID)
	case infoCharacter:
		title = "Character"
		page = newCharacterInfo(iw, entityID)
	case infoConstellation:
		title = "Constellation"
		page = newConstellationInfo(iw, entityID)
	case infoCorporation:
		title = "Corporation"
		page = newCorporationInfo(iw, entityID)
	case infoInventoryType:
		x := newInventoryTypeInfo(iw, entityID, characterID)
		x.setTitle = func(s string) { ab.SetTitle(makeAppBarTitle(s)) }
		page = x
		title = "Item"
	case infoRace:
		title = "Race"
		page = newRaceInfo(iw, entityID)
	case infoRegion:
		title = "Region"
		page = newRegionInfo(iw, entityID)
	case infoSolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, entityID)
	case infoLocation:
		title = "Location"
		page = newLocationInfo(iw, entityID)
	default:
		iw.u.ShowInformationDialog(
			"Warning",
			"Can't show info window for unknown category",
			iw.w,
		)
		return
	}
	ab = iwidget.NewAppBar(makeAppBarTitle(title), page)
	ab.HideBackground = !iw.isMobile
	if iw.nav == nil {
		w := fyne.CurrentApp().NewWindow(iw.u.MakeWindowTitle("Information"))
		iw.w = w
		iw.sb = iwidget.NewSnackbar(w)
		iw.sb.Start()
		iw.nav = iwidget.NewNavigator(ab)
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
	}
	go func() {
		err := page.update(context.Background())
		if err != nil {
			slog.Error("info widget load", "variant", v, "id", entityID, "error", err)
			fyne.Do(func() {
				page.setError("ERROR: " + iw.u.HumanizeError(err))
			})
		}
	}()
}

func (iw *InfoWindow) showZoomWindow(title string, id int64, load func(int64, int, func(fyne.Resource)), w fyne.Window) {
	w2, created := iw.u.GetOrCreateWindow(fmt.Sprintf("zoom-window-%d", id), title)
	if !created {
		w2.Show()
		return
	}
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	image := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(s))
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

func (iw *InfoWindow) makeZKillboardIcon(id int64, v infoVariant) *iwidget.TappableIcon {
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
	icon := iwidget.NewTappableIcon(icons.ZkillboardPng, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoWindow) makeDotlanIcon(id int64, v infoVariant) *iwidget.TappableIcon {
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
	icon := iwidget.NewTappableIcon(icons.DotlanAvatarPng, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoWindow) makeEveWhoIcon(id int64, v infoVariant) *iwidget.TappableIcon {
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
	icon := iwidget.NewTappableIcon(icons.Characterplaceholder32Jpeg, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoWindow) renderIconSize() fyne.Size {
	var s float32
	if iw.isMobile {
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

func InfoWindowSupportedEveEntities() set.Set[app.EveEntityCategory] {
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
	supportedCategories := InfoWindowSupportedEveEntities()
	l := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			value := widget.NewLabel("Value")
			value.Truncation = fyne.TextTruncateEllipsis
			value.Alignment = fyne.TextAlignTrailing
			label := widget.NewLabel("Label")
			icon := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
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
						w.iw.ShowEveEntity(x)
					}
				}
			case *app.EveLocation:
				if x != nil {
					f = func() {
						w.iw.ShowLocation(x.ID)
					}
				}
			case *app.EveRace:
				if x != nil {
					f = func() {
						w.iw.ShowRace(x.ID)
					}
				}
			}
			iconBox := border[2].(*fyne.Container)
			if f != nil {
				iconBox.Objects[1].(*iwidget.TappableIcon).OnTapped = f
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
				w.iw.sb.Show(fmt.Sprintf("ERROR: Failed to open URL: %s", w.iw.u.HumanizeError(err)))
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
			text := iwidget.NewRichText()
			text.Truncation = fyne.TextTruncateEllipsis
			icon := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
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
			icon := border1[1].(*fyne.Container).Objects[1].(*iwidget.TappableIcon)
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
			text := border2[1].(*fyne.Container).Objects[0].(*iwidget.RichText)
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
	logo := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(logoUnitSize))
	return logo
}

func newLabelWithWrapAndSelectable(s string) *widget.Label {
	description := widget.NewLabel(s)
	description.Wrapping = fyne.TextWrapWord
	description.Selectable = true
	return description
}
