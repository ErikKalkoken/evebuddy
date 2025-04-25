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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// locationInfo shows public information about a character.
type locationInfo struct {
	widget.BaseWidget

	description *widget.Label
	id          int64
	iw          *InfoWindow
	name        *widget.Label
	owner       *kxwidget.TappableLabel
	ownerLogo   *canvas.Image
	location    *entityList
	tabs        *container.AppTabs
	typeImage   *kxwidget.TappableImage
	typeInfo    *kxwidget.TappableLabel
}

func newLocationInfo(iw *InfoWindow, id int64) *locationInfo {
	typeInfo := kxwidget.NewTappableLabel("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := kxwidget.NewTappableLabel("", nil)
	owner.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	typeImage := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(iw.renderIconSize())
	a := &locationInfo{
		description: description,
		id:          id,
		iw:          iw,
		name:        makeInfoName(),
		owner:       owner,
		ownerLogo:   iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		typeImage:   typeImage,
		typeInfo:    typeInfo,
	}
	a.ExtendBaseWidget(a)
	a.location = newEntityList(a.iw.show)
	location := container.NewTabItem("Location", a.location)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		location,
	)
	a.tabs.Select(location)
	return a
}

func (a *locationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("location info update failed", "locationID", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", a.iw.u.ErrorDisplay(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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

func (a *locationInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateLocationESI(ctx, a.id)
	if err != nil {
		return err
	}
	go func() {
		r, err := a.iw.u.EveImageService().InventoryTypeRender(o.Type.ID, renderIconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load portrait", "location", o, "error", err)
			return
		}
		fyne.Do(func() {
			a.typeImage.SetResource(r)
		})
	}()
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(o.Owner.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load corp logo", "owner", o.Owner, "error", err)
			return
		}
		fyne.Do(func() {
			a.ownerLogo.Resource = r
			a.ownerLogo.Refresh()
		})
	}()
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.typeInfo.SetText(o.Type.Name)
		a.typeInfo.OnTapped = func() {
			a.iw.ShowEveEntity(o.Type.ToEveEntity())
		}
		a.owner.SetText(o.Owner.Name)
		a.owner.OnTapped = func() {
			a.iw.ShowEveEntity(o.Owner)
		}
		a.typeImage.OnTapped = func() {
			a.iw.showZoomWindow(o.Name, o.Type.ID, a.iw.u.EveImageService().InventoryTypeRender, a.iw.w)
		}
		description := o.Type.Description
		if description == "" {
			description = o.Type.Name
		}
		a.description.SetText(description)
	})

	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.u.App().Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
			a.tabs.Refresh()
		})
	}
	fyne.Do(func() {
		a.location.set(
			newEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.Region.ToEveEntity(), ""),
			newEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.ToEveEntity(), ""),
			newEntityItemFromEveSolarSystem(o.SolarSystem),
		)
	})
	if o.Variant() == app.EveLocationStation {
		services := container.NewTabItem("Services", widget.NewLabel(""))
		fyne.DoAndWait(func() {
			a.tabs.Append(services)
			a.tabs.Refresh()
		})
		go func() {
			ss, err := a.iw.u.EveUniverseService().GetStationServicesESI(ctx, int32(a.id))
			if err != nil {
				slog.Error("Failed to fetch station services", "stationID", o.ID, "error", err)
				return
			}
			items := xslices.Map(ss, func(s string) entityItem {
				s2 := strings.ReplaceAll(s, "-", " ")
				titler := cases.Title(language.English)
				name := titler.String(s2)
				return newEntityItem(0, "Service", name, infoNotSupported)
			})
			fyne.Do(func() {
				services.Content = newEntityListFromItems(nil, items...)
				a.tabs.Refresh()
			})
		}()
	}

	return nil
}
