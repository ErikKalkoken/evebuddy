package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"

	islices "github.com/ErikKalkoken/evebuddy/internal/helper/slices"
)

// headerArea is the UI area showing the list of mail headers.
type headerArea struct {
	content       fyne.CanvasObject
	currentFolder node
	infoText      *widget.Label
	list          *widget.List
	lastSelected  widget.ListItemID
	mailIDs       binding.IntList
	ui            *ui
}

func (u *ui) NewHeaderArea() *headerArea {
	a := headerArea{
		infoText: widget.NewLabel(""),
		mailIDs:  binding.NewIntList(),
		ui:       u,
	}

	foregroundColor := theme.ForegroundColor()
	subjectSize := theme.TextSize() * 1.15
	list := widget.NewListWithData(
		a.mailIDs,
		func() fyne.CanvasObject {
			from := canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor)
			timestamp := canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor)
			subject := canvas.NewText("subject", foregroundColor)
			subject.TextSize = subjectSize
			return container.NewPadded(container.NewPadded(container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp),
				subject,
			)))
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			characterID := u.CurrentCharID()
			if characterID == 0 {
				return
			}
			mailID, err := di.(binding.Int).Get()
			if err != nil {
				panic(err)
			}
			m, err := u.service.GetMail(characterID, int32(mailID))
			if err != nil {
				if !errors.Is(err, storage.ErrNotFound) {
					slog.Error("Failed to get mail", "error", err)
				}
				return
			}
			parent := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)

			from := top.Objects[0].(*canvas.Text)
			var t string
			if u.headerArea != nil && u.headerArea.currentFolder.isSent() {
				t = strings.Join(m.RecipientNames(), ", ")
			} else {
				t = m.From.Name
			}
			from.Text = t
			from.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			from.Refresh()

			timestamp := top.Objects[2].(*canvas.Text)
			timestamp.Text = m.Timestamp.Format(myDateTime)
			timestamp.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			timestamp.Refresh()

			subject := parent.Objects[1].(*canvas.Text)
			subject.Text = m.Subject
			subject.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			subject.Refresh()
		})
	list.OnSelected = func(id widget.ListItemID) {
		di, err := a.mailIDs.GetItem(id)
		if err != nil {
			panic((err))
		}
		mailID, err := di.(binding.Int).Get()
		if err != nil {
			panic(err)
		}
		u.mailArea.SetMail(int32(mailID), id)
		u.headerArea.lastSelected = id
	}

	a.content = container.NewBorder(a.infoText, nil, nil, nil, list)
	a.list = list
	return &a
}

func (a *headerArea) SetFolder(folder node) {
	a.currentFolder = folder
	a.Refresh()
	a.list.ScrollToTop()
	a.list.UnselectAll()
	a.ui.mailArea.Clear()
}

func (a *headerArea) Refresh() {
	a.updateMails()
	a.list.Refresh()
}

func (a *headerArea) updateMails() {
	folder := a.currentFolder
	if folder.MyCharacterID == 0 {
		return
	}
	var mailIDs []int32
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		mailIDs, err = a.ui.service.ListMailIDsForLabelOrdered(folder.MyCharacterID, folder.ObjID)
	case nodeCategoryList:
		mailIDs, err = a.ui.service.ListMailIDsForListOrdered(folder.MyCharacterID, folder.ObjID)
	}
	x := islices.ConvertNumeric[int32, int](mailIDs)
	if err := a.mailIDs.Set(x); err != nil {
		panic(err)
	}
	var s string
	var i widget.Importance
	if err != nil {
		slog.Error("Failed to fetch mail", "characterID", folder.MyCharacterID, "error", err)
		s = "ERROR"
		i = widget.DangerImportance
	}
	s, i = a.makeTopText()
	a.infoText.Text = s
	a.infoText.Importance = i
	a.infoText.Refresh()

	if len(mailIDs) == 0 {
		a.ui.mailArea.Clear()
		return
	}
}

func (a *headerArea) makeTopText() (string, widget.Importance) {
	hasData, err := a.ui.service.SectionWasUpdated(a.ui.CurrentCharID(), model.UpdateSectionSkillqueue)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data yet...", widget.LowImportance
	}
	s := fmt.Sprintf("%d mails", a.mailIDs.Length())
	return s, widget.MediumImportance

}
