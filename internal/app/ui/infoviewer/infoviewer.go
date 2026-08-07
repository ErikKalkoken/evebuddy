// Package infoviewer provides a window for displaying information about Eve objects.
package infoviewer

import (
	"context"
	"strings"

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
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"

	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

//go:generate go tool stringer -type=Kind

// Kind of entity for showing an info window.
type Kind uint

const (
	Undefined Kind = iota
	Alliance
	Bloodline
	Character
	Constellation
	Corporation
	Faction
	Location
	Race
	Region
	SolarSystem
	Type
)

var eveEntityCategory2InfoVariant = map[app.EveEntityCategory]Kind{
	app.EveEntityAlliance:      Alliance,
	app.EveEntityCharacter:     Character,
	app.EveEntityConstellation: Constellation,
	app.EveEntityCorporation:   Corporation,
	app.EveEntityFaction:       Faction,
	app.EveEntityInventoryType: Type,
	app.EveEntityRegion:        Region,
	app.EveEntitySolarSystem:   SolarSystem,
	app.EveEntityStation:       Location,
}

func eveEntity2InfoVariant(ee *app.EveEntity) Kind {
	v, ok := eveEntityCategory2InfoVariant[ee.Category]
	if !ok {
		return Undefined
	}
	return v

}

// SupportedCategories returns which EveEntity categories are supported.
func SupportedCategories() set.Set[app.EveEntityCategory] {
	return set.Collect(maps.Keys(eveEntityCategory2InfoVariant))

}

type baseUI interface {
	Character() *characterservice.CharacterService
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	ErrorDisplay(err error) string
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	Janice() *janiceservice.JaniceService
	MainWindow() fyne.Window
	Settings() *settings.Settings
}

// InfoViewer represents a dedicated window for showing information about Eve objects
// similar to the in-game info window.
type InfoViewer struct {
	current       *showParams // parameters for currently shown info window (if any)
	nav           *xwidget.Navigator
	onClosedFuncs []func() // f runs when the window is closed. Useful for cleanup.
	sb            *xwidget.Snackbar
	u             baseUI
	w             fyne.Window
}

const (
	windowHeight        = 600
	windowWidth         = 600
	logoUnitSize        = 64
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	zoomImagePixelSize  = 512
)

// New returns a new InfoViewer.
func New(u baseUI) *InfoViewer {
	iw := &InfoViewer{
		u: u,
		w: u.MainWindow(),
	}
	return iw
}

// Show display an info window for an entity.
//
// Note that not all entity categories are supported. See also SupportedCategories().
func (iw *InfoViewer) Show(o *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(o), o.ID)
}

// Show2 displays an info window for an object.
//
// It is designed to take IDs from a showinfo scheme link.
// The typeID is mandatory, itemID and characterID are optional.
// The characterID is used to fetch a token when trying to show a structure location.
func (iw *InfoViewer) Show2(typeID, itemID, characterID int64) {
	slog.Debug("Showing info window", "typeID", typeID, "itemID", itemID)
	w := iw.w
	if w == nil {
		w = iw.u.MainWindow()
	}

	showError := func(err error) {
		ui.ShowErrorAndLog("Can't show info window", err, iw.u.IsDeveloperMode(), w)
	}

	if typeID == 0 {
		showError(fmt.Errorf("missing type ID"))
		return
	}
	if itemID == 0 {
		iw.show(Type, typeID)
		return
	}

	switch typeID {
	case app.EveTypeAlliance:
		iw.show(Alliance, itemID)
		return
	case app.EveTypeCharacter:
		iw.show(Character, itemID)
		return
	case app.EveTypeConstellation:
		iw.show(Constellation, itemID)
		return
	case app.EveTypeCorporation:
		iw.show(Corporation, itemID)
		return
	case app.EveTypeRegion:
		iw.show(Region, itemID)
		return
	case app.EveTypeSolarSystem:
		iw.show(SolarSystem, itemID)
		return
	case app.EveTypeCaldariLogisticsStation:
		iw.show(Location, itemID)
		return
	}

	ctx := context.Background()
	et, err := iw.u.EVEUniverse().GetOrCreateTypeESI(ctx, typeID)
	if err != nil {
		showError(err)
		return
	}
	switch et.Group.Category.ID {
	case app.EveCategoryStation:
		iw.show(Location, itemID)
		return
	case app.EveCategoryStructure:
		iw.show2(showParams{
			variant:     Location,
			entityID:    itemID,
			characterID: characterID,
		})
		return
	}
	switch et.Group.ID {
	case app.EveGroupCharacter:
		iw.show(Character, itemID)
		return
	}

	showError(fmt.Errorf("not supported"))
}

func (iw *InfoViewer) ShowBloodline(id int64) {
	iw.show(Bloodline, id)
}

