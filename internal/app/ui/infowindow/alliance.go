package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget

	attributes *attributeList
	dotlan     *kxwidget.TappableIcon
	hq         *kxwidget.TappableLabel
	id         int32
	iw         *InfoWindow
	logo       *canvas.Image
	members    *entityList
	name       *widget.Label
	tabs       *container.AppTabs
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
	}
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.members = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Attributes", a.attributes),
		container.NewTabItem("Members", a.members),
	)
	a.dotlan = kxwidget.NewTappableIcon(icons.DotlanAvatarPng, nil)
	a.dotlan.Hide()
	return a
}

func (a *allianceInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("alliance info update failed", "alliance", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load alliance: %s", a.iw.u.ErrorDisplay(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			a.logo,
			container.NewHBox(
				layout.NewSpacer(),
				kxwidget.NewTappableIcon(icons.ZkillboardPng, func() {
					a.iw.openURL(fmt.Sprintf("https://zkillboard.com/alliance/%d/", a.id))
				}),
				a.dotlan,
				layout.NewSpacer(),
			),
		),
		nil,
		a.name,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *allianceInfo) load() error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.EveImageService().AllianceLogo(a.id, app.IconPixelSize)
		if err != nil {
			slog.Error("alliance info: Failed to load logo", "allianceID", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	}()

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
		fyne.Do(func() {
			a.members.set(entityItemsFromEveEntities(members)...)
		})
	}()
	o, err := a.iw.u.EveUniverseService().GetAllianceESI(ctx, a.id)
	if err != nil {
		return err
	}

	// Attributes
	attributes := make([]attributeItem, 0)
	if o.ExecutorCorporation != nil {
		attributes = append(attributes, newAttributeItem("Executor", o.ExecutorCorporation))
	}
	if o.Ticker != "" {
		attributes = append(attributes, newAttributeItem("Short Name", o.Ticker))
	}
	if o.CreatorCorporation != nil {
		attributes = append(attributes, newAttributeItem("Created By Corporation", o.CreatorCorporation))
	}
	if o.Creator != nil {
		attributes = append(attributes, newAttributeItem("Created By", o.Creator))
	}
	if !o.DateFounded.IsZero() {
		attributes = append(attributes, newAttributeItem("Start Date", o.DateFounded))
	}
	if o.Faction != nil {
		attributes = append(attributes, newAttributeItem("Faction", o.Faction))
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.attributes.set(attributes)
		a.dotlan.OnTapped = func() {
			name := strings.ReplaceAll(o.Name, "_", " ")
			a.iw.openURL(fmt.Sprintf("https://evemaps.dotlan.net/alliance/%s", name))
		}
		a.dotlan.Show()
	})
	return nil
}
