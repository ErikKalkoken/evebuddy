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
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// allianceInfoArea represents an area that shows public information about a character.
type allianceInfoArea struct {
	Content fyne.CanvasObject

	hq   *kxwidget.TappableLabel
	iw   InfoWindow
	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
}

func newAllianceInfoArea(iw InfoWindow, allianceID int32) *allianceInfoArea {
	alliance := widget.NewLabel("")
	alliance.Truncation = fyne.TextTruncateEllipsis
	corporation := widget.NewLabel("Loading...")
	corporation.Truncation = fyne.TextTruncateEllipsis
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Truncation = fyne.TextTruncateEllipsis
	corporationLogo := iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(defaultIconUnitSize))
	s := float32(defaultIconPixelSize) * logoZoomFactor
	corporationLogo.SetMinSize(fyne.NewSquareSize(s))
	a := &allianceInfoArea{
		iw:   iw,
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

func (a *allianceInfoArea) load(allianceID int32) error {
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
					a.iw.ShowEveEntity(m)
				}
			},
		)
		a.tabs.Append(container.NewTabItem("Members", memberList))
		a.tabs.Refresh()
	}()
	a.tabs.Refresh()
	return nil
}
