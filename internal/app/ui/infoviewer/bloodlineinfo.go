package infoviewer

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
)

type eveBloodlineShort app.EntityShort

type bloodlineInfo struct {
	widget.BaseWidget
	baseInfo

	id          int64
	logo        *canvas.Image
	tabs        *container.AppTabs
	description *widget.Label
}

func newBloodlineInfo(iw *InfoViewer, id int64) *bloodlineInfo {
	a := &bloodlineInfo{
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

func (a *bloodlineInfo) CreateRenderer() fyne.WidgetRenderer {
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

func (a *bloodlineInfo) update(ctx context.Context) error {
	o, err := a.iw.u.EVEUniverse().GetOrCreateBloodlineESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.Description)
	})
	if r, ok := o.Logo(); ok {
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	}
	attributes := []attributeItem{
		newAttributeItem("Race", o.Race),
		newAttributeItem("Starter corporation", o.Corporation),
	}
	if typeID, ok := o.ShipTypeID.Value(); ok {
		et, err := a.iw.u.EVEUniverse().GetOrCreateTypeESI(ctx, typeID)
		if err != nil {
			slog.Error("Failed to load ship type for bloodline", "typeID", typeID, "bloodlineID", o.ID)
		} else {
			attributes = append(attributes, newAttributeItem("Starter ship", et))
		}
	}
	if a.iw.u.IsDeveloperMode() {
		attributes = append(attributes, newAttributeItem("EVE ID", fmt.Sprint(o.ID)))
	}
	attributesTab := container.NewTabItem("Attributes", newAttributeList(a.iw, attributes...))
	fyne.Do(func() {
		a.tabs.Append(attributesTab)
	})

	// 	if a.iw.u.IsDeveloperMode() {
	// 		x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
	// 		x.Action = func(v any) {
	// 			fyne.CurrentApp().Clipboard().SetContent(v.(string))
	// 		}
	// 	}
	// 	fyne.Do(func() {
	// 		a.name.SetText(o.Name)
	// 		a.description.SetText(o.Description)
	// 		a.tabs.Refresh()
	// 	})
	// 	return nil
	// })
	return nil
}
