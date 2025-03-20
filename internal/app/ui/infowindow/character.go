package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
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
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
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
	iw              InfoWindow
	w               fyne.Window
}

func newCharacterInfo(iw InfoWindow, characterID int32, w fyne.Window) *characterInfo {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Truncation = fyne.TextTruncateEllipsis
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Truncation = fyne.TextTruncateEllipsis
	portrait := kxwidget.NewTappableImage(icons.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	title := widget.NewLabel("")
	title.Truncation = fyne.TextTruncateEllipsis
	a := &characterInfo{
		alliance:        alliance,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		iw:              iw,
		id:              characterID,
		membership:      widget.NewLabel(""),
		name:            name,
		portrait:        portrait,
		security:        widget.NewLabel(""),
		tabs:            container.NewAppTabs(),
		title:           title,
		w:               w,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *characterInfo) CreateRenderer() fyne.WidgetRenderer {
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

	go func() {
		err := a.load(a.id)
		if err != nil {
			slog.Error("character info update failed", "character", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return widget.NewSimpleRenderer(c)
}

func (a *characterInfo) load(characterID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.u.EveImageService().CharacterPortrait(characterID, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "charaterID", characterID, "error", err)
			return
		}
		a.portrait.SetResource(r)
	}()
	o, err := a.iw.u.EveUniverseService().GetCharacterESI(ctx, characterID)
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
		go a.iw.showZoomWindow(o.Name, characterID, a.iw.u.EveImageService().CharacterPortrait, a.w)
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
		NewAtributeItem("Race", o.Race.Name),
	}
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributes = append(attributes, x)
	}
	attributeList := NewAttributeList(attributes...)
	attributeList.ShowInfoWindow = a.iw.ShowEveEntity
	attributesTab := container.NewTabItem("Attributes", attributeList)
	a.tabs.Append(attributesTab)
	a.tabs.Refresh()
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(o.Corporation.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "charaterID", characterID, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	go func() {
		history, err := a.iw.u.EveUniverseService().GetCharacterCorporationHistory(ctx, characterID)
		if err != nil {
			slog.Error("character info: Failed to load corporation history", "charaterID", characterID, "error", err)
			return
		}
		if len(history) == 0 {
			a.membership.Hide()
			return
		}
		current := history[0]
		duration := humanize.RelTime(current.StartDate, time.Now(), "", "")
		a.membership.SetText(fmt.Sprintf("for %s", duration))
		items := slices.Collect(xiter.MapSlice(history, historyItem2EntityItem))
		historyList := NewEntityListFromItems(a.iw.show, items...)
		a.tabs.Append(container.NewTabItem("Employment History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
