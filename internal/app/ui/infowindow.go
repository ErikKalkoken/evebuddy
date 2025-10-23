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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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

// infoWindow represents a dedicated window for showing information similar to the in-game info windows.
type infoWindow struct {
	nav           *iwidget.Navigator
	onClosedFuncs []func() // f runs when the window is closed. Useful for cleanup.
	sb            *iwidget.Snackbar
	u             *baseUI
	w             fyne.Window
}

// newInfoWindow returns a configured InfoWindow.
func newInfoWindow(u *baseUI) *infoWindow {
	iw := &infoWindow{
		u: u,
		w: u.MainWindow(),
	}
	return iw
}

// Show shows a new info window for an EveEntity.
func (iw *infoWindow) showEveEntity(ee *app.EveEntity) {
	iw.show(eveEntity2InfoVariant(ee), int64(ee.ID))
}

// Show shows a new info window for an EveEntity.
func (iw *infoWindow) Show(c app.EveEntityCategory, id int32) {
	iw.show(eveEntity2InfoVariant(&app.EveEntity{Category: c}), int64(id))
}

func (iw *infoWindow) showLocation(id int64) {
	iw.show(infoLocation, id)
}

func (iw *infoWindow) showRace(id int32) {
	iw.show(infoRace, int64(id))
}

// infoWidget defines common functionality for all info widgets.
type infoWidget interface {
	fyne.CanvasObject
	update() error
	setError(string)
}

func (iw *infoWindow) show(v infoVariant, id int64) {
	iw.showWithCharacterID(v, id, 0)
}

