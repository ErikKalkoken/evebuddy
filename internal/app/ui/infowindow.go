package ui

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	infoWindowHeight    = 600
	infoWindowWidth     = 600
	logoUnitSize        = 64
	renderIconPixelSize = 256
	renderIconUnitSize  = 128
	zoomImagePixelSize  = 512
)

// InfoWindow represents a dedicated window for showing information similar to the in-game info windows.
type InfoWindow struct {
	nav           *iwidget.Navigator
	onClosedFuncs []func() // f runs when the window is closed. Useful for cleanup.
	u             *BaseUI
	w             fyne.Window
}

// newInfoWindow returns a configured InfoWindow.
func newInfoWindow(u *BaseUI) *InfoWindow {
	iw := &InfoWindow{
		u: u,
		w: u.MainWindow(),
	}
	return iw
}

// Show shows a new info window for an EveEntity.
func (iw *InfoWindow) showEveEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), int64(ee.ID))
}

// Show shows a new info window for an EveEntity.
func (iw *InfoWindow) Show(c app.EveEntityCategory, id int32) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), int64(id))
}

func (iw *InfoWindow) showLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw *InfoWindow) showRace(id int32) {
	iw.show(infoRace, int64(id))
}

func (iw *InfoWindow) show(t infoVariant, id int64) {
	if iw.u.IsOffline() {
		iw.u.ShowInformationDialog(
			"Offline",
			"Can't show info window when offline",
			iw.w,
		)
		return
	}
	var title string
	var page fyne.CanvasObject
	switch t {
	case infoAlliance:
		title = "Alliance"
		page = newAllianceInfo(iw, int32(id))
	case infoCharacter:
		title = "Character"
		page = newCharacterInfo(iw, int32(id))
	case infoConstellation:
		title = "Constellation"
		page = newConstellationInfo(iw, int32(id))
	case infoCorporation:
		title = "Corporation"
		page = newCorporationInfo(iw, int32(id))
	case infoInventoryType:
		// TODO: Restructure, so that window is first drawn empty and content loaded in background (as other info windo)
		a, err := NewInventoryTypeInfo(iw, int32(id), iw.u.CurrentCharacterID())
		if err != nil {
			iw.u.ShowInformationDialog("ERROR", "Something whent wrong when trying to show info for type", iw.w)
			slog.Error("show type", "error", err)
			return
		}
		title = a.title()
		page = a
	case infoRace:
		title = "Race"
		page = newRaceInfo(iw, int32(id))
	case infoRegion:
		title = "Region"
		page = newRegionInfo(iw, int32(id))
	case infoSolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, int32(id))
	case infoLocation:
		title = "Location"
		page = newLocationInfo(iw, id)
	default:
		iw.u.ShowInformationDialog(
			"Warning",
			"Can't show info window for unknown category",
			iw.w,
		)
		return
	}
	ab := iwidget.NewAppBar(title+": Information", page)
	if iw.nav == nil {
		w := iw.u.App().NewWindow(iw.u.MakeWindowTitle("Information"))
		iw.w = w
		iw.nav = iwidget.NewNavigatorWithAppBar(ab)
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
		w.Show()
	} else {
		iw.nav.Push(ab)
	}
}

func (iw *InfoWindow) showZoomWindow(title string, id int32, load func(int32, int) (fyne.Resource, error), w fyne.Window) {
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	r, err := load(id, zoomImagePixelSize)
	if err != nil {
		iw.u.ShowErrorDialog("Failed to load image", err, w)
	}
	i := iwidget.NewImageFromResource(r, fyne.NewSquareSize(s))
	p := theme.Padding()
	w2 := iw.u.App().NewWindow(iw.u.MakeWindowTitle(title))
	w2.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
	w2.Show()
}

func (iw *InfoWindow) openURL(s string) {
	x, err := url.ParseRequestURI(s)
	if err != nil {
		slog.Error("Construcing URL", "url", s, "error", err)
		return
	}
	err = iw.u.App().OpenURL(x)
	if err != nil {
		slog.Error("Opening URL", "url", x, "error", err)
		return
	}
}

