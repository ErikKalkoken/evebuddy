package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"github.com/dustin/go-humanize"
)

// walletTransactionArea is the UI area that shows the skillqueue
type walletTransactionArea struct {
	content *fyne.Container
	ui      *ui
}

func (u *ui) NewWalletTransactionArea() *walletTransactionArea {
	c := walletTransactionArea{ui: u, content: container.NewStack()}
	return &c
}

func (a *walletTransactionArea) Redraw() {
	a.content.RemoveAll()
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	ee, err := a.ui.service.ListWalletJournalEntries(characterID)
	if err != nil {
		slog.Error("failed to fetch wallet journal", "err", err)
		a.content.Add(makeMessage("Failed to fetch wallet journal", widget.DangerImportance))
		return
	}
	const myFloatFormat = "#,###.##"
	table := widgets.NewStaticTable(
		func() (rows int, cols int) {
			return len(ee), 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("2024-05-08 18:59")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			e := ee[tci.Row]
			switch tci.Col {
			case 0:
				l.SetText(e.Date.Format(myDateTime))
			case 1:
				l.SetText(e.Type())
			case 2:
				l.SetText(humanize.FormatFloat(myFloatFormat, e.Amount))
				l.Alignment = fyne.TextAlignTrailing
				switch {
				case e.Amount < 0:
					l.Importance = widget.DangerImportance
				case e.Amount > 0:
					l.Importance = widget.SuccessImportance
				default:
					l.Importance = widget.MediumImportance
				}
			case 3:
				l.SetText(humanize.FormatFloat(myFloatFormat, e.Balance))
				l.Alignment = fyne.TextAlignTrailing
			case 4:
				l.SetText(e.Description)
				l.Truncation = fyne.TextTruncateEllipsis
			}
		},
	)
	table.SetColumnWidth(0, 130)
	table.SetColumnWidth(1, 130)
	table.SetColumnWidth(2, 130)
	table.SetColumnWidth(3, 130)
	table.SetColumnWidth(4, 450)

	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	table.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		var s string
		switch tci.Col {
		case 0:
			s = "Date"
		case 1:
			s = "Type"
		case 2:
			s = "Amount"
		case 3:
			s = "Balance"
		case 4:
			s = "Description"
		}
		co.(*widget.Label).SetText(s)
	}

	var s string
	var i widget.Importance
	if len(ee) > 0 {
		s = fmt.Sprintf("Total: %s", humanize.FormatFloat(myFloatFormat, ee[0].Balance))
		i = widget.MediumImportance
	} else {
		s = "No entries"
		i = widget.WarningImportance
	}
	x := widget.NewLabel(s)
	x.Importance = i
	bottom := container.NewVBox(widget.NewSeparator(), x)

	content := container.NewBorder(nil, bottom, nil, nil, table)
	a.content.Add(content)
}

func (a *walletTransactionArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := a.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				if !a.ui.service.SectionUpdatedExpired(characterID, service.UpdateSectionWalletJournal) {
					return
				}
				count, err := a.ui.service.UpdateWalletJournalEntryESI(characterID)
				if err != nil {
					slog.Error(err.Error())
					return
				}
				if count > 0 {
					a.Redraw()
				}
			}()
			<-ticker.C
		}
	}()
}

func makeMessage(msg string, importance widget.Importance) *fyne.Container {
	var c color.Color
	switch importance {
	case widget.DangerImportance:
		c = theme.ErrorColor()
	case widget.WarningImportance:
		c = theme.WarningColor()
	default:
		c = theme.ForegroundColor()
	}
	t := canvas.NewText(msg, c)
	return container.NewHBox(layout.NewSpacer(), t, layout.NewSpacer())
}
