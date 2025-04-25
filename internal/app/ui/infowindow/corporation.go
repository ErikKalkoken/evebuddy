package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// corporationInfo shows public information about a character.
type corporationInfo struct {
	widget.BaseWidget

	alliance        *kxwidget.TappableLabel
	allianceHistory *entityList
	allianceLogo    *canvas.Image
	attributes      *attributeList
	description     *widget.Label
	dotlan          *kxwidget.TappableIcon
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
	a.dotlan = kxwidget.NewTappableIcon(icons.DotlanAvatarPng, nil)
	a.dotlan.Hide()
	return a
}

func (a *corporationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("corporation info update failed", "corporation", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", a.iw.u.ErrorDisplay(err))
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
			a.logo,
			container.NewHBox(
				layout.NewSpacer(),
				kxwidget.NewTappableIcon(icons.ZkillboardPng, func() {
					a.iw.openURL(fmt.Sprintf("https://zkillboard.com/corporation/%d/", a.id))
				}),
				a.dotlan,
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
		r, err := a.iw.u.EveImageService().CorporationLogo(a.id, app.IconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	}()
	o, err := a.iw.u.EveUniverseService().GetCorporationESI(ctx, a.id)
	if err != nil {
		return err
	}
	attributes := a.makeAttributes(o)
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.DescriptionPlain())
		a.attributes.set(attributes)
		a.dotlan.OnTapped = func() {
			name := strings.ReplaceAll(o.Name, "_", " ")
			a.iw.openURL(fmt.Sprintf("https://evemaps.dotlan.net/corp/%s", name))
		}
		a.dotlan.Show()
	})
	fyne.Do(func() {
		if o.Alliance == nil {
			a.alliance.Hide()
			a.allianceLogo.Hide()
			return
		}
		a.alliance.SetText("Member of " + o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(o.Alliance)
		}
		go func() {
			r, err := a.iw.u.EveImageService().AllianceLogo(o.Alliance.ID, app.IconPixelSize)
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
			a.iw.ShowEveEntity(o.HomeStation)
		}
	})
	go func() {
		history, err := a.iw.u.EveUniverseService().GetCorporationAllianceHistory(ctx, a.id)
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
