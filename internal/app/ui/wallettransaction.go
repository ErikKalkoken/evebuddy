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

// walletTransactionArea is the UI area that shows the skillqueue
type walletTransactionArea struct {
	content      *fyne.Container
	transactions []walletTransaction
	table        *widget.Table
	top          *widget.Label
	u            *ui
}

func (u *ui) newWalletTransactionArea() *walletTransactionArea {
	a := walletTransactionArea{
		top:          widget.NewLabel(""),
		transactions: make([]walletTransaction, 0),
		u:            u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *walletTransactionArea) makeTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Date", 130},
		{"Quantity", 130},
		{"Type", 200},
		{"Unit Price", 130},
		{"Total", 130},
		{"Client", 250},
		{"Where", 250},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.transactions), len(headers)
		},
		func() fyne.CanvasObject {
			x := widget.NewLabel("Template")
			x.Truncation = fyne.TextTruncateEllipsis
			return x
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			if tci.Row >= len(a.transactions) {
				return
			}
			w := a.transactions[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = w.date.Format(app.TimeDefaultFormat)
			case 1:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.Comma(int64(w.quantity))
			case 2:
				l.Text = w.eveType
			case 3:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.FormatFloat(myFloatFormat, w.unitPrice)
			case 4:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = humanize.FormatFloat(myFloatFormat, w.total)
				switch {
				case w.total < 0:
					l.Importance = widget.DangerImportance
				case w.total > 0:
					l.Importance = widget.SuccessImportance
				default:
					l.Importance = widget.MediumImportance
				}
			case 5:
				l.Text = w.client
			case 6:
				l.Text = w.location
			}
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	return t
}

func (a *walletTransactionArea) refresh() {
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
	a.table.Refresh()
}

func (a *walletTransactionArea) makeTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance
	}
	characterID := a.u.characterID()
	hasData := a.u.StatusCacheService.CharacterSectionExists(characterID, app.SectionWalletTransactions)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.transactions)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *walletTransactionArea) updateEntries() error {
	if !a.u.hasCharacter() {
		a.transactions = make([]walletTransaction, 0)
		return nil
	}
	characterID := a.u.characterID()
	ww, err := a.u.CharacterService.ListCharacterWalletTransactions(context.TODO(), characterID)
	if err != nil {
		return fmt.Errorf("failed to fetch wallet journal for character %d: %w", characterID, err)
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
	a.transactions = transactions
	return nil
}
