package infowindow

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// AllianceInfoArea represents an area that shows public information about a character.
type AllianceInfoArea struct {
	Content fyne.CanvasObject

	eus            *eveuniverse.EveUniverseService
	eis            app.EveImageService
	showInfoWindow func(*app.EveEntity)

	name *widget.Label
	logo *canvas.Image
	hq   *kxwidget.TappableLabel
	tabs *container.AppTabs
}

func NewAllianceInfoArea(
	eus *eveuniverse.EveUniverseService,
	eis app.EveImageService,
	showInfoWindow func(*app.EveEntity),
	allianceID int32,
) *AllianceInfoArea {
	alliance := widget.NewLabel("")
	alliance.Truncation = fyne.TextTruncateEllipsis
	corporation := widget.NewLabel("Loading...")
	corporation.Truncation = fyne.TextTruncateEllipsis
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Truncation = fyne.TextTruncateEllipsis
	corporationLogo := iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(defaultIconUnitSize))
	s := float32(defaultIconPixelSize) * logoZoomFactor
	corporationLogo.SetMinSize(fyne.NewSquareSize(s))
	a := &AllianceInfoArea{
		eis:            eis,
		eus:            eus,
		showInfoWindow: showInfoWindow,

		name: corporation,
		logo: corporationLogo,
		hq:   hq,
		tabs: container.NewAppTabs(),
	}

	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, a.name)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(allianceID)
		if err != nil {
			slog.Error("alliance info update failed", "id", allianceID, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load alliance: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *AllianceInfoArea) load(allianceID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.eis.AllianceLogo(allianceID, defaultIconPixelSize)
		if err != nil {
			slog.Error("alliance info: Failed to load logo", "allianceID", allianceID, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
	o, err := a.eus.GetEveAllianceESI(ctx, allianceID)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)

	// Attributes
	attributes := make([]appwidget.AtributeItem, 0)
	if o.ExecutorCorporation != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Executor", o.ExecutorCorporation))
	}
	if o.Ticker != "" {
		attributes = append(attributes, appwidget.NewAtributeItem("Short Name", o.Ticker))
	}
	if o.CreatorCorporation != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Created By Corporation", o.CreatorCorporation))
	}
	if o.Creator != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Created By", o.Creator))
	}
	if !o.DateFounded.IsZero() {
		attributes = append(attributes, appwidget.NewAtributeItem("Start Date", o.DateFounded))
	}
	if o.Faction != nil {
		attributes = append(attributes, appwidget.NewAtributeItem("Faction", o.Faction))
	}
	attributeList := appwidget.NewAttributeList()
	attributeList.ShowInfoWindow = a.showInfoWindow
	attributeList.Set(attributes)
	a.tabs.Append(container.NewTabItem("Attributes", attributeList))

	// Members
	go func() {
		members, err := a.eus.GetEveAllianceCorporationsESI(ctx, allianceID)
		if err != nil {
			slog.Error("alliance info: Failed to load corporations", "allianceID", allianceID, "error", err)
			return
		}
		if len(members) == 0 {
			return
		}
		memberList := widget.NewList(
			func() int {
				return len(members)
			},
			func() fyne.CanvasObject {
				name := kxwidget.NewTappableLabel("Template", nil)
				name.Truncation = fyne.TextTruncateEllipsis
				return container.NewBorder(nil, nil, nil, widget.NewIcon(theme.InfoIcon()), name)
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				if id >= len(members) {
					return
				}
				m := members[id]
				border := co.(*fyne.Container).Objects
				l := border[0].(*kxwidget.TappableLabel)
				l.SetText(m.Name)
				l.OnTapped = func() {
					a.showInfoWindow(m)
				}
			},
		)
		a.tabs.Append(container.NewTabItem("Members", memberList))
		a.tabs.Refresh()
	}()
	a.tabs.Refresh()
	return nil
}
