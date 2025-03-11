package ui

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
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
)

// CharacterInfoArea represents an area that shows public information about a character.
type CharacterInfoArea struct {
	Content fyne.CanvasObject

	alliance        *widget.Label
	historyList     *widget.List
	character       *widget.Label
	corporationLogo *canvas.Image
	corporation     *kxwidget.TappableLabel
	membership      *widget.Label
	portrait        *kxwidget.TappableImage
	security        *widget.Label
	title           *widget.Label
	historyItems    []app.MembershipHistoryItem
	tabs            *container.AppTabs

	u *BaseUI
}

func NewCharacterInfoArea(u *BaseUI, characterID int32) *CharacterInfoArea {
	alliance := widget.NewLabel("")
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
	a := &CharacterInfoArea{
		alliance:        alliance,
		character:       character,
		corporationLogo: iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize)),
		corporation:     corporation,
		membership:      widget.NewLabel(""),
		portrait:        portrait,
		historyItems:    make([]app.MembershipHistoryItem, 0),
		security:        widget.NewLabel(""),
		title:           title,
		u:               u,
	}

	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.character,
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
	a.historyList = a.makeHistory()
	top := container.NewBorder(nil, nil, container.NewVBox(a.portrait), nil, main)
	a.tabs = container.NewAppTabs()
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(characterID)
		if err != nil {
			slog.Error("character info update failed", "characterID", characterID, "error", err)
			a.character.Text = fmt.Sprintf("ERROR: Failed to load character: %s", ihumanize.Error(err))
			a.character.Importance = widget.DangerImportance
			a.character.Refresh()
		}
	}()
	return a
}

func (a *CharacterInfoArea) makeHistory() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.historyItems)
		},
		func() fyne.CanvasObject {
			l := widget.NewRichText()
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.historyItems) {
				return
			}
			it := a.historyItems[id]
			const dateFormat = "2006.01.02 15:04"
			var endDateStr string
			if !it.EndDate.IsZero() {
				endDateStr = it.EndDate.Format(dateFormat)
			} else {
				endDateStr = "this day"
			}
			var closed string
			if it.IsDeleted {
				closed = " (closed)"
			}
			text := fmt.Sprintf(
				"%s%s   **%s** to **%s** (%s days)",
				it.Organization.Name,
				closed,
				it.StartDate.Format(dateFormat),
				endDateStr,
				humanize.Comma(int64(it.Days)),
			)
			co.(*widget.RichText).ParseMarkdown(text)
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.historyItems) {
			return
		}
		it := a.historyItems[id]
		a.u.ShowCorporaitonInfoWindow(it.Organization.ID)
	}
	return l
}

func (a *CharacterInfoArea) load(characterID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.u.EveImageService.CharacterPortrait(characterID, 256)
		if err != nil {
			slog.Error("character info: Failed to load portrait", "charaterID", characterID, "error", err)
			return
		}
		a.portrait.SetResource(r)
	}()
	c, err := a.u.EveUniverseService.GetEveCharacterESI(ctx, characterID)
	if err != nil {
		return err
	}
	a.character.SetText(c.Name)
	if c.HasAlliance() {
		a.alliance.SetText(c.Alliance.Name)
	} else {
		a.alliance.Hide()
	}
	a.security.SetText(fmt.Sprintf("Security Status: %.1f", c.SecurityStatus))
	a.corporation.SetText(fmt.Sprintf("Member of %s", c.Corporation.Name))
	a.corporation.OnTapped = func() {
		a.u.ShowCorporaitonInfoWindow(c.Corporation.ID)
	}
	a.portrait.OnTapped = func() {
		w := a.u.FyneApp.NewWindow(a.u.MakeWindowTitle(c.Name))
		size := 512
		s := float32(size) / w.Canvas().Scale()
		i := NewImageResourceAsync(icon.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
			return a.u.EveImageService.CharacterPortrait(characterID, size)
		})
		p := theme.Padding()
		w.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
		w.Show()
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
		r, err := a.u.EveImageService.CorporationLogo(c.Corporation.ID, DefaultIconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "charaterID", characterID, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	go func() {
		history, err := a.u.EveUniverseService.CharacterCorporationHistory(ctx, characterID)
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
		a.historyItems = history
		a.tabs.Append(container.NewTabItem("Employment History", a.historyList))
		a.tabs.Refresh()
	}()
	return nil
}
