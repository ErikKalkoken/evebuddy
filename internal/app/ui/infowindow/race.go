package infowindow

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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
)

type raceInfo struct {
	widget.BaseWidget

	id int32
	iw *InfoWindow

	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
}

func newRaceInfo(iw *InfoWindow, id int32) *raceInfo {
	a := &raceInfo{
		iw:   iw,
		id:   id,
		logo: makeInfoLogo(),
		name: makeInfoName(),
		tabs: container.NewAppTabs(),
	}
	a.logo.Resource = icons.BlankSvg
	a.ExtendBaseWidget(a)
	return a
}

func (a *raceInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("race info update failed", "race", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load race: %s", a.iw.u.ErrorDisplay(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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

func (a *raceInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateRaceESI(ctx, a.id)
	if err != nil {
		return err
	}
	factionID, found := o.FactionID()
	if found {
		go func() {
			r, err := a.iw.u.EveImageService().FactionLogo(factionID, app.IconPixelSize)
			if err != nil {
				slog.Error("race info: Failed to load logo", "corporationID", a.id, "error", err)
				return
			}
			fyne.Do(func() {
				a.logo.Resource = r
				a.logo.Refresh()
			})
		}()
	}
	fyne.Do(func() {
		desc := widget.NewLabel(o.Description)
		desc.Wrapping = fyne.TextWrapWord
		a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(desc)))
		a.name.SetText(o.Name)
	})
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			a.iw.u.App().Clipboard().SetContent(v.(string))
		}
		attributeList := NewAttributeList(a.iw, []AttributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
		})
	}
	return nil
}
