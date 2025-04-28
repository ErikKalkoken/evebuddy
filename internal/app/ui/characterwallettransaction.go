package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type CharacterWalletTransaction struct {
	widget.BaseWidget

	rows []*app.CharacterWalletTransaction
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewCharacterWalletTransaction(u *BaseUI) *CharacterWalletTransaction {
	headers := []iwidget.HeaderDef{
		{Text: "Date", Width: 150},
		{Text: "Quantity", Width: 130},
		{Text: "Type", Width: 200},
		{Text: "Unit Price", Width: 200},
		{Text: "Total", Width: 200},
		{Text: "Client", Width: 250},
		{Text: "Where", Width: 350},
	}
	a := &CharacterWalletTransaction{
		top:  appwidget.MakeTopLabel(),
		rows: make([]*app.CharacterWalletTransaction, 0),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r *app.CharacterWalletTransaction) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.Date.Format(app.DateTimeFormat))
		case 1:
			return iwidget.NewRichTextSegmentFromText(humanize.Comma(int64(r.Quantity)),
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
		case 2:
			return iwidget.NewRichTextSegmentFromText(r.EveType.Name)
		case 3:
			return iwidget.NewRichTextSegmentFromText(
				humanize.FormatFloat(app.FloatFormat, r.UnitPrice),
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
		case 4:
			total := r.UnitPrice * float64(r.Quantity)
			if r.IsBuy {
				total = total * -1
			}
			text := humanize.FormatFloat(app.FloatFormat, total)
			var color fyne.ThemeColorName
			switch {
			case total < 0:
				color = theme.ColorNameError
			case total > 0:
				color = theme.ColorNameSuccess
			default:
				color = theme.ColorNameForeground
			}
			return iwidget.NewRichTextSegmentFromText(text, widget.RichTextStyle{
				ColorName: color,
				Alignment: fyne.TextAlignTrailing,
			})
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.Client.Name)
		case 6:
			return r.Location.DisplayRichText()
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.isDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeCell, func(column int, r *app.CharacterWalletTransaction) {
			switch column {
			case 2:
				a.u.ShowTypeInfoWindow(r.EveType.ID)
			case 5:
				a.u.ShowEveEntityInfoWindow(r.Client)
			case 6:
				a.u.ShowLocationInfoWindow(r.Location.ID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeCell, func(r *app.CharacterWalletTransaction) {
			a.u.ShowTypeInfoWindow(r.EveType.ID)
		})
	}
	return a
}

func (a *CharacterWalletTransaction) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterWalletTransaction) update() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.body.Refresh()
	})
}

func (a *CharacterWalletTransaction) makeTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance
	}
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.CharacterSectionExists(characterID, app.SectionWalletTransactions)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.rows)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *CharacterWalletTransaction) updateEntries() error {
	if !a.u.hasCharacter() {
		a.rows = make([]*app.CharacterWalletTransaction, 0)
		return nil
	}
	characterID := a.u.currentCharacterID()
	ww, err := a.u.cs.ListWalletTransactions(context.TODO(), characterID)
	if err != nil {
		return err
	}
	a.rows = ww
	return nil
}
