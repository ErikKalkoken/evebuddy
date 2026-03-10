package infowindow

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// allianceInfo shows public information about a character.
type allianceInfo struct {
	widget.BaseWidget
	baseInfo

	attributes *attributeList
	hq         *widget.Hyperlink
	id         int64
	logo       *canvas.Image
	members    *entityList
	tabs       *container.AppTabs
}

func newAllianceInfo(iw *InfoWindow, id int64) *allianceInfo {
	hq := widget.NewHyperlink("", nil)
	hq.Wrapping = fyne.TextWrapWord
	a := &allianceInfo{
		id:   id,
		logo: makeInfoLogo(),
		hq:   hq,
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.attributes = newAttributeList(a.iw)
	a.members = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Attributes", a.attributes),
		container.NewTabItem("Members", a.members),
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
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoAlliance),
				a.iw.makeDotlanIcon(a.id, infoAlliance),
				a.iw.makeEveWhoIcon(a.id, infoAlliance),
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
	g := new(errgroup.Group)
	g.Go(func() error {
		o, err := a.iw.u.EVEUniverse().FetchAlliance(ctx, a.id)
		if err != nil {
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
			x := newAttributeItem("EVE ID", o.ID)
			x.Action = func(_ any) {
				fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
			}
			attributes = append(attributes, x)
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.attributes.set(attributes)
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		members, err := a.iw.u.EVEUniverse().FetchAllianceCorporations(ctx, a.id)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return nil
		}
		fyne.Do(func() {
			a.members.set(entityItemsFromEveEntities(members)...)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}
