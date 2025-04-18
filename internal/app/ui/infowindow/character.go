package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// characterInfo shows public information about a character.
type characterInfo struct {
	widget.BaseWidget

	id              int32
	alliance        *kxwidget.TappableLabel
	name            *widget.Label
	corporationLogo *canvas.Image
	corporation     *kxwidget.TappableLabel
	membership      *widget.Label
	portrait        *kxwidget.TappableImage
	security        *widget.Label
	title           *widget.Label
	tabs            *container.AppTabs
	iw              *InfoWindow
}

func newCharacterInfo(iw *InfoWindow, id int32) *characterInfo {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Wrapping = fyne.TextWrapWord
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Wrapping = fyne.TextWrapWord
	portrait := kxwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(renderIconUnitSize))
	title := widget.NewLabel("")
	title.Wrapping = fyne.TextWrapWord
	a := &characterInfo{
		alliance:        alliance,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		iw:              iw,
		id:              id,
		membership:      widget.NewLabel(""),
		name:            makeInfoName(),
		portrait:        portrait,
		security:        widget.NewLabel(""),
		tabs:            container.NewAppTabs(),
		title:           title,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *characterInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("character info update failed", "character", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", a.iw.u.ErrorDisplay(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.title,
		),
		container.NewBorder(
			nil,
			nil,
			a.corporationLogo,
			nil,
			container.New(
				layout.NewCustomPaddedVBoxLayout(-2*p),
				a.corporation,
				a.membership,
			),
		),
		widget.NewSeparator(),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.alliance,
			a.security,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.portrait), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *characterInfo) load() error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.EveImageService().CharacterPortrait(a.id, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "characterID", a.id, "error", err)
			return
		}
		a.portrait.SetResource(r)
	}()
	o, err := a.iw.u.EveUniverseService().GetCharacterESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	if o.HasAlliance() {
		a.alliance.SetText(o.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(o.Alliance)
		}
	} else {
		a.alliance.Hide()
	}
	a.security.SetText(fmt.Sprintf("Security Status: %.1f", o.SecurityStatus))
	a.corporation.SetText(fmt.Sprintf("Member of %s", o.Corporation.Name))
	a.corporation.OnTapped = func() {
		a.iw.ShowEveEntity(o.Corporation)
	}
	a.portrait.OnTapped = func() {
		go a.iw.showZoomWindow(o.Name, a.id, a.iw.u.EveImageService().CharacterPortrait, a.iw.w)
	}
	if s := o.DescriptionPlain(); s != "" {
		bio := widget.NewLabel(s)
		bio.Wrapping = fyne.TextWrapWord
		a.tabs.Append(container.NewTabItem("Bio", container.NewVScroll(bio)))
	}
	if o.Title != "" {
		a.title.SetText("Title: " + o.Title)
	} else {
		a.title.Hide()
	}

	desc := widget.NewLabel(o.RaceDescription())
	desc.Wrapping = fyne.TextWrapWord
	a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(desc)))

	attributes := []AttributeItem{
		NewAtributeItem("Corporation", o.Corporation),
		NewAtributeItem("Race", o.Race),
	}
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.iw.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	attributeList := NewAttributeList(a.iw, attributes...)
	attributesTab := container.NewTabItem("Attributes", attributeList)
	a.tabs.Append(attributesTab)
	a.tabs.Refresh()
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(o.Corporation.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "characterID", a.id, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	go func() {
		history, err := a.iw.u.EveUniverseService().GetCharacterCorporationHistory(ctx, a.id)
		if err != nil {
			slog.Error("character info: Failed to load corporation history", "characterID", a.id, "error", err)
			return
		}
		if len(history) == 0 {
			a.membership.Hide()
			return
		}
		current := history[0]
		duration := humanize.RelTime(current.StartDate, time.Now(), "", "")
		a.membership.SetText(fmt.Sprintf("for %s", duration))
		items := xslices.Map(history, historyItem2EntityItem)
		historyList := NewEntityListFromItems(a.iw.show, items...)
		a.tabs.Append(container.NewTabItem("Employment History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