func (iw *InfoWindow) makeZkillboardIcon(id int32, v infoVariant) *iwidget.TappableIcon {
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

func (iw *InfoWindow) makeDotlanIcon(id int32, v infoVariant) *iwidget.TappableIcon {
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

func (iw *InfoWindow) makeEveWhoIcon(id int32, v infoVariant) *iwidget.TappableIcon {
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
	if iw.u.IsMobile() {
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

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget

	attributes *attributeList
	hq         *kxwidget.TappableLabel
	id         int32
	iw         *InfoWindow
	logo       *canvas.Image
	members    *entityList
	name       *widget.Label
	tabs       *container.AppTabs
}

func newAllianceInfo(iw *InfoWindow, id int32) *allianceInfo {
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Wrapping = fyne.TextWrapWord
	a := &allianceInfo{
		iw:   iw,
		id:   id,
		name: makeInfoName(),
		logo: makeInfoLogo(),
		hq:   hq,
	}
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
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("alliance info update failed", "alliance", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load alliance: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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
				a.iw.makeZkillboardIcon(a.id, infoAlliance),
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

func (a *allianceInfo) load() error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.eis.AllianceLogo(a.id, app.IconPixelSize)
		if err != nil {
			slog.Error("alliance info: Failed to load logo", "allianceID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	}()

	// Members
	go func() {
		members, err := a.iw.u.eus.FetchAllianceCorporations(ctx, a.id)
		if err != nil {
			slog.Error("alliance info: Failed to load corporations", "allianceID", a.id, "error", err)
			return
		}
		if len(members) == 0 {
			return
		}
		fyne.Do(func() {
			a.members.set(entityItemsFromEveEntities(members)...)
		})
	}()
	o, err := a.iw.u.eus.FetchAlliance(ctx, a.id)
	if err != nil {
		return err
	}

	// Attributes
	attributes := make([]attributeItem, 0)
	if o.ExecutorCorporation != nil {
		attributes = append(attributes, newAttributeItem("Executor", o.ExecutorCorporation))
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
	if o.Faction != nil {
		attributes = append(attributes, newAttributeItem("Faction", o.Faction))
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.attributes.set(attributes)
	})
	return nil
}

// characterInfo shows public information about a character.
type characterInfo struct {
	widget.BaseWidget

	alliance        *kxwidget.TappableLabel
	bio             *widget.Label
	corporation     *kxwidget.TappableLabel
	corporationLogo *canvas.Image
	description     *widget.Label
	employeeHistory *entityList
	id              int32
	iw              *InfoWindow
	membership      *widget.Label
	name            *widget.Label
	portrait        *kxwidget.TappableImage
	security        *widget.Label
	tabs            *container.AppTabs
	title           *widget.Label
	attributes      *attributeList
}

func newCharacterInfo(iw *InfoWindow, id int32) *characterInfo {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Wrapping = fyne.TextWrapWord
	portrait := kxwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(iw.renderIconSize())
	title := widget.NewLabel("")
	title.Wrapping = fyne.TextWrapWord
	bio := widget.NewLabel("")
	bio.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &characterInfo{
		alliance:        alliance,
		bio:             bio,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:     description,
		id:              id,
		iw:              iw,
		membership:      widget.NewLabel(""),
		name:            makeInfoName(),
		portrait:        portrait,
		security:        widget.NewLabel(""),
		title:           title,
	}
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.employeeHistory = newEntityListFromItems(a.iw.show)
	attributes := container.NewTabItem("Attributes", a.attributes)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Bio", container.NewVScroll(a.bio)),
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		attributes,
		container.NewTabItem("Employment History", a.employeeHistory),
	)
	a.tabs.Select(attributes)
	return a
}

func (a *characterInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("character info update failed", "character", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
	p := theme.Padding()
	main := container.NewVBox(
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
				a.corporation,
				a.membership,
			),
		),
		widget.NewSeparator(),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.alliance,
			a.security,
		),
	)
	name := a.iw.u.scs.CharacterName(a.id)
	name = strings.ReplaceAll(name, " ", "_")
	forums := iwidget.NewTappableIcon(icons.EvelogoPng, func() {
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
				a.iw.makeZkillboardIcon(a.id, infoCharacter),
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

func (a *characterInfo) load() error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.eis.CharacterPortrait(a.id, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "characterID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.portrait.SetResource(r)
		})
	}()
	go func() {
		history, err := a.iw.u.eus.FetchCharacterCorporationHistory(ctx, a.id)
		if err != nil {
			slog.Error("character info: Failed to load corporation history", "characterID", a.id, "error", err)
			return
		}
		if len(history) == 0 {
			fyne.Do(func() {
				a.membership.Hide()
			})
			return
		}
		items := xslices.Map(history, historyItem2EntityItem)
		fyne.Do(func() {
			a.employeeHistory.set(items...)
			current := history[0]
			duration := humanize.RelTime(current.StartDate, time.Now(), "", "")
			a.membership.SetText(fmt.Sprintf("for %s", duration))
		})
	}()
	o, err := a.iw.u.eus.GetOrCreateCharacterESI(ctx, a.id)
	if err != nil {
		return err
	}
	go func() {
		r, err := a.iw.u.eis.CorporationLogo(o.Corporation.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "characterID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.corporationLogo.Resource = r
			a.corporationLogo.Refresh()
		})
	}()
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.security.SetText(fmt.Sprintf("Security Status: %.1f", o.SecurityStatus))
		a.corporation.SetText(fmt.Sprintf("Member of %s", o.Corporation.Name))
		a.corporation.OnTapped = func() {
			a.iw.showEveEntity(o.Corporation)
		}
		a.portrait.OnTapped = func() {
			go fyne.Do(func() {
				a.iw.showZoomWindow(o.Name, a.id, a.iw.u.eis.CharacterPortrait, a.iw.w)
			})
		}
	})
	fyne.Do(func() {
		a.bio.SetText(o.DescriptionPlain())
		a.description.SetText(o.RaceDescription())
	})
	fyne.Do(func() {
		if !o.HasAlliance() {
			a.alliance.Hide()
			return
		}
		a.alliance.SetText(o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.showEveEntity(o.Alliance)
		}
	})
	fyne.Do(func() {
		if o.Title == "" {
			a.title.Hide()
			return
		}
		a.title.SetText("Title: " + o.Title)
	})
	attributes := []attributeItem{
		newAttributeItem("Born", o.Birthday.Format(app.DateTimeFormat)),
		newAttributeItem("Race", o.Race),
		newAttributeItem("Security Status", fmt.Sprintf("%.1f", o.SecurityStatus)),
		newAttributeItem("Corporation", o.Corporation),
	}
	if o.Alliance != nil {
		attributes = append(attributes, newAttributeItem("Alliance", o.Alliance))
	}
	if o.Faction != nil {
		attributes = append(attributes, newAttributeItem("Faction", o.Faction))
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	fyne.Do(func() {
		a.attributes.set(attributes)
	})
	return nil
}

type constellationInfo struct {
	widget.BaseWidget

	iw *InfoWindow

	id      int32
	region  *kxwidget.TappableLabel
	logo    *canvas.Image
	name    *widget.Label
	tabs    *container.AppTabs
	systems *entityList
}

func newConstellationInfo(iw *InfoWindow, id int32) *constellationInfo {
	region := kxwidget.NewTappableLabel("", nil)
	region.Wrapping = fyne.TextWrapWord
	a := &constellationInfo{
		iw:     iw,
		id:     id,
		logo:   makeInfoLogo(),
		name:   makeInfoName(),
		region: region,
		tabs:   container.NewAppTabs(),
	}
	a.ExtendBaseWidget(a)
	a.logo.Resource = icons.Constellation64Png
	a.systems = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Solar Systems", a.systems),
	)
	return a
}

