package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	infoWindowHeight    = 600
	infoWindowWidth     = 600
	logoUnitSize        = 64
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	zoomImagePixelSize  = 512
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

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
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

type InfoWindowParams struct {
	cs       CS
	eis      EIS
	eus      EUS
	isMobile bool
	js       *janiceservice.JaniceService
	scs      SCS
	settings Settings
	u        UIService
	w        fyne.Window
}

// NewInfoWindow returns a configured InfoWindow.
func NewInfoWindow(arg InfoWindowParams) *InfoWindow {
	iw := &InfoWindow{
		cs:       arg.cs,
		eis:      arg.eis,
		eus:      arg.eus,
		isMobile: arg.isMobile,
		js:       arg.js,
		scs:      arg.scs,
		settings: arg.settings,
		u:        arg.u,
		w:        arg.w,
	}
	return iw
}

// ShowEveEntity shows a new info window for an EveEntity.
func (iw *InfoWindow) ShowEveEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), int64(ee.ID))
}

// Show shows a new info window for an EveEntity.
func (iw *InfoWindow) Show(c app.EveEntityCategory, id int64) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), int64(id))
}

func (iw *InfoWindow) ShowLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw *InfoWindow) ShowRace(id int64) {
	iw.show(infoRace, int64(id))
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
		page = newAllianceInfo(iw, int64(entityID))
	case infoCharacter:
		title = "Character"
		page = newCharacterInfo(iw, int64(entityID))
	case infoConstellation:
		title = "Constellation"
		page = newConstellationInfo(iw, int64(entityID))
	case infoCorporation:
		title = "Corporation"
		page = newCorporationInfo(iw, int64(entityID))
	case infoInventoryType:
		x := newInventoryTypeInfo(iw, int64(entityID), characterID)
		x.setTitle = func(s string) { ab.SetTitle(makeAppBarTitle(s)) }
		page = x
		title = "Item"
	case infoRace:
		title = "Race"
		page = newRaceInfo(iw, int64(entityID))
	case infoRegion:
		title = "Region"
		page = newRegionInfo(iw, int64(entityID))
	case infoSolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, int64(entityID))
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

func (iw *InfoWindow) showZoomWindow(title string, id int64, load loadFuncAsync, w fyne.Window) {
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

func infoWindowSupportedEveEntities() set.Set[app.EveEntityCategory] {
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

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget
	baseInfo

	attributes *attributeList
	hq         *widget.Hyperlink
	id         int64
	logo       *canvas.Image
	members    *entityList
	tabs       *container.AppTabs
}

func newAllianceInfo(iw *InfoWindow, id int64) *allianceInfo {
	hq := widget.NewHyperlink("", nil)
	hq.Wrapping = fyne.TextWrapWord
	a := &allianceInfo{
		id:   id,
		logo: makeInfoLogo(),
		hq:   hq,
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.members = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Attributes", a.attributes),
		container.NewTabItem("Members", a.members),
	)
	return a
}

func (a *allianceInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	top := container.NewBorder(
		nil,
		nil,
		container.New(
			layout.NewCustomPaddedVBoxLayout(2*p),
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoAlliance),
				a.iw.makeDotlanIcon(a.id, infoAlliance),
				a.iw.makeEveWhoIcon(a.id, infoAlliance),
				layout.NewSpacer(),
			),
		),
		nil,
		a.name,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *allianceInfo) update(ctx context.Context) error {
	fyne.Do(func() {
		a.iw.eis.AllianceLogoAsync(a.id, app.IconPixelSize, func(r fyne.Resource) {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	})
	g := new(errgroup.Group)
	g.Go(func() error {
		o, err := a.iw.eus.FetchAlliance(ctx, a.id)
		if err != nil {
			return err
		}
		// Attributes
		var attributes []attributeItem
		if v, ok := o.ExecutorCorporation.Value(); ok {
			attributes = append(attributes, newAttributeItem("Executor", v))
		}
		if o.Ticker != "" {
			attributes = append(attributes, newAttributeItem("Short Name", o.Ticker))
		}
		if o.CreatorCorporation != nil {
			attributes = append(attributes, newAttributeItem("Created By Corporation", o.CreatorCorporation))
		}
		if o.Creator != nil {
			attributes = append(attributes, newAttributeItem("Created By", o.Creator))
		}
		if !o.DateFounded.IsZero() {
			attributes = append(attributes, newAttributeItem("Start Date", o.DateFounded))
		}
		if v, ok := o.Faction.Value(); ok {
			attributes = append(attributes, newAttributeItem("Faction", v))
		}
		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", o.ID)
			x.Action = func(_ any) {
				fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
			}
			attributes = append(attributes, x)
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.attributes.set(attributes)
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		members, err := a.iw.eus.FetchAllianceCorporations(ctx, a.id)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return nil
		}
		fyne.Do(func() {
			a.members.set(entityItemsFromEveEntities(members)...)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}

// characterInfo shows public information about a character.
type characterInfo struct {
	widget.BaseWidget
	baseInfo

	alliance        *widget.Hyperlink
	attributes      *attributeList
	bio             *widget.Label
	corporation     *widget.Hyperlink
	corporationLogo *canvas.Image
	description     *widget.Label
	employeeHistory *entityList
	id              int64
	isOwned         bool
	membership      *widget.Label
	ownedIcon       *ttwidget.Icon
	portrait        *iwidget.TappableImage
	security        *widget.Label
	tabs            *container.AppTabs
	title           *widget.Label
}

func newCharacterInfo(iw *InfoWindow, id int64) *characterInfo {
	alliance := widget.NewHyperlink("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	corporation := widget.NewHyperlink("", nil)
	corporation.Wrapping = fyne.TextWrapWord
	portrait := iwidget.NewTappableImage(icons.BlankSvg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(iw.renderIconSize())
	title := widget.NewLabel("")
	title.Wrapping = fyne.TextWrapWord
	bio := widget.NewLabel("")
	bio.Wrapping = fyne.TextWrapWord
	ownedIcon := ttwidget.NewIcon(theme.NewSuccessThemedResource(icons.CheckDecagramSvg))
	ownedIcon.SetToolTip("You own this character")
	a := &characterInfo{
		alliance:        alliance,
		bio:             bio,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:     newLabelWithWrapAndSelectable(""),
		id:              id,
		isOwned:         iw.scs.ListCharacterIDs().Contains(id),
		membership:      widget.NewLabel(""),
		ownedIcon:       ownedIcon,
		portrait:        portrait,
		security:        widget.NewLabel(""),
		title:           title,
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.employeeHistory = newEntityListFromItems(a.iw.show)
	attributes := container.NewTabItem("Attributes", a.attributes)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Bio", container.NewVScroll(a.bio)),
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		attributes,
	)
	ee := app.EveEntity{ID: id, Category: app.EveEntityCharacter}
	if !ee.IsNPC().ValueOrZero() {
		a.tabs.Append(container.NewTabItem("Employment History", a.employeeHistory))
	}
	a.tabs.Select(attributes)
	if !a.isOwned {
		a.ownedIcon.Hide()
	}
	return a
}

func (a *characterInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			nil,
			container.NewPadded(a.ownedIcon),
			container.NewVBox(
				container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
					a.name,
					a.title,
				),
				container.NewBorder(
					nil,
					nil,
					a.corporationLogo,
					nil,
					container.New(
						layout.NewCustomPaddedVBoxLayout(-2*p),
						container.NewBorder(
							nil,
							nil,
							container.New(layout.NewCustomPaddedLayout(0, 0, 0, -3*p), widget.NewLabel("Member of")),
							nil,
							a.corporation,
						),
						a.membership,
					),
				),
			),
		),
		widget.NewSeparator(),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.alliance,
			a.security,
		),
	)
	forums := iwidget.NewTappableIcon(icons.EvelogoPng, func() {
		ec, err := a.iw.eus.GetCharacterESI(context.Background(), a.id)
		if err != nil {
			a.iw.sb.Show("Failed to get character for forum: " + a.iw.u.HumanizeError(err))
			return
		}
		name := strings.ReplaceAll(ec.Name, " ", "_")
		a.iw.openURL(fmt.Sprintf("https://forums.eveonline.com/u/%s/summary", name))
	})
	forums.SetToolTip("Show on forums.eveonline.com")
	top := container.NewBorder(
		nil,
		nil,
		container.New(
			layout.NewCustomPaddedVBoxLayout(2*p),
			a.portrait,
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoCharacter),
				a.iw.makeEveWhoIcon(a.id, infoCharacter),
				forums,
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *characterInfo) update(ctx context.Context) error {
	fyne.Do(func() {
		a.iw.eis.CharacterPortraitAsync(a.id, 256, func(r fyne.Resource) {
			a.portrait.SetResource(r)
		})
	})
	o, _, err := a.iw.eus.GetOrCreateCharacterESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.iw.eis.CorporationLogoAsync(o.Corporation.ID, app.IconPixelSize, func(r fyne.Resource) {
			a.corporationLogo.Resource = r
			a.corporationLogo.Refresh()
		})
	})

	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.security.SetText(fmt.Sprintf("Security Status: %s", o.SecurityStatus.StringFunc("?", func(v float64) string {
			return fmt.Sprintf("%.1f", v)
		})))
		a.corporation.SetText(o.Corporation.Name)
		a.corporation.OnTapped = func() {
			a.iw.ShowEveEntity(o.Corporation)
		}
		a.portrait.OnTapped = func() {
			a.iw.showZoomWindow(o.Name, a.id, a.iw.eis.CharacterPortraitAsync, a.iw.w)
		}
	})
	fyne.Do(func() {
		a.bio.SetText(o.DescriptionPlain())
		a.description.SetText(o.Race.Description)
		a.tabs.Refresh()
	})
	fyne.Do(func() {
		v, ok := o.Alliance.Value()
		if !ok {
			a.alliance.Hide()
			return
		}
		a.alliance.SetText(v.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(v)
		}
	})
	fyne.Do(func() {
		v, ok := o.Title.Value()
		if !ok {
			a.title.Hide()
			return
		}
		a.title.SetText("Title: " + v)
	})
	attributes, err := a.makeAttributes(o)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.attributes.set(attributes)
		a.tabs.Refresh()
	})

	history, err := a.iw.eus.FetchCharacterCorporationHistory(ctx, a.id)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		fyne.Do(func() {
			a.membership.Hide()
		})
		return nil
	}
	items := xslices.Map(history, historyItem2EntityItem)
	duration := humanize.RelTime(history[0].StartDate, time.Now(), "", "")
	fyne.Do(func() {
		a.employeeHistory.set(items...)
		a.membership.SetText(fmt.Sprintf("for %s", duration))
		a.tabs.Refresh()
	})
	return nil
}

