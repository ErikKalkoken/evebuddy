package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type trainingCharacter struct {
	id            int32
	name          string
	totalSP       optional.Optional[int]
	training      optional.Optional[time.Duration]
	unallocatedSP optional.Optional[int]
}

// trainingArea is the UI area that shows an overview of all the user's characters.
type trainingArea struct {
	characters []trainingCharacter
	content    *fyne.Container
	table      *widget.Table
	top        *widget.Label
	u          *UI
}

func (u *UI) newTrainingArea() *trainingArea {
	a := trainingArea{
		characters: make([]trainingCharacter, 0),
		top:        widget.NewLabel(""),
		u:          u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *trainingArea) makeTable() *widget.Table {
	var headers = []struct {
		text     string
		maxChars int
	}{
		{"Name", 20},
		{"SP", 5},
		{"Unall. SP", 5},
		{"Training", 5},
	}

	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.characters), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			if tci.Row >= len(a.characters) || tci.Row < 0 {
				return
			}
			c := a.characters[tci.Row]
			l.Alignment = fyne.TextAlignLeading
			l.Importance = widget.MediumImportance
			var text string
			switch tci.Col {
			case 0:
				text = c.name
			case 1:
				text = ihumanize.Optional(c.totalSP, "?")
				l.Alignment = fyne.TextAlignTrailing
			case 2:
				text = ihumanize.Optional(c.unallocatedSP, "?")
				l.Alignment = fyne.TextAlignTrailing
			case 3:
				if c.training.IsEmpty() {
					text = "Inactive"
					l.Importance = widget.WarningImportance
				} else {
					text = ihumanize.Duration(c.training.ValueOrZero())
				}
			}
			l.Text = text
			l.Truncation = fyne.TextTruncateClip
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		label := co.(*widget.Label)
		label.SetText(s.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
	}

	for i, h := range headers {
		x := widget.NewLabel(strings.Repeat("w", h.maxChars))
		w := x.MinSize().Width
		t.SetColumnWidth(i, w)
	}
	return t
}

func (a *trainingArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		totalSP, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.characters) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		spText := ihumanize.Optional(totalSP, "?")
		s := fmt.Sprintf("%d characters • %s Total SP", len(a.characters), spText)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh overview UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.table.Refresh()
}

func (a *trainingArea) updateCharacters() (optional.Optional[int], error) {
	var totalSP optional.Optional[int]
	var err error
	ctx := context.TODO()
	mycc, err := a.u.CharacterService.ListCharacters(ctx)
	if err != nil {
		return totalSP, err
	}
	cc := make([]trainingCharacter, len(mycc))
	for i, m := range mycc {
		c := trainingCharacter{
			id:            m.ID,
			name:          m.EveCharacter.Name,
			totalSP:       m.TotalSP,
			unallocatedSP: m.UnallocatedSP,
		}
		cc[i] = c
	}
	for i, c := range cc {
		v, err := a.u.CharacterService.GetCharacterTotalTrainingTime(ctx, c.id)
		if err != nil {
			return totalSP, err
		}
		cc[i].training = v
	}
	for _, c := range cc {
		if !c.totalSP.IsEmpty() {
			totalSP.Set(totalSP.ValueOrZero() + c.totalSP.ValueOrZero())
		}
	}
	a.characters = cc
	return totalSP, nil
}
