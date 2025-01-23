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

type notificationFolder struct {
	folder evenotification.Folder
	name   string
	unread int
}

// NotificationsArea is the UI area that shows the skillqueue
type NotificationsArea struct {
	Content       fyne.CanvasObject
	Detail        *fyne.Container
	Notifications fyne.CanvasObject
	Toolbar       *widget.Toolbar

	folders          []notificationFolder
	folderList       *widget.List
	current          *app.CharacterNotification
	notificationList *widget.List
	notifications    []*app.CharacterNotification
	notificationsTop *widget.Label
	top              *widget.Label
	u                *BaseUI
}

func (u *BaseUI) NewNotificationsArea() *NotificationsArea {
	a := NotificationsArea{
		folders:          make([]notificationFolder, 0),
		notifications:    make([]*app.CharacterNotification, 0),
		notificationsTop: widget.NewLabel(""),
		top:              widget.NewLabel(""),
		u:                u,
	}
	a.Toolbar = a.makeToolbar()
	a.Toolbar.Hide()
	a.Detail = container.NewVBox()
	a.notificationList = a.makeNotificationList()
	a.Notifications = container.NewBorder(a.notificationsTop, nil, nil, nil, a.notificationList)
	split1 := container.NewHSplit(
		a.Notifications,
		container.NewBorder(a.Toolbar, nil, nil, nil, a.Detail),
	)
	split1.Offset = 0.35
	a.folderList = a.makeFolderList()
	split2 := container.NewHSplit(
		container.NewBorder(a.top, nil, nil, nil, a.folderList),
		split1,
	)
	split2.Offset = 0.15
	a.Content = split2
	return &a
}

func (a *NotificationsArea) makeFolderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.folders)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("template"), layout.NewSpacer(), kwidget.NewBadge("999"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.folders) {
				return
			}
			c := a.folders[id]
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
		if id >= len(a.folders) {
			l.UnselectAll()
			return
		}
		o := a.folders[id]
		a.clearDetail()
		if err := a.setNotifications(o.folder); err != nil {
			slog.Error("Failed to load notifications", "err", err)
			l.UnselectAll()
		}
	}
	return l
}

func (a *NotificationsArea) makeNotificationList() *widget.List {
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

func (a *NotificationsArea) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.current == nil {
				return
			}
			a.u.Window.Clipboard().SetContent(a.current.String())
		}),
	)
	return toolbar
}

func (a *NotificationsArea) Refresh() {
	a.clearDetail()
	a.notifications = make([]*app.CharacterNotification, 0)
	a.notificationList.Refresh()
	a.notificationList.UnselectAll()
	a.notificationsTop.SetText("")
	var counts map[evenotification.Folder]int
	if characterID := a.u.CharacterID(); characterID != 0 {
		var err error
		counts, err = a.u.CharacterService.CountCharacterNotificationUnreads(context.TODO(), characterID)
		if err != nil {
			slog.Error("Failed to fetch notification unread counts", "error", err)
		}
	}
	categories := make([]notificationFolder, 0)
	var unreadTotal int
	for _, c := range evenotification.Folders() {
		nc := notificationFolder{
			folder: c,
			name:   c.String(),
			unread: counts[c],
		}
		categories = append(categories, nc)
		unreadTotal += counts[c]
	}
	slices.SortFunc(categories, func(a, b notificationFolder) int {
		return cmp.Compare(a.name, b.name)
	})
	c1 := notificationFolder{
		folder: evenotification.Unread,
		name:   "Unread",
		unread: unreadTotal,
	}
	categories = slices.Insert(categories, 0, c1)
	c2 := notificationFolder{
		folder: evenotification.All,
		name:   "All",
		unread: unreadTotal,
	}
	categories = append(categories, c2)
	a.folders = categories
	a.folderList.Refresh()
	a.folderList.UnselectAll()
	a.top.Text, a.top.Importance = a.makeTopText()
	a.top.Refresh()
}

func (a *NotificationsArea) makeTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for data to load...", widget.WarningImportance
	}
	return fmt.Sprintf("%d categories", len(a.folders)), widget.MediumImportance
}

func (a *NotificationsArea) setNotifications(nc evenotification.Folder) error {
	ctx := context.TODO()
	characterID := a.u.CharacterID()
	var notifications []*app.CharacterNotification
	var err error
	switch nc {
	case evenotification.All:
		notifications, err = a.u.CharacterService.ListCharacterNotificationsAll(ctx, characterID)
	case evenotification.Unread:
		notifications, err = a.u.CharacterService.ListCharacterNotificationsUnread(ctx, characterID)
	default:
		types := evenotification.FolderTypes[nc]
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

func (a *NotificationsArea) clearDetail() {
	a.Detail.RemoveAll()
	a.Toolbar.Hide()
	a.current = nil
}

func (a *NotificationsArea) setDetail(n *app.CharacterNotification) {
	if n.RecipientName == "" && a.u.HasCharacter() {
		n.RecipientName = a.u.CurrentCharacter().EveCharacter.Name
	}
	a.Detail.RemoveAll()
	subject := widget.NewLabel(n.TitleDisplay())
	subject.TextStyle.Bold = true
	a.Detail.Add(subject)
	a.Detail.Add(widget.NewLabel(n.Header()))
	body := widget.NewRichTextFromMarkdown(n.Body.ValueOrZero())
	body.Wrapping = fyne.TextWrapWord
	if n.Body.IsEmpty() {
		body.ParseMarkdown("*This notification type is not fully supported yet*")
	}
	a.Detail.Add(body)
	a.current = n
	a.Toolbar.Show()
}
