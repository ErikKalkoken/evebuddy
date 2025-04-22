package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type NotificationFolder struct {
	group  app.NotificationGroup
	Name   string
	Unread optional.Optional[int]
	Total  optional.Optional[int]
}

type CharacterCommunications struct {
	widget.BaseWidget

	Detail        *fyne.Container
	Notifications fyne.CanvasObject
	OnSelected    func()
	OnUpdate      func(count optional.Optional[int])
	Toolbar       *widget.Toolbar

	current            *app.CharacterNotification
	folderList         *widget.List
	folders            []NotificationFolder
	foldersTop         *widget.Label
	notificationList   *widget.List
	notifications      []*app.CharacterNotification
	notificationsTop   *widget.Label
	notificationsCount optional.Optional[int]
	u                  *BaseUI
}

func NewCharacterCommunications(u *BaseUI) *CharacterCommunications {
	a := &CharacterCommunications{
		folders:          make([]NotificationFolder, 0),
		notifications:    make([]*app.CharacterNotification, 0),
		notificationsTop: widget.NewLabel(""),
		foldersTop:       widget.NewLabel(""),
		u:                u,
	}
	a.ExtendBaseWidget(a)
	a.Toolbar = a.makeToolbar()
	a.Toolbar.Hide()
	a.folderList = a.makeFolderList()
	a.Detail = container.NewVBox()
	a.notificationList = a.makeNotificationList()
	a.Notifications = container.NewBorder(a.notificationsTop, nil, nil, nil, a.notificationList)
	return a
}

func (a *CharacterCommunications) CreateRenderer() fyne.WidgetRenderer {
	split1 := container.NewHSplit(
		a.Notifications,
		container.NewBorder(a.Toolbar, nil, nil, nil, a.Detail),
	)
	split1.Offset = 0.35
	split2 := container.NewHSplit(
		container.NewBorder(a.foldersTop, nil, nil, nil, a.folderList),
		split1,
	)
	split2.Offset = 0.15
	p := theme.Padding()
	c := container.NewBorder(
		widget.NewSeparator(),
		nil,
		nil,
		nil,
		container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), split2),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterCommunications) MakeFolderMenu() []*fyne.MenuItem {
	items2 := make([]*fyne.MenuItem, 0)
	for _, f := range a.folders {
		s := f.Name
		if f.Unread.ValueOrZero() > 0 {
			s += fmt.Sprintf(" (%s)", ihumanize.OptionalComma(f.Unread, "?"))
		}
		it := fyne.NewMenuItem(s, func() {
			a.setCurrentFolder(f.group)
		})
		items2 = append(items2, it)
	}
	return items2
}

func (a *CharacterCommunications) makeFolderList() *widget.List {
	maxGroup := slices.MaxFunc(app.NotificationGroups(), func(a, b app.NotificationGroup) int {
		return strings.Compare(a.String(), b.String())
	})
	l := widget.NewList(
		func() int {
			return len(a.folders)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(maxGroup.String()+"WWWWWWWW"), layout.NewSpacer(), kwidget.NewBadge("999"),
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
			text := c.Name
			if c.Unread.ValueOrZero() > 0 {
				label.TextStyle.Bold = true
				badge.SetText(ihumanize.OptionalComma(c.Unread, "?"))
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
		a.setCurrentFolder(o.group)
	}
	return l
}

func (a *CharacterCommunications) makeNotificationList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.notifications)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItem(a.u.EveImageService())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.notifications) {
				return
			}
			n := a.notifications[id]
			item := co.(*MailHeaderItem)
			item.Set(n.Sender, n.TitleDisplay(), n.Timestamp, n.IsRead)
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

func (a *CharacterCommunications) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.current == nil {
				return
			}
			a.u.App().Clipboard().SetContent(a.current.String())
		}),
	)
	return toolbar
}

