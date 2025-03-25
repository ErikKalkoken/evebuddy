package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"slices"

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

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// corporationInfo shows public information about a character.
type corporationInfo struct {
	widget.BaseWidget

	id           int32
	iw           *InfoWindow
	alliance     *kxwidget.TappableLabel
	allianceLogo *canvas.Image
	name         *widget.Label
	logo         *canvas.Image
	hq           *kxwidget.TappableLabel
	tabs         *container.AppTabs
}

func newCorporationInfo(iw *InfoWindow, id int32) *corporationInfo {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Wrapping = fyne.TextWrapWord
	a := &corporationInfo{
		id:           id,
		alliance:     alliance,
		allianceLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		name:         makeInfoName(),
		logo:         makeInfoLogo(),
		hq:           hq,
		tabs:         container.NewAppTabs(),
		iw:           iw,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *corporationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("corporation info update failed", "corporation", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
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
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetCorporationESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(a.id, app.IconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", a.id, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
	if o.Alliance != nil {
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
			a.allianceLogo.Resource = r
			a.allianceLogo.Refresh()
		}()
	} else {
		a.alliance.Hide()
		a.allianceLogo.Hide()
	}
	desc := o.DescriptionPlain()
	if desc != "" {
		description := widget.NewLabel(desc)
		description.Wrapping = fyne.TextWrapWord
		a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(description)))
	}
	if o.HomeStation != nil {
		a.hq.SetText("Headquarters: " + o.HomeStation.Name)
		a.hq.OnTapped = func() {
			a.iw.ShowEveEntity(o.HomeStation)
		}
	} else {
		a.hq.Hide()
	}
	attributes := make([]AttributeItem, 0)
	if o.Ceo != nil {
		attributes = append(attributes, NewAtributeItem("CEO", o.Ceo))
	}
	if o.Creator != nil {
		attributes = append(attributes, NewAtributeItem("Founder", o.Creator))
	}
	if o.Alliance != nil {
		attributes = append(attributes, NewAtributeItem("Alliance", o.Alliance))
	}
	if o.Ticker != "" {
		attributes = append(attributes, NewAtributeItem("Ticker Name", o.Ticker))
	}
	if o.Faction != nil {
		attributes = append(attributes, NewAtributeItem("Faction", o.Faction))
	}
	if o.Shares != 0 {
		attributes = append(attributes, NewAtributeItem("Shares", o.Shares))
	}
	if o.MemberCount != 0 {
		attributes = append(attributes, NewAtributeItem("Member Count", o.MemberCount))
	}
	if o.TaxRate != 0 {
		attributes = append(attributes, NewAtributeItem("ISK Tax Rate", o.TaxRate))
	}
	attributes = append(attributes, NewAtributeItem("War Eligability", o.WarEligible))
	if o.URL != "" {
		u, err := url.ParseRequestURI(o.URL)
		if err == nil && u.Host != "" {
			attributes = append(attributes, NewAtributeItem("URL", u))
		}
	}
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	attributeList := NewAttributeList(attributes...)
	attributeList.ShowInfoWindow = a.iw.ShowEveEntity
	attributesTab := container.NewTabItem("Attributes", attributeList)
	a.tabs.Append(attributesTab)
	a.tabs.Select(attributesTab)
	go func() {
		history, err := a.iw.u.EveUniverseService().GetCorporationAllianceHistory(ctx, a.id)
		if err != nil {
			slog.Error("corporation info: Failed to load alliance history", "corporationID", a.id, "error", err)
			return
		}
		if len(history) == 0 {
			return
		}
		history2 := xslices.Filter(history, func(v app.MembershipHistoryItem) bool {
			return v.Organization != nil && v.Organization.Category.IsKnown()
		})
		items := xslices.Map(history2, historyItem2EntityItem)
		oldest := slices.MinFunc(history, func(a, b app.MembershipHistoryItem) int {
			return a.StartDate.Compare(b.StartDate)
		})
		items = append(items, NewEntityItem(
			0,
			"Corporation Founded",
			fmt.Sprintf("**%s**", oldest.StartDate.Format(app.DateFormat)),
			infoNotSupported,
		))
		historyList := NewEntityListFromItems(a.iw.show, items...)
		a.tabs.Append(container.NewTabItem("Alliance History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
