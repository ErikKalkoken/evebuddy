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
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// characterArea represents an area that shows public information about a character.
type characterArea struct {
	Content fyne.CanvasObject

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

func newCharacterArea(iw InfoWindow, character *app.EveEntity, w fyne.Window) *characterArea {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Truncation = fyne.TextTruncateEllipsis
	name := widget.NewLabel(character.Name)
	name.Truncation = fyne.TextTruncateEllipsis
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Truncation = fyne.TextTruncateEllipsis
	portrait := kxwidget.NewTappableImage(icon.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	title := widget.NewLabel("")
	title.Truncation = fyne.TextTruncateEllipsis
	a := &characterArea{
		alliance:        alliance,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(defaultIconUnitSize)),
		iw:              iw,
		membership:      widget.NewLabel(""),
		name:            name,
		portrait:        portrait,
		security:        widget.NewLabel(""),
		tabs:            container.NewAppTabs(),
		title:           title,
		w:               w,
	}

	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.name,
		a.title,
		container.NewBorder(
			nil,
			nil,
			a.corporationLogo,
			nil,
			container.New(
				layout.NewCustomPaddedVBoxLayout(0),
				a.corporation,
				a.membership,
			),
		),
		widget.NewSeparator(),
		a.alliance,
		a.security,
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.portrait), nil, main)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(character)
		if err != nil {
			slog.Error("character info update failed", "character", character, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *characterArea) load(character *app.EveEntity) error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.eis.CharacterPortrait(character.ID, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "charaterID", character, "error", err)
			return
		}
		a.portrait.SetResource(r)
	}()
	c, err := a.iw.eus.GetEveCharacterESI(ctx, character.ID)
	if err != nil {
		return err
	}
	a.name.SetText(c.Name)
	if c.HasAlliance() {
		a.alliance.SetText(c.Alliance.Name)
		a.alliance.OnTapped = func() {
			a.iw.ShowEveEntity(c.Alliance)
		}
	} else {
		a.alliance.Hide()
	}
	a.security.SetText(fmt.Sprintf("Security Status: %.1f", c.SecurityStatus))
	a.corporation.SetText(fmt.Sprintf("Member of %s", c.Corporation.Name))
	a.corporation.OnTapped = func() {
		a.iw.ShowEveEntity(c.Corporation)
	}
	a.portrait.OnTapped = func() {
		go a.iw.showZoomWindow(c.Name, character.ID, a.iw.eis.CharacterPortrait, a.w)
	}
	if s := c.DescriptionPlain(); s != "" {
		bio := widget.NewLabel(s)
		bio.Wrapping = fyne.TextWrapWord
		a.tabs.Append(container.NewTabItem("Bio", container.NewVScroll(bio)))
	}
	desc := widget.NewLabel(c.RaceDescription())
	desc.Wrapping = fyne.TextWrapWord
	a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(desc)))
	if c.Title != "" {
		a.title.SetText("Title: " + c.Title)
	} else {
		a.title.Hide()
	}
	a.tabs.Refresh()
	go func() {
		r, err := a.iw.eis.CorporationLogo(c.Corporation.ID, defaultIconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "charaterID", character, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	go func() {
		history, err := a.iw.eus.GetCharacterCorporationHistory(ctx, character.ID)
		if err != nil {
			slog.Error("character info: Failed to load corporation history", "charaterID", character, "error", err)
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
		historyList := NewEntityListFromItems(a.iw.ShowEveEntity, items...)
		a.tabs.Append(container.NewTabItem("Employment History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
