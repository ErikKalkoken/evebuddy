package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type walletTransaction struct {
	client    string
	date      time.Time
	location  string
	quantity  int32
	total     float64
	eveType   string
	unitPrice float64
}

// WalletTransactionArea is the UI area that shows the skillqueue
type WalletTransactionArea struct {
	Content fyne.CanvasObject

	rows []walletTransaction
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewWalletTransactionArea(u *BaseUI) *WalletTransactionArea {
	a := WalletTransactionArea{
		top:  makeTopLabel(),
		rows: make([]walletTransaction, 0),
		u:    u,
	}
	makeDataLabel := func(col int, w walletTransaction) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = w.date.Format(app.TimeDefaultFormat)
		case 1:
			align = fyne.TextAlignTrailing
			text = humanize.Comma(int64(w.quantity))
		case 2:
			text = w.eveType
		case 3:
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(MyFloatFormat, w.unitPrice)
		case 4:
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(MyFloatFormat, w.total)
			switch {
			case w.total < 0:
				importance = widget.DangerImportance
			case w.total > 0:
				importance = widget.SuccessImportance
			default:
				importance = widget.MediumImportance
			}
		case 5:
			text = w.client
		case 6:
			text = w.location
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
		a.body = makeDataTableForDesktop(headers, &a.rows, makeDataLabel, nil)
	} else {
		a.body = makeDataTableForMobile(headers, &a.rows, makeDataLabel, nil)
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
	characterID := a.u.CharacterID()
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
		a.rows = make([]walletTransaction, 0)
		return nil
	}
	characterID := a.u.CharacterID()
	ww, err := a.u.CharacterService.ListCharacterWalletTransactions(context.TODO(), characterID)
	if err != nil {
		return err
	}
	transactions := make([]walletTransaction, len(ww))
	for i, w := range ww {
		var tx walletTransaction
		tx.client = w.Client.Name
		tx.date = w.Date
		tx.eveType = w.EveType.Name
		tx.location = w.Location.Name
		tx.quantity = w.Quantity
		tx.unitPrice = w.UnitPrice
		tx.total = w.UnitPrice * float64(w.Quantity)
		if w.IsBuy {
			tx.total *= -1
		}
		transactions[i] = tx
	}
	a.rows = transactions
	return nil
}