func (a *characterInfo) makeAttributes(o *app.EveCharacter) ([]attributeItem, error) {
	attributes := []attributeItem{
		newAttributeItem("Born", o.Birthday.Format(app.DateTimeFormat)),
		newAttributeItem("Race", o.Race),
		newAttributeItem("Security Status", o.SecurityStatus.StringFunc("?", func(v float64) string {
			return fmt.Sprintf("%.1f", v)
		})),
		newAttributeItem("Corporation", o.Corporation),
	}
	if v, ok := o.Alliance.Value(); ok {
		attributes = append(attributes, newAttributeItem("Alliance", v))
	}
	if v, ok := o.Alliance.Value(); ok {
		attributes = append(attributes, newAttributeItem("Faction", v))
	}
	var u any
	if v, ok := o.EveEntity().IsNPC().Value(); ok {
		u = v
	} else {
		u = "?"
	}
	attributes = append(attributes, newAttributeItem("NPC", u))
	if a.isOwned {
		c, err := a.iw.cs.GetCharacter(context.Background(), a.id)
		if err != nil {
			return nil, err
		}
		if v, ok := c.Home.Value(); ok {
			attributes = append(attributes, newAttributeItem("Home", v))
		}
		if v, ok := c.Location.Value(); ok {
			attributes = append(attributes, newAttributeItem("Location", v))
		}
		if v, ok := c.LastLoginAt.Value(); ok {
			attributes = append(attributes, newAttributeItem("Last Login", v))
		}
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	return attributes, nil
}

type constellationInfo struct {
	widget.BaseWidget
	baseInfo

	id      int64
	logo    *canvas.Image
	region  *widget.Hyperlink
	systems *entityList
	tabs    *container.AppTabs
}

func newConstellationInfo(iw *InfoWindow, id int64) *constellationInfo {
	region := widget.NewHyperlink("", nil)
	region.Wrapping = fyne.TextWrapWord
	a := &constellationInfo{

		id:     id,
		logo:   makeInfoLogo(),
		region: region,
		tabs:   container.NewAppTabs(),
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.logo.Resource = icons.Constellation64Png
	a.systems = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Solar Systems", a.systems),
	)
	return a
}

func (a *constellationInfo) CreateRenderer() fyne.WidgetRenderer {
	columns := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(columns, widget.NewLabel("Region"), a.region),
		))
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *constellationInfo) update(ctx context.Context) error {
	o, err := a.iw.eus.GetOrCreateConstellationESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.region.SetText(o.Region.Name)
		a.region.OnTapped = func() {
			a.iw.ShowEveEntity(o.Region.EveEntity())
		}

		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			a.tabs.Append(attributesTab)
			a.tabs.Refresh()
		}
	})
	oo, err := a.iw.eus.GetConstellationSolarSystemsESI(ctx, o.ID)
	if err != nil {
		return err
	}
	xx := xslices.Map(oo, newEntityItemFromEveSolarSystem)
	fyne.Do(func() {
		a.systems.set(xx...)
		a.tabs.Refresh()
	})
	return nil
}

