package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/dustin/go-humanize"
)

type notificationCategory struct {
	id     app.NotificationCategory
	name   string
	unread int
}

// notificationsArea is the UI area that shows the skillqueue
type notificationsArea struct {
	content          *fyne.Container
	notifications    []*app.CharacterNotification
	top              *widget.Label
	ui               *ui
	notificationList *widget.List
	categoryList     *widget.List
	categories       []notificationCategory
}

func (u *ui) newNotificationsArea() *notificationsArea {
	a := notificationsArea{
		notifications: make([]*app.CharacterNotification, 0),
		top:           widget.NewLabel(""),
		ui:            u,
		categories:    make([]notificationCategory, 0),
	}
	a.top.TextStyle.Bold = true
	a.categoryList = a.makeCategoryList()
	a.notificationList = a.makeNotificationList()
	top := container.NewVBox(a.top, widget.NewSeparator())
	split := container.NewHSplit(a.categoryList, a.notificationList)
	split.Offset = 0.2
	a.content = container.NewBorder(top, nil, nil, nil, split)
	return &a
}

func (a *notificationsArea) makeCategoryList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.categories)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("title")
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.categories) {
				return
			}
			c := a.categories[id]
			label := co.(*widget.Label)
			text := c.name
			if c.unread > 0 {
				text += fmt.Sprintf(" (%s)", humanize.Comma(int64(c.unread)))
				label.TextStyle.Bold = true
			} else {
				label.TextStyle.Bold = false
			}
			label.Text = text
			label.Refresh()
		})
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.categories) {
			l.UnselectAll()
			return
		}
		o := a.categories[id]
		if err := a.loadNotifications(o.id); err != nil {
			slog.Error("Failed to load notifications", "err", err)
			l.UnselectAll()
		}
	}
	return l
}

func (a *notificationsArea) makeNotificationList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.notifications)
		},
		func() fyne.CanvasObject {
			return widgets.NewMailHeaderItem(myDateTime)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.notifications) {
				return
			}
			n := a.notifications[id]
			item := co.(*widgets.MailHeaderItem)
			item.Set(n.Sender.Name, n.Title(), n.Timestamp, n.IsRead)
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
	a.notifications = make([]*app.CharacterNotification, 0)
	a.notificationList.Refresh()
	var counts map[app.NotificationCategory]int
	if characterID := a.ui.characterID(); characterID != 0 {
		var err error
		counts, err = a.ui.CharacterService.CalcCharacterNotificationUnreadCounts(context.TODO(), characterID)
		if err != nil {
			slog.Error("Failed to fetch notification unread counts", "error", err)
		}
	}
	categories := make([]notificationCategory, 0)
	for id, name := range app.NotificationCategoryNames {
		nc := notificationCategory{
			id:     id,
			name:   name,
			unread: counts[id],
		}
		categories = append(categories, nc)
	}
	slices.SortFunc(categories, func(a, b notificationCategory) int {
		return cmp.Compare(a.name, b.name)
	})
	a.categories = categories
	a.categoryList.Refresh()
}

func (a *notificationsArea) loadNotifications(nc app.NotificationCategory) error {
	types := app.NotificationCategoryTypes[nc]
	notifications, err := a.ui.CharacterService.ListCharacterNotifications(context.TODO(), a.ui.characterID(), types)
	if err != nil {
		return err
	}
	a.notifications = notifications
	a.notificationList.Refresh()
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
