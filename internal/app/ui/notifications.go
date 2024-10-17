package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/dustin/go-humanize"
)

type notificationCategory struct {
	category evenotification.Category
	name     string
	unread   int
}

// notificationsArea is the UI area that shows the skillqueue
type notificationsArea struct {
	content *container.Split
	ui      *ui

	categories   []notificationCategory
	categoryList *widget.List
	categoryTop  *widget.Label

	notifications    []*app.CharacterNotification
	notificationList *widget.List
	notificationsTop *widget.Label

	detail *fyne.Container
}

func (u *ui) newNotificationsArea() *notificationsArea {
	a := notificationsArea{
		ui:               u,
		categories:       make([]notificationCategory, 0),
		categoryTop:      widget.NewLabel(""),
		notifications:    make([]*app.CharacterNotification, 0),
		notificationsTop: widget.NewLabel(""),
	}
	a.detail = container.NewVBox()
	a.notificationList = a.makeNotificationList()
	split1 := container.NewHSplit(
		container.NewBorder(a.notificationsTop, nil, nil, nil, a.notificationList),
		a.detail,
	)
	split1.Offset = 0.35
	a.categoryList = a.makeCategoryList()
	a.content = container.NewHSplit(
		container.NewBorder(a.categoryTop, nil, nil, nil, a.categoryList),
		split1,
	)
	a.content.Offset = 0.15
	return &a
}

func (a *notificationsArea) makeCategoryList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.categories)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("template"), layout.NewSpacer(), kwidget.NewBadge("999"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.categories) {
				return
			}
			c := a.categories[id]
			hbox := co.(*fyne.Container).Objects
			label := hbox[0].(*widget.Label)
			badge := hbox[2].(*kwidget.Badge)
			text := c.name
			if c.unread > 0 {
				label.TextStyle.Bold = true
				badge.SetText(strconv.Itoa(c.unread))
				badge.Show()
			} else {
				label.TextStyle.Bold = false
				badge.Hide()
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
		if err := a.setNotifications(o.category); err != nil {
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
			return widgets.NewMailHeaderItem(app.TimeDefaultFormat)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.notifications) {
				return
			}
			n := a.notifications[id]
			item := co.(*widgets.MailHeaderItem)
			item.Set(n.Sender.Name, n.TitleDisplay(), n.Timestamp, n.IsRead)
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
	a.notificationList.UnselectAll()
	a.notificationsTop.SetText("")
	var counts map[evenotification.Category]int
	if characterID := a.ui.characterID(); characterID != 0 {
		var err error
		counts, err = a.ui.CharacterService.CalcCharacterNotificationUnreadCounts(context.TODO(), characterID)
		if err != nil {
			slog.Error("Failed to fetch notification unread counts", "error", err)
		}
	}
	categories := make([]notificationCategory, 0)
	var unreadTotal int
	for _, c := range evenotification.Categories() {
		nc := notificationCategory{
			category: c,
			name:     c.String(),
			unread:   counts[c],
		}
		categories = append(categories, nc)
		unreadTotal += counts[c]
	}
	slices.SortFunc(categories, func(a, b notificationCategory) int {
		return cmp.Compare(a.name, b.name)
	})
	unread := notificationCategory{name: "Unread", unread: unreadTotal}
	categories = slices.Insert(categories, 0, unread)
	a.categories = categories
	a.categoryList.Refresh()
	a.categoryList.UnselectAll()
	a.categoryTop.SetText(fmt.Sprintf("%s total", humanize.Comma(int64(999))))
}

func (a *notificationsArea) setNotifications(nc evenotification.Category) error {
	ctx := context.TODO()
	characterID := a.ui.characterID()
	var notifications []*app.CharacterNotification
	var err error
	if nc == 0 {
		notifications, err = a.ui.CharacterService.ListCharacterNotificationsUnread(ctx, characterID)
	} else {
		types := evenotification.CategoryTypes[nc]
		notifications, err = a.ui.CharacterService.ListCharacterNotificationsTypes(ctx, characterID, types)
	}
	if err != nil {
		return err
	}
	a.notifications = notifications
	a.notificationList.Refresh()
	a.notificationsTop.SetText(fmt.Sprintf("%s notifications", humanize.Comma(int64(len(notifications)))))
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
	subject := widget.NewLabel(n.TitleDisplay())
	subject.TextStyle.Bold = true
	a.detail.Add(subject)
	header := fmt.Sprintf("From: %s\nSent: %s", n.Sender.Name, n.Timestamp.Format(app.TimeDefaultFormat))
	a.detail.Add(widget.NewLabel(header))
	body := widget.NewRichTextFromMarkdown(n.Body.ValueOrZero())
	body.Wrapping = fyne.TextWrapWord
	if n.Body.IsEmpty() {
		body.ParseMarkdown("*This notification type is not fully supported yet*")
	}
	a.detail.Add(body)
}
