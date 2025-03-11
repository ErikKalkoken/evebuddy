package ui

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
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// CorporationInfoArea represents an area that shows public information about a character.
type CorporationInfoArea struct {
	Content fyne.CanvasObject

	alliance        *kxwidget.TappableLabel
	allianceLogo    *canvas.Image
	corporation     *widget.Label
	corporationLogo *canvas.Image
	hq              *kxwidget.TappableLabel
	tabs            *container.AppTabs
	u               *BaseUI
}

func NewCorporationInfoArea(u *BaseUI, corporationID int32) *CorporationInfoArea {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Truncation = fyne.TextTruncateEllipsis
	corporation := widget.NewLabel("Loading...")
	corporation.Truncation = fyne.TextTruncateEllipsis
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Truncation = fyne.TextTruncateEllipsis
	corporationLogo := iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize))
	s := float32(DefaultIconPixelSize) * 1.3 / u.Window.Canvas().Scale()
	corporationLogo.SetMinSize(fyne.NewSquareSize(s))
	a := &CorporationInfoArea{
		alliance:        alliance,
		allianceLogo:    iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize)),
		corporation:     corporation,
		corporationLogo: corporationLogo,
		hq:              hq,
		tabs:            container.NewAppTabs(),
		u:               u,
	}

	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.corporation,
		a.hq,
		container.NewBorder(
			nil,
			nil,
			a.allianceLogo,
			nil,
			a.alliance,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.corporationLogo), nil, main)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(corporationID)
		if err != nil {
			slog.Error("corporation info update failed", "id", corporationID, "error", err)
			a.corporation.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", ihumanize.Error(err))
			a.corporation.Importance = widget.DangerImportance
			a.corporation.Refresh()
		}
	}()
	return a
}

func (a *CorporationInfoArea) load(corporationID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.u.EveImageService.CorporationLogo(corporationID, DefaultIconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", corporationID, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	o, err := a.u.EveUniverseService.GetEveCorporationESI(ctx, corporationID)
	if err != nil {
		return err
	}
	a.corporation.SetText(o.Name)
	if o.Alliance != nil {
		a.alliance.SetText("Member of " + o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.u.ShowInfoWindow(o.Alliance)
		}
		go func() {
			r, err := a.u.EveImageService.AllianceLogo(o.Alliance.ID, DefaultIconPixelSize)
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
			a.u.ShowInfoWindow(o.HomeStation)
		}
	} else {
		a.hq.Hide()
	}
	attributes := make([]appwidget.AtributeItem, 0)
	if o.Ceo != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("CEO", o.Ceo))
	}
	if o.Creator != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Founder", o.Creator))
	}
	if o.Alliance != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Alliance", o.Alliance))
	}
	if o.Ticker != "" {
		attributes = append(attributes, appwidget.NewAtributeItem("Ticker Name", o.Ticker))
	}
	if o.Faction != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Faction", o.Faction))
	}
	if o.Shares != 0 {
		attributes = append(attributes, appwidget.NewAtributeItem("Shares", o.Shares))
	}
	if o.MemberCount != 0 {
		attributes = append(attributes, appwidget.NewAtributeItem("Member Count", o.MemberCount))
	}
	if o.TaxRate != 0 {
		attributes = append(attributes, appwidget.NewAtributeItem("ISK Tax Rate", o.TaxRate))
	}
	attributes = append(attributes, appwidget.NewAtributeItem("War Eligability", o.WarEligible))
	if o.URL != "" {
		u, err := url.ParseRequestURI(o.URL)
		if err == nil {
			attributes = append(attributes, appwidget.NewAtributeItem("URL", u))
		}
	}
	attributeList := appwidget.NewAttributeList()
	attributeList.ShowInfoWindow = a.u.ShowInfoWindow
	attributeList.OpenURL = a.u.FyneApp.OpenURL
	attributeList.Set(attributes)
	a.tabs.Append(container.NewTabItem("Attributes", attributeList))
	a.tabs.Refresh()
	go func() {
		history, err := a.u.EveUniverseService.GetCorporationAllianceHistory(ctx, corporationID)
		if err != nil {
			slog.Error("corporation info: Failed to load alliance history", "corporationID", corporationID, "error", err)
			return
		}
		if len(history) == 0 {
			return
		}
		historyList := appwidget.NewMembershipHistoryList()
		historyList.IsFoundedShown = true
		historyList.ShowInfoWindow = a.u.ShowInfoWindow
		historyList.Set(slices.Collect(xiter.FilterSlice(history, func(v app.MembershipHistoryItem) bool {
			return v.Organization != nil || v.IsOldest
		})))
		a.tabs.Append(container.NewTabItem("Alliance History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