func (a *constellationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("constellation info update failed", "solarSystem", a.id, "error", err)
			fyne.Do(func() {
				fyne.Do(func() {
					a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", a.iw.u.humanizeError(err))
					a.name.Importance = widget.DangerImportance
					a.name.Refresh()
				})
			})
		}
	}()
	colums := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(colums, widget.NewLabel("Region"), a.region),
		))
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *constellationInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateConstellationESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.region.SetText(o.Region.Name)
		a.region.OnTapped = func() {
			a.iw.showEveEntity(o.Region.ToEveEntity())
		}

		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				a.iw.u.App().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			a.tabs.Append(attributesTab)
		}
	})
	go func() {
		oo, err := a.iw.u.eus.GetConstellationSolarSystemsESI(ctx, o.ID)
		if err != nil {
			slog.Error("constellation info: Failed to load constellations", "region", o.ID, "error", err)
			return
		}
		xx := xslices.Map(oo, newEntityItemFromEveSolarSystem)
		fyne.Do(func() {
			a.systems.set(xx...)
		})
	}()
	return nil
}

// corporationInfo shows public information about a character.
type corporationInfo struct {
	widget.BaseWidget

	alliance        *kxwidget.TappableLabel
	allianceHistory *entityList
	allianceLogo    *canvas.Image
	attributes      *attributeList
	description     *widget.Label
	hq              *kxwidget.TappableLabel
	id              int32
	iw              *InfoWindow
	logo            *canvas.Image
	name            *widget.Label
	tabs            *container.AppTabs
}

