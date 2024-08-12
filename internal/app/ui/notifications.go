package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// notificationsArea is the UI area that shows the skillqueue
type notificationsArea struct {
	content       *fyne.Container
	notifications []*app.CharacterNotification
	top           *widget.Label
	ui            *ui
	list          *widget.List
}

func (u *ui) newNotificationsArea() *notificationsArea {
	a := notificationsArea{
		notifications: make([]*app.CharacterNotification, 0),
		top:           widget.NewLabel(""),
		ui:            u,
	}
	a.top.TextStyle.Bold = true
	a.list = a.makeNotificationList()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.list)
	return &a
}

func (a *notificationsArea) makeNotificationList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.notifications)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("time"),
				layout.NewSpacer(),
				widget.NewLabel("title"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.notifications) {
				return
			}
			n := a.notifications[id]
			row := co.(*fyne.Container)
			timestamp := row.Objects[0].(*widget.Label)
			timestamp.Text = n.Timestamp.Format(time.RFC3339)
			timestamp.TextStyle.Bold = !n.IsRead
			timestamp.Refresh()
			title := row.Objects[2].(*widget.Label)
			title.Text = n.Type
			title.TextStyle.Bold = !n.IsRead
			title.Refresh()
		})
	// l.OnSelected = func(id widget.ListItemID) {
	// 	defer l.UnselectAll()
	// 	if id >= len(a.notifications) {
	// 		return
	// 	}
	// 	o := a.notifications[id]
	// 	a.ui.showTypeInfoWindow(o.EveType.ID, a.ui.characterID())
	// }
	return l
}

func (a *notificationsArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateNotifications(); err != nil {
		slog.Error("Failed to refresh notifications UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *notificationsArea) updateNotifications() error {
	if !a.ui.hasCharacter() {
		a.notifications = make([]*app.CharacterNotification, 0)
		return nil
	}
	notifications, err := a.ui.CharacterService.ListCharacterNotifications(context.TODO(), a.ui.characterID())
	if err != nil {
		return err
	}
	a.notifications = notifications
	a.list.Refresh()
	return nil
}

func (a *notificationsArea) makeTopText() (string, widget.Importance) {
	hasData := a.ui.StatusCacheService.CharacterSectionExists(a.ui.characterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	var unread int
	for _, n := range a.notifications {
		if !n.IsRead {
			unread++
		}
	}
	return fmt.Sprintf("%d notifications (%d unread)", len(a.notifications), unread), widget.MediumImportance
}
