package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
)

// WalletTransactionArea is the UI area that shows the skillqueue
type WalletTransactionArea struct {
	Content fyne.CanvasObject

	rows []*app.CharacterWalletTransaction
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewWalletTransactionArea(u *BaseUI) *WalletTransactionArea {
	a := WalletTransactionArea{
		top:  makeTopLabel(),
		rows: make([]*app.CharacterWalletTransaction, 0),
		u:    u,
	}
	makeDataLabel := func(col int, r *app.CharacterWalletTransaction) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = r.Date.Format(app.DateTimeFormat)
		case 1:
			align = fyne.TextAlignTrailing
			text = humanize.Comma(int64(r.Quantity))
		case 2:
			text = r.EveType.Name
		case 3:
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(app.FloatFormat, r.UnitPrice)
		case 4:
			total := r.UnitPrice * float64(r.Quantity)
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(app.FloatFormat, total)
			switch {
			case total < 0:
				importance = widget.DangerImportance
			case total > 0:
				importance = widget.SuccessImportance
			default:
				importance = widget.MediumImportance
			}
		case 5:
			text = r.Client.Name
		case 6:
			text = r.Location.Name
		}
		return text, align, importance
	}
	var headers = []headerDef{
		{"Date", 150},
		{"Quantity", 130},
		{"Type", 200},
		{"Unit Price", 200},
		{"Total", 200},
		{"Client", 250},
		{"Where", 250},
	}
	if a.u.IsDesktop() {
		a.body = makeDataTableForDesktop(headers, &a.rows, makeDataLabel, func(column int, r *app.CharacterWalletTransaction) {
			switch column {
			case 2:
				a.u.ShowTypeInfoWindow(r.EveType.ID)
			case 5:
				a.u.ShowEveEntityInfoWindow(r.Client)
			case 6:
				a.u.ShowInfoWindow(infowindow.Location, r.Location.ID)
			}
		})
	} else {
		a.body = makeDataTableForMobile(headers, &a.rows, makeDataLabel, func(r *app.CharacterWalletTransaction) {
			a.u.ShowTypeInfoWindow(r.EveType.ID)
		})
	}
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.body)
	return &a
}

func (a *WalletTransactionArea) Refresh() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
}

func (a *WalletTransactionArea) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	characterID := a.u.CurrentCharacterID()
	hasData := a.u.StatusCacheService.CharacterSectionExists(characterID, app.SectionWalletTransactions)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.rows)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *WalletTransactionArea) updateEntries() error {
	if !a.u.HasCharacter() {
		a.rows = make([]*app.CharacterWalletTransaction, 0)
		return nil
	}
	characterID := a.u.CurrentCharacterID()
	ww, err := a.u.CharacterService.ListCharacterWalletTransactions(context.TODO(), characterID)
	if err != nil {
		return err
	}
	a.rows = ww
	return nil
}
