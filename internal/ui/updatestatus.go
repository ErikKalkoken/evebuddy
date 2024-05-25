package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

func (u *ui) showStatusDialog(c model.CharacterShort) {
	content := makeCharacterStatus(u, c)
	d1 := dialog.NewCustom("Character update status", "Close", content, u.window)
	d1.Show()
	d1.Resize(fyne.Size{Width: 800, Height: 500})
}

func makeCharacterStatus(u *ui, c model.CharacterShort) fyne.CanvasObject {
	data := u.service.CharacterListUpdateStatus(c.ID)
	var headers = []struct {
		text  string
		width float32
	}{
		{"Section", 150},
		{"Timeout", 150},
		{"Last Update", 150},
		{"Status", 150},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(data), len(headers)

		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("Placeholder")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			d := data[tci.Row]
			label := co.(*widget.Label)
			var s string
			i := widget.MediumImportance
			switch tci.Col {
			case 0:
				s = d.Section
			case 1:
				now := time.Now()
				s = humanize.RelTime(now.Add(d.Timeout), now, "", "")
			case 2:
				s = humanizedNullTime(d.LastUpdatedAt, "?")
			case 3:
				if d.IsOK() {
					s = "OK"
					i = widget.SuccessImportance
				} else {
					s = d.ErrorMessage
					i = widget.DangerImportance
				}
			}
			label.Text = s
			label.Importance = i
			label.Refresh()
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
	t.OnSelected = func(id widget.TableCellID) {
		t.UnselectAll()
	}

	top := widget.NewLabel(fmt.Sprintf("Update status for %s", c.Name))
	top.TextStyle.Bold = true
	return container.NewBorder(top, nil, nil, nil, t)
}
