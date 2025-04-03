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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget

	id   int32
	hq   *kxwidget.TappableLabel
	iw   *InfoWindow
	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
}

func newAllianceInfo(iw *InfoWindow, id int32) *allianceInfo {
	hq := kxwidget.NewTappableLabel("", nil)
	hq.Wrapping = fyne.TextWrapWord
	a := &allianceInfo{
		iw:   iw,
		id:   id,
		name: makeInfoName(),
		logo: makeInfoLogo(),
		hq:   hq,
		tabs: container.NewAppTabs(),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *allianceInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("alliance info update failed", "alliance", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load alliance: %s", a.iw.u.ErrorDisplay(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, a.name)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *allianceInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetAllianceESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	go func() {
		r, err := a.iw.u.EveImageService().AllianceLogo(a.id, app.IconPixelSize)
		if err != nil {
			slog.Error("alliance info: Failed to load logo", "allianceID", a.id, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
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
			a.iw.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	attributeList := NewAttributeList(a.iw, attributes...)
	a.tabs.Append(container.NewTabItem("Attributes", attributeList))

	// Members
	go func() {
		members, err := a.iw.u.EveUniverseService().GetAllianceCorporationsESI(ctx, a.id)
		if err != nil {
			slog.Error("alliance info: Failed to load corporations", "allianceID", a.id, "error", err)
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