func newCorporationInfo(iw *InfoWindow, id int32) *corporationInfo {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &corporationInfo{
		alliance:     alliance,
		allianceLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:  description,
		hq:           hq,
		id:           id,
		iw:           iw,
		logo:         makeInfoLogo(),
		name:         makeInfoName(),
	}
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.allianceHistory = newEntityListFromItems(a.iw.show)
	attributes := container.NewTabItem("Attributes", a.attributes)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		attributes,
		container.NewTabItem("Alliance History", a.allianceHistory),
	)
	a.tabs.Select(attributes)
	return a
}

func (a *corporationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("corporation info update failed", "corporation", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.hq,
		),
		container.NewBorder(
			nil,
			nil,
			a.allianceLogo,
			nil,
			a.alliance,
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
				a.iw.makeZkillboardIcon(a.id, infoCorporation),
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

func (a *corporationInfo) load() error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.eis.CorporationLogo(a.id, app.IconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	}()
	o, err := a.iw.u.eus.GetCorporationESI(ctx, a.id)
	if err != nil {
		return err
	}
	attributes := a.makeAttributes(o)
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.DescriptionPlain())
		a.attributes.set(attributes)
	})
	fyne.Do(func() {
		if o.Alliance == nil {
			a.alliance.Hide()
			a.allianceLogo.Hide()
			return
		}
		a.alliance.SetText("Member of " + o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.showEveEntity(o.Alliance)
		}
		go func() {
			r, err := a.iw.u.eis.AllianceLogo(o.Alliance.ID, app.IconPixelSize)
			if err != nil {
				slog.Error("corporation info: Failed to load alliance logo", "allianceID", o.Alliance.ID, "error", err)
				return
			}
			fyne.Do(func() {
				a.allianceLogo.Resource = r
				a.allianceLogo.Refresh()
			})
		}()
	})
	fyne.Do(func() {
		if o.HomeStation == nil {
			a.hq.Hide()
			return
		}
		a.hq.SetText("Headquarters: " + o.HomeStation.Name)
		a.hq.OnTapped = func() {
			a.iw.showEveEntity(o.HomeStation)
		}
	})
	go func() {
		history, err := a.iw.u.eus.FetchCorporationAllianceHistory(ctx, a.id)
		if err != nil {
			slog.Error("corporation info: Failed to load alliance history", "corporationID", a.id, "error", err)
			return
		}
		var items []entityItem
		if len(history) > 0 {
			history2 := xslices.Filter(history, func(v app.MembershipHistoryItem) bool {
				return v.Organization != nil && v.Organization.Category.IsKnown()
			})
			items = append(items, xslices.Map(history2, historyItem2EntityItem)...)
		}
		var founded string
		if o.DateFounded.IsZero() {
			founded = "?"
		} else {
			founded = fmt.Sprintf("**%s**", o.DateFounded.Format(app.DateFormat))
		}
		items = append(items, newEntityItem(0, "Corporation Founded", founded, infoNotSupported))
		fyne.Do(func() {
			a.allianceHistory.set(items...)
		})
	}()
	return nil
}