// corporationInfo shows public information about a character.
type corporationInfo struct {
	widget.BaseWidget
	baseInfo

	alliance        *widget.Hyperlink
	allianceHistory *entityList
	allianceLogo    *canvas.Image
	allianceBox     fyne.CanvasObject
	attributes      *attributeList
	description     *widget.Label
	hq              *widget.Hyperlink
	id              int64
	logo            *canvas.Image
	tabs            *container.AppTabs
}

func newCorporationInfo(iw *InfoWindow, id int64) *corporationInfo {
	alliance := widget.NewHyperlink("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	hq := widget.NewHyperlink("", nil)
	hq.Wrapping = fyne.TextWrapWord
	a := &corporationInfo{
		alliance:     alliance,
		allianceLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:  newLabelWithWrapAndSelectable(""),
		hq:           hq,
		id:           id,
		logo:         makeInfoLogo(),
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.allianceHistory = newEntityListFromItems(a.iw.show)
	attributes := container.NewTabItem("Attributes", a.attributes)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		attributes,
	)
	ee := app.EveEntity{ID: id, Category: app.EveEntityCorporation}
	if !ee.IsNPC().ValueOrZero() {
		a.tabs.Append(container.NewTabItem("Alliance History", a.allianceHistory))
	}
	a.tabs.Select(attributes)
	p := theme.Padding()
	a.allianceBox = container.NewBorder(
		nil,
		nil,
		container.NewHBox(a.allianceLogo, container.New(layout.NewCustomPaddedLayout(0, 0, 0, -3*p), widget.NewLabel("Member of"))),
		nil,
		a.alliance,
	)
	return a
}

func (a *corporationInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.hq,
		),
		a.allianceBox,
	)
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoCorporation),
				a.iw.makeDotlanIcon(a.id, infoCorporation),
				a.iw.makeEveWhoIcon(a.id, infoCorporation),
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationInfo) update(ctx context.Context) error {
	fyne.Do(func() {
		a.iw.eis.CorporationLogoAsync(a.id, app.IconPixelSize, func(r fyne.Resource) {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	})
	o, err := a.iw.eus.GetOrCreateCorporationESI(ctx, a.id)
	if err != nil {
		return err
	}
	attributes := a.makeAttributes(o)
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.DescriptionPlain())
		a.attributes.set(attributes)
		a.tabs.Refresh()
	})
	fyne.Do(func() {
		v, ok := o.Alliance.Value()
		if !ok {
			a.allianceBox.Hide()
			return
		}
		a.alliance.SetText(v.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(v)
		}
		a.iw.eis.AllianceLogoAsync(v.ID, app.IconPixelSize, func(r fyne.Resource) {
			a.allianceLogo.Resource = r
			a.allianceLogo.Refresh()
		})
	})
	fyne.Do(func() {
		v, ok := o.HomeStation.Value()
		if !ok {
			a.hq.Hide()
			return
		}
		a.hq.SetText("Headquarters: " + v.Name)
		a.hq.OnTapped = func() {
			a.iw.ShowEveEntity(v)
		}
	})
	g := new(errgroup.Group)
	g.Go(func() error {
		history, err := a.iw.eus.FetchCorporationAllianceHistory(ctx, a.id)
		if err != nil {
			return err
		}
		var items []entityItem
		if len(history) > 0 {
			history2 := xslices.Filter(history, func(v app.MembershipHistoryItem) bool {
				return v.Organization != nil && v.Organization.Category.IsKnown()
			})
			items = append(items, xslices.Map(history2, historyItem2EntityItem)...)
		}
		var founded string
		if v, ok := o.DateFounded.Value(); ok {
			founded = fmt.Sprintf("**%s**", v.Format(app.DateFormat))
		} else {
			founded = "?"
		}
		items = append(items, newEntityItem(0, "Corporation Founded", founded, infoNotSupported))
		fyne.Do(func() {
			a.allianceHistory.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}

func (a *corporationInfo) makeAttributes(o *app.EveCorporation) []attributeItem {
	var attributes []attributeItem
	if v, ok := o.Ceo.Value(); ok {
		attributes = append(attributes, newAttributeItem("CEO", v))
	}
	if v, ok := o.Creator.Value(); ok {
		attributes = append(attributes, newAttributeItem("Founder", v))
	}
	if v, ok := o.Alliance.Value(); ok {
		attributes = append(attributes, newAttributeItem("Alliance", v))
	}
	if o.Ticker != "" {
		attributes = append(attributes, newAttributeItem("Ticker Name", o.Ticker))
	}
	if v, ok := o.Faction.Value(); ok {
		attributes = append(attributes, newAttributeItem("Faction", v))
	}
	var u any
	if v, ok := o.EveEntity().IsNPC().Value(); ok {
		u = v
	} else {
		u = "?"
	}
	attributes = append(attributes, newAttributeItem("NPC", u))
	if v, ok := o.Shares.Value(); ok {
		attributes = append(attributes, newAttributeItem("Shares", v))
	}
	attributes = append(attributes, newAttributeItem("Member Count", o.MemberCount))
	attributes = append(attributes, newAttributeItem("ISK Tax Rate", fmt.Sprintf("%.1f %%", o.TaxRate*100)))
	attributes = append(attributes, newAttributeItem("War Eligibility", o.WarEligible))
	if v, ok := o.URL.Value(); ok {
		if u, err := url.ParseRequestURI(v); err == nil && u.Host != "" {
			attributes = append(attributes, newAttributeItem("URL", u))
		}
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	return attributes
}

// locationInfo shows public information about a character.
type locationInfo struct {
	widget.BaseWidget
	baseInfo

	description *widget.Label
	id          int64
	location    *entityList
	owner       *widget.Hyperlink
	ownerLogo   *canvas.Image
	services    *entityList
	tabs        *container.AppTabs
	typeImage   *iwidget.TappableImage
	typeInfo    *widget.Hyperlink
}

func newLocationInfo(iw *InfoWindow, id int64) *locationInfo {
	typeInfo := widget.NewHyperlink("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := widget.NewHyperlink("", nil)
	owner.Wrapping = fyne.TextWrapWord
	typeImage := iwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(iw.renderIconSize())
	a := &locationInfo{
		description: newLabelWithWrapAndSelectable(""),
		id:          id,
		owner:       owner,
		ownerLogo:   iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		typeImage:   typeImage,
		typeInfo:    typeInfo,
	}
	a.ExtendBaseWidget(a)
	a.initBase(iw)
	a.location = newEntityList(a.iw.show)
	location := container.NewTabItem("Location", a.location)
	a.services = newEntityList(a.iw.show)
	services := container.NewTabItem("Services", a.services)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		location,
		services,
	)
	a.tabs.Select(location)
	return a
}

func (a *locationInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.typeInfo,
		),
		container.NewBorder(
			nil,
			nil,
			a.ownerLogo,
			nil,
			a.owner,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.typeImage), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *locationInfo) update(ctx context.Context) error {
	o, err := a.iw.eus.GetOrCreateLocationESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
	})
	if et, ok := o.Type.Value(); ok {
		fyne.Do(func() {
			a.iw.eis.InventoryTypeRenderAsync(et.ID, renderIconPixelSize, func(r fyne.Resource) {
				a.typeImage.SetResource(r)
			})
			a.typeInfo.SetText(et.Name)
			a.typeInfo.OnTapped = func() {
				a.iw.ShowEveEntity(et.EveEntity())
			}
			a.typeImage.OnTapped = func() {
				a.iw.showZoomWindow(o.Name, et.ID, a.iw.eis.InventoryTypeRenderAsync, a.iw.w)
			}
			description := et.Description
			if description == "" {
				description = et.Name
			}
			a.description.SetText(description)
		})
	}
	if v, ok := o.Owner.Value(); ok {
		fyne.Do(func() {
			a.iw.eis.CorporationLogoAsync(v.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.ownerLogo.Resource = r
				a.ownerLogo.Refresh()
			})
			a.owner.SetText(v.Name)
			a.owner.OnTapped = func() {
				a.iw.ShowEveEntity(v)
			}
		})
	}
	if es, ok := o.SolarSystem.Value(); ok {
		fyne.Do(func() {
			a.location.set(
				newEntityItemFromEveEntityWithText(es.Constellation.Region.EveEntity(), ""),
				newEntityItemFromEveEntityWithText(es.Constellation.EveEntity(), ""),
				newEntityItemFromEveSolarSystem(es),
			)
		})
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
		})
	}
	fyne.Do(func() {
		a.tabs.Refresh()
	})
	g := new(errgroup.Group)
	g.Go(func() error {
		if o.Variant() != app.EveLocationStation {
			return nil
		}
		ss, err := a.iw.eus.GetStationServicesESI(ctx, int64(a.id))
		if err != nil {
			return err
		}
		items := xslices.Map(ss, func(s string) entityItem {
			s2 := strings.ReplaceAll(s, "-", " ")
			name := xstrings.Title(s2)
			return newEntityItem(0, "Service", name, infoNotSupported)
		})
		fyne.Do(func() {
			a.services.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

type raceInfo struct {
	widget.BaseWidget
	baseInfo

	id          int64
	logo        *canvas.Image
	tabs        *container.AppTabs
	description *widget.Label
}

func newRaceInfo(iw *InfoWindow, id int64) *raceInfo {
	a := &raceInfo{
		description: newLabelWithWrapAndSelectable(""),
		id:          id,
		logo:        makeInfoLogo(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.BlankSvg
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
	)
	return a
}

func (a *raceInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *raceInfo) update(ctx context.Context) error {
	o, err := a.iw.eus.GetOrCreateRaceESI(ctx, a.id)
	if err != nil {
		return err
	}
	if factionID, ok := o.FactionID(); ok {
		fyne.Do(func() {
			a.iw.eis.FactionLogoAsync(factionID, app.IconPixelSize, func(r fyne.Resource) {
				a.logo.Resource = r
				a.logo.Refresh()
			})
		})
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			fyne.Do(func() {
				a.tabs.Append(attributesTab)
			})
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.description.SetText(o.Description)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}

type regionInfo struct {
	widget.BaseWidget
	baseInfo

	description    *widget.Label
	constellations *entityList
	id             int64
	logo           *canvas.Image
	tabs           *container.AppTabs
}

func newRegionInfo(iw *InfoWindow, id int64) *regionInfo {
	a := &regionInfo{
		id:          id,
		description: newLabelWithWrapAndSelectable(""),
		logo:        makeInfoLogo(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.Region64Png
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.constellations = newEntityList(a.iw.show)
	constellations := container.NewTabItem("Constellations", a.constellations)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		constellations,
	)
	a.tabs.Select(constellations)
	return a
}

func (a *regionInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
	)
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoRegion),
				a.iw.makeDotlanIcon(a.id, infoRegion),
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *regionInfo) update(ctx context.Context) error {
	o, err := a.iw.eus.GetOrCreateRegionESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if !a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			fyne.Do(func() {
				a.tabs.Append(attributesTab)
			})
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.description.SetText(o.DescriptionPlain())
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		oo, err := a.iw.eus.GetRegionConstellationsESI(ctx, o.ID)
		if err != nil {
			return err
		}
		items := xslices.Map(oo, newEntityItemFromEveEntity)
		fyne.Do(func() {
			a.constellations.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	return nil
}

type solarSystemInfo struct {
	widget.BaseWidget
	baseInfo

	constellation *widget.Hyperlink
	id            int64
	logo          *canvas.Image
	planets       *entityList
	region        *widget.Hyperlink
	security      *widget.Label
	stargates     *entityList
	stations      *entityList
	structures    *entityList
	tabs          *container.AppTabs
}

func newSolarSystemInfo(iw *InfoWindow, id int64) *solarSystemInfo {
	region := widget.NewHyperlink("", nil)
	region.Wrapping = fyne.TextWrapWord
	constellation := widget.NewHyperlink("", nil)
	constellation.Wrapping = fyne.TextWrapWord
	a := &solarSystemInfo{
		id:            id,
		region:        region,
		constellation: constellation,
		logo:          makeInfoLogo(),
		security:      widget.NewLabel(""),
		tabs:          container.NewAppTabs(),
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.planets = newEntityList(a.iw.show)
	a.stargates = newEntityList(a.iw.show)
	a.stations = newEntityList(a.iw.show)
	a.structures = newEntityList(a.iw.show)
	note := widget.NewLabel("Only contains structures known through characters")
	note.Importance = widget.LowImportance
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Stargates", a.stargates),
		container.NewTabItem("Planets", a.planets),
		container.NewTabItem("Stations", a.stations),
		container.NewTabItem("Structures", container.NewBorder(
			nil,
			note,
			nil,
			nil,
			a.structures,
		)),
	)
	return a
}

func (a *solarSystemInfo) CreateRenderer() fyne.WidgetRenderer {
	columns := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Solar System"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(columns, widget.NewLabel("Region"), a.region),
			container.New(columns, widget.NewLabel("Constellation"), a.constellation),
			container.New(columns, widget.NewLabel("Security"), a.security),
		),
	)
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoSolarSystem),
				a.iw.makeDotlanIcon(a.id, infoSolarSystem),
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *solarSystemInfo) update(ctx context.Context) error {
	o, err := a.iw.eus.GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	starID, planets, stargateIDs, stations, structures, err := a.iw.eus.GetSolarSystemInfoESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(a.id))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			fyne.Do(func() {
				a.tabs.Append(attributesTab)
			})
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.region.SetText(o.Constellation.Region.Name)
			a.region.OnTapped = func() {
				a.iw.ShowEveEntity(o.Constellation.Region.EveEntity())
			}
			a.constellation.SetText(o.Constellation.Name)
			a.constellation.OnTapped = func() {
				a.iw.ShowEveEntity(o.Constellation.EveEntity())
			}
			a.security.Text = o.SecurityStatusDisplay()
			a.security.Importance = o.SecurityType().ToImportance()
			a.security.Refresh()
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		stationItems := entityItemsFromEveEntities(stations)
		structureItems := xslices.Map(structures, func(x *app.EveLocation) entityItem {
			return newEntityItem(
				x.ID,
				x.Name,
				"Structure",
				infoLocation,
			)
		})
		fyne.Do(func() {
			a.stations.set(stationItems...)
			a.structures.set(structureItems...)
			a.tabs.Refresh()
		})
		return nil
	})
	if v, ok := starID.Value(); ok {
		g.Go(func() error {
			id, err := a.iw.eus.GetStarTypeID(ctx, v)
			if err != nil {
				return err
			}
			fyne.Do(func() {
				a.iw.eis.InventoryTypeIconAsync(id, app.IconPixelSize, func(r fyne.Resource) {
					a.logo.Resource = r
					a.logo.Refresh()
				})
			})
			return nil
		})
	}
	g.Go(func() error {
		ss, err := a.iw.eus.GetStargatesSolarSystemsESI(ctx, stargateIDs)
		if err != nil {
			return err
		}
		items := xslices.Map(ss, newEntityItemFromEveSolarSystem)
		fyne.Do(func() {
			a.stargates.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		pp, err := a.iw.eus.GetSolarSystemPlanets(ctx, planets)
		if err != nil {
			return err
		}
		items := xslices.Map(pp, newEntityItemFromEvePlanet)
		fyne.Do(func() {
			a.planets.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}

// inventoryTypeInfo displays information about Eve Online inventory types.
type inventoryTypeInfo struct {
	widget.BaseWidget
	baseInfo

	characterIcon    *canvas.Image
	characterID      int64
	characterName    *widget.Hyperlink
	checkIcon        *widget.Icon
	description      *widget.Label
	eveMarketBrowser *fyne.Container
	janice           *fyne.Container
	setTitle         func(string) // for setting the title during update
	tabs             *container.AppTabs
	typeIcon         *iwidget.TappableImage
	typeID           int64
}

func newInventoryTypeInfo(iw *InfoWindow, typeID, characterID int64) *inventoryTypeInfo {
	typeIcon := iwidget.NewTappableImage(icons.BlankSvg, nil)
	typeIcon.SetFillMode(canvas.ImageFillContain)
	typeIcon.SetMinSize(fyne.NewSquareSize(logoUnitSize))
	a := &inventoryTypeInfo{
		characterIcon: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		characterID:   characterID,
		checkIcon:     widget.NewIcon(icons.BlankSvg),
		description:   newLabelWithWrapAndSelectable(""),
		typeIcon:      typeIcon,
		typeID:        typeID,
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)

	a.checkIcon.Hide()
	a.characterIcon.Hide()
	a.characterName = widget.NewHyperlink("", nil)
	a.characterName.Wrapping = fyne.TextWrapWord
	a.characterName.Hide()

	emb := iwidget.NewTappableIcon(icons.EvemarketbrowserJpg, func() {
		a.iw.openURL(fmt.Sprintf("https://evemarketbrowser.com/region/0/type/%d", a.typeID))
	})
	emb.SetToolTip("Show on evemarketbrowser.com")
	a.eveMarketBrowser = container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameButton)), emb)
	a.eveMarketBrowser.Hide()

	janice := iwidget.NewTappableIcon(icons.JanicePng, func() {
		a.iw.openURL(fmt.Sprintf("https://janice.e-351.com/i/%d", a.typeID))
	})
	janice.SetToolTip("Show on janice.e-351.com")
	a.janice = container.NewStack(canvas.NewRectangle(color.White), janice)
	a.janice.Hide()

	a.tabs = container.NewAppTabs(container.NewTabItem("Description", container.NewVScroll(a.description)))
	return a
}

func (a *inventoryTypeInfo) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.typeIcon),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*theme.Padding()),
				layout.NewSpacer(),
				a.eveMarketBrowser,
				a.janice,
				layout.NewSpacer(),
			),
		),
		nil,
		container.NewVBox(
			a.name,
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(a.checkIcon, a.characterIcon),
				nil,
				a.characterName,
			)),
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *inventoryTypeInfo) update(ctx context.Context) error {
	et, err := a.iw.eus.GetOrCreateTypeESI(ctx, a.typeID)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(et.Name)
		a.setTitle(et.Group.Name)
		if et.IsTradeable() {
			a.eveMarketBrowser.Show()
			a.janice.Show()
		}
		s := et.DescriptionPlain()
		if s == "" {
			s = et.Name
		}
		a.description.SetText(s)
	})
	fyne.Do(func() {
		if et.IsSKIN() {
			a.iw.eis.InventoryTypeSKINAsync(et.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.typeIcon.SetResource(r)
			})
		} else if et.IsBlueprint() {
			a.iw.eis.InventoryTypeBPOAsync(et.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.typeIcon.SetResource(r)
			})
		} else {
			a.iw.eis.InventoryTypeIconAsync(et.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.typeIcon.SetResource(r)
			})
		}
	})
	if et.HasRender() {
		a.typeIcon.OnTapped = func() {
			a.iw.showZoomWindow(et.Name, a.typeID, a.iw.eis.InventoryTypeRenderAsync, a.iw.w)
		}
	}

	var character *app.EveEntity
	if a.characterID != 0 {
		ee, err := a.iw.eus.GetOrCreateEntityESI(ctx, a.characterID)
		if err != nil {
			return err
		}
		character = ee
		fyne.Do(func() {
			a.iw.eis.CharacterPortraitAsync(character.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.characterIcon.Resource = r
				a.characterIcon.Refresh()
			})
			a.characterIcon.Show()
			a.characterName.OnTapped = func() {
				a.iw.ShowEveEntity(character)
			}
			a.characterName.SetText(character.Name)
			a.characterName.Show()
		})
	}

	oo, err := a.iw.eus.ListTypeDogmaAttributesForType(ctx, et.ID)
	if err != nil {
		return err
	}
	dogmaAttributes := make(map[int64]*app.EveTypeDogmaAttribute)
	for _, o := range oo {
		dogmaAttributes[o.DogmaAttribute.ID] = o
	}

	var requiredSkills []requiredSkill
	if a.characterID != 0 {
		skills, err := a.calcRequiredSkills(ctx, a.characterID, dogmaAttributes)
		if err != nil {
			return err
		}
		requiredSkills = skills
	}
	hasRequiredSkills := true
	for _, o := range requiredSkills {
		if o.requiredLevel > o.activeLevel {
			hasRequiredSkills = false
			break
		}
	}
	if character != nil && character.IsCharacter() && len(requiredSkills) > 0 {
		fyne.Do(func() {
			a.checkIcon.SetResource(boolIconResource(hasRequiredSkills))
			a.checkIcon.Show()
		})
	}

	// tabs
	attributeTab := a.makeAttributeTab(ctx, dogmaAttributes, et)
	if attributeTab != nil {
		fyne.Do(func() {
			a.tabs.Append(attributeTab)
		})
	}
	fittingTab := a.makeFittingTab(ctx, dogmaAttributes)
	if fittingTab != nil {
		fyne.Do(func() {
			a.tabs.Append(fittingTab)
		})
	}
	requirementsTab := a.makeRequirementsTab(requiredSkills)
	if requirementsTab != nil {
		fyne.Do(func() {
			a.tabs.Append(requirementsTab)
		})
	}
	marketTab := a.makeMarketTab(ctx, et)
	if marketTab != nil {
		fyne.Do(func() {
			a.tabs.Append(marketTab)
		})
	}

	// Set initial tab
	fyne.Do(func() {
		if marketTab != nil && a.iw.settings.PreferMarketTab() {
			a.tabs.Select(marketTab)
		} else if requirementsTab != nil && et.Group.Category.ID == app.EveCategorySkill {
			a.tabs.Select(requirementsTab)
		} else if attributeTab != nil &&
			set.Of[int64](
				app.EveCategoryDrone,
				app.EveCategoryFighter,
				app.EveCategoryOrbitals,
				app.EveCategoryShip,
				app.EveCategoryStructure,
			).Contains(et.Group.Category.ID) {
			a.tabs.Select(attributeTab)
		}
		a.tabs.Refresh()
	})
	return nil
}

