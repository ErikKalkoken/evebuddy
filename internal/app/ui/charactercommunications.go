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

type notificationFolder struct {
	group  app.NotificationGroup
	Name   string
	Unread optional.Optional[int]
	Total  optional.Optional[int]
}

type characterCommunications struct {
	widget.BaseWidget

	Detail        *fyne.Container
	Notifications *fyne.Container
	OnSelected    func()
	OnUpdate      func(count optional.Optional[int])
	Toolbar       *widget.Toolbar

	current          *app.CharacterNotification
	folderList       *widget.List
	folders          []notificationFolder
	foldersTop       *widget.Label
	notificationList *widget.List
	notifications    []*app.CharacterNotification
	notificationsTop *widget.Label
	u                *baseUI
}

func newCharacterCommunications(u *baseUI) *characterCommunications {
	a := &characterCommunications{
		folders:          make([]notificationFolder, 0),
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

func (a *characterCommunications) CreateRenderer() fyne.WidgetRenderer {
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

func (a *characterCommunications) makeFolderMenu() []*fyne.MenuItem {
	items2 := make([]*fyne.MenuItem, 0)
	for _, f := range a.folders {
		s := f.Name
		if f.Unread.ValueOrZero() > 0 {
			s += fmt.Sprintf(" (%s)", ihumanize.OptionalWithComma(f.Unread, "?"))
		}
		it := fyne.NewMenuItem(s, func() {
			a.setCurrentFolder(f.group)
		})
		items2 = append(items2, it)
	}
	return items2
}

func (a *characterCommunications) makeFolderList() *widget.List {
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
				badge.SetText(ihumanize.OptionalWithComma(c.Unread, "?"))
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

func (a *characterCommunications) makeNotificationList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.notifications)
		},
		func() fyne.CanvasObject {
			return newMailHeaderItem(a.u.eis)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.notifications) {
				return
			}
			n := a.notifications[id]
			item := co.(*mailHeaderItem)
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

// TODO: Refactor to avoid recreating the container every time
func (a *characterCommunications) setDetail(n *app.CharacterNotification) {
	if n.RecipientName == "" && a.u.hasCharacter() {
		n.RecipientName = a.u.currentCharacter().EveCharacter.Name
	}
	a.Detail.RemoveAll()
	subject := widget.NewLabel(n.TitleDisplay())
	subject.SizeName = theme.SizeNameSubHeadingText
	subject.Wrapping = fyne.TextWrapWord
	a.Detail.Add(subject)
	h := newMailHeader(a.u.eis, a.u.ShowEveEntityInfoWindow)
	h.Set(n.Sender, n.Timestamp, a.u.currentCharacter().EveCharacter.ToEveEntity())
	a.Detail.Add(h)
	s, err := n.BodyPlain() // using markdown blocked by #61
	if err != nil {
		slog.Warn("failed to convert markdown", "notificationID", n.ID, "text", n.Body.ValueOrZero())
	}
	body := iwidget.NewRichTextWithText(s.ValueOrZero())
	body.Wrapping = fyne.TextWrapWord
	if n.Body.IsEmpty() {
		body.ParseMarkdown("*This notification type is not fully supported yet*")
	}
	a.Detail.Add(body)
	a.current = n
	a.Toolbar.Show()
}

func (a *characterCommunications) makeToolbar() *widget.Toolbar {
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

func (a *characterCommunications) update() {
	var err error
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasCharacterSection(a.u.currentCharacterID(), app.SectionCharacterNotifications)
	groups := make([]notificationFolder, 0)
	var unreadCount, totalCount optional.Optional[int]
	if characterID != 0 && hasData {
		groupCounts, err2 := a.u.cs.CountNotifications(context.Background(), characterID)
		if err2 != nil {
			slog.Error("communications update", "error", err)
			err = err2
		}

		for _, g := range app.NotificationGroups() {
			nf := notificationFolder{
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
		slices.SortFunc(groups, func(a, b notificationFolder) int {
			return cmp.Compare(a.Name, b.Name)
		})
		if unreadCount.ValueOrZero() > 0 {
			groups = slices.Insert(groups, 0, notificationFolder{
				group:  app.GroupUnread,
				Name:   "Unread",
				Unread: unreadCount,
			})
		}
		groups = append(groups, notificationFolder{
			group:  app.GroupAll,
			Name:   "All",
			Unread: unreadCount,
		})
	}
	t, i := a.u.makeTopTextCharacter(characterID, hasData, err, func() (string, widget.Importance) {
		return fmt.Sprintf("%s messages", ihumanize.OptionalWithComma(totalCount, "?")), widget.MediumImportance
	})
	a.resetCurrentFolder()
	fyne.Do(func() {
		a.clearDetail()
		a.folders = groups
		a.foldersTop.Text, a.foldersTop.Importance = t, i
		a.foldersTop.Refresh()
		a.folderList.Refresh()
		a.folderList.UnselectAll()
	})
	if a.OnUpdate != nil {
		a.OnUpdate(unreadCount)
	}
}

func (a *characterCommunications) resetCurrentFolder() {
	a.setCurrentFolder(app.GroupUnread)
	fyne.Do(func() {
		a.notificationList.UnselectAll()
	})
}

func (a *characterCommunications) setCurrentFolder(nc app.NotificationGroup) {
	var err error
	characterID := a.u.currentCharacterID()
	notifications := make([]*app.CharacterNotification, 0)
	hasData := a.u.scs.HasCharacterSection(a.u.currentCharacterID(), app.SectionCharacterNotifications)
	if hasData {
		var err2 error
		var n []*app.CharacterNotification
		ctx := context.Background()
		switch nc {
		case app.GroupAll:
			n, err2 = a.u.cs.ListNotificationsAll(ctx, characterID)
		case app.GroupUnread:
			n, err2 = a.u.cs.ListNotificationsUnread(ctx, characterID)
		default:
			n, err2 = a.u.cs.ListNotificationsTypes(ctx, characterID, nc)
		}
		if err2 != nil {
			slog.Error("communications set group", "characterID", characterID, "error", err2)
		} else {
			notifications = n
			err = err2
		}
	}
	t, i := a.u.makeTopTextCharacter(characterID, hasData, err, func() (string, widget.Importance) {
		s := humanize.Comma(int64(len(notifications)))
		return fmt.Sprintf("%s • %s messages", nc.String(), s), widget.MediumImportance
	})
	fyne.Do(func() {
		a.notificationsTop.Text, a.notificationsTop.Importance = t, i
		a.notificationsTop.Refresh()
	})
	fyne.Do(func() {
		a.notifications = notifications
		a.notificationList.Refresh()
		a.notificationList.ScrollToTop()
	})
}

func (a *characterCommunications) clearDetail() {
	a.Detail.RemoveAll()
	a.Toolbar.Hide()
	a.current = nil
}
