package infoviewer

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

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// locationInfo shows public information about a character.
type locationInfo struct {
	widget.BaseWidget
	baseInfo

	description *widget.Label
	itemID      int64
	characterID int64
	location    *entityList
	owner       *widget.Hyperlink
	ownerLogo   *canvas.Image
	services    *entityList
	tabs        *container.AppTabs
	typeImage   *xwidget.TappableImage
	typeInfo    *widget.Hyperlink
}

func newLocationInfo(iw *InfoViewer, itemID, characterID int64) *locationInfo {
	typeInfo := widget.NewHyperlink("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := widget.NewHyperlink("", nil)
	owner.Wrapping = fyne.TextWrapWord
	typeImage := xwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(iw.renderIconSize())
	a := &locationInfo{
		description: newLabelWithWrapAndSelectable(""),
		itemID:      itemID,
		characterID: characterID,
		owner:       owner,
		ownerLogo:   xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(ui.IconUnitSize)),
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
	if a.characterID != 0 {
		ts, err := a.iw.u.Character().TokenSource(ctx, a.characterID, set.Of(goesi.ScopeUniverseReadStructuresV1))
		if err != nil {
			return err
		}
		ctx = xgoesi.NewContextWithAuth(ctx, a.characterID, ts)
	}
	o, err := a.iw.u.EVEUniverse().GetOrCreateLocationESI(ctx, a.itemID)
	if err != nil {
		return err
	}
	if o.Name == "" {
		return fmt.Errorf("not allowed to access structure")
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
	})
	if et, ok := o.Type.Value(); ok {
		fyne.Do(func() {
			a.iw.u.EVEImage().InventoryTypeRenderAsync(et.ID, renderIconPixelSize, func(r fyne.Resource) {
				a.typeImage.SetResource(r)
			})
			a.typeInfo.SetText(et.Name)
			a.typeInfo.OnTapped = func() {
				a.iw.Show(et.ToEveEntity())
			}
			a.typeImage.OnTapped = func() {
				a.iw.showZoomWindow(o.Name, et.ID, a.iw.u.EVEImage().InventoryTypeRenderAsync, a.iw.w)
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
			a.iw.u.EVEImage().CorporationLogoAsync(v.ID, ui.IconPixelSize, func(r fyne.Resource) {
				a.ownerLogo.Resource = r
				a.ownerLogo.Refresh()
			})
			a.owner.SetText(v.Name)
			a.owner.OnTapped = func() {
				a.iw.Show(v)
			}
		})
	}
	if es, ok := o.SolarSystem.Value(); ok {
		fyne.Do(func() {
			a.location.set(
				newEntityItemFromEveEntityWithText(es.Constellation.Region.ToEveEntity(), ""),
				newEntityItemFromEveEntityWithText(es.Constellation.ToEveEntity(), ""),
				newEntityItemFromEveSolarSystem(es),
			)
		})
	}
	if a.iw.u.IsDeveloperMode() {
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
		ss, err := a.iw.u.EVEUniverse().GetStationServicesESI(ctx, a.itemID)
		if err != nil {
			return err
		}
		items := xslices.Map(ss, func(s string) entityItem {
			s2 := strings.ReplaceAll(s, "-", " ")
			name := xstrings.Title(s2)
			return newEntityItem(0, "Service", name, Undefined)
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
