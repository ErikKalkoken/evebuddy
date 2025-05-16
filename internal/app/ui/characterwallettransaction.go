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
	headers := []headerDef{
		{Text: "Date", Width: 150},
		{Text: "Quantity", Width: 130},
		{Text: "Type", Width: 200},
		{Text: "Unit Price", Width: 200},
		{Text: "Total", Width: 200},
		{Text: "Client", Width: 250},
		{Text: "Where", Width: 350},
	}
	a := &CharacterWalletTransaction{
		top:  makeTopLabel(),
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
	if a.u.isDesktop {
		a.body = makeDataTableForDesktop(headers, &a.rows, makeCell, func(column int, r *app.CharacterWalletTransaction) {
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
		a.body = makeDataTableForMobile(headers, &a.rows, makeCell, func(r *app.CharacterWalletTransaction) {
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
	var err error
	entries := make([]*app.CharacterWalletTransaction, 0)
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionWalletTransactions)
	if hasData {
		entries2, err2 := a.u.cs.ListWalletTransactions(context.Background(), characterID)
		if err2 != nil {
			slog.Error("Failed to refresh wallet transaction UI", "err", err2)
			err = err2
		} else {
			entries = entries2
		}
	}
	t, i := makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		t := humanize.Comma(int64(len(entries)))
		s := fmt.Sprintf("Entries: %s", t)
		return s, widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = entries
		a.body.Refresh()
	})
}
