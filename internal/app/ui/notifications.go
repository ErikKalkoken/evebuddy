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

type NotificationFolder struct {
	Folder      evenotification.Folder
	Name        string
	UnreadCount int
}

// NotificationsArea is the UI area that shows the skillqueue
type NotificationsArea struct {
	Content       fyne.CanvasObject
	Detail        *fyne.Container
	Notifications fyne.CanvasObject
	Toolbar       *widget.Toolbar
	OnSelected    func()
	OnRefresh     func(count int)

	Folders          []NotificationFolder
	folderList       *widget.List
	current          *app.CharacterNotification
	notificationList *widget.List
	notifications    []*app.CharacterNotification
	notificationsTop *widget.Label
	folderTop        *widget.Label
	u                *BaseUI
}

func (u *BaseUI) NewNotificationsArea() *NotificationsArea {
	a := NotificationsArea{
		Folders:          make([]NotificationFolder, 0),
		notifications:    make([]*app.CharacterNotification, 0),
		notificationsTop: widget.NewLabel(""),
		folderTop:        widget.NewLabel(""),
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
		container.NewBorder(a.folderTop, nil, nil, nil, a.folderList),
		split1,
	)
	split2.Offset = 0.15
	a.Content = split2
	return &a
}

func (a *NotificationsArea) makeFolderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.Folders)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("template"), layout.NewSpacer(), kwidget.NewBadge("999"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.Folders) {
				return
			}
			c := a.Folders[id]
			hbox := co.(*fyne.Container).Objects
			label := hbox[0].(*widget.Label)
			badge := hbox[2].(*kwidget.Badge)
			text := c.Name
			if c.UnreadCount > 0 {
				label.TextStyle.Bold = true
				badge.SetText(strconv.Itoa(c.UnreadCount))
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
		if id >= len(a.Folders) {
			l.UnselectAll()
			return
		}
		o := a.Folders[id]
		a.clearDetail()
		a.SetFolder(o.Folder)
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
			l.UnselectAll()
			return
		}
		a.setDetail(a.notifications[id])
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
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
	folders := make([]NotificationFolder, 0)
	var unreadTotal int
	for _, c := range evenotification.Folders() {
		nc := NotificationFolder{
			Folder:      c,
			Name:        c.String(),
			UnreadCount: counts[c],
		}
		folders = append(folders, nc)
		unreadTotal += counts[c]
	}
	slices.SortFunc(folders, func(a, b NotificationFolder) int {
		return cmp.Compare(a.Name, b.Name)
	})
	f1 := NotificationFolder{
		Folder:      evenotification.Unread,
		Name:        "Unread",
		UnreadCount: unreadTotal,
	}
	folders = slices.Insert(folders, 0, f1)
	f2 := NotificationFolder{
		Folder:      evenotification.All,
		Name:        "All",
		UnreadCount: unreadTotal,
	}
	folders = append(folders, f2)
	a.Folders = folders
	a.folderList.Refresh()
	a.folderList.UnselectAll()
	a.folderTop.Text, a.folderTop.Importance = a.makeFolderTopText()
	a.folderTop.Refresh()
	if a.OnRefresh != nil {
		a.OnRefresh(unreadTotal)
	}
}

func (a *NotificationsArea) makeFolderTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for data to load...", widget.WarningImportance
	}
	return fmt.Sprintf("%d folders", len(a.Folders)), widget.MediumImportance
}

func (a *NotificationsArea) ResetFolders() {
	a.SetFolder(evenotification.Unread)
}

func (a *NotificationsArea) SetFolder(nc evenotification.Folder) {
	ctx := context.Background()
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
	a.notifications = notifications
	var top string
	var importance widget.Importance
	if err != nil {
		slog.Error("set notifications", "characterID", characterID, "error", err)
		top = "Something went wrong"
		importance = widget.DangerImportance
	} else {
		top = fmt.Sprintf("%s • %s notifications", nc.String(), humanize.Comma(int64(len(notifications))))
	}
	a.notificationsTop.Text = top
	a.notificationsTop.Importance = importance
	a.notificationsTop.Refresh()
	a.Notifications.Refresh()
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
	subject := widgets.NewSubHeading(n.TitleDisplay())
	subject.Wrapping = fyne.TextWrapWord
	a.Detail.Add(subject)
	a.Detail.Add(widget.NewLabel(n.Header()))
	body := widget.NewRichTextFromMarkdown(markdownStripLinks(n.Body.ValueOrZero()))
	body.Wrapping = fyne.TextWrapWord
	if n.Body.IsEmpty() {
		body.ParseMarkdown("*This notification type is not fully supported yet*")
	}
	a.Detail.Add(body)
	a.current = n
	a.Toolbar.Show()
}
