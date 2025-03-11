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
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// characterInfoArea represents an area that shows public information about a character.
type characterInfoArea struct {
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
}

func newCharacterInfoArea(iw InfoWindow, characterID int32) *characterInfoArea {
	alliance := kxwidget.NewTappableLabel("", nil)
	alliance.Truncation = fyne.TextTruncateEllipsis
	character := widget.NewLabel("Loading...")
	character.Truncation = fyne.TextTruncateEllipsis
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Truncation = fyne.TextTruncateEllipsis
	portrait := kxwidget.NewTappableImage(icon.Characterplaceholder64Jpeg, nil)
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	title := widget.NewLabel("")
	title.Truncation = fyne.TextTruncateEllipsis
	a := &characterInfoArea{
		alliance:        alliance,
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(defaultIconUnitSize)),
		iw:              iw,
		membership:      widget.NewLabel(""),
		name:            character,
		portrait:        portrait,
		security:        widget.NewLabel(""),
		tabs:            container.NewAppTabs(),
		title:           title,
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
		err := a.load(characterID)
		if err != nil {
			slog.Error("character info update failed", "characterID", characterID, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *characterInfoArea) load(characterID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.iw.eis.CharacterPortrait(characterID, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "charaterID", characterID, "error", err)
			return
		}
		a.portrait.SetResource(r)
	}()
	c, err := a.iw.eus.GetEveCharacterESI(ctx, characterID)
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
		showZoomWindow(c.Name, characterID, a.iw.eis.CharacterPortrait)
	}
	bioText := c.DescriptionPlain()
	if bioText != "" {
		bio := widget.NewLabel(bioText)
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
			slog.Error("character info: Failed to load corp logo", "charaterID", characterID, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	go func() {
		history, err := a.iw.eus.GetCharacterCorporationHistory(ctx, characterID)
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
		historyList := appwidget.NewMembershipHistoryList()
		historyList.ShowInfoWindow = a.iw.ShowEveEntity
		historyList.Set(history)
		a.tabs.Append(container.NewTabItem("Employment History", historyList))
		a.tabs.Refresh()
	}()
	return nil
}