func (iw *InfoViewer) ShowLocation(id int64) {
	iw.show(Location, id)
}

func (iw *InfoViewer) ShowRace(id int64) {
	iw.show(Race, id)
}

func (iw *InfoViewer) ShowType(typeID, characterID int64) {
	iw.show2(showParams{
		variant:     Type,
		entityID:    typeID,
		characterID: characterID,
	})
}

func (iw *InfoViewer) show(v Kind, id int64) {
	iw.show2(showParams{
		variant:  v,
		entityID: id,
	})
}

// infoWidget defines common functionality for all info widgets.
type infoWidget interface {
	fyne.CanvasObject
	update(context.Context) error
	setError(string)
}

type showParams struct {
	variant     Kind
	entityID    int64
	characterID int64
}

func (iw *InfoViewer) show2(arg showParams) {
	// iw.w is nil after an info window has been closed; fall back to the main window
	// so that error/informational dialogs always have a valid parent.
	parentW := iw.w
	if parentW == nil {
		parentW = iw.u.MainWindow()
	}

	if arg.entityID == 0 {
		ui.ShowErrorAndLog("Can't show info window", fmt.Errorf("no ID provided"), iw.u.IsDeveloperMode(), parentW)
		return
	}

	if arg.variant == Undefined {
		ui.ShowErrorAndLog("Can't show info window", fmt.Errorf("not supported"), iw.u.IsDeveloperMode(), parentW)
		return
	}

	if iw.u.IsOffline() {
		ui.ShowInformation(
			"Offline",
			"Can't show info window when offline",
			parentW,
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
		if iw.u.IsMobile() {
			return s
		}
		return s + ": Information"
	}

	if arg.variant == Location {
		switch app.LocationVariantFromID(arg.entityID) {
		case app.EveLocationSolarSystem:
			arg.variant = SolarSystem
		case app.EveLocationUnknown:
			ui.ShowInformation(
				"Unknown location",
				"Can't show info window for an unknown location",
				parentW,
			)
			return
		}
	}

	var title string
	var page infoWidget
	var ab *xwidget.AppBar
	switch arg.variant {
	case Alliance:
		title = "Alliance"
		page = newAllianceInfo(iw, arg.entityID)
	case Bloodline:
		title = "Bloodline"
		page = newBloodlineInfo(iw, arg.entityID)
	case Character:
		title = "Character"
		page = newCharacterInfo(iw, arg.entityID)
	case Constellation:
		title = "Constellation"
		page = newConstellationInfo(iw, arg.entityID)
	case Corporation:
		title = "Corporation"
		page = newCorporationInfo(iw, arg.entityID)
	case Faction:
		title = "Faction"
		page = newFactionInfo(iw, arg.entityID)
	case Location:
		title = "Location"
		page = newLocationInfo(iw, arg.entityID, arg.characterID)
	case Race:
		title = "Race"
		page = newRaceInfo(iw, arg.entityID)
	case Region:
		title = "Region"
		page = newRegionInfo(iw, arg.entityID)
	case SolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, arg.entityID)
	case Type:
		x := newInventoryTypeInfo(iw, arg.entityID, arg.characterID)
		x.setTitle = func(s string) { ab.SetTitle(makeAppBarTitle(s)) }
		page = x
		title = "Item"
	default:
		ui.ShowInformation(
			"Warning",
			"Can't show info window for unknown category",
			parentW,
		)
		return
	}
	ab = xwidget.NewAppBar(makeAppBarTitle(title), page)
	ab.HideBackground = !iw.u.IsMobile()
	if iw.nav == nil {
		w, _, onClosed := iw.u.GetOrCreateWindowWithOnClosed("", "Information")
		iw.w = w
		iw.sb = xwidget.NewSnackbar(w.Canvas())
		iw.sb.Start()
		iw.nav = xwidget.NewNavigator(ab)
		w.SetOnClosed(func() {
			for _, f := range iw.onClosedFuncs {
				f()
			}
			iw.nav = nil
			iw.w = nil
			if onClosed != nil {
				onClosed()
			}
		})
		w.SetCloseIntercept(func() {
			w.Close()
			fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
		})
		if fyne.CurrentDevice().IsMobile() {
			w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
				if ev.Name == mobile.KeyBack {
					if iw.nav.Pop() == nil {
						w.Close()
					}
				}
			})
		}
		w.SetContent(fynetooltip.AddWindowToolTipLayer(iw.nav, w.Canvas()))
		w.Resize(fyne.NewSize(windowWidth, windowHeight))
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
				page.setError("ERROR: " + iw.u.ErrorDisplay(err))
			})
		}
	}()
}