func (a *corporationInfo) makeAttributes(o *app.EveCorporation) []attributeItem {
	attributes := make([]attributeItem, 0)
	if o.Ceo != nil {
		attributes = append(attributes, newAttributeItem("CEO", o.Ceo))
	}
	if o.Creator != nil {
		attributes = append(attributes, newAttributeItem("Founder", o.Creator))
	}
	if o.Alliance != nil {
		attributes = append(attributes, newAttributeItem("Alliance", o.Alliance))
	}
	if o.Ticker != "" {
		attributes = append(attributes, newAttributeItem("Ticker Name", o.Ticker))
	}
	if o.Faction != nil {
		attributes = append(attributes, newAttributeItem("Faction", o.Faction))
	}
	if o.Shares != 0 {
		attributes = append(attributes, newAttributeItem("Shares", o.Shares))
	}
	if o.MemberCount != 0 {
		attributes = append(attributes, newAttributeItem("Member Count", o.MemberCount))
	}
	if o.TaxRate != 0 {
		attributes = append(attributes, newAttributeItem("ISK Tax Rate", o.TaxRate))
	}
	attributes = append(attributes, newAttributeItem("War Eligability", o.WarEligible))
	if o.URL != "" {
		u, err := url.ParseRequestURI(o.URL)
		if err == nil && u.Host != "" {
			attributes = append(attributes, newAttributeItem("URL", u))
		}
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	return attributes
}

// locationInfo shows public information about a character.
type locationInfo struct {
	widget.BaseWidget

	description *widget.Label
	id          int64
	iw          *InfoWindow
	name        *widget.Label
	owner       *kxwidget.TappableLabel
	ownerLogo   *canvas.Image
	location    *entityList
	tabs        *container.AppTabs
	typeImage   *kxwidget.TappableImage
	typeInfo    *kxwidget.TappableLabel
}

func newLocationInfo(iw *InfoWindow, id int64) *locationInfo {
	typeInfo := kxwidget.NewTappableLabel("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := kxwidget.NewTappableLabel("", nil)
	owner.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	typeImage := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(iw.renderIconSize())
	a := &locationInfo{
		description: description,
		id:          id,
		iw:          iw,
		name:        makeInfoName(),
		owner:       owner,
		ownerLogo:   iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		typeImage:   typeImage,
		typeInfo:    typeInfo,
	}
	a.ExtendBaseWidget(a)
	a.location = newEntityList(a.iw.show)
	location := container.NewTabItem("Location", a.location)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		location,
	)
	a.tabs.Select(location)
	return a
}

func (a *locationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("location info update failed", "locationID", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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

func (a *locationInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateLocationESI(ctx, a.id)
	if err != nil {
		return err
	}
	go func() {
		r, err := a.iw.u.eis.InventoryTypeRender(o.Type.ID, renderIconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load portrait", "location", o, "error", err)
			return
		}
		fyne.Do(func() {
			a.typeImage.SetResource(r)
		})
	}()
	go func() {
		r, err := a.iw.u.eis.CorporationLogo(o.Owner.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load corp logo", "owner", o.Owner, "error", err)
			return
		}
		fyne.Do(func() {
			a.ownerLogo.Resource = r
			a.ownerLogo.Refresh()
		})
	}()
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.typeInfo.SetText(o.Type.Name)
		a.typeInfo.OnTapped = func() {
			a.iw.showEveEntity(o.Type.ToEveEntity())
		}
		a.owner.SetText(o.Owner.Name)
		a.owner.OnTapped = func() {
			a.iw.showEveEntity(o.Owner)
		}
		a.typeImage.OnTapped = func() {
			a.iw.showZoomWindow(o.Name, o.Type.ID, a.iw.u.eis.InventoryTypeRender, a.iw.w)
		}
		description := o.Type.Description
		if description == "" {
			description = o.Type.Name
		}
		a.description.SetText(description)
	})

	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
			a.tabs.Refresh()
		})
	}
	fyne.Do(func() {
		a.location.set(
			newEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.Region.ToEveEntity(), ""),
			newEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.ToEveEntity(), ""),
			newEntityItemFromEveSolarSystem(o.SolarSystem),
		)
	})
	if o.Variant() == app.EveLocationStation {
		services := container.NewTabItem("Services", widget.NewLabel(""))
		fyne.DoAndWait(func() {
			a.tabs.Append(services)
			a.tabs.Refresh()
		})
		go func() {
			ss, err := a.iw.u.eus.GetStationServicesESI(ctx, int32(a.id))
			if err != nil {
				slog.Error("Failed to fetch station services", "stationID", o.ID, "error", err)
				return
			}
			items := xslices.Map(ss, func(s string) entityItem {
				s2 := strings.ReplaceAll(s, "-", " ")
				titler := cases.Title(language.English)
				name := titler.String(s2)
				return newEntityItem(0, "Service", name, infoNotSupported)
			})
			fyne.Do(func() {
				services.Content = newEntityListFromItems(nil, items...)
				a.tabs.Refresh()
			})
		}()
	}

	return nil
}

