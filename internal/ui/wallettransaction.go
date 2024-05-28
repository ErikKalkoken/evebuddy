package ui

import (
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

const walletTransactionUpdateTicker = 60 * time.Second

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
	total   *widget.Label
	ui      *ui
}

func (u *ui) NewWalletTransactionArea() *walletTransactionArea {
	a := walletTransactionArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		total:   widget.NewLabel(""),
	}
	a.total.TextStyle.Bold = true
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

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, t)
	a.table = t
	return &a
}

func (a *walletTransactionArea) Refresh() {
	a.updateEntries()
	s, i := a.makeTopText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
	a.table.Refresh()
}

func (a *walletTransactionArea) makeTopText() (string, widget.Importance) {
	c := a.ui.CurrentChar()
	if c == nil {
		return "No data yet...", widget.LowImportance
	}
	hasData, err := a.ui.service.CharacterSectionWasUpdated(c.ID, model.CharacterSectionWalletTransactions)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data yet...", widget.LowImportance
	}
	return "", widget.MediumImportance
}

func (a *walletTransactionArea) updateEntries() error {
	if !a.ui.HasCharacter() {
		x := make([]any, 0)
		err := a.entries.Set(x)
		if err != nil {
			return err
		}
	}
	characterID := a.ui.CurrentCharID()
	ww, err := a.ui.service.ListCharacterWalletTransactions(characterID)
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

func (a *walletTransactionArea) StartUpdateTicker() {
	ticker := time.NewTicker(walletTransactionUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *walletTransactionArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionWalletTransactions)
	if err != nil {
		slog.Error("Failed to update wallet transactions", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}
