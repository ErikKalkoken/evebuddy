package infowindow

import (
	"context"
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

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

func newCorporationInfo(iw *infoWindow, id int64) *corporationInfo {
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
			a.iw.showEveEntity(v)
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
			a.iw.showEveEntity(v)
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
