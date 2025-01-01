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

type contractEntry struct {
	from    string
	info    string
	issued  time.Time
	name    string
	expired time.Time
	to      string
	type_   string
}

// func (e contractEntry) refTypeOutput() string {
// 	s := strings.ReplaceAll(e.refType, "_", " ")
// 	c := cases.Title(language.English)
// 	s = c.String(s)
// 	return s
// }

// contractsArea is the UI area that shows the skillqueue
type contractsArea struct {
	content *fyne.Container
	entries []contractEntry
	table   *widget.Table
	top     *widget.Label
	u       *UI
}

func (u *UI) newContractsArea() *contractsArea {
	a := contractsArea{
		entries: make([]contractEntry, 0),
		top:     widget.NewLabel(""),
		u:       u,
	}

	a.top.TextStyle.Bold = true
	a.table = a.makeTable()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *contractsArea) makeTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Contract", 150},
		{"Type", 150},
		{"From", 200},
		{"To", 200},
		{"Date Issued", 150},
		{"Time Left", 150},
		{"Info by Issuer", 200},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.entries), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			l.Truncation = fyne.TextTruncateOff
			if tci.Row >= len(a.entries) || tci.Row < 0 {
				return
			}
			w := a.entries[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = w.name
			case 1:
				l.Text = w.type_
			case 2:
				l.Text = w.from
			case 3:
				l.Text = w.to
			case 4:
				l.Text = w.issued.Format(app.TimeDefaultFormat)
			case 5:
				if w.expired.After(time.Now()) {
					l.Text = "EXPIRED"
				} else {
					l.Text = humanize.Time(w.expired)
				}
				l.Importance = widget.DangerImportance
			case 6:
				l.Text = w.info
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
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if tci.Row >= len(a.entries) || tci.Row < 0 {
			return
		}
		// TODO
	}
	return t
}

func (a *contractsArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
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

func (a *contractsArea) makeTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.currentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionContracts)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.entries)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *contractsArea) updateEntries() error {
	if !a.u.hasCharacter() {
		a.entries = make([]contractEntry, 0)
		return nil
	}
	characterID := a.u.characterID()
	oo, err := a.u.CharacterService.ListCharacterContracts(context.TODO(), characterID)
	if err != nil {
		return err
	}
	entries := make([]contractEntry, len(oo))
	for i, o := range oo {
		var e contractEntry
		e.name = "[Multiple Items]"
		e.type_ = o.TypeDisplay()
		e.from = o.Issuer.Name
		if o.Assignee != nil {
			e.to = o.Assignee.Name
		}
		e.issued = o.DateIssued
		e.expired = o.DateExpired
		e.info = o.Title
		entries[i] = e
	}
	a.entries = entries
	return nil
}
