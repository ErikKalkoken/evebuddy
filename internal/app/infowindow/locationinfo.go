package infowindow

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// locationInfo shows public information about a character.
type locationInfo struct {
	widget.BaseWidget
	baseInfo

	description *widget.Label
	id          int64
	location    *entityList
	owner       *widget.Hyperlink
	ownerLogo   *canvas.Image
	services    *entityList
	tabs        *container.AppTabs
	typeImage   *xwidget.TappableImage
	typeInfo    *widget.Hyperlink
}

func newLocationInfo(iw *InfoWindow, id int64) *locationInfo {
	typeInfo := widget.NewHyperlink("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := widget.NewHyperlink("", nil)
	owner.Wrapping = fyne.TextWrapWord
	typeImage := xwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(iw.renderIconSize())
	a := &locationInfo{
		description: newLabelWithWrapAndSelectable(""),
		id:          id,
		owner:       owner,
		ownerLogo:   xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		typeImage:   typeImage,
		typeInfo:    typeInfo,
	}
	a.ExtendBaseWidget(a)
	a.initBase(iw)
	a.location = newEntityList(a.iw.show)
	location := container.NewTabItem("Location", a.location)
	a.services = newEntityList(a.iw.show)
	services := container.NewTabItem("Services", a.services)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		location,
		services,
	)
	a.tabs.Select(location)
	return a
}

func (a *locationInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.typeInfo,
		),
		container.NewBorder(
			nil,
			nil,
			a.ownerLogo,
			nil,
			a.owner,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.typeImage), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *locationInfo) update(ctx context.Context) error {
	o, err := a.iw.s.EVEUniverse().GetOrCreateLocationESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
	})
	if et, ok := o.Type.Value(); ok {
		fyne.Do(func() {
			a.iw.s.EVEImage().InventoryTypeRenderAsync(et.ID, renderIconPixelSize, func(r fyne.Resource) {
				a.typeImage.SetResource(r)
			})
			a.typeInfo.SetText(et.Name)
			a.typeInfo.OnTapped = func() {
				a.iw.ShowEntity(et.EveEntity())
			}
			a.typeImage.OnTapped = func() {
				a.iw.showZoomWindow(o.Name, et.ID, a.iw.s.EVEImage().InventoryTypeRenderAsync, a.iw.w)
			}
			description := et.Description
			if description == "" {
				description = et.Name
			}
			a.description.SetText(description)
		})
	}
	if v, ok := o.Owner.Value(); ok {
		fyne.Do(func() {
			a.iw.s.EVEImage().CorporationLogoAsync(v.ID, app.IconPixelSize, func(r fyne.Resource) {
				a.ownerLogo.Resource = r
				a.ownerLogo.Refresh()
			})
			a.owner.SetText(v.Name)
			a.owner.OnTapped = func() {
				a.iw.ShowEntity(v)
			}
		})
	}
	if es, ok := o.SolarSystem.Value(); ok {
		fyne.Do(func() {
			a.location.set(
				newEntityItemFromEveEntityWithText(es.Constellation.Region.EveEntity(), ""),
				newEntityItemFromEveEntityWithText(es.Constellation.EveEntity(), ""),
				newEntityItemFromEveSolarSystem(es),
			)
		})
	}
	if app.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			fyne.CurrentApp().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
		})
	}
	fyne.Do(func() {
		a.tabs.Refresh()
	})
	g := new(errgroup.Group)
	g.Go(func() error {
		if o.Variant() != app.EveLocationStation {
			return nil
		}
		ss, err := a.iw.s.EVEUniverse().GetStationServicesESI(ctx, a.id)
		if err != nil {
			return err
		}
		items := xslices.Map(ss, func(s string) entityItem {
			s2 := strings.ReplaceAll(s, "-", " ")
			name := xstrings.Title(s2)
			return newEntityItem(0, "Service", name, infoNotSupported)
		})
		fyne.Do(func() {
			a.services.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

type raceInfo struct {
	widget.BaseWidget
	baseInfo

	id          int64
	logo        *canvas.Image
	tabs        *container.AppTabs
	description *widget.Label
}

func newRaceInfo(iw *InfoWindow, id int64) *raceInfo {
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
	o, err := a.iw.s.EVEUniverse().GetOrCreateRaceESI(ctx, a.id)
	if err != nil {
		return err
	}
	if factionID, ok := o.FactionID(); ok {
		fyne.Do(func() {
			a.iw.s.EVEImage().FactionLogoAsync(factionID, app.IconPixelSize, func(r fyne.Resource) {
				a.logo.Resource = r
				a.logo.Refresh()
			})
		})
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if app.IsDeveloperMode() {
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