func (a *inventoryTypeInfo) makeAttributeTab(ctx context.Context, dogmaAttributes map[int64]*app.EveTypeDogmaAttribute, et *app.EveType) *container.TabItem {
	attributes := a.calcAttributesData(ctx, et, dogmaAttributes)
	if len(attributes) == 0 {
		return nil
	}
	list := widget.NewList(
		func() int {
			return len(attributes)
		},
		func() fyne.CanvasObject {
			return newTypeAttributeItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(attributes) {
				return
			}
			r := attributes[id]
			item := co.(*typeAttributeItem)
			if r.isTitle {
				item.SetTitle(r.label)
			} else {
				item.SetRegular(r.icon, r.label, r.value)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(attributes) {
			return
		}
		r := attributes[id]
		if r.action != nil {
			r.action(r.value)
		}
	}
	return container.NewTabItem("Attributes", list)
}

// attributeGroup represents a group of dogma attributes.
//
// Used for rendering the attributes and fitting tabs for inventory type info
type attributeGroup string

func (ag attributeGroup) DisplayName() string {
	return xstrings.Title(string(ag))
}

const (
	attributeGroupArmor                 attributeGroup = "armor"
	attributeGroupCapacitor             attributeGroup = "capacitor"
	attributeGroupElectronicResistances attributeGroup = "electronic resistances"
	attributeGroupFitting               attributeGroup = "fitting"
	attributeGroupFighter               attributeGroup = "fighter squadron facilities"
	attributeGroupJumpDrive             attributeGroup = "jump drive systems"
	attributeGroupMiscellaneous         attributeGroup = "miscellaneous"
	attributeGroupPropulsion            attributeGroup = "propulsion"
	attributeGroupShield                attributeGroup = "shield"
	attributeGroupStructure             attributeGroup = "structure"
	attributeGroupTargeting             attributeGroup = "targeting"
)

// attribute groups to show in order on attributes tab
var attributeGroups = []attributeGroup{
	attributeGroupStructure,
	attributeGroupArmor,
	attributeGroupShield,
	attributeGroupElectronicResistances,
	attributeGroupCapacitor,
	attributeGroupTargeting,
	attributeGroupFighter,
	attributeGroupJumpDrive,
	attributeGroupPropulsion,
	attributeGroupMiscellaneous,
}

// assignment of attributes to groups
var attributeGroupsMap = map[attributeGroup][]int64{
	attributeGroupStructure: {
		app.EveDogmaAttributeStructureHitpoints,
		app.EveDogmaAttributeCapacity,
		app.EveDogmaAttributeDroneCapacity,
		app.EveDogmaAttributeDroneBandwidth,
		app.EveDogmaAttributeMass,
		app.EveDogmaAttributeInertiaModifier,
		app.EveDogmaAttributeStructureEMDamageResistance,
		app.EveDogmaAttributeStructureThermalDamageResistance,
		app.EveDogmaAttributeStructureKineticDamageResistance,
		app.EveDogmaAttributeStructureExplosiveDamageResistance,
	},
	attributeGroupArmor: {
		app.EveDogmaAttributeArmorHitpoints,
		app.EveDogmaAttributeArmorEMDamageResistance,
		app.EveDogmaAttributeArmorThermalDamageResistance,
		app.EveDogmaAttributeArmorKineticDamageResistance,
		app.EveDogmaAttributeArmorExplosiveDamageResistance,
	},
	attributeGroupShield: {
		app.EveDogmaAttributeShieldCapacity,
		app.EveDogmaAttributeShieldRechargeTime,
		app.EveDogmaAttributeShieldEMDamageResistance,
		app.EveDogmaAttributeShieldThermalDamageResistance,
		app.EveDogmaAttributeShieldKineticDamageResistance,
		app.EveDogmaAttributeShieldExplosiveDamageResistance,
	},
	attributeGroupElectronicResistances: {
		app.EveDogmaAttributeCargoScanResistance,
		app.EveDogmaAttributeCapacitorWarfareResistance,
		app.EveDogmaAttributeSensorWarfareResistance,
		app.EveDogmaAttributeWeaponDisruptionResistance,
		app.EveDogmaAttributeTargetPainterResistance,
		app.EveDogmaAttributeStasisWebifierResistance,
		app.EveDogmaAttributeRemoteLogisticsImpedance,
		app.EveDogmaAttributeRemoteElectronicAssistanceImpedance,
		app.EveDogmaAttributeECMResistance,
		app.EveDogmaAttributeCapacitorWarfareResistanceBonus,
		app.EveDogmaAttributeStasisWebifierResistanceBonus,
	},
	attributeGroupCapacitor: {
		app.EveDogmaAttributeCapacitorCapacity,
		app.EveDogmaAttributeCapacitorRechargeTime,
	},
	attributeGroupTargeting: {
		app.EveDogmaAttributeMaximumTargetingRange,
		app.EveDogmaAttributeMaximumLockedTargets,
		app.EveDogmaAttributeSignatureRadius,
		app.EveDogmaAttributeScanResolution,
		app.EveDogmaAttributeRADARSensorStrength,
		app.EveDogmaAttributeLadarSensorStrength,
		app.EveDogmaAttributeMagnetometricSensorStrength,
		app.EveDogmaAttributeGravimetricSensorStrength,
	},
	attributeGroupPropulsion: {
		app.EveDogmaAttributeMaxVelocity,
		app.EveDogmaAttributeShipWarpSpeed,
	},
	attributeGroupJumpDrive: {
		app.EveDogmaAttributeJumpDriveCapacitorNeed,
		app.EveDogmaAttributeMaximumJumpRange,
		app.EveDogmaAttributeJumpDriveFuelNeed,
		app.EveDogmaAttributeJumpDriveConsumptionAmount,
		app.EveDogmaAttributeFuelBayCapacity,
	},
	attributeGroupFighter: {
		app.EveDogmaAttributeFighterHangarCapacity,
		app.EveDogmaAttributeFighterSquadronLaunchTubes,
		app.EveDogmaAttributeLightFighterSquadronLimit,
		app.EveDogmaAttributeSupportFighterSquadronLimit,
		app.EveDogmaAttributeHeavyFighterSquadronLimit,
	},
	attributeGroupFitting: {
		app.EveDogmaAttributeCPUOutput,
		app.EveDogmaAttributeCPUusage,
		app.EveDogmaAttributePowergridOutput,
		app.EveDogmaAttributeCalibration,
		app.EveDogmaAttributeRigSlots,
		app.EveDogmaAttributeLauncherHardpoints,
		app.EveDogmaAttributeTurretHardpoints,
		app.EveDogmaAttributeHighSlots,
		app.EveDogmaAttributeMediumSlots,
		app.EveDogmaAttributeLowSlots,
		app.EveDogmaAttributeServiceSlots,
	},
	attributeGroupMiscellaneous: {
		app.EveDogmaAttributeImplantSlot,
		app.EveDogmaAttributeCharismaModifier,
		app.EveDogmaAttributeIntelligenceModifier,
		app.EveDogmaAttributeMemoryModifier,
		app.EveDogmaAttributePerceptionModifier,
		app.EveDogmaAttributeWillpowerModifier,
		app.EveDogmaAttributePrimaryAttribute,
		app.EveDogmaAttributeSecondaryAttribute,
		app.EveDogmaAttributeTrainingTimeMultiplier,
		app.EveDogmaAttributeTechLevel,
	},
}

type typeAttributeRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
	action  func(v string)
}

func (a *inventoryTypeInfo) calcAttributesData(ctx context.Context, et *app.EveType, attributes map[int64]*app.EveTypeDogmaAttribute) []typeAttributeRow {
	droneCapacity, ok := attributes[app.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	jumpDrive, ok := attributes[app.EveDogmaAttributeOnboardJumpDrive]
	hasJumpDrive := ok && jumpDrive.Value == 1.0

	groupedRows := make(map[attributeGroup][]typeAttributeRow)

	for _, ag := range attributeGroups {
		var attributeSelection []*app.EveTypeDogmaAttribute
		for _, da := range attributeGroupsMap[ag] {
			o, ok := attributes[da]
			if !ok {
				continue
			}
			if ag == attributeGroupElectronicResistances {
				s := attributeGroupsMap[ag]
				found := slices.Index(s, o.DogmaAttribute.ID) == -1
				if found && o.Value == 0 {
					continue
				}
			}
			switch o.DogmaAttribute.ID {
			case app.EveDogmaAttributeCapacity, app.EveDogmaAttributeMass:
				if o.Value == 0 {
					continue
				}
			case app.EveDogmaAttributeDroneCapacity,
				app.EveDogmaAttributeDroneBandwidth:
				if !hasDrones {
					continue
				}
			case app.EveDogmaAttributeMaximumJumpRange,
				app.EveDogmaAttributeJumpDriveFuelNeed:
				if !hasJumpDrive {
					continue
				}
			case app.EveDogmaAttributeSupportFighterSquadronLimit:
				if o.Value == 0 {
					continue
				}
			}
			attributeSelection = append(attributeSelection, o)
		}
		if len(attributeSelection) == 0 {
			continue
		}
		for _, o := range attributeSelection {
			value := o.Value
			switch o.DogmaAttribute.ID {
			case app.EveDogmaAttributeShipWarpSpeed:
				x := attributes[app.EveDogmaAttributeWarpSpeedMultiplier]
				value = value * x.Value
			}
			v, substituteIcon := a.iw.eus.FormatDogmaValue(ctx, value, o.DogmaAttribute.Unit)
			var iconID int64
			if substituteIcon != 0 {
				iconID = substituteIcon
			} else {
				iconID = o.DogmaAttribute.IconID.ValueOrZero()
			}
			r, _ := eveicon.FromID(iconID)
			groupedRows[ag] = append(groupedRows[ag], typeAttributeRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName.ValueOrZero(),
				value: v,
			})
		}
	}
	var rows []typeAttributeRow
	if v, ok := et.Volume.Value(); ok {
		value, _ := a.iw.eus.FormatDogmaValue(ctx, v, app.EveUnitVolume)
		if pv, ok := et.PackagedVolume.Value(); ok && !optional.Equal(et.Volume, et.PackagedVolume) {
			s, _ := a.iw.eus.FormatDogmaValue(ctx, pv, app.EveUnitVolume)
			value += fmt.Sprintf(" (%s Packaged)", s)
		}
		r := typeAttributeRow{
			icon:  eveicon.FromName(eveicon.Structure),
			label: "Volume",
			value: value,
		}
		var ag attributeGroup
		if len(groupedRows[attributeGroupStructure]) > 0 {
			ag = attributeGroupStructure
		} else {
			ag = attributeGroupMiscellaneous
		}
		groupedRows[ag] = append([]typeAttributeRow{r}, groupedRows[ag]...)
	}
	usedGroupsCount := 0
	for _, ag := range attributeGroups {
		if len(groupedRows[ag]) > 0 {
			usedGroupsCount++
		}
	}
	for _, ag := range attributeGroups {
		if len(groupedRows[ag]) > 0 {
			if usedGroupsCount > 1 {
				rows = append(rows, typeAttributeRow{label: ag.DisplayName(), isTitle: true})
			}
			rows = append(rows, groupedRows[ag]...)
		}
	}
	if a.iw.u.IsDeveloperMode() {
		rows = append(rows, typeAttributeRow{label: "Developer Mode", isTitle: true})
		rows = append(rows, typeAttributeRow{
			label: "EVE ID",
			value: fmt.Sprint(et.ID),
			action: func(v string) {
				fyne.CurrentApp().Clipboard().SetContent(v)
			},
		})
	}
	return rows
}

func (a *inventoryTypeInfo) makeFittingTab(ctx context.Context, dogmaAttributes map[int64]*app.EveTypeDogmaAttribute) *container.TabItem {
	fittingData := a.calcFittingData(ctx, dogmaAttributes)
	if len(fittingData) == 0 {
		return nil
	}
	list := widget.NewList(
		func() int {
			return len(fittingData)
		},
		func() fyne.CanvasObject {
			return newTypeAttributeItem()
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			r := fittingData[lii]
			item := co.(*typeAttributeItem)
			item.SetRegular(r.icon, r.label, r.value)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll()
	}
	return container.NewTabItem("Fittings", list)
}

func (a *inventoryTypeInfo) calcFittingData(ctx context.Context, dogmaAttributes map[int64]*app.EveTypeDogmaAttribute) []typeAttributeRow {
	var data []typeAttributeRow
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := dogmaAttributes[da]
		if !ok {
			continue
		}
		r, _ := eveicon.FromID(o.DogmaAttribute.IconID.ValueOrZero())
		v, _ := a.iw.eus.FormatDogmaValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, typeAttributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName.ValueOrZero(),
			value: v,
		})
	}
	return data
}

