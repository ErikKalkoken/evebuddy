package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/model"
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
	content *fyne.Container
	entries binding.UntypedList // []walletTransaction
	table   *widget.Table
	top     *widget.Label
	ui      *ui
}

func (u *ui) newWalletTransactionArea() *walletTransactionArea {
	a := walletTransactionArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		top:     widget.NewLabel(""),
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
			return a.entries.Length(), len(headers)
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
			w, err := getItemUntypedList[walletTransaction](a.entries, tci.Row)
			if err != nil {
				slog.Error("failed to render cell in wallet transaction table", "err", err)
				l.Text = "failed to render"
				l.Importance = widget.DangerImportance
				l.Refresh()
				return
			}
			switch tci.Col {
			case 0:
				l.Text = w.date.Format(myDateTime)
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
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.updateEntries(); err != nil {
			return "", 0, err
		}
		return a.makeTopText()
	}()
	if err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.table.Refresh()
}

func (a *walletTransactionArea) makeTopText() (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	characterID := a.ui.characterID()
	hasData, err := a.ui.sv.Characters.CharacterSectionWasUpdated(
		context.Background(), characterID, model.CharacterSectionWalletTransactions)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return "", widget.MediumImportance, nil
}

func (a *walletTransactionArea) updateEntries() error {
	if !a.ui.hasCharacter() {
		x := make([]any, 0)
		err := a.entries.Set(x)
		if err != nil {
			return err
		}
	}
	characterID := a.ui.characterID()
	ww, err := a.ui.sv.Characters.ListCharacterWalletTransactions(context.Background(), characterID)
	if err != nil {
		return fmt.Errorf("failed to fetch wallet journal for character %d: %w", characterID, err)
	}
	entries := make([]walletTransaction, len(ww))
	for i, w := range ww {
		var w2 walletTransaction
		w2.client = w.Client.Name
		w2.date = w.Date
		w2.eveType = w.EveType.Name
		w2.location = w.Location.Name
		w2.quantity = w.Quantity
		w2.unitPrice = w.UnitPrice
		w2.total = w.UnitPrice * float64(w.Quantity)
		if w.IsBuy {
			w2.total *= -1
		}
		entries[i] = w2
	}
	if err := a.entries.Set(copyToUntypedSlice(entries)); err != nil {
		return err
	}
	return nil
}
