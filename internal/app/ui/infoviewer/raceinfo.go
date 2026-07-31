package infoviewer

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
	"github.com/ErikKalkoken/evebuddy/internal/icons"
)

type raceInfo struct {
	widget.BaseWidget
	baseInfo

	id          int64
	logo        *canvas.Image
	tabs        *container.AppTabs
	description *widget.Label
}

func newRaceInfo(iw *InfoViewer, id int64) *raceInfo {
	a := &raceInfo{
		description: newLabelWithWrapAndSelectable(""),
		id:          id,
		logo:        makeInfoLogo(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.BlankSvg
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
	)
	return a
}

func (a *raceInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *raceInfo) update(ctx context.Context) error {
	o, err := a.iw.u.EVEUniverse().GetOrCreateRaceESI(ctx, a.id)
	if err != nil {
		return err
	}
	if factionID, ok := o.FactionID(); ok {
		fyne.Do(func() {
			a.iw.u.EVEImage().FactionLogoAsync(factionID, ui.IconPixelSize, func(r fyne.Resource) {
				a.logo.Resource = r
				a.logo.Refresh()
			})
		})
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			fyne.Do(func() {
				a.tabs.Append(attributesTab)
			})
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.description.SetText(o.Description)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}