func (a *inventoryTypeInfo) makeRequirementsTab(requiredSkills []requiredSkill) *container.TabItem {
	if len(requiredSkills) == 0 {
		return nil
	}
	list := widget.NewList(
		func() int {
			return len(requiredSkills)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Placeholder"),
				layout.NewSpacer(),
				widget.NewLabel("Check"),
				awidget.NewSkillLevel(),
				widget.NewIcon(icons.QuestionmarkSvg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := requiredSkills[id]
			row := co.(*fyne.Container).Objects
			skill := row[0].(*widget.Label)
			text := row[2].(*widget.Label)
			level := row[3].(*awidget.SkillLevel)
			icon := row[4].(*widget.Icon)
			skill.SetText(app.SkillDisplayName(o.name, o.requiredLevel))
			if o.activeLevel == 0 && o.trainedLevel == 0 {
				text.Text = "Skill not injected"
				text.Importance = widget.DangerImportance
				text.Refresh()
				text.Show()
				level.Hide()
				icon.Hide()
			} else if o.activeLevel >= o.requiredLevel {
				icon.SetResource(boolIconResource(true))
				icon.Show()
				text.Hide()
				level.Hide()
			} else {
				level.Set(o.activeLevel, o.trainedLevel, o.requiredLevel)
				text.Refresh()
				text.Hide()
				icon.Hide()
				level.Show()
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		r := requiredSkills[id]
		a.iw.show(infoInventoryType, int64(r.typeID))
	}
	return container.NewTabItem("Requirements", list)
}

func (a *inventoryTypeInfo) makeMarketTab(ctx context.Context, et *app.EveType) *container.TabItem {
	if !et.IsTradeable() {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	a.iw.onClosedFuncs = append(a.iw.onClosedFuncs, cancel)
	marketTab := container.NewTabItem("Market", widget.NewLabel("Fetching prices..."))
	go func() {
		const (
			priceFormat    = "#,###.##"
			currencySuffix = " ISK"
		)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
	L:
		for {
			var items []attributeItem

			var averagePrice string
			p, err := a.iw.eus.MarketPrice(ctx, et.ID)
			if err != nil {
				slog.Error("average price", "typeID", et.ID, "error", err)
				averagePrice = "ERROR: " + a.iw.u.HumanizeError(err)
			} else {
				averagePrice = p.StringFunc("?", func(v float64) string {
					return humanize.FormatFloat(priceFormat, v) + currencySuffix
				})
			}
			items = append(items, newAttributeItem("Average price", averagePrice))

			j, err := a.iw.js.FetchPrices(ctx, a.typeID)
			if err != nil {
				slog.Error("janice pricer", "typeID", et.ID, "error", err)
				s := "ERROR: " + a.iw.u.HumanizeError(err)
				items = append(items, newAttributeItem("Janice prices", s))
			} else {
				items2 := []attributeItem{
					newAttributeItem("Jita sell price", humanize.FormatFloat(
						priceFormat,
						j.ImmediatePrices.SellPrice)+currencySuffix,
					),
					newAttributeItem("Jita buy price", humanize.FormatFloat(
						priceFormat,
						j.ImmediatePrices.BuyPrice)+currencySuffix,
					),
					newAttributeItem("Jita sell volume", ihumanize.Comma(j.SellVolume)),
					newAttributeItem("Jita buy volume", ihumanize.Comma(j.BuyVolume)),
				}
				items = slices.Concat(items, items2)
			}
			c := newAttributeList(a.iw, items...)
			fyne.Do(func() {
				marketTab.Content = c
				a.tabs.Refresh()
			})
			select {
			case <-ctx.Done():
				break L
			case <-ticker.C:
			}
		}
		slog.Debug("market update type for canceled", "name", et.Name)
	}()
	return marketTab
}

type requiredSkill struct {
	rank          int
	name          string
	typeID        int64
	activeLevel   int64
	requiredLevel int64
	trainedLevel  int64
}

func (a *inventoryTypeInfo) calcRequiredSkills(ctx context.Context, characterID int64, attributes map[int64]*app.EveTypeDogmaAttribute) ([]requiredSkill, error) {
	var skills []requiredSkill
	skillAttributes := []struct {
		id    int64
		level int64
	}{
		{app.EveDogmaAttributePrimarySkillID, app.EveDogmaAttributePrimarySkillLevel},
		{app.EveDogmaAttributeSecondarySkillID, app.EveDogmaAttributeSecondarySkillLevel},
		{app.EveDogmaAttributeTertiarySkillID, app.EveDogmaAttributeTertiarySkillLevel},
		{app.EveDogmaAttributeQuaternarySkillID, app.EveDogmaAttributeQuaternarySkillLevel},
		{app.EveDogmaAttributeQuinarySkillID, app.EveDogmaAttributeQuinarySkillLevel},
		{app.EveDogmaAttributeSenarySkillID, app.EveDogmaAttributeSenarySkillLevel},
	}
	for i, x := range skillAttributes {
		daID, ok := attributes[x.id]
		if !ok {
			continue
		}
		typeID := int64(daID.Value)
		daLevel, ok := attributes[x.level]
		if !ok {
			continue
		}
		requiredLevel := int64(daLevel.Value)
		et, err := a.iw.eus.GetType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := a.iw.cs.GetSkill(ctx, characterID, typeID)
		if errors.Is(err, app.ErrNotFound) {
			// do nothing
		} else if err != nil {
			return nil, err
		} else {
			skill.activeLevel = cs.ActiveSkillLevel
			skill.trainedLevel = cs.TrainedSkillLevel
		}
		skills = append(skills, skill)
	}
	return skills, nil
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
	supportedCategories := infoWindowSupportedEveEntities()
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
		id:          int64(o.ID),
		category:    "Planet",
		text:        o.Name,
		infoVariant: infoNotSupported,
	}
}

func newEntityItemFromEveSolarSystem(o *app.EveSolarSystem) entityItem {
	ee := o.EveEntity()
	return entityItem{
		id:           int64(ee.ID),
		category:     ee.CategoryDisplay(),
		textSegments: o.DisplayRichText(),
		infoVariant:  eveEntity2InfoVariant(ee),
	}
}

func newEntityItemFromEveEntity(ee *app.EveEntity) entityItem {
	return newEntityItem(int64(ee.ID), ee.CategoryDisplay(), ee.Name, eveEntity2InfoVariant(ee))
}

func newEntityItemFromEveEntityWithText(ee *app.EveEntity, text string) entityItem {
	if text == "" {
		text = ee.Name
	}
	return newEntityItem(int64(ee.ID), ee.CategoryDisplay(), text, eveEntity2InfoVariant(ee))
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
