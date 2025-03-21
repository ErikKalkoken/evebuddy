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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget

	id   int32
	hq   *kxwidget.TappableLabel
	iw   InfoWindow
	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
	w    fyne.Window
}

func newAllianceInfo(iw InfoWindow, allianceID int32, w fyne.Window) *allianceInfo {
	name := widget.NewLabel("")
	name.Wrapping = fyne.TextWrapWord
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Wrapping = fyne.TextWrapWord
	s := float32(app.IconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(s))
	a := &allianceInfo{
		iw:   iw,
		id:   allianceID,
		name: name,
		logo: logo,
		hq:   hq,
		tabs: container.NewAppTabs(),
		w:    w,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *allianceInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load(a.id)
		if err != nil {
			slog.Error("alliance info update failed", "alliance", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load alliance: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, a.name)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *allianceInfo) load(allianceID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.EveImageService().AllianceLogo(allianceID, app.IconPixelSize)
		if err != nil {
			slog.Error("alliance info: Failed to load logo", "allianceID", allianceID, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
	o, err := a.iw.u.EveUniverseService().GetAllianceESI(ctx, allianceID)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)

	// Attributes
	attributes := make([]AttributeItem, 0)
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
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	attributeList := NewAttributeList(attributes...)
	attributeList.ShowInfoWindow = a.iw.ShowEveEntity
	a.tabs.Append(container.NewTabItem("Attributes", attributeList))

	// Members
	go func() {
		members, err := a.iw.u.EveUniverseService().GetAllianceCorporationsESI(ctx, allianceID)
		if err != nil {
			slog.Error("alliance info: Failed to load corporations", "allianceID", allianceID, "error", err)
			return
		}
		if len(members) == 0 {
			return
		}
		memberList := NewEntityListFromEntities(a.iw.show, members...)
		a.tabs.Append(container.NewTabItem("Members", memberList))
		a.tabs.Refresh()
	}()
	a.tabs.Refresh()
	return nil
}
