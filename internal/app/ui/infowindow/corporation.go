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
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// corporationArea represents an area that shows public information about a character.
type corporationArea struct {
	Content      fyne.CanvasObject
	iw           InfoWindow
	alliance     *kxwidget.TappableLabel
	allianceLogo *canvas.Image
	name         *widget.Label
	logo         *canvas.Image
	hq           *kxwidget.TappableLabel
	tabs         *container.AppTabs
	w            fyne.Window
}

func newCorporationArea(iw InfoWindow, corporationID int32, w fyne.Window) *corporationArea {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Truncation = fyne.TextTruncateEllipsis
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Truncation = fyne.TextTruncateEllipsis
	s := float32(app.DefaultIconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(s))
	a := &corporationArea{
		alliance:     alliance,
		allianceLogo: iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(app.DefaultIconUnitSize)),
		name:         name,
		logo:         logo,
		hq:           hq,
		tabs:         container.NewAppTabs(),
		iw:           iw,
		w:            w,
	}
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
	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, main)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(corporationID)
		if err != nil {
			slog.Error("corporation info update failed", "corporation", corporationID, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *corporationArea) load(corporationID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.eis.CorporationLogo(corporationID, app.DefaultIconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", corporationID, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
	o, err := a.iw.eus.GetEveCorporationESI(ctx, corporationID)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	if o.Alliance != nil {
		a.alliance.SetText("Member of " + o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(o.Alliance)
		}
		go func() {
			r, err := a.iw.eis.AllianceLogo(o.Alliance.ID, app.DefaultIconPixelSize)
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
	attributes := make([]AtributeItem, 0)
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
		if err == nil {
			attributes = append(attributes, NewAtributeItem("URL", u))
		}
	}
	attributeList := NewAttributeList()
	attributeList.ShowInfoWindow = a.iw.ShowEveEntity
	attributeList.Set(attributes)
	attributesTab := container.NewTabItem("Attributes", attributeList)
	a.tabs.Append(attributesTab)
	a.tabs.Select(attributesTab)
	go func() {
		history, err := a.iw.eus.GetCorporationAllianceHistory(ctx, corporationID)
		if err != nil {
			slog.Error("corporation info: Failed to load alliance history", "corporationID", corporationID, "error", err)
			return
		}
		if len(history) == 0 {
			return
		}
		history2 := slices.Collect(xiter.FilterSlice(history, func(v app.MembershipHistoryItem) bool {
			return v.Organization != nil
		}))
		items := slices.Collect(xiter.MapSlice(history2, historyItem2EntityItem))
		oldest := slices.MinFunc(history, func(a, b app.MembershipHistoryItem) int {
			return a.StartDate.Compare(b.StartDate)
		})
		items = append(items, NewEntityItem(
			0,
			"Corporation Founded",
			fmt.Sprintf("**%s**", oldest.StartDate.Format(app.DateTimeDefaultFormat)),
			None,
		))
		historyList := NewEntityListFromItems(a.iw.Show, items...)
		a.tabs.Append(container.NewTabItem("Alliance History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
