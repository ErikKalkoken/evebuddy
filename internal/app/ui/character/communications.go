package character

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xdialog"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type notificationFolder struct {
	group  app.EveNotificationGroup
	Name   string
	Unread optional.Optional[int]
	Total  optional.Optional[int]
}

type Communications struct {
	widget.BaseWidget

	Detail        *communicationDetail
	Notifications *fyne.Container
	OnSelected    func()
	OnUpdate      func(count optional.Optional[int])
	Toolbar       *widget.Toolbar

	character        atomic.Pointer[app.Character]
	current          *app.CharacterNotification
	folderList       *widget.List
	folders          []notificationFolder
	foldersTop       *widget.Label
	notificationList *widget.List
	notifications    []*app.CharacterNotification
	notificationsTop *widget.Label
	u                baseUI
}

func NewCommunications(u baseUI) *Communications {
	a := &Communications{
		notificationsTop: widget.NewLabel(""),
		foldersTop:       widget.NewLabel(""),
		u:                u,
	}
	a.ExtendBaseWidget(a)
	a.Toolbar = a.makeToolbar()
	a.Toolbar.Hide()
	a.folderList = a.makeFolderList()
	a.Detail = newCommunicationDetail(awidget.LoadEveEntityIconFunc(u.EVEImage()), u.InfoWindow().Show)
	a.notificationList = a.makeNotificationList()
	a.Notifications = container.NewBorder(a.notificationsTop, nil, nil, nil, a.notificationList)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterNotifications {
			a.update(ctx)
		}
	})
	return a
}

func (a *Communications) CreateRenderer() fyne.WidgetRenderer {
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

func (a *Communications) MakeFolderMenu() []*fyne.MenuItem {
	var items2 []*fyne.MenuItem
	for _, f := range a.folders {
		s := f.Name
		if f.Unread.ValueOrZero() > 0 {
			s += fmt.Sprintf(" (%s)", ihumanize.OptionalWithComma(f.Unread, "?"))
		}
		it := fyne.NewMenuItem(s, func() {
			go a.setCurrentFolder(context.Background(), f.group)
		})
		items2 = append(items2, it)
	}
	return items2
}

func (a *Communications) makeFolderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.folders)
		},
		func() fyne.CanvasObject {
			return newFolderItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.folders) {
				return
			}
			co.(*folderItem).set(a.folders[id])
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		a.notificationList.UnselectAll()
		if id >= len(a.folders) {
			l.UnselectAll()
			return
		}
		o := a.folders[id]
		a.clearDetail()
		go a.setCurrentFolder(context.Background(), o.group)
	}
	return l
}

func (a *Communications) makeNotificationList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.notifications)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItem(awidget.LoadEveEntityIconFunc(a.u.EVEImage()))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.notifications) {
				return
			}
			n := a.notifications[id]
			item := co.(*MailHeaderItem)
			item.Set(
				n.Sender,
				n.TitleDisplay(),
				n.Timestamp,
				n.IsRead.ValueOrZero(),
			)
		},
	)
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

func (a *Communications) setDetail(n *app.CharacterNotification) {
	if a.character.Load() == nil {
		return
	}
	err := a.Detail.set(n, a.u.Character().NotificationRecipient(n))
	if err != nil {
		slog.Warn("Failed to set notification detail", "err", err)
		fyne.Do(func() {
			a.Detail.setError(a.u.ErrorDisplay(err))
		})
		return
	}
	a.current = n
	a.Toolbar.Show()
	a.Detail.Show()
}

func (a *Communications) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.current == nil {
				return
			}
			processErr := func(err error) {
				xdialog.ShowErrorAndLog(
					"Failed to generated notification for clipboard",
					err,
					a.u.IsDeveloperMode(),
					a.u.MainWindow(),
				)
			}
			cn := a.current
			recipient := a.u.Character().NotificationRecipient(cn)
			header := fmt.Sprintf(
				"From: %s\nSent: %s\nTo: %s",
				cn.Sender.Name,
				cn.Timestamp.Format(app.DateTimeFormat),
				recipient.Name,
			)
			s := cn.TitleDisplay() + "\n" + header
			b, err := cn.BodyPlain()
			if err != nil {
				processErr(err)
				return
			}
			s += "\n\n"
			if v, ok := b.Value(); ok {
				s += v
			} else {
				s += "(no body)"
			}
			fyne.CurrentApp().Clipboard().SetContent(s)
		}),
	)
	if a.u.IsDeveloperMode() {
		toolbar.Append(widget.NewToolbarAction(theme.NewThemedResource(icons.TeddyBearSvg), func() {
			if a.current == nil {
				return
			}
			if a.character.Load() == nil {
				return
			}
			title, content := a.u.Character().RenderNotificationSummary(a.current)
			fyne.CurrentApp().SendNotification(fyne.NewNotification(title, content))
		}))
	}
	return toolbar
}

