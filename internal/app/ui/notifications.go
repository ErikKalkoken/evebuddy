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
	content *container.Split
	ui      *ui

	categories   []notificationCategory
	categoryList *widget.List

	notifications    []*app.CharacterNotification
	notificationList *widget.List

	detail *fyne.Container
}

func (u *ui) newNotificationsArea() *notificationsArea {
	a := notificationsArea{
		notifications: make([]*app.CharacterNotification, 0),
		ui:            u,
		categories:    make([]notificationCategory, 0),
	}
	a.detail = container.NewVBox()
	a.notificationList = a.makeNotificationList()
	split1 := container.NewHSplit(a.notificationList, a.detail)
	split1.Offset = 0.35
	a.categoryList = a.makeCategoryList()
	a.content = container.NewHSplit(a.categoryList, split1)
	a.content.Offset = 0.15
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
		a.notificationList.UnselectAll()
		if id >= len(a.categories) {
			l.UnselectAll()
			return
		}
		o := a.categories[id]
		a.detail.RemoveAll()
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
			item.Set(n.Sender.Name, n.FakeTitle(), n.Timestamp, n.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		a.detail.RemoveAll()
		if id >= len(a.notifications) {
			defer l.UnselectAll()
			return
		}
		a.setDetails(a.notifications[id])
	}
	return l
}

func (a *notificationsArea) refresh() {
	a.detail.RemoveAll()
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
	var unreadTotal int
	for id, name := range app.NotificationCategoryNames {
		nc := notificationCategory{
			id:     id,
			name:   name,
			unread: counts[id],
		}
		categories = append(categories, nc)
		unreadTotal += counts[id]
	}
	slices.SortFunc(categories, func(a, b notificationCategory) int {
		return cmp.Compare(a.name, b.name)
	})
	unread := notificationCategory{name: "Unread", unread: unreadTotal}
	categories = slices.Insert(categories, 0, unread)
	a.categories = categories
	a.categoryList.Refresh()
}

func (a *notificationsArea) loadNotifications(nc app.NotificationCategory) error {
	ctx := context.TODO()
	characterID := a.ui.characterID()
	var notifications []*app.CharacterNotification
	var err error
	if nc == 0 {
		notifications, err = a.ui.CharacterService.ListCharacterNotificationsUnread(ctx, characterID)
	} else {
		types := app.NotificationCategoryTypes[nc]
		notifications, err = a.ui.CharacterService.ListCharacterNotificationsTypes(ctx, characterID, types)
	}
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

func (a *notificationsArea) setDetails(n *app.CharacterNotification) {
	a.detail.RemoveAll()
	subject := widget.NewLabel(n.FakeTitle())
	subject.TextStyle.Bold = true
	a.detail.Add(subject)
	header := fmt.Sprintf("From: %s\nSent: %s", n.Sender.Name, n.Timestamp.Format(myDateTime))
	a.detail.Add(widget.NewLabel(header))
	a.detail.Add(widget.NewLabel(n.Text))
}
