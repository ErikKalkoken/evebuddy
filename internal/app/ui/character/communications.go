package character

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type NotificationGroup struct {
	Group       app.NotificationGroup
	Name        string
	UnreadCount int
}

type Communications struct {
	widget.BaseWidget

	Detail        *fyne.Container
	Notifications fyne.CanvasObject
	Toolbar       *widget.Toolbar
	OnSelected    func()
	OnUpdate      func(count int)

	Groups           []NotificationGroup
	groupList        *widget.List
	current          *app.CharacterNotification
	notificationList *widget.List
	notifications    []*app.CharacterNotification
	notificationsTop *widget.Label
	groupsTop        *widget.Label
	u                app.UI
}

func NewCommunications(u app.UI) *Communications {
	a := &Communications{
		Groups:           make([]NotificationGroup, 0),
		notifications:    make([]*app.CharacterNotification, 0),
		notificationsTop: widget.NewLabel(""),
		groupsTop:        widget.NewLabel(""),
		u:                u,
	}
	a.ExtendBaseWidget(a)
	a.Toolbar = a.makeToolbar()
	a.Toolbar.Hide()
	a.groupList = a.makeGroupList()
	a.Detail = container.NewVBox()
	a.notificationList = a.makeNotificationList()
	a.Notifications = container.NewBorder(a.notificationsTop, nil, nil, nil, a.notificationList)
	return a
}

func (a *Communications) CreateRenderer() fyne.WidgetRenderer {
	split1 := container.NewHSplit(
		a.Notifications,
		container.NewBorder(a.Toolbar, nil, nil, nil, a.Detail),
	)
	split1.Offset = 0.35
	split2 := container.NewHSplit(
		container.NewBorder(a.groupsTop, nil, nil, nil, a.groupList),
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

func (a *Communications) makeGroupList() *widget.List {
	maxGroup := slices.MaxFunc(app.NotificationGroups(), func(a, b app.NotificationGroup) int {
		return strings.Compare(a.String(), b.String())
	})
	l := widget.NewList(
		func() int {
			return len(a.Groups)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(maxGroup.String()+"WWWWWWWW"), layout.NewSpacer(), kwidget.NewBadge("999"),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.Groups) {
				return
			}
			c := a.Groups[id]
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
		if id >= len(a.Groups) {
			l.UnselectAll()
			return
		}
		o := a.Groups[id]
		a.clearDetail()
		a.SetGroup(o.Group)
	}
	return l
}

func (a *Communications) makeNotificationList() *widget.List {
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

func (a *Communications) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.current == nil {
				return
			}
			a.u.MainWindow().Clipboard().SetContent(a.current.String())
		}),
	)
	return toolbar
}

func (a *Communications) Update() {
	a.clearDetail()
	a.notifications = make([]*app.CharacterNotification, 0)
	a.notificationList.Refresh()
	a.notificationList.UnselectAll()
	a.notificationsTop.SetText("")
	var counts map[app.NotificationGroup]int
	if characterID := a.u.CurrentCharacterID(); characterID != 0 {
		var err error
		counts, err = a.u.CharacterService().CountCharacterNotificationUnreads(context.TODO(), characterID)
		if err != nil {
			slog.Error("Failed to fetch notification unread counts", "error", err)
		}
	}
	groups := make([]NotificationGroup, 0)
	var unreadTotal int
	for _, c := range app.NotificationGroups() {
		nc := NotificationGroup{
			Group:       c,
			Name:        c.String(),
			UnreadCount: counts[c],
		}
		groups = append(groups, nc)
		unreadTotal += counts[c]
	}
	slices.SortFunc(groups, func(a, b NotificationGroup) int {
		return cmp.Compare(a.Name, b.Name)
	})
	f1 := NotificationGroup{
		Group:       app.GroupUnread,
		Name:        "Unread",
		UnreadCount: unreadTotal,
	}
	groups = slices.Insert(groups, 0, f1)
	f2 := NotificationGroup{
		Group:       app.GroupAll,
		Name:        "All",
		UnreadCount: unreadTotal,
	}
	groups = append(groups, f2)
	a.Groups = groups
	a.groupList.Refresh()
	a.groupList.UnselectAll()
	a.groupsTop.Text, a.groupsTop.Importance = a.makeGroupTopText()
	a.groupsTop.Refresh()
	if a.OnUpdate != nil {
		a.OnUpdate(unreadTotal)
	}
}

func (a *Communications) makeGroupTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService().CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for data to load...", widget.WarningImportance
	}
	return fmt.Sprintf("%d groups", len(a.Groups)), widget.MediumImportance
}

func (a *Communications) ResetGroups() {
	a.SetGroup(app.GroupUnread)
}

func (a *Communications) SetGroup(nc app.NotificationGroup) {
	ctx := context.Background()
	characterID := a.u.CurrentCharacterID()
	var notifications []*app.CharacterNotification
	var err error
	switch nc {
	case app.GroupAll:
		notifications, err = a.u.CharacterService().ListCharacterNotificationsAll(ctx, characterID)
	case app.GroupUnread:
		notifications, err = a.u.CharacterService().ListCharacterNotificationsUnread(ctx, characterID)
	default:
		notifications, err = a.u.CharacterService().ListCharacterNotificationsTypes(ctx, characterID, nc)
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

func (a *Communications) clearDetail() {
	a.Detail.RemoveAll()
	a.Toolbar.Hide()
	a.current = nil
}

// TODO: Refactor to avoid recreating the container every time
func (a *Communications) setDetail(n *app.CharacterNotification) {
	if n.RecipientName == "" && a.u.HasCharacter() {
		n.RecipientName = a.u.CurrentCharacter().EveCharacter.Name
	}
	a.Detail.RemoveAll()
	subject := iwidget.NewLabelWithSize(n.TitleDisplay(), theme.SizeNameSubHeadingText)
	subject.Wrapping = fyne.TextWrapWord
	a.Detail.Add(subject)
	h := NewMailHeader(a.u.EveImageService(), a.u.ShowEveEntityInfoWindow)
	h.Set(n.Sender, n.Timestamp, a.u.CurrentCharacter().EveCharacter.ToEveEntity())
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