type raceInfo struct {
	widget.BaseWidget

	id          int32
	iw          *InfoWindow
	logo        *canvas.Image
	name        *widget.Label
	tabs        *container.AppTabs
	description *widget.Label
}

func newRaceInfo(iw *InfoWindow, id int32) *raceInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &raceInfo{
		description: description,
		id:          id,
		iw:          iw,
		logo:        makeInfoLogo(),
		name:        makeInfoName(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.BlankSvg
	a.ExtendBaseWidget(a)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
	)
	return a
}

func (a *raceInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("race info update failed", "race", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load race: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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

func (a *raceInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateRaceESI(ctx, a.id)
	if err != nil {
		return err
	}
	factionID, found := o.FactionID()
	if found {
		go func() {
			r, err := a.iw.u.eis.FactionLogo(factionID, app.IconPixelSize)
			if err != nil {
				slog.Error("race info: Failed to load logo", "corporationID", a.id, "error", err)
				return
			}
			fyne.Do(func() {
				a.logo.Resource = r
				a.logo.Refresh()
			})
		}()
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.Description)
	})
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			a.iw.u.App().Clipboard().SetContent(v.(string))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
		})
	}
	return nil
}

type regionInfo struct {
	widget.BaseWidget

	description    *widget.Label
	constellations *entityList
	id             int32
	iw             *InfoWindow
	logo           *canvas.Image
	name           *widget.Label
	tabs           *container.AppTabs
}

func newRegionInfo(iw *InfoWindow, id int32) *regionInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &regionInfo{
		iw:          iw,
		id:          id,
		description: description,
		logo:        makeInfoLogo(),
		name:        makeInfoName(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.Region64Png
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
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("region info update failed", "solarSystem", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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
				a.iw.makeZkillboardIcon(a.id, infoRegion),
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

func (a *regionInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateRegionESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.DescriptionPlain())
	})
	fyne.Do(func() {
		if !a.iw.u.IsDeveloperMode() {
			return
		}
		x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			a.iw.u.App().Clipboard().SetContent(v.(string))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	})
	go func() {
		oo, err := a.iw.u.eus.GetRegionConstellationsESI(ctx, o.ID)
		if err != nil {
			slog.Error("region info: Failed to load constellations", "region", o.ID, "error", err)
			return
		}
		items := xslices.Map(oo, NewEntityItemFromEveEntity)
		fyne.Do(func() {
			a.constellations.set(items...)
		})
	}()
	return nil
}

type solarSystemInfo struct {
	widget.BaseWidget

	constellation *kxwidget.TappableLabel
	id            int32
	iw            *InfoWindow
	logo          *canvas.Image
	name          *widget.Label
	planets       *entityList
	region        *kxwidget.TappableLabel
	security      *widget.Label
	stargates     *entityList
	stations      *entityList
	structures    *entityList
	tabs          *container.AppTabs
}