func (a *Communications) update(ctx context.Context) {
	var err error
	characterID := a.character.Load().IDOrZero()
	hasData := a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterNotifications)
	var groups []notificationFolder
	var unreadCount, totalCount optional.Optional[int]
	if characterID != 0 && hasData {
		groupCounts, err2 := a.u.Character().CountNotifications(ctx, characterID)
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
	t, i := makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		return fmt.Sprintf("%s messages", ihumanize.OptionalWithComma(totalCount, "?")), widget.MediumImportance
	})
	a.ResetCurrentFolder(ctx)
	fyne.Do(func() {
		a.clearDetail()
		a.folders = groups
		a.foldersTop.Text, a.foldersTop.Importance = t, i
		a.foldersTop.Refresh()
		a.folderList.Refresh()
		a.folderList.UnselectAll()
		if a.OnUpdate != nil {
			a.OnUpdate(unreadCount)
		}
	})
}

func (a *Communications) ResetCurrentFolder(ctx context.Context) {
	a.setCurrentFolder(ctx, app.GroupUnread)
	fyne.Do(func() {
		a.notificationList.UnselectAll()
	})
}

func (a *Communications) setCurrentFolder(ctx context.Context, nc app.EveNotificationGroup) {
	var err error
	characterID := a.character.Load().IDOrZero()
	var notifications []*app.CharacterNotification
	hasData := a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterNotifications)
	if hasData {
		var err2 error
		var n []*app.CharacterNotification
		switch nc {
		case app.GroupAll:
			n, err2 = a.u.Character().ListNotificationsAll(ctx, characterID)
		case app.GroupUnread:
			n, err2 = a.u.Character().ListNotificationsUnread(ctx, characterID)
		default:
			n, err2 = a.u.Character().ListNotificationsForGroup(ctx, characterID, nc)
		}
		if err2 != nil {
			slog.Error("communications set group", "characterID", characterID, "error", err2)
		} else {
			notifications = n
			err = err2
		}
	}
	t, i := makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		s := humanize.Comma(int64(len(notifications)))
		return fmt.Sprintf("%s • %s messages", nc.String(), s), widget.MediumImportance
	})
	fyne.Do(func() {
		a.notificationsTop.Text, a.notificationsTop.Importance = t, i
		a.notificationsTop.Refresh()
	})
	// Replace generic corporations && alliances in notifications
	if c := a.character.Load(); c != nil {
		for _, n := range notifications {
			if n.Sender == nil {
				continue
			}
			switch n.Sender.ID {
			case app.EveTypeAlliance:
				n.Sender = c.EveCharacter.Alliance.ValueOrFallback(&app.EveEntity{
					ID:       1,
					Name:     "Unknown",
					Category: app.EveEntityCorporation,
				})
			case app.EveTypeCorporation:
				n.Sender = c.EveCharacter.Corporation
			}
		}
	}
	fyne.Do(func() {
		a.notifications = notifications
		a.notificationList.Refresh()
		a.notificationList.ScrollToTop()
	})
}

func (a *Communications) clearDetail() {
	a.Detail.Hide()
	a.Toolbar.Hide()
	a.current = nil
}

type folderItem struct {
	widget.BaseWidget

	name   *widget.Label
	unread *kxwidget.Badge
}

func newFolderItem() *folderItem {
	w := &folderItem{
		name:   widget.NewLabel(""),
		unread: kxwidget.NewBadge(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *folderItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, nil, w.unread, w.name)
	return widget.NewSimpleRenderer(c)
}

func (w *folderItem) set(r notificationFolder) {
	text := r.Name
	if r.Unread.ValueOrZero() > 0 {
		w.name.TextStyle.Bold = true
		w.unread.SetText(ihumanize.OptionalWithComma(r.Unread, "?"))
		w.unread.Show()
	} else {
		w.name.TextStyle.Bold = false
		w.unread.Hide()
	}
	w.name.Text = text
	w.name.Refresh()
}

// communicationDetail shows the complete communication for a character.
type communicationDetail struct {
	widget.BaseWidget

	body    *widget.Label
	header  *MailHeader
	subject *widget.Label
}

func newCommunicationDetail(loadIcon awidget.EveEntityIconLoader, show func(*app.EveEntity)) *communicationDetail {
	subject := widget.NewLabel("")
	subject.SizeName = theme.SizeNameSubHeadingText
	subject.Wrapping = fyne.TextWrapWord
	subject.Selectable = true
	body := widget.NewLabel("")
	body.Wrapping = fyne.TextWrapWord
	body.Selectable = true
	w := &communicationDetail{
		body:    body,
		header:  NewMailHeader(loadIcon, show),
		subject: subject,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *communicationDetail) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(w.subject, w.header, w.body)
	return widget.NewSimpleRenderer(c)
}

func (w *communicationDetail) set(n *app.CharacterNotification, recipient *app.EveEntity) error {
	w.subject.SetText(n.TitleDisplay())
	w.header.Set(n.Sender, n.Timestamp, recipient)
	b, err := n.BodyPlain() // using markdown blocked by #61
	if err != nil {
		return fmt.Errorf("failed to convert markdown for notification %+v: %w", n, err)
	}
	w.body.Text = b.StringFunc("[This notification type is not fully supported yet]", func(v string) string {
		return v
	})
	w.body.Importance = widget.MediumImportance
	w.body.Refresh()
	return nil
}

func (w *communicationDetail) setError(text string) {
	w.subject.SetText("ERROR")
	w.header.Clear()
	w.body.Text = text
	w.body.Importance = widget.DangerImportance
	w.body.Refresh()
}
