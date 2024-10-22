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
	"fyne.io/fyne/v2/theme"
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
	u       *UI

	categories   []notificationCategory
	categoryList *widget.List
	top          *widget.Label

	notifications    []*app.CharacterNotification
	notificationList *widget.List
	notificationsTop *widget.Label

	current *app.CharacterNotification
	detail  *fyne.Container
	toolbar *widget.Toolbar
}

func (u *UI) newNotificationsArea() *notificationsArea {
	a := notificationsArea{
		categories:       make([]notificationCategory, 0),
		notifications:    make([]*app.CharacterNotification, 0),
		notificationsTop: widget.NewLabel(""),
		top:              widget.NewLabel(""),
		u:                u,
	}
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	a.detail = container.NewVBox()
	a.notificationList = a.makeNotificationList()
	split1 := container.NewHSplit(
		container.NewBorder(a.notificationsTop, nil, nil, nil, a.notificationList),
		container.NewBorder(a.toolbar, nil, nil, nil, a.detail),
	)
	split1.Offset = 0.35
	a.categoryList = a.makeCategoryList()
	a.content = container.NewHSplit(
		container.NewBorder(a.top, nil, nil, nil, a.categoryList),
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
		a.clearDetail()
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
		a.clearDetail()
		if id >= len(a.notifications) {
			defer l.UnselectAll()
			return
		}
		a.setDetail(a.notifications[id])
	}
	return l
}

func (a *notificationsArea) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.current == nil {
				return
			}
			a.u.window.Clipboard().SetContent(a.current.String())
		}),
	)
	return toolbar
}

func (a *notificationsArea) refresh() {
	a.clearDetail()
	a.notifications = make([]*app.CharacterNotification, 0)
	a.notificationList.Refresh()
	a.notificationList.UnselectAll()
	a.notificationsTop.SetText("")
	var counts map[evenotification.Category]int
	if characterID := a.u.characterID(); characterID != 0 {
		var err error
		counts, err = a.u.CharacterService.CountCharacterNotificationUnreads(context.TODO(), characterID)
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
	c1 := notificationCategory{
		category: evenotification.Unread,
		name:     "Unread",
		unread:   unreadTotal,
	}
	categories = slices.Insert(categories, 0, c1)
	c2 := notificationCategory{
		category: evenotification.All,
		name:     "All",
		unread:   unreadTotal,
	}
	categories = append(categories, c2)
	a.categories = categories
	a.categoryList.Refresh()
	a.categoryList.UnselectAll()
	a.top.Text, a.top.Importance = a.makeTopText()
	a.top.Refresh()
}

func (a *notificationsArea) makeTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.characterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for data to load...", widget.WarningImportance
	}
	return fmt.Sprintf("%d categories", len(a.categories)), widget.MediumImportance
}

func (a *notificationsArea) setNotifications(nc evenotification.Category) error {
	ctx := context.TODO()
	characterID := a.u.characterID()
	var notifications []*app.CharacterNotification
	var err error
	switch nc {
	case evenotification.All:
		notifications, err = a.u.CharacterService.ListCharacterNotificationsAll(ctx, characterID)
	case evenotification.Unread:
		notifications, err = a.u.CharacterService.ListCharacterNotificationsUnread(ctx, characterID)
	default:
		types := evenotification.CategoryTypes[nc]
		notifications, err = a.u.CharacterService.ListCharacterNotificationsTypes(ctx, characterID, types)
	}
	if err != nil {
		return err
	}
	a.notifications = notifications
	a.notificationList.Refresh()
	a.notificationsTop.SetText(fmt.Sprintf("%s notifications", humanize.Comma(int64(len(notifications)))))
	return nil
}

func (a *notificationsArea) clearDetail() {
	a.detail.RemoveAll()
	a.toolbar.Hide()
	a.current = nil
}

func (a *notificationsArea) setDetail(n *app.CharacterNotification) {
	if n.RecipientName == "" && a.u.hasCharacter() {
		n.RecipientName = a.u.currentCharacter().EveCharacter.Name
	}
	a.detail.RemoveAll()
	subject := widget.NewLabel(n.TitleDisplay())
	subject.TextStyle.Bold = true
	a.detail.Add(subject)
	a.detail.Add(widget.NewLabel(n.Header()))
	body := widget.NewRichTextFromMarkdown(n.Body.ValueOrZero())
	body.Wrapping = fyne.TextWrapWord
	if n.Body.IsEmpty() {
		body.ParseMarkdown("*This notification type is not fully supported yet*")
	}
	a.detail.Add(body)
	a.current = n
	a.toolbar.Show()
}
