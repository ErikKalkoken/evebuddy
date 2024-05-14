package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// headerArea is the UI area showing the list of mail headers.
type headerArea struct {
	content       fyne.CanvasObject
	currentFolder node
	infoText      *widget.Label
	list          *widget.List
	lastSelected  widget.ListItemID
	mailIDs       []int32
	ui            *ui
}

func (u *ui) NewHeaderArea() *headerArea {
	a := headerArea{
		mailIDs:  make([]int32, 0),
		ui:       u,
		infoText: widget.NewLabel(""),
	}

	foregroundColor := theme.ForegroundColor()
	subjectSize := theme.TextSize() * 1.15
	list := widget.NewList(
		func() int {
			return len(a.mailIDs)
		},
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
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			characterID := u.CurrentCharID()
			if characterID == 0 {
				return
			}
			m, err := u.service.GetMail(characterID, a.mailIDs[lii])
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
		mailID := a.mailIDs[id]
		u.mailArea.SetMail(mailID, id)
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

	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		a.mailIDs, err = a.ui.service.ListMailIDsForLabelOrdered(folder.MyCharacterID, folder.ObjID)
	case nodeCategoryList:
		a.mailIDs, err = a.ui.service.ListMailIDsForListOrdered(folder.MyCharacterID, folder.ObjID)
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

	if len(a.mailIDs) == 0 {
		a.ui.mailArea.Clear()
		return
	}
}

func (a *headerArea) makeTopText() (string, widget.Importance) {
	hasData, err := a.ui.service.SectionWasUpdated(a.ui.CurrentCharID(), service.UpdateSectionSkillqueue)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data yet...", widget.LowImportance
	}
	s := fmt.Sprintf("%d mails", len(a.mailIDs))
	return s, widget.MediumImportance

}