func (iw *InfoViewer) showZoomWindow(title string, id int64, load func(int64, int, func(fyne.Resource)), w fyne.Window) {
	w2, created := iw.u.GetOrCreateWindow(fmt.Sprintf("infowindow-zoom-%s-%d", title, id), title)
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

func (iw *InfoViewer) openURL(s string) {
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

func (iw *InfoViewer) makeZKillboardIcon(id int64, v Kind) *xwidget.TappableIcon {
	m := map[Kind]string{
		Alliance:    "alliance",
		Character:   "character",
		Corporation: "corporation",
		Region:      "region",
		SolarSystem: "system",
	}
	var f func()
	var title string
	partial, ok := m[v]
	if ok {
		f = func() {
			iw.openURL(fmt.Sprintf("https://zkillboard.com/%s/%d/", partial, id))
		}
		title = fmt.Sprintf("Show %s on zKillboard.com", strings.ToLower(v.String()))
	}
	icon := xwidget.NewTappableIcon(icons.ZkillboardPng, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoViewer) makeDotlanIcon(id int64, v Kind) *xwidget.TappableIcon {
	m := map[Kind]string{
		Alliance:    "alliance",
		Corporation: "corp",
		Region:      "region",
		SolarSystem: "system",
	}
	var f func()
	var title string
	partial, ok := m[v]
	if ok {
		f = func() {
			iw.openURL(fmt.Sprintf("https://evemaps.dotlan.net/%s/%d", partial, id))
		}
		title = fmt.Sprintf("Show %s on evemaps.dotlan.net", strings.ToLower(v.String()))
	}
	icon := xwidget.NewTappableIcon(icons.DotlanAvatarPng, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoViewer) makeEveWhoIcon(id int64, v Kind) *xwidget.TappableIcon {
	m := map[Kind]string{
		Alliance:    "alliance",
		Corporation: "corporation",
		Character:   "character",
	}
	var f func()
	var title string
	partial, ok := m[v]
	if ok {
		f = func() {
			iw.openURL(fmt.Sprintf("https://evewho.com/%s/%d", partial, id))
		}
		title = fmt.Sprintf("Show %s on evewho.com", strings.ToLower(v.String()))
	}
	icon := xwidget.NewTappableIcon(icons.Characterplaceholder32Jpeg, f)
	if title != "" {
		icon.SetToolTip(title)
	}
	return icon
}

func (iw *InfoViewer) renderIconSize() fyne.Size {
	var s float32
	if iw.u.IsMobile() {
		s = logoUnitSize
	} else {
		s = renderIconUnitSize
	}
	return fyne.NewSquareSize(s)
}

// baseInfo represents shared functionality between all info widgets.
type baseInfo struct {
	name *widget.Label
	iw   *InfoViewer
}

func (b *baseInfo) initBase(iw *InfoViewer) {
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
	iw      *InfoViewer
	openURL func(*url.URL) error
}

func newAttributeList(iw *InfoViewer, items ...attributeItem) *attributeList {
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
			case *app.EveFaction:
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
			case *eveBloodlineShort:
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
			case *app.EveType:
				if x == nil {
					s = "?"
					break
				}
				s = x.Name
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
						w.iw.Show(x)
					}
				}
			case *app.EveLocation:
				if x != nil {
					f = func() {
						w.iw.show(Location, x.ID)
					}
				}
			case *eveBloodlineShort:
				if x != nil {
					f = func() {
						w.iw.show(Bloodline, x.ID)
					}
				}

			case *app.EveFaction:
				if x != nil {
					f = func() {
						w.iw.show(Faction, x.ID)
					}
				}
			case *app.EveRace:
				if x != nil {
					f = func() {
						w.iw.show(Race, x.ID)
					}
				}
			case *app.EveType:
				if x != nil {
					f = func() {
						w.iw.show(Type, x.ID)
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
	infoVariant  Kind
}

func newEntityItem(id int64, category, text string, v Kind) entityItem {
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
		infoVariant: Undefined,
	}
}

func newEntityItemFromEveSolarSystem(o *app.EveSolarSystem) entityItem {
	ee := o.ToEveEntity()
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
	showInfo func(Kind, int64)
}

func entityItemsFromEveEntities(ee []*app.EveEntity) []entityItem {
	items := xslices.Map(ee, func(ee *app.EveEntity) entityItem {
		return newEntityItemFromEveEntityWithText(ee, "")
	})
	return items
}

func newEntityList(show func(Kind, int64)) *entityList {
	var items []entityItem
	return newEntityListFromItems(show, items...)
}

func newEntityListFromItems(show func(Kind, int64), items ...entityItem) *entityList {
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
			if it.infoVariant == Undefined {
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
	l.OnSelected = func(_ widget.ListItemID) {
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