func (iw *infoWindow) showWithCharacterID(v infoVariant, entityID int64, characterID int32) {
	if iw.u.IsOffline() {
		iw.u.ShowInformationDialog(
			"Offline",
			"Can't show info window when offline",
			iw.w,
		)
		return
	}

	makeAppBarTitle := func(s string) string {
		if !iw.u.isDesktop {
			return s
		}
		return s + ": Information"
	}

	var title string
	var page infoWidget
	var ab *iwidget.AppBar
	switch v {
	case infoAlliance:
		title = "Alliance"
		page = newAllianceInfo(iw, int32(entityID))
	case infoCharacter:
		title = "Character"
		page = newCharacterInfo(iw, int32(entityID))
	case infoConstellation:
		title = "Constellation"
		page = newConstellationInfo(iw, int32(entityID))
	case infoCorporation:
		title = "Corporation"
		page = newCorporationInfo(iw, int32(entityID))
	case infoInventoryType:
		x := newInventoryTypeInfo(iw, int32(entityID), characterID)
		x.setTitle = func(s string) { ab.SetTitle(makeAppBarTitle(s)) }
		page = x
		title = "Item"
	case infoRace:
		title = "Race"
		page = newRaceInfo(iw, int32(entityID))
	case infoRegion:
		title = "Region"
		page = newRegionInfo(iw, int32(entityID))
	case infoSolarSystem:
		title = "Solar System"
		page = newSolarSystemInfo(iw, int32(entityID))
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
	if iw.nav == nil {
		w := iw.u.App().NewWindow(iw.u.makeWindowTitle("Information"))
		iw.w = w
		iw.sb = iwidget.NewSnackbar(w)
		iw.sb.Start()
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
		err := page.update()
		if err != nil {
			slog.Error("info widget load", "variant", v, "id", entityID, "error", err)
			fyne.Do(func() {
				page.setError("ERROR: " + iw.u.humanizeError(err))
			})
		}
	}()
}

func (iw *infoWindow) showZoomWindow(title string, id int32, load func(int32, int) (fyne.Resource, error), w fyne.Window) {
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	r, err := load(id, zoomImagePixelSize)
	if err != nil {
		iw.u.showErrorDialog("Failed to load image", err, w)
	}
	i := iwidget.NewImageFromResource(r, fyne.NewSquareSize(s))
	p := theme.Padding()
	w2, created := iw.u.getOrCreateWindow(fmt.Sprintf("zoom-window-%d", id), title)
	if created {
		w2.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
	}
	w2.Show()
}

func (iw *infoWindow) openURL(s string) {
	x, err := url.ParseRequestURI(s)
	if err != nil {
		slog.Error("Constructing URL", "url", s, "error", err)
		return
	}
	err = iw.u.App().OpenURL(x)
	if err != nil {
		slog.Error("Opening URL", "url", x, "error", err)
		return
	}
}

func (iw *infoWindow) makeZKillboardIcon(id int32, v infoVariant) *iwidget.TappableIcon {
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

func (iw *infoWindow) makeDotlanIcon(id int32, v infoVariant) *iwidget.TappableIcon {
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

func (iw *infoWindow) makeEveWhoIcon(id int32, v infoVariant) *iwidget.TappableIcon {
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

func (iw *infoWindow) renderIconSize() fyne.Size {
	var s float32
	if !iw.u.isDesktop {
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
	name *kxwidget.TappableLabel
	iw   *infoWindow
}

func (b *baseInfo) initBase(iw *infoWindow) {
	b.iw = iw
	b.name = kxwidget.NewTappableLabel("Loading...", func() {
		b.iw.u.app.Clipboard().SetContent(b.name.Text)
		b.iw.sb.Show(fmt.Sprintf("\"%s\" added to clipboard", b.name.Text))
	})
	b.name.Wrapping = fyne.TextWrapWord
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
	id         int32
	logo       *canvas.Image
	members    *entityList
	tabs       *container.AppTabs
}

func newAllianceInfo(iw *infoWindow, id int32) *allianceInfo {
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

func (a *allianceInfo) update() error {
	ctx := context.Background()
	g := new(errgroup.Group)
	g.Go(func() error {
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
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		r, err := a.iw.u.eis.AllianceLogo(a.id, app.IconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		members, err := a.iw.u.eus.FetchAllianceCorporations(ctx, a.id)
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
	id              int32
	isOwned         bool
	membership      *widget.Label
	ownedIcon       *ttwidget.Icon
	portrait        *kxwidget.TappableImage
	security        *widget.Label
	tabs            *container.AppTabs
	title           *widget.Label
}

func newCharacterInfo(iw *infoWindow, id int32) *characterInfo {
	alliance := widget.NewHyperlink("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	corporation := widget.NewHyperlink("", nil)
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
	ownedIcon := ttwidget.NewIcon(theme.NewSuccessThemedResource(icons.CheckDecagramSvg))
	ownedIcon.SetToolTip("You own this character")
	a := &characterInfo{
		alliance:        alliance,
		bio:             bio,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:     description,
		id:              id,
		isOwned:         iw.u.scs.ListCharacterIDs().Contains(id),
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
		ec, err := a.iw.u.eus.GetCharacterESI(context.Background(), a.id)
		if err != nil {
			a.iw.u.snackbar.Show("Failed to get character for forum: " + a.iw.u.humanizeError(err))
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

func (a *characterInfo) update() error {
	ctx := context.Background()
	o, _, err := a.iw.u.eus.GetOrCreateCharacterESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		r, err := a.iw.u.eis.CharacterPortrait(a.id, 256)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.portrait.SetResource(r)
		})
		return nil
	})
	g.Go(func() error {
		history, err := a.iw.u.eus.FetchCharacterCorporationHistory(ctx, a.id)
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
	})
	g.Go(func() error {
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.security.SetText(fmt.Sprintf("Security Status: %.1f", o.SecurityStatus))
			a.corporation.SetText(o.Corporation.Name)
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
			a.tabs.Refresh()
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
		attributes, err := a.makeAttributes(o)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.attributes.set(attributes)
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		r, err := a.iw.u.eis.CorporationLogo(o.Corporation.ID, app.IconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.corporationLogo.Resource = r
			a.corporationLogo.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		found, err := a.iw.u.cs.HasCharacter(ctx, a.id)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		roles, err := a.iw.u.cs.ListRoles(ctx, a.id)
		if err != nil {
			return err
		}
		tab, search := a.makeRolesTab(roles)
		fyne.Do(func() {
			a.tabs.Append(tab)
			a.tabs.OnSelected = func(ti *container.TabItem) {
				if ti != tab {
					return
				}
				a.iw.w.Canvas().Focus(search)
			}
			a.tabs.Refresh()
		})
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (a *characterInfo) makeAttributes(o *app.EveCharacter) ([]attributeItem, error) {
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
	var u any
	if v := o.EveEntity().IsNPC(); v.IsEmpty() {
		u = "?"
	} else {
		u = v.ValueOrZero()
	}
	attributes = append(attributes, newAttributeItem("NPC", u))
	if a.isOwned {
		c, err := a.iw.u.cs.GetCharacter(context.Background(), a.id)
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, newAttributeItem("Home", c.Home))
		attributes = append(attributes, newAttributeItem("Location", c.Location))
		attributes = append(attributes, newAttributeItem("Last Login", c.LastLoginAt.ValueOrZero()))
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	return attributes, nil
}

func (a *characterInfo) makeRolesTab(roles []app.CharacterRole) (*container.TabItem, *widget.Entry) {
	rolesFiltered := slices.Clone(roles)
	list := widget.NewList(
		func() int {
			return len(rolesFiltered)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Template")
			name.Wrapping = fyne.TextWrapWord
			return container.NewBorder(
				nil,
				nil,
				nil,
				widget.NewIcon(icons.BlankSvg),
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(rolesFiltered) {
				return
			}
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(rolesFiltered[id].Role.Display())
			border[1].(*widget.Icon).SetResource(boolIconResource(rolesFiltered[id].Granted))
		},
	)
	search := widget.NewEntry()
	search.PlaceHolder = "Search roles"
	search.OnChanged = func(s string) {
		if len(s) < 2 {
			rolesFiltered = slices.Clone(roles)
			list.Refresh()
			return
		}
		rolesFiltered = xslices.Filter(roles, func(x app.CharacterRole) bool {
			return strings.Contains(x.Role.String(), strings.ToLower(s))
		})
		list.Refresh()
	}
	search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		search.SetText("")
	})
	rolesTab := container.NewTabItem("Roles", container.NewBorder(search, nil, nil, nil, list))
	return rolesTab, search
}

type constellationInfo struct {
	widget.BaseWidget
	baseInfo

	id      int32
	logo    *canvas.Image
	region  *widget.Hyperlink
	systems *entityList
	tabs    *container.AppTabs
}

func newConstellationInfo(iw *infoWindow, id int32) *constellationInfo {
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

func (a *constellationInfo) update() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateConstellationESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.region.SetText(o.Region.Name)
		a.region.OnTapped = func() {
			a.iw.showEveEntity(o.Region.EveEntity())
		}

		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				a.iw.u.App().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			a.tabs.Append(attributesTab)
			a.tabs.Refresh()
		}
	})
	oo, err := a.iw.u.eus.GetConstellationSolarSystemsESI(ctx, o.ID)
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
	id              int32
	logo            *canvas.Image
	tabs            *container.AppTabs
}

func newCorporationInfo(iw *infoWindow, id int32) *corporationInfo {
	alliance := widget.NewHyperlink("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	hq := widget.NewHyperlink("", nil)
	hq.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &corporationInfo{
		alliance:     alliance,
		allianceLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		description:  description,
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

func (a *corporationInfo) update() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateCorporationESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		r, err := a.iw.u.eis.CorporationLogo(a.id, app.IconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		attributes := a.makeAttributes(o)
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.description.SetText(o.DescriptionPlain())
			a.attributes.set(attributes)
			a.tabs.Refresh()
		})
		fyne.Do(func() {
			if o.Alliance == nil {
				a.allianceBox.Hide()
				return
			}
			a.alliance.SetText(o.Alliance.Name)
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
		return nil
	})
	g.Go(func() error {
		history, err := a.iw.u.eus.FetchCorporationAllianceHistory(ctx, a.id)
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
		if o.DateFounded.IsEmpty() {
			founded = "?"
		} else {
			founded = fmt.Sprintf("**%s**", o.DateFounded.ValueOrZero().Format(app.DateFormat))
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
	var u any
	if v := o.EveEntity().IsNPC(); v.IsEmpty() {
		u = "?"
	} else {
		u = v.ValueOrZero()
	}
	attributes = append(attributes, newAttributeItem("NPC", u))
	if !o.Shares.IsEmpty() {
		attributes = append(attributes, newAttributeItem("Shares", o.Shares))
	}
	if o.MemberCount != 0 {
		attributes = append(attributes, newAttributeItem("Member Count", o.MemberCount))
	}
	if o.TaxRate != 0 {
		attributes = append(attributes, newAttributeItem("ISK Tax Rate", o.TaxRate))
	}
	attributes = append(attributes, newAttributeItem("War Eligibility", o.WarEligible))
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
	baseInfo

	description *widget.Label
	id          int64
	location    *entityList
	owner       *widget.Hyperlink
	ownerLogo   *canvas.Image
	services    *entityList
	tabs        *container.AppTabs
	typeImage   *kxwidget.TappableImage
	typeInfo    *widget.Hyperlink
}

func newLocationInfo(iw *infoWindow, id int64) *locationInfo {
	typeInfo := widget.NewHyperlink("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := widget.NewHyperlink("", nil)
	owner.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	typeImage := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(iw.renderIconSize())
	a := &locationInfo{
		description: description,
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

func (a *locationInfo) update() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateLocationESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		r, err := a.iw.u.eis.InventoryTypeRender(o.Type.ID, renderIconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.typeImage.SetResource(r)
		})
		return nil
	})
	g.Go(func() error {
		r, err := a.iw.u.eis.CorporationLogo(o.Owner.ID, app.IconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.ownerLogo.Resource = r
			a.ownerLogo.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.typeInfo.SetText(o.Type.Name)
			a.typeInfo.OnTapped = func() {
				a.iw.showEveEntity(o.Type.EveEntity())
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
			a.tabs.Refresh()
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
				newEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.Region.EveEntity(), ""),
				newEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.EveEntity(), ""),
				newEntityItemFromEveSolarSystem(o.SolarSystem),
			)
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		if o.Variant() != app.EveLocationStation {
			return nil
		}
		ss, err := a.iw.u.eus.GetStationServicesESI(ctx, int32(a.id))
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
	return g.Wait()
}

type raceInfo struct {
	widget.BaseWidget
	baseInfo

	id          int32
	logo        *canvas.Image
	tabs        *container.AppTabs
	description *widget.Label
}

func newRaceInfo(iw *infoWindow, id int32) *raceInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &raceInfo{
		description: description,
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

func (a *raceInfo) update() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateRaceESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		factionID, found := o.FactionID()
		if !found {
			return nil
		}
		r, err := a.iw.u.eis.FactionLogo(factionID, app.IconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
		return nil
	})
	g.Go(func() error {
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
	id             int32
	logo           *canvas.Image
	tabs           *container.AppTabs
}

func newRegionInfo(iw *infoWindow, id int32) *regionInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &regionInfo{
		id:          id,
		description: description,
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

func (a *regionInfo) update() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateRegionESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if !a.iw.u.IsDeveloperMode() {
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
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.description.SetText(o.DescriptionPlain())
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		oo, err := a.iw.u.eus.GetRegionConstellationsESI(ctx, o.ID)
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
	id            int32
	logo          *canvas.Image
	planets       *entityList
	region        *widget.Hyperlink
	security      *widget.Label
	stargates     *entityList
	stations      *entityList
	structures    *entityList
	tabs          *container.AppTabs
}

func newSolarSystemInfo(iw *infoWindow, id int32) *solarSystemInfo {
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

func (a *solarSystemInfo) update() error {
	ctx := context.Background()
	o, err := a.iw.u.eus.GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	starID, planets, stargateIDs, stations, structures, err := a.iw.u.eus.GetSolarSystemInfoESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
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
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.region.SetText(o.Constellation.Region.Name)
			a.region.OnTapped = func() {
				a.iw.showEveEntity(o.Constellation.Region.EveEntity())
			}
			a.constellation.SetText(o.Constellation.Name)
			a.constellation.OnTapped = func() {
				a.iw.showEveEntity(o.Constellation.EveEntity())
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
	g.Go(func() error {
		id, err := a.iw.u.eus.GetStarTypeID(ctx, starID)
		if err != nil {
			return err
		}
		r, err := a.iw.u.eis.InventoryTypeIcon(id, app.IconPixelSize)
		if err != nil {
			return err
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		ss, err := a.iw.u.eus.GetStargatesSolarSystemsESI(ctx, stargateIDs)
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
		pp, err := a.iw.u.eus.GetSolarSystemPlanets(ctx, planets)
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
	characterID      int32
	characterName    *widget.Hyperlink
	checkIcon        *widget.Icon
	description      *widget.Label
	eveMarketBrowser *fyne.Container
	janice           *fyne.Container
	setTitle         func(string) // for setting the title during update
	tabs             *container.AppTabs
	typeIcon         *kxwidget.TappableImage
	typeID           int32
}

func newInventoryTypeInfo(iw *infoWindow, typeID, characterID int32) *inventoryTypeInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	typeIcon := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeIcon.SetFillMode(canvas.ImageFillContain)
	typeIcon.SetMinSize(fyne.NewSquareSize(logoUnitSize))
	a := &inventoryTypeInfo{
		characterIcon: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		characterID:   characterID,
		checkIcon:     widget.NewIcon(icons.BlankSvg),
		description:   description,
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

func (a *inventoryTypeInfo) update() error {
	ctx := context.Background()
	et, err := a.iw.u.eus.GetOrCreateTypeESI(ctx, a.typeID)
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

	iwidget.RefreshTappableImageAsync(a.typeIcon, func() (fyne.Resource, error) {
		if et.IsSKIN() {
			return a.iw.u.eis.InventoryTypeSKIN(et.ID, app.IconPixelSize)
		} else if et.IsBlueprint() {
			return a.iw.u.eis.InventoryTypeBPO(et.ID, app.IconPixelSize)
		} else {
			return a.iw.u.eis.InventoryTypeIcon(et.ID, app.IconPixelSize)
		}
	})
	if et.HasRender() {
		a.typeIcon.OnTapped = func() {
			fyne.Do(func() {
				a.iw.showZoomWindow(et.Name, a.typeID, a.iw.u.eis.InventoryTypeRender, a.iw.w)
			})
		}
	}

	var character *app.EveEntity
	if a.characterID != 0 {
		ee, err := a.iw.u.eus.GetOrCreateEntityESI(ctx, a.characterID)
		if err != nil {
			return err
		}
		character = ee
		iwidget.RefreshImageAsync(a.characterIcon, func() (fyne.Resource, error) {
			return a.iw.u.eis.CharacterPortrait(character.ID, app.IconPixelSize)
		})
		fyne.Do(func() {
			a.characterIcon.Show()
			a.characterName.OnTapped = func() {
				a.iw.showEveEntity(character)
			}
			a.characterName.SetText(character.Name)
			a.characterName.Show()
		})
	}

	oo, err := a.iw.u.eus.ListTypeDogmaAttributesForType(ctx, et.ID)
	if err != nil {
		return err
	}
	dogmaAttributes := make(map[int32]*app.EveTypeDogmaAttribute)
	for _, o := range oo {
		dogmaAttributes[o.DogmaAttribute.ID] = o
	}

	var requiredSkills []requiredSkill
	if a.characterID != 0 {
		skills, err := a.calcRequiredSkills(ctx, a.characterID, dogmaAttributes, a.iw.u)
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
		if marketTab != nil && a.iw.u.settings.PreferMarketTab() {
			a.tabs.Select(marketTab)
		} else if requirementsTab != nil && et.Group.Category.ID == app.EveCategorySkill {
			a.tabs.Select(requirementsTab)
		} else if attributeTab != nil &&
			set.Of[int32](
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

func (a *inventoryTypeInfo) makeAttributeTab(ctx context.Context, dogmaAttributes map[int32]*app.EveTypeDogmaAttribute, et *app.EveType) *container.TabItem {
	attributes := a.calcAttributesData(ctx, et, dogmaAttributes, a.iw.u)
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
var attributeGroupsMap = map[attributeGroup][]int32{
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

type attributeRow struct {
	icon    fyne.Resource
	label   string
	value   string
	isTitle bool
	action  func(v string)
}

func (*inventoryTypeInfo) calcAttributesData(ctx context.Context, et *app.EveType, attributes map[int32]*app.EveTypeDogmaAttribute, u *baseUI) []attributeRow {
	droneCapacity, ok := attributes[app.EveDogmaAttributeDroneCapacity]
	hasDrones := ok && droneCapacity.Value > 0

	jumpDrive, ok := attributes[app.EveDogmaAttributeOnboardJumpDrive]
	hasJumpDrive := ok && jumpDrive.Value == 1.0

	groupedRows := make(map[attributeGroup][]attributeRow)

	for _, ag := range attributeGroups {
		attributeSelection := make([]*app.EveTypeDogmaAttribute, 0)
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
			v, substituteIcon := u.eus.FormatDogmaValue(ctx, value, o.DogmaAttribute.Unit)
			var iconID int32
			if substituteIcon != 0 {
				iconID = substituteIcon
			} else {
				iconID = o.DogmaAttribute.IconID
			}
			r, _ := eveicon.FromID(iconID)
			groupedRows[ag] = append(groupedRows[ag], attributeRow{
				icon:  r,
				label: o.DogmaAttribute.DisplayName,
				value: v,
			})
		}
	}
	rows := make([]attributeRow, 0)
	if et.Volume > 0 {
		v, _ := u.eus.FormatDogmaValue(ctx, et.Volume, app.EveUnitVolume)
		if et.Volume != et.PackagedVolume {
			v2, _ := u.eus.FormatDogmaValue(ctx, et.PackagedVolume, app.EveUnitVolume)
			v += fmt.Sprintf(" (%s Packaged)", v2)
		}
		r := attributeRow{
			icon:  eveicon.FromName(eveicon.Structure),
			label: "Volume",
			value: v,
		}
		var ag attributeGroup
		if len(groupedRows[attributeGroupStructure]) > 0 {
			ag = attributeGroupStructure
		} else {
			ag = attributeGroupMiscellaneous
		}
		groupedRows[ag] = append([]attributeRow{r}, groupedRows[ag]...)
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
				rows = append(rows, attributeRow{label: ag.DisplayName(), isTitle: true})
			}
			rows = append(rows, groupedRows[ag]...)
		}
	}
	if u.IsDeveloperMode() {
		rows = append(rows, attributeRow{label: "Developer Mode", isTitle: true})
		rows = append(rows, attributeRow{
			label: "EVE ID",
			value: fmt.Sprint(et.ID),
			action: func(v string) {
				u.App().Clipboard().SetContent(v)
			},
		})
	}
	return rows
}

func (a *inventoryTypeInfo) makeFittingTab(ctx context.Context, dogmaAttributes map[int32]*app.EveTypeDogmaAttribute) *container.TabItem {
	fittingData := a.calcFittingData(ctx, dogmaAttributes, a.iw.u)
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

func (*inventoryTypeInfo) calcFittingData(ctx context.Context, dogmaAttributes map[int32]*app.EveTypeDogmaAttribute, u *baseUI) []attributeRow {
	data := make([]attributeRow, 0)
	for _, da := range attributeGroupsMap[attributeGroupFitting] {
		o, ok := dogmaAttributes[da]
		if !ok {
			continue
		}
		iconID := o.DogmaAttribute.IconID
		r, _ := eveicon.FromID(iconID)
		v, _ := u.eus.FormatDogmaValue(ctx, o.Value, o.DogmaAttribute.Unit)
		data = append(data, attributeRow{
			icon:  r,
			label: o.DogmaAttribute.DisplayName,
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
				newSkillLevel(),
				widget.NewIcon(icons.QuestionmarkSvg),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			o := requiredSkills[id]
			row := co.(*fyne.Container).Objects
			skill := row[0].(*widget.Label)
			text := row[2].(*widget.Label)
			level := row[3].(*skillLevel)
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
	L:
		for {
			var items []attributeItem

			var averagePrice string
			p, err := a.iw.u.eus.MarketPrice(ctx, et.ID)
			if err != nil {
				slog.Error("average price", "typeID", et.ID, "error", err)
				averagePrice = "ERROR: " + a.iw.u.humanizeError(err)
			} else {
				averagePrice = p.StringFunc("?", func(v float64) string {
					return humanize.FormatFloat(priceFormat, v) + currencySuffix
				})
			}
			items = append(items, newAttributeItem("Average price", averagePrice))

			j, err := a.iw.u.js.FetchPrices(ctx, a.typeID)
			if err != nil {
				slog.Error("janice pricer", "typeID", et.ID, "error", err)
				s := "ERROR: " + a.iw.u.humanizeError(err)
				items = append(items, newAttributeItem("Janice prices", s))
			} else {
				items2 := []attributeItem{
					newAttributeItem("Jita sell price", humanize.FormatFloat(priceFormat, j.ImmediatePrices.SellPrice)+currencySuffix),
					newAttributeItem("Jita buy price", humanize.FormatFloat(priceFormat, j.ImmediatePrices.BuyPrice)+currencySuffix),
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
	typeID        int32
	activeLevel   int
	requiredLevel int
	trainedLevel  int
}

func (*inventoryTypeInfo) calcRequiredSkills(ctx context.Context, characterID int32, attributes map[int32]*app.EveTypeDogmaAttribute, u *baseUI) ([]requiredSkill, error) {
	skills := make([]requiredSkill, 0)
	skillAttributes := []struct {
		id    int32
		level int32
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
		typeID := int32(daID.Value)
		daLevel, ok := attributes[x.level]
		if !ok {
			continue
		}
		requiredLevel := int(daLevel.Value)
		et, err := u.eus.GetType(ctx, typeID)
		if err != nil {
			return nil, err
		}
		skill := requiredSkill{
			rank:          i + 1,
			requiredLevel: requiredLevel,
			name:          et.Name,
			typeID:        typeID,
		}
		cs, err := u.cs.GetSkill(ctx, characterID, typeID)
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
	iw      *infoWindow
	openURL func(*url.URL) error
}

func newAttributeList(iw *infoWindow, items ...attributeItem) *attributeList {
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
						w.iw.showEveEntity(x)
					}
				}
			case *app.EveLocation:
				if x != nil {
					f = func() {
						w.iw.showLocation(x.ID)
					}
				}
			case *app.EveRace:
				if x != nil {
					f = func() {
						w.iw.showRace(x.ID)
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
				w.iw.u.ShowSnackbar(fmt.Sprintf("ERROR: Failed to open URL: %s", w.iw.u.humanizeError(err)))
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