func newSolarSystemInfo(iw *InfoWindow, id int32) *solarSystemInfo {
	region := kxwidget.NewTappableLabel("", nil)
	region.Wrapping = fyne.TextWrapWord
	constellation := kxwidget.NewTappableLabel("", nil)
	constellation.Wrapping = fyne.TextWrapWord
	a := &solarSystemInfo{
		id:            id,
		region:        region,
		constellation: constellation,
		iw:            iw,
		logo:          makeInfoLogo(),
		name:          makeInfoName(),
		security:      widget.NewLabel(""),
		tabs:          container.NewAppTabs(),
	}
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
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("solar system info update failed", "solarSystem", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", a.iw.u.humanizeError(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
	colums := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Solar System"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(colums, widget.NewLabel("Region"), a.region),
			container.New(colums, widget.NewLabel("Constellation"), a.constellation),
			container.New(colums, widget.NewLabel("Security"), a.security),
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
				a.iw.makeZkillboardIcon(a.id, infoSolarSystem),
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

func (a *solarSystemInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.region.SetText(o.Constellation.Region.Name)
		a.region.OnTapped = func() {
			a.iw.showEveEntity(o.Constellation.Region.ToEveEntity())
		}
		a.constellation.SetText(o.Constellation.Name)
		a.constellation.OnTapped = func() {
			a.iw.showEveEntity(o.Constellation.ToEveEntity())
		}
		a.security.Text = o.SecurityStatusDisplay()
		a.security.Importance = o.SecurityType().ToImportance()
		a.security.Refresh()
	})

	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", fmt.Sprint(a.id))
		x.Action = func(v any) {
			a.iw.u.App().Clipboard().SetContent(v.(string))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
		})
	}

	a.tabs.Refresh()

	go func() {
		starID, planets, stargateIDs, stations, structures, err := a.iw.u.eus.GetSolarSystemInfoESI(ctx, a.id)
		if err != nil {
			slog.Error("solar system info: Failed to load system info", "solarSystem", a.id, "error", err)
			return
		}
		items := entityItemsFromEveEntities(stations)
		fyne.Do(func() {
			a.stations.set(items...)
		})
		oo := xslices.Map(structures, func(x *app.EveLocation) entityItem {
			return newEntityItem(
				x.ID,
				x.Name,
				"Structure",
				infoLocation,
			)
		})
		fyne.Do(func() {
			a.structures.set(oo...)
		})

		id, err := a.iw.u.eus.GetStarTypeID(ctx, starID)
		if err != nil {
			return
		}
		r, err := a.iw.u.eis.InventoryTypeIcon(id, app.IconPixelSize)
		if err != nil {
			slog.Error("solar system info: Failed to load logo", "solarSystem", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})

		go func() {
			ss, err := a.iw.u.eus.GetStargateSolarSystemsESI(ctx, stargateIDs)
			if err != nil {
				slog.Error("solar system info: Failed to load adjacent systems", "solarSystem", a.id, "error", err)
				return
			}
			items := xslices.Map(ss, newEntityItemFromEveSolarSystem)
			fyne.Do(func() {
				a.stargates.set(items...)
			})
		}()

		go func() {
			pp, err := a.iw.u.eus.GetSolarSystemPlanets(ctx, planets)
			if err != nil {
				slog.Error("solar system info: Failed to load planets", "solarSystem", a.id, "error", err)
				return
			}
			items := xslices.Map(pp, newEntityItemFromEvePlanet)
			fyne.Do(func() {
				a.planets.set(items...)
			})
		}()

	}()
	return nil
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
				w.iw.showEveEntity(x)
			}
		case *app.EveRace:
			w.iw.showRace(x.ID)
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

type entityItem struct {
	id           int64
	category     string
	text         string                   // text in markdown
	textSegments []widget.RichTextSegment // takes precendence over text when not empty
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
	ee := o.ToEveEntity()
	return entityItem{
		id:           int64(ee.ID),
		category:     ee.CategoryDisplay(),
		textSegments: o.DisplayRichText(),
		infoVariant:  eveEntity2InfoVariant(ee),
	}
}

func NewEntityItemFromEveEntity(ee *app.EveEntity) entityItem {
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
	items := make([]entityItem, 0)
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
			category := iwidget.NewLabelWithSize("Category", theme.SizeNameCaptionText)
			text := widget.NewRichText()
			text.Truncation = fyne.TextTruncateEllipsis
			icon := widget.NewIcon(theme.InfoIcon())
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
			icon := border1[1].(*fyne.Container).Objects[1]
			category := border2[0].(*fyne.Container).Objects[0].(*iwidget.Label)
			category.SetText(it.category)
			if it.infoVariant == infoNotSupported {
				icon.Hide()
			} else {
				icon.Show()
			}
			text := border2[1].(*fyne.Container).Objects[0].(*widget.RichText)
			if len(it.textSegments) != 0 {
				text.Segments = it.textSegments
				text.Refresh()
			} else {
				text.ParseMarkdown(it.text)
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
		if it.infoVariant == infoNotSupported {
			return
		}
		w.showInfo(it.infoVariant, it.id)
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
	if hi.IsDeleted {
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

func makeInfoName() *widget.Label {
	name := widget.NewLabel("Loading...")
	name.Wrapping = fyne.TextWrapWord
	return name
}
