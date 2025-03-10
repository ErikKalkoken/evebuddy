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

	alliance     *widget.Label
	bio          *widget.Label
	history      *widget.List
	character    *widget.Label
	corpIcon     *canvas.Image
	description  *widget.Label
	membership   *widget.Label
	portrait     *kxwidget.TappableImage
	security     *widget.Label
	title        *widget.Label
	historyItems []app.CharacterCorporationHistoryItem

	u *BaseUI
}

func NewCharacterInfoArea(u *BaseUI, characterID int32) *CharacterInfoArea {
	bio := widget.NewLabel("")
	bio.Wrapping = fyne.TextWrapWord
	description := widget.NewLabel("Loading...")
	description.Wrapping = fyne.TextWrapWord
	alliance := widget.NewLabel("")
	alliance.Truncation = fyne.TextTruncateEllipsis
	character := widget.NewLabel("Loading...")
	character.Truncation = fyne.TextTruncateEllipsis
	corporation := widget.NewLabel("")
	corporation.Truncation = fyne.TextTruncateEllipsis
	portrait := kxwidget.NewTappableImage(icon.Characterplaceholder64Jpeg, func() {})
	portrait.SetFillMode(canvas.ImageFillContain)
	portrait.SetMinSize(fyne.NewSquareSize(128))
	title := widget.NewLabel("")
	title.Truncation = fyne.TextTruncateEllipsis
	a := &CharacterInfoArea{
		alliance:     alliance,
		bio:          bio,
		character:    character,
		corpIcon:     iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize)),
		description:  description,
		membership:   corporation,
		portrait:     portrait,
		historyItems: make([]app.CharacterCorporationHistoryItem, 0),
		security:     widget.NewLabel(""),
		title:        title,
		u:            u,
	}

	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.character,
		a.title,
		container.NewBorder(
			nil,
			nil,
			a.corpIcon,
			nil,
			a.membership,
		),
		widget.NewSeparator(),
		a.alliance,
		a.security,
	)
	a.history = widget.NewList(
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
			text := fmt.Sprintf(
				"%s **%s** to **%s** (%d days)",
				it.Corporation.Name,
				it.StartDate.Format(dateFormat),
				endDateStr,
				it.Days(),
			)
			co.(*widget.RichText).ParseMarkdown(text)
		},
	)
	a.history.HideSeparators = true
	a.history.OnSelected = func(id widget.ListItemID) {
		a.history.UnselectAll()
	}
	top := container.NewBorder(nil, nil, container.NewVBox(a.portrait), nil, main)
	tabs := container.NewAppTabs(
		container.NewTabItem("Bio", container.NewVScroll(a.bio)),
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		container.NewTabItem("Employment History", container.NewVScroll(a.history)),
	)
	a.Content = container.NewBorder(top, nil, nil, nil, tabs)

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
	c, err := a.u.EveUniverseService.GetOrCreateEveCharacterESI(ctx, characterID)
	if err != nil {
		return err
	}
	a.character.SetText(c.Name)
	if c.HasAlliance() {
		a.alliance.SetText(c.AllianceName())
	} else {
		a.alliance.Hide()
	}
	a.security.SetText(fmt.Sprintf("Security Status: %.1f", c.SecurityStatus))
	a.bio.SetText(c.DescriptionPlain())
	a.description.SetText(c.RaceDescription())
	if c.Title != "" {
		a.title.SetText("Title: " + c.Title)
	} else {
		a.title.Hide()
	}
	a.membership.SetText(fmt.Sprintf("Member of %s\nfor %s", c.Corporation.Name, "?"))
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
	go func() {
		r, err := a.u.EveImageService.CorporationLogo(c.Corporation.ID, DefaultIconPixelSize)
		if err != nil {
			slog.Error("character info: Failed to load corp logo", "charaterID", characterID, "error", err)
			return
		}
		a.corpIcon.Resource = r
		a.corpIcon.Refresh()
	}()
	go func() {
		history, err := a.u.CharacterService.CorporationHistory(ctx, characterID)
		if err != nil {
			slog.Error("character info: Failed to load corporation history", "charaterID", characterID, "error", err)
			return
		}
		var duration string
		if len(history) > 0 {
			current := history[0]
			duration = humanize.RelTime(current.StartDate, time.Now(), "", "")
		} else {
			duration = "?"
		}
		a.membership.SetText(fmt.Sprintf("Member of %s\nfor %s", c.Corporation.Name, duration))
		a.historyItems = history
		a.history.Refresh()
	}()
	return nil
}
