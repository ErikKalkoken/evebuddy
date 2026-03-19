package characters

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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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

	character              atomic.Pointer[app.Character]
	current                *app.CharacterNotification
	folderList             *widget.List
	folders                []notificationFolder
	foldersTop             *widget.Label
	notificationList       *widget.List
	notifications          []*app.CharacterNotification
	notificationsTop       *widget.Label
	sendNotificationAction *widget.ToolbarAction
	u                      baseUI
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
	a.Detail = newCommunicationDetail(u.EVEImage().EveEntityLogoAsync, u.InfoViewer().Show)
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
			return NewMailHeaderItem(a.u.EVEImage().EveEntityLogoAsync)
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
	err := a.Detail.set(n, notificationRecipient(n, a.character.Load().NameOrZero()))
	if err != nil {
		slog.Warn("Failed to set notification detail", "err", err)
		fyne.Do(func() {
			a.Detail.setError(a.u.ErrorDisplay(err))
		})
		return
	}
	if a.u.IsDeveloperMode() {
		a.sendNotificationAction.ToolbarObject().Show()
	} else {
		a.sendNotificationAction.ToolbarObject().Hide()
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
				ui.ShowErrorAndLog(
					"Failed to generated notification for clipboard",
					err,
					a.u.IsDeveloperMode(),
					a.u.MainWindow(),
				)
			}
			cn := a.current
			recipient := notificationRecipient(cn, a.character.Load().NameOrZero())
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
	a.sendNotificationAction = widget.NewToolbarAction(theme.NewThemedResource(icons.TeddyBearSvg), func() {
		if a.current == nil {
			return
		}
		if a.character.Load() == nil {
			return
		}
		go a.u.Character().SendDesktopNotification(context.Background(), a.current)
	})
	a.sendNotificationAction.ToolbarObject().Hide()
	toolbar.Append(a.sendNotificationAction)
	return toolbar
}

func (a *Communications) update(ctx context.Context) {
	reset := func() {
		a.ResetCurrentFolder(ctx)
		fyne.Do(func() {
			a.clearDetail()
			a.folders = xslices.Reset(a.folders)
			a.folderList.Refresh()
			a.folderList.UnselectAll()
			if a.OnUpdate != nil {
				a.OnUpdate(optional.Optional[int]{})
			}
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.foldersTop.Text, a.foldersTop.Importance = s, i
			a.foldersTop.Refresh()
		})
	}

	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		reset()
		setTop("No character", widget.LowImportance)
		return
	}

	hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterNotifications)
	if err != nil {
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	if !hasData {
		reset()
		setTop("No data", widget.WarningImportance)
		return
	}

	folders, unreadCount, totalCount, err := a.fetchFolders(ctx, characterID)
	if err != nil {
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	top := fmt.Sprintf("%s messages", ihumanize.OptionalWithComma(totalCount, "?"))
	setTop(top, widget.MediumImportance)
	a.ResetCurrentFolder(ctx)

	fyne.Do(func() {
		a.clearDetail()
		a.folders = folders
		a.folderList.Refresh()
		a.folderList.UnselectAll()
		if a.OnUpdate != nil {
			a.OnUpdate(unreadCount)
		}
	})
}

func (a *Communications) fetchFolders(ctx context.Context, characterID int64) ([]notificationFolder, optional.Optional[int], optional.Optional[int], error) {
	groupCounts, err := a.u.Character().CountNotifications(ctx, characterID)
	if err != nil {
		return nil, optional.Optional[int]{}, optional.Optional[int]{}, err
	}

	var folders []notificationFolder
	var unreadCount, totalCount optional.Optional[int]
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
			folders = append(folders, nf)
		}
	}
	slices.SortFunc(folders, func(a, b notificationFolder) int {
		return cmp.Compare(a.Name, b.Name)
	})
	if unreadCount.ValueOrZero() > 0 {
		folders = slices.Insert(folders, 0, notificationFolder{
			group:  app.GroupUnread,
			Name:   "Unread",
			Unread: unreadCount,
		})
	}
	folders = append(folders, notificationFolder{
		group:  app.GroupAll,
		Name:   "All",
		Unread: unreadCount,
	})
	return folders, unreadCount, totalCount, err
}

func (a *Communications) ResetCurrentFolder(ctx context.Context) {
	a.setCurrentFolder(ctx, app.GroupUnread)
	fyne.Do(func() {
		a.notificationList.UnselectAll()
	})
}

func (a *Communications) setCurrentFolder(ctx context.Context, nc app.EveNotificationGroup) {
	reset := func() {
		fyne.Do(func() {
			a.notifications = xslices.Reset(a.notifications)
			a.notificationList.Refresh()
			a.notificationList.ScrollToTop()
		})

	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.notificationsTop.Text, a.notificationsTop.Importance = s, i
			a.notificationsTop.Refresh()
		})
	}
	character := a.character.Load()
	if character == nil {
		reset()
		setTop("No character", widget.LowImportance)
		return
	}

	hasData, err := a.u.Character().HasSection(ctx, character.ID, app.SectionCharacterNotifications)
	if err != nil {
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	if !hasData {
		reset()
		setTop("No character", widget.WarningImportance)
	}

	notifications, err := a.fetchNotifications(ctx, nc, character)
	if err != nil {
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	top := fmt.Sprintf("%s • %s messages", nc.String(), ihumanize.Comma(len(notifications)))
	setTop(top, widget.MediumImportance)

	fyne.Do(func() {
		a.notifications = notifications
		a.notificationList.Refresh()
		a.notificationList.ScrollToTop()
	})
}

func (a *Communications) fetchNotifications(ctx context.Context, nc app.EveNotificationGroup, character *app.Character) ([]*app.CharacterNotification, error) {
	var err error
	var oo []*app.CharacterNotification
	switch nc {
	case app.GroupAll:
		oo, err = a.u.Character().ListNotificationsAll(ctx, character.ID)
	case app.GroupUnread:
		oo, err = a.u.Character().ListNotificationsUnread(ctx, character.ID)
	default:
		oo, err = a.u.Character().ListNotificationsForGroup(ctx, character.ID, nc)
	}
	if err != nil {
		slog.Error("Fetch notifications for UI", "characterID", character.ID, "error", err)
		return nil, err
	}

	// Replace generic corporations && alliances in notifications
	for _, n := range oo {
		if n.Sender == nil {
			continue
		}
		switch n.Sender.ID {
		case app.EveTypeAlliance:
			n.Sender = character.EveCharacter.Alliance.ValueOrFallback(&app.EveEntity{
				ID:       1,
				Name:     "Unknown",
				Category: app.EveEntityCorporation,
			})
		case app.EveTypeCorporation:
			n.Sender = character.EveCharacter.Corporation
		}
	}
	return oo, nil
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

// NotificationRecipient returns a valid recipient for a notification.
func notificationRecipient(cn *app.CharacterNotification, characterName string) *app.EveEntity {
	return cn.Recipient.ValueOrFallback(&app.EveEntity{
		ID:       cn.CharacterID,
		Name:     characterName,
		Category: app.EveEntityCharacter,
	})
}

// communicationDetail shows the complete communication for a character.
type communicationDetail struct {
	widget.BaseWidget

	body    *widget.Label
	header  *MailHeader
	subject *widget.Label
}

func newCommunicationDetail(loadIcon ui.EveEntityIconLoader, show func(*app.EveEntity)) *communicationDetail {
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