func (a *CharacterCommunications) update() {
	a.notifications = make([]*app.CharacterNotification, 0)
	fyne.Do(func() {
		a.notificationList.Refresh()
		a.notificationList.UnselectAll()
		a.notificationsTop.SetText("")
		a.clearDetail()
	})

	var groupCounts map[app.NotificationGroup][]int
	if characterID := a.u.CurrentCharacterID(); characterID != 0 {
		var err error
		groupCounts, err = a.u.CharacterService().CountNotifications(context.Background(), characterID)
		if err != nil {
			slog.Error("communications update", "error", err)
		}
	}

	groups := make([]NotificationFolder, 0)
	var unreadCount, totalCount optional.Optional[int]
	for _, g := range app.NotificationGroups() {
		nf := NotificationFolder{
			group: g,
			Name:  g.String(),
		}
		gc, ok := groupCounts[g]
		if ok {
			nf.Total.Set(gc[0])
			nf.Unread.Set(gc[1])
			totalCount.Set(totalCount.ValueOrZero() + gc[0])
			unreadCount.Set(unreadCount.ValueOrZero() + gc[1])
		}
		if nf.Total.ValueOrZero() > 0 {
			groups = append(groups, nf)
		}
	}
	slices.SortFunc(groups, func(a, b NotificationFolder) int {
		return cmp.Compare(a.Name, b.Name)
	})
	if unreadCount.ValueOrZero() > 0 {
		groups = slices.Insert(groups, 0, NotificationFolder{
			group:  app.GroupUnread,
			Name:   "Unread",
			Unread: unreadCount,
		})
	}
	groups = append(groups, NotificationFolder{
		group:  app.GroupAll,
		Name:   "All",
		Unread: unreadCount,
	})
	a.notificationsCount = totalCount
	a.folders = groups
	a.foldersTop.Text, a.foldersTop.Importance = a.makeFolderTopText()
	fyne.Do(func() {
		a.folderList.Refresh()
		a.folderList.UnselectAll()
		a.foldersTop.Refresh()
	})
	if a.OnUpdate != nil {
		a.OnUpdate(unreadCount)
	}
}

func (a *CharacterCommunications) makeFolderTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService().CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for data to load...", widget.WarningImportance
	}
	return fmt.Sprintf("%s messages", ihumanize.OptionalComma(a.notificationsCount, "?")), widget.MediumImportance
}

func (a *CharacterCommunications) resetCurrentFolder() {
	a.setCurrentFolder(app.GroupUnread)
}

func (a *CharacterCommunications) setCurrentFolder(nc app.NotificationGroup) {
	ctx := context.Background()
	characterID := a.u.CurrentCharacterID()
	var notifications []*app.CharacterNotification
	var err error
	switch nc {
	case app.GroupAll:
		notifications, err = a.u.CharacterService().ListNotificationsAll(ctx, characterID)
	case app.GroupUnread:
		notifications, err = a.u.CharacterService().ListNotificationsUnread(ctx, characterID)
	default:
		notifications, err = a.u.CharacterService().ListNotificationsTypes(ctx, characterID, nc)
	}
	a.notifications = notifications
	var top string
	var importance widget.Importance
	if err != nil {
		slog.Error("communications set group", "characterID", characterID, "error", err)
		top = "Something went wrong"
		importance = widget.DangerImportance
	} else {
		top = fmt.Sprintf("%s • %s messages", nc.String(), humanize.Comma(int64(len(notifications))))
	}
	a.notificationsTop.Text = top
	a.notificationsTop.Importance = importance
	a.notificationsTop.Refresh()
	a.Notifications.Refresh()
}

func (a *CharacterCommunications) clearDetail() {
	a.Detail.RemoveAll()
	a.Toolbar.Hide()
	a.current = nil
}

// TODO: Refactor to avoid recreating the container every time
func (a *CharacterCommunications) setDetail(n *app.CharacterNotification) {
	if n.RecipientName == "" && a.u.hasCharacter() {
		n.RecipientName = a.u.currentCharacter().EveCharacter.Name
	}
	a.Detail.RemoveAll()
	subject := iwidget.NewLabelWithSize(n.TitleDisplay(), theme.SizeNameSubHeadingText)
	subject.Wrapping = fyne.TextWrapWord
	a.Detail.Add(subject)
	h := NewMailHeader(a.u.EveImageService(), a.u.ShowEveEntityInfoWindow)
	h.Set(n.Sender, n.Timestamp, a.u.currentCharacter().EveCharacter.ToEveEntity())
	a.Detail.Add(h)
	s, err := n.BodyPlain() // using markdown blocked by #61
	if err != nil {
		slog.Warn("failed to convert markdown", "notificationID", n.ID, "text", n.Body.ValueOrZero())
	}
	body := widget.NewRichTextWithText(s.ValueOrZero())
	body.Wrapping = fyne.TextWrapWord
	if n.Body.IsEmpty() {
		body.ParseMarkdown("*This notification type is not fully supported yet*")
	}
	a.Detail.Add(body)
	a.current = n
	a.Toolbar.Show()
}
