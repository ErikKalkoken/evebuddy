package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// CorporationInfoArea represents an area that shows public information about a character.
type CorporationInfoArea struct {
	Content fyne.CanvasObject

	alliance        *widget.Label
	history         *widget.List
	corporationLogo *canvas.Image
	allianceLogo    *canvas.Image
	description     *widget.Label
	corporation     *widget.Label
	headquarters    *widget.Label
	historyItems    []app.CharacterCorporationHistoryItem

	u *BaseUI
}

func NewCorporationInfoArea(u *BaseUI, corporationID int32) *CorporationInfoArea {
	description := widget.NewLabel("Loading...")
	description.Wrapping = fyne.TextWrapWord
	alliance := widget.NewLabel("")
	alliance.Truncation = fyne.TextTruncateEllipsis
	corporation := widget.NewLabel("Loading...")
	corporation.Truncation = fyne.TextTruncateEllipsis
	title := widget.NewLabel("")
	title.Truncation = fyne.TextTruncateEllipsis
	corporationLogo := iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize))
	s := float32(DefaultIconPixelSize) * 1.3 / u.Window.Canvas().Scale()
	corporationLogo.SetMinSize(fyne.NewSquareSize(s))
	a := &CorporationInfoArea{
		alliance:        alliance,
		allianceLogo:    iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize)),
		corporationLogo: corporationLogo,
		description:     description,
		corporation:     corporation,
		historyItems:    make([]app.CharacterCorporationHistoryItem, 0),
		headquarters:    title,
		u:               u,
	}

	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.corporation,
		a.headquarters,
		container.NewBorder(
			nil,
			nil,
			a.allianceLogo,
			nil,
			a.alliance,
		),
	)
	a.history = a.makeHistory()
	top := container.NewBorder(nil, nil, container.NewVBox(a.corporationLogo), nil, main)
	tabs := container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		container.NewTabItem("Attributes", widget.NewLabel("PLACEHOLDER")),
		container.NewTabItem("Alliance History", container.NewVScroll(a.history)),
	)
	a.Content = container.NewBorder(top, nil, nil, nil, tabs)

	go func() {
		err := a.load(corporationID)
		if err != nil {
			slog.Error("corporation info update failed", "id", corporationID, "error", err)
			a.corporation.Text = fmt.Sprintf("ERROR: Failed to load corporation: %s", ihumanize.Error(err))
			a.corporation.Importance = widget.DangerImportance
			a.corporation.Refresh()
		}
	}()
	return a
}

func (a *CorporationInfoArea) makeHistory() *widget.List {
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
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *CorporationInfoArea) load(corporationID int32) error {
	ctx := context.Background()
	go func() {
		r, err := a.u.EveImageService.CorporationLogo(corporationID, DefaultIconPixelSize)
		if err != nil {
			slog.Error("corporation info: Failed to load logo", "corporationID", corporationID, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	c, err := a.u.EveUniverseService.GetEveCorporation(ctx, corporationID)
	if err != nil {
		return err
	}
	a.corporation.SetText(c.Name)
	if c.HasAlliance() {
		a.alliance.SetText("Member of " + c.Alliance.Name)
	} else {
		a.alliance.Hide()
	}
	a.description.SetText(c.DescriptionPlain())
	if c.HomeStation != nil {
		a.headquarters.SetText("Headquarters: " + c.HomeStation.Name)
	} else {
		a.headquarters.Hide()
	}
	if c.HasAlliance() {
		go func() {
			r, err := a.u.EveImageService.AllianceLogo(c.Alliance.ID, DefaultIconPixelSize)
			if err != nil {
				slog.Error("corporation info: Failed to load alliance logo", "allianceID", c.Alliance.ID, "error", err)
				return
			}
			a.allianceLogo.Resource = r
			a.allianceLogo.Refresh()
		}()
	}
	// go func() {
	// 	history, err := a.u.CharacterService.CorporationHistory(ctx, corporationID)
	// 	if err != nil {
	// 		slog.Error("corporation info: Failed to load corporation history", "corporationID", corporationID, "error", err)
	// 		return
	// 	}
	// 	var duration string
	// 	if len(history) > 0 {
	// 		current := history[0]
	// 		duration = humanize.RelTime(current.StartDate, time.Now(), "", "")
	// 	} else {
	// 		duration = "?"
	// 	}
	// 	a.corporation.SetText(fmt.Sprintf("Member of %s\nfor %s", c.Corporation.Name, duration))
	// 	a.historyItems = history
	// 	a.history.Refresh()
	// }()
	return nil
}
