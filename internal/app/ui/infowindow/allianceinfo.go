package infowindow

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// allianceArea represents an area that shows public information about a character.
type allianceArea struct {
	Content fyne.CanvasObject

	hq   *kxwidget.TappableLabel
	iw   InfoWindow
	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
	w    fyne.Window
}

func newAlliancArea(iw InfoWindow, alliance *app.EveEntity, w fyne.Window) *allianceArea {
	name := widget.NewLabel(alliance.Name)
	name.Truncation = fyne.TextTruncateEllipsis
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Truncation = fyne.TextTruncateEllipsis
	logo := iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(defaultIconUnitSize))
	s := float32(defaultIconPixelSize) * logoZoomFactor
	logo.SetMinSize(fyne.NewSquareSize(s))
	a := &allianceArea{
		iw:   iw,
		name: name,
		logo: logo,
		hq:   hq,
		tabs: container.NewAppTabs(),
		w:    w,
	}

	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, a.name)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(alliance.ID)
		if err != nil {
			slog.Error("alliance info update failed", "alliance", alliance, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load alliance: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *allianceArea) load(allianceID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.eis.AllianceLogo(allianceID, defaultIconPixelSize)
		if err != nil {
			slog.Error("alliance info: Failed to load logo", "allianceID", allianceID, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
	o, err := a.iw.eus.GetEveAllianceESI(ctx, allianceID)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)

	// Attributes
	attributes := make([]AtributeItem, 0)
	if o.ExecutorCorporation != nil {
		attributes = append(attributes, NewAtributeItem("Executor", o.ExecutorCorporation))
	}
	if o.Ticker != "" {
		attributes = append(attributes, NewAtributeItem("Short Name", o.Ticker))
	}
	if o.CreatorCorporation != nil {
		attributes = append(attributes, NewAtributeItem("Created By Corporation", o.CreatorCorporation))
	}
	if o.Creator != nil {
		attributes = append(attributes, NewAtributeItem("Created By", o.Creator))
	}
	if !o.DateFounded.IsZero() {
		attributes = append(attributes, NewAtributeItem("Start Date", o.DateFounded))
	}
	if o.Faction != nil {
		attributes = append(attributes, NewAtributeItem("Faction", o.Faction))
	}
	attributeList := NewAttributeList()
	attributeList.ShowInfoWindow = a.iw.ShowEveEntity
	attributeList.Set(attributes)
	a.tabs.Append(container.NewTabItem("Attributes", attributeList))

	// Members
	go func() {
		members, err := a.iw.eus.GetEveAllianceCorporationsESI(ctx, allianceID)
		if err != nil {
			slog.Error("alliance info: Failed to load corporations", "allianceID", allianceID, "error", err)
			return
		}
		if len(members) == 0 {
			return
		}
		memberList := NewEntityListFromEntities(members...)
		memberList.ShowEveEntity = a.iw.ShowEveEntity
		a.tabs.Append(container.NewTabItem("Members", memberList))
		a.tabs.Refresh()
	}()
	a.tabs.Refresh()
	return nil
}
