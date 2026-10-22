package infoviewer

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget
	baseInfo

	attributes *attributeList
	id         int64
	logo       *canvas.Image
	members    *entityList
	tabs       *container.AppTabs
}

func newAllianceInfo(iw *InfoViewer, id int64) *allianceInfo {
	a := &allianceInfo{
		id:   id,
		logo: makeInfoLogo(),
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.members = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		ui.NewTabItem("Attributes", a.attributes),
		ui.NewTabItem("Members", a.members),
	)
	return a
}

func (a *allianceInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	top := container.NewBorder(
		nil,
		nil,
		container.New(
			layout.NewCustomPaddedVBoxLayout(2*p),
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(linkIconPaddingFactor*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, Alliance),
				a.iw.makeDotlanIcon(a.id, Alliance),
				a.iw.makeEveWhoIcon(a.id, Alliance),
				layout.NewSpacer(),
			),
		),
		nil,
		a.name,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *allianceInfo) update(ctx context.Context) error {
	fyne.Do(func() {
		a.iw.u.EVEImage().AllianceLogoAsync(a.id, ui.IconPixelSize, func(r fyne.Resource) {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	})
	var o *app.EveAlliance
	var members []*app.EveEntity
	g := new(errgroup.Group)
	g.Go(func() error {
		var err error
		o, err = a.iw.u.EVEUniverse().FetchAlliance(ctx, a.id)
		return err
	})
	g.Go(func() error {
		var err error
		members, err = a.iw.u.EVEUniverse().FetchAllianceCorporations(ctx, a.id)
		return err
	})
	if err := g.Wait(); err != nil {
		return err
	}
	// Attributes
	var attributes []attributeItem
	if v, ok := o.ExecutorCorporation.Value(); ok {
		attributes = append(attributes, newAttributeItem("Executor", v))
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
	if v, ok := o.Faction.Value(); ok {
		attributes = append(attributes, newAttributeItem("Faction", v))
	}
	if a.iw.u.IsDeveloperMode() {
		attributes = append(attributes, newEVEIDAttributeItem(o.ID))
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.attributes.set(attributes)
		if len(members) > 0 {
			a.members.set(entityItemsFromEveEntities(members)...)
		}
		a.tabs.Refresh()
	})
	return nil
}
