package characters

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type Communications struct {
	widget.BaseWidget

	MessagePane    *communicationsMessagePane
	NavigationPane *communicationsNavigationPane
	ReadingPane    *communicationsReadingPane

	character atomic.Pointer[app.Character]
	u         baseUI
}

func NewCommunications(u baseUI) *Communications {

	a := &Communications{
		u: u,
	}
	a.ExtendBaseWidget(a)

	a.ReadingPane = newCommunicationsReadingPane(a)
	a.MessagePane = newCommunicationsMessagePane(a)
	a.NavigationPane = newCommunicationsNavigationPane(a)

	// Signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		fyne.Do(func() {
			a.ReadingPane.clear()
		})
		a.NavigationPane.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterNotifications {
			a.NavigationPane.update(ctx)
		}
	})

	return a
}

func (a *Communications) CreateRenderer() fyne.WidgetRenderer {
	split1 := container.NewHSplit(a.MessagePane, a.ReadingPane)
	split1.Offset = 0.35
	split2 := container.NewHSplit(a.NavigationPane, split1)
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

type notificationFolder struct {
	folder app.EveNotificationGroup
	name   string
	unread optional.Optional[int]
	total  optional.Optional[int]
}
type communicationsNavigationPane struct {
	widget.BaseWidget

	OnUpdate func(count optional.Optional[int])

	co          *Communications
	folderList  *widget.List
	folders     []notificationFolder
	footerLabel *widget.Label
}

func newCommunicationsNavigationPane(co *Communications) *communicationsNavigationPane {
	a := &communicationsNavigationPane{
		footerLabel: widget.NewLabel(""),
		co:          co,
	}
	a.ExtendBaseWidget(a)
	a.folderList = a.makeFolderList()
	return a
}

func (a *communicationsNavigationPane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, a.footerLabel, nil, nil, a.folderList)
	return widget.NewSimpleRenderer(c)
}

func (a *communicationsNavigationPane) MakeFolderMenu() []*fyne.MenuItem {
	var items2 []*fyne.MenuItem
	for _, f := range a.folders {
		s := f.name
		if f.unread.ValueOrZero() > 0 {
			s += fmt.Sprintf(" (%s)", ihumanize.OptionalWithComma(f.unread, "?"))
		}
		it := fyne.NewMenuItem(s, func() {
			go a.co.MessagePane.set(context.Background(), f.folder)
		})
		items2 = append(items2, it)
	}
	return items2
}

func (a *communicationsNavigationPane) makeFolderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.folders)
		},
		func() fyne.CanvasObject {
			return newFolderItemWidget()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.folders) {
				return
			}
			co.(*folderItemWidget).set(a.folders[id])
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.folders) {
			l.UnselectAll()
			return
		}
		o := a.folders[id]
		go a.co.MessagePane.set(context.Background(), o.folder)
	}
	return l
}

func (a *communicationsNavigationPane) update(ctx context.Context) {
	reset := func() {
		a.co.MessagePane.ResetHeaders(ctx)
		fyne.Do(func() {
			a.folders = xslices.Reset(a.folders)
			a.folderList.Refresh()
			a.folderList.UnselectAll()
			a.folderList.Select(0)
			if a.OnUpdate != nil {
				a.OnUpdate(optional.Optional[int]{})
			}
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footerLabel.Text, a.footerLabel.Importance = s, i
			a.footerLabel.Refresh()
		})
	}

	characterID := a.co.character.Load().IDOrZero()
	if characterID == 0 {
		reset()
		setTop("No character", widget.LowImportance)
		return
	}

	hasData, err := a.co.u.Character().HasSection(ctx, characterID, app.SectionCharacterNotifications)
	if err != nil {
		reset()
		setTop("ERROR: "+a.co.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	if !hasData {
		reset()
		setTop("No data", widget.WarningImportance)
		return
	}

	folders, unreadCount, totalCount, err := a.fetchFolders(ctx)
	if err != nil {
		reset()
		setTop("ERROR: "+a.co.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	top := fmt.Sprintf("%s total", ihumanize.OptionalWithComma(totalCount, "?"))
	setTop(top, widget.MediumImportance)
	a.co.MessagePane.ResetHeaders(ctx)

	fyne.Do(func() {
		a.folders = folders
		a.folderList.Refresh()
		a.folderList.UnselectAll()
		a.folderList.Select(0)
		if a.OnUpdate != nil {
			a.OnUpdate(unreadCount)
		}
	})
}

func (a *communicationsNavigationPane) setFoldersUnread(current app.EveNotificationGroup) {
	currentIdx, allIdx := -1, -1
	for i, f := range a.folders {
		if f.folder == current {
			currentIdx = i
		}
		if f.folder == app.GroupAll {
			allIdx = i
		}
	}
	a.folders[currentIdx].unread.Set(a.folders[currentIdx].unread.ValueOrZero() - 1)
	a.folders[allIdx].unread.Set(a.folders[allIdx].unread.ValueOrZero() - 1)
	a.folderList.Refresh()
	if a.OnUpdate != nil {
		a.OnUpdate(a.folders[allIdx].unread)
	}
}

func (a *communicationsNavigationPane) fetchFolders(ctx context.Context) ([]notificationFolder, optional.Optional[int], optional.Optional[int], error) {
	character := a.co.character.Load()
	if character == nil {
		return nil, optional.Optional[int]{}, optional.Optional[int]{}, fmt.Errorf("no character: %s", app.ErrInvalid)
	}
	groupCounts, err := a.co.u.Character().CountNotifications(ctx, character.ID)
	if err != nil {
		return nil, optional.Optional[int]{}, optional.Optional[int]{}, err
	}

	var folders []notificationFolder
	var unreadCount, totalCount optional.Optional[int]
	for _, g := range app.NotificationGroups() {
		nf := notificationFolder{
			folder: g,
			name:   g.String(),
		}
		gc, ok := groupCounts[g]
		if ok {
			nf.total.Set(gc[0])
			nf.unread.Set(gc[1])
			totalCount.Set(totalCount.ValueOrZero() + gc[0])
			unreadCount.Set(unreadCount.ValueOrZero() + gc[1])
		}
		if nf.total.ValueOrZero() > 0 {
			folders = append(folders, nf)
		}
	}
	slices.SortFunc(folders, func(a, b notificationFolder) int {
		return cmp.Compare(a.name, b.name)
	})
	folders = slices.Insert(folders, 0, notificationFolder{
		folder: app.GroupAll,
		name:   "All",
		unread: unreadCount,
	})
	return folders, unreadCount, totalCount, err
}

type folderItemWidget struct {
	widget.BaseWidget

	name   *widget.Label
	unread *kxwidget.Badge
}

func newFolderItemWidget() *folderItemWidget {
	w := &folderItemWidget{
		name:   widget.NewLabel(""),
		unread: kxwidget.NewBadge(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *folderItemWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, nil, w.unread, w.name)
	return widget.NewSimpleRenderer(c)
}

func (w *folderItemWidget) set(r notificationFolder) {
	if r.unread.ValueOrZero() > 0 {
		w.name.TextStyle.Bold = true
		w.unread.SetText(ihumanize.OptionalWithComma(r.unread, "?"))
		w.unread.Show()
	} else {
		w.name.TextStyle.Bold = false
		w.unread.Hide()
	}
	w.name.Text = r.name
	w.name.Refresh()
}

// -----------------

const (
	communicationsColTimestamp = iota + 1
	communicationsColSender
	communicationsColType
)

type notificationMessageRow struct {
	characterID      int64
	id               int64
	isRead           bool
	isRead2          bool // updated by user, can be newer then isRead
	notificationID   int64
	notificationType string
	searchTarget     string
	sender           *app.EveEntity
	subject          string
	timestamp        time.Time
}

func (n notificationMessageRow) IsZero() bool {
	return n.notificationID == 0
}

type communicationsMessagePane struct {
	widget.BaseWidget

	OnSelected func()

	co            *Communications
	columnSorter  *xwidget.ColumnSorter[notificationMessageRow]
	currentFolder app.EveNotificationGroup
	footerLabel   *widget.Label
	messageList   *widget.List
	moreButton    *kxwidget.IconButton
	rows          []notificationMessageRow
	rowsFiltered  []notificationMessageRow
	searchEntry   *widget.Entry
	sortButton    *xwidget.SortButton
	topLabel      *widget.Label
	unreadChip    *kxwidget.FilterChip
}

func newCommunicationsMessagePane(co *Communications) *communicationsMessagePane {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[notificationMessageRow]{{
		ID:    communicationsColTimestamp,
		Label: "Date",
		Sort: func(a, b notificationMessageRow) int {
			return a.timestamp.Compare(b.timestamp)
		},
	}, {
		ID:    communicationsColSender,
		Label: "Sender",
		Sort: func(a, b notificationMessageRow) int {
			return strings.Compare(a.sender.Name, b.sender.Name)
		},
	}, {
		ID:    communicationsColType,
		Label: "Type",
		Sort: func(a, b notificationMessageRow) int {
			return strings.Compare(a.notificationType, b.notificationType)
		},
	}}),
		communicationsColTimestamp,
		xwidget.SortDesc,
	)
	a := &communicationsMessagePane{
		columnSorter: columnSorter,
		footerLabel:  widget.NewLabel(""),
		topLabel:     widget.NewLabel(""),
		searchEntry:  widget.NewEntry(),
		co:           co,
	}
	a.ExtendBaseWidget(a)
	a.messageList = a.makeHeaderList()
	a.topLabel.SizeName = theme.SizeNameSubHeadingText
	a.searchEntry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.searchEntry.SetText("")
		a.filterRowsAsync()
	})
	a.searchEntry.OnChanged = func(_ string) {
		a.filterRowsAsync()
	}
	a.searchEntry.PlaceHolder = "Search communications"
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.co.u.MainWindow())
	a.unreadChip = kxwidget.NewFilterChip("Unread", func(on bool) {
		if on {
			for i := range a.rows {
				a.rows[i].isRead = a.rows[i].isRead2
			}
		}
		a.filterRowsAsync()
	})
	a.moreButton = kxwidget.NewIconButtonWithMenu(
		theme.MoreHorizontalIcon(),
		fyne.NewMenu("", fyne.NewMenuItem("Mark folder as read", a.markCurrentFolderRead)),
	)
	return a
}

func (a *communicationsMessagePane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			container.NewHBox(a.topLabel, layout.NewSpacer(), a.moreButton),
			container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(a.unreadChip, a.sortButton),
				a.searchEntry,
			),
		),
		a.footerLabel,
		nil,
		nil,
		a.messageList,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *communicationsMessagePane) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItemWidget(a.co.u.EVEImage().EveEntityLogoAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			item := co.(*MailHeaderItemWidget)
			item.Set(
				r.sender,
				r.subject,
				r.timestamp,
				r.isRead2,
			)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.rowsFiltered) {
			a.co.ReadingPane.clear()
			l.UnselectAll()
			return
		}
		a.co.ReadingPane.set(a.rowsFiltered[id])
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *communicationsMessagePane) ResetHeaders(ctx context.Context) {
	a.set(ctx, app.GroupAll)
}

func (a *communicationsMessagePane) markCurrentFolderRead() {
	var ids set.Set[int64]
	for _, h := range a.rows {
		if !h.isRead2 {
			ids.Add(h.id)
		}
	}
	go func() {
		reportError := func(err error) {
			fyne.Do(func() {
				ui.ShowErrorAndLog("Failed to mark folder as read", err, a.co.u.IsDeveloperMode(), a.co.u.MainWindow())
			})
		}

		ctx := context.Background()
		err := a.co.u.Character().SetNotificationsAsRead(ctx, ids)
		if err != nil {
			reportError(err)
			return
		}
		folders, unreadCount, _, err := a.co.NavigationPane.fetchFolders(ctx)
		if err != nil {
			reportError(err)
			return
		}
		fyne.Do(func() {
			for i, h := range a.rows {
				if !h.isRead2 {
					a.rows[i].isRead2 = true
				}
			}
			a.filterRowsAsync()
			a.co.NavigationPane.folders = folders
			a.co.NavigationPane.folderList.Refresh()
			if a.co.NavigationPane.OnUpdate != nil {
				a.co.NavigationPane.OnUpdate(unreadCount)
			}
		})
	}()
}

func (a *communicationsMessagePane) setNotificationRead(ctx context.Context, id int64) {
	err := a.co.u.Character().SetNotificationsAsRead(ctx, set.Of(id))
	if err != nil {
		slog.Error("Failed to set notification as read", "ID", id)
		return
	}
	fyne.DoAndWait(func() {
		a.co.NavigationPane.setFoldersUnread(a.currentFolder)
	})
	fyne.Do(func() {
		for i, r := range a.rows {
			if id == r.id {
				a.rows[i].isRead2 = true
			}
		}
		a.filterRowsAsync()
	})
}

func (a *communicationsMessagePane) filterRowsAsync() {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	unread := a.unreadChip.On
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)
	go func() {
		// filter
		if unread {
			rows = slices.DeleteFunc(rows, func(r notificationMessageRow) bool {
				return r.isRead
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r notificationMessageRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		// sort
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// refresh
		id2idx := make(map[int64]int)
		for i, r := range rows {
			id2idx[r.id] = i
		}

		footer := fmt.Sprintf("Showing %d / %d messages", len(rows), totalRows)
		fyne.Do(func() {
			a.footerLabel.Text = footer
			a.footerLabel.Importance = widget.MediumImportance
			a.footerLabel.Refresh()
			a.rowsFiltered = rows
			a.messageList.Refresh()
			a.messageList.UnselectAll()
			var notClear bool
			if cn := a.co.ReadingPane.currentNotification; cn != nil {
				// try to update selection for current message
				if idx, ok := id2idx[cn.ID]; ok {
					a.messageList.Select(idx)
					a.messageList.ScrollTo(idx)
					notClear = true
				} else {
					a.messageList.ScrollToTop()
				}
			}
			if !notClear {
				a.co.ReadingPane.clear()
			}
		})
	}()
}

func (a *communicationsMessagePane) set(ctx context.Context, ng app.EveNotificationGroup) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync()
		})

	}

	character := a.co.character.Load()
	if character == nil {
		reset()
		fyne.Do(func() {
			a.topLabel.SetText("ERROR: " + "No character")
		})
		return
	}

	hasData, err := a.co.u.Character().HasSection(ctx, character.ID, app.SectionCharacterNotifications)
	if err != nil {
		reset()
		fyne.Do(func() {
			a.topLabel.SetText(a.co.u.ErrorDisplay(err))
		})
		return
	}

	if !hasData {
		reset()
		fyne.Do(func() {
			a.topLabel.SetText("ERROR: " + "No data")
		})
	}

	notificationRows, err := a.fetchRows(ctx, ng, character)
	if err != nil {
		reset()
		fyne.Do(func() {
			a.topLabel.SetText("ERROR" + a.co.u.ErrorDisplay(err))
		})
		return
	}

	fyne.Do(func() {
		a.co.ReadingPane.clear()
		a.currentFolder = ng
		a.topLabel.SetText(ng.String())
		a.rows = notificationRows
		a.filterRowsAsync()
	})
}

func (a *communicationsMessagePane) fetchRows(ctx context.Context, nc app.EveNotificationGroup, character *app.Character) ([]notificationMessageRow, error) {
	var err error
	var oo []*app.CharacterNotification
	switch nc {
	case app.GroupAll:
		oo, err = a.co.u.Character().ListNotificationsAll(ctx, character.ID)
	case app.GroupUnread:
		oo, err = a.co.u.Character().ListNotificationsUnread(ctx, character.ID)
	default:
		oo, err = a.co.u.Character().ListNotificationsForGroup(ctx, character.ID, nc)
	}
	if err != nil {
		slog.Error("Fetch notifications for UI", "characterID", character.ID, "error", err)
		return nil, err
	}

	var rows []notificationMessageRow

	for _, n := range oo {
		// Replace generic corporations && alliances in notifications
		var sender *app.EveEntity
		switch n.Sender.ID {
		case app.EveTypeAlliance:
			sender = character.EveCharacter.Alliance.ValueOrFallback(&app.EveEntity{
				ID:       1,
				Name:     "Unknown",
				Category: app.EveEntityCorporation,
			})
		case app.EveTypeCorporation:
			sender = character.EveCharacter.Corporation
		default:
			sender = n.Sender
		}
		subject := n.TitleDisplay()
		r := notificationMessageRow{
			characterID:      n.CharacterID,
			id:               n.ID,
			isRead:           n.IsRead.ValueOrZero(),
			notificationID:   n.NotificationID,
			notificationType: n.Type.Display(),
			searchTarget:     strings.ToLower(fmt.Sprintf("%s-%s", subject, sender.Name)),
			sender:           sender,
			subject:          subject,
			timestamp:        n.Timestamp,
		}
		r.isRead2 = r.isRead
		rows = append(rows, r)
	}
	return rows, nil
}

// -----------------

type communicationsReadingPane struct {
	widget.BaseWidget

	bodyText            *xwidget.RichText
	co                  *Communications
	currentNotification *app.CharacterNotification
	developerAction     *widget.ToolbarAction
	headerWidget        *MailHeaderWidget
	subjectLabel        *widget.Label
	toolbar             *widget.Toolbar
}

func newCommunicationsReadingPane(co *Communications) *communicationsReadingPane {
	header := NewMailHeaderWidget(co.u.EVEImage().EveEntityLogoAsync, co.u.InfoViewer().Show)
	a := &communicationsReadingPane{
		subjectLabel: widget.NewLabel(""),
		bodyText:     xwidget.NewRichText(),
		headerWidget: header,
		co:           co,
	}
	a.ExtendBaseWidget(a)
	a.subjectLabel.SizeName = theme.SizeNameSubHeadingText
	a.subjectLabel.Wrapping = fyne.TextWrapWord
	a.bodyText.Wrapping = fyne.TextWrapWord

	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	return a
}

func (a *communicationsReadingPane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.toolbar,
		nil,
		nil,
		nil,
		container.NewVBox(a.subjectLabel, a.headerWidget, a.bodyText),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *communicationsReadingPane) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.currentNotification == nil {
				return
			}

			cn := a.currentNotification
			recipient := a.notificationRecipient(cn)
			header := fmt.Sprintf(
				"From: %s\nSent: %s\nTo: %s",
				cn.Sender.Name,
				cn.Timestamp.Format(app.DateTimeFormat),
				recipient.Name,
			)
			s := cn.TitleDisplay() + "\n" + header
			b, err := cn.BodyPlain()
			if err != nil {
				ui.ShowErrorAndLog(
					"Failed to copy notification to clipboard",
					err,
					a.co.u.IsDeveloperMode(),
					a.co.u.MainWindow(),
				)
				return
			}
			s += "\n\n"
			if v, ok := b.Value(); ok {
				s += v
			} else {
				s += "(no body)"
			}
			fyne.CurrentApp().Clipboard().SetContent(s)
			a.co.u.ShowSnackbar("Communication copied to clipboard")
		}),
	)
	items := []*fyne.MenuItem{
		fyne.NewMenuItem(
			"Send test notification",
			func() {
				if a.currentNotification == nil {
					return
				}
				if a.co.character.Load() == nil {
					return
				}
				go a.co.u.Character().SendDesktopNotification(context.Background(), a.currentNotification)
			},
		),
		fyne.NewMenuItem(
			"Copy notification object to clipboard",
			func() {
				b, err := a.currentNotification.ToJSON()
				if err != nil {
					slog.Error("Failed to convert notification to JSON", "characterID", a.currentNotification.CharacterID, "notificationID", a.currentNotification.NotificationID, "error", err)
					a.co.u.ShowSnackbar("ERROR: Failed to convert data: " + err.Error())
					return
				}
				if len(b) == 0 {
					return
				}
				fyne.CurrentApp().Clipboard().SetContent(string(b))
				a.co.u.ShowSnackbar("Notification object copied to clipboard")
			},
		),
	}
	a.developerAction = kxwidget.NewToolbarActionMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("", items...))
	a.developerAction.ToolbarObject().Hide()
	toolbar.Append(widget.NewToolbarSpacer())
	toolbar.Append(a.developerAction)
	return toolbar
}

func (a *communicationsReadingPane) clear() {
	a.currentNotification = nil
	a.bodyText.Hide()
	a.headerWidget.Hide()
	a.subjectLabel.Hide()
	a.toolbar.Hide()
}

func (a *communicationsReadingPane) set(r notificationMessageRow) {
	ctx := context.Background()
	if !r.isRead2 {
		r.isRead2 = true
		go a.co.MessagePane.setNotificationRead(ctx, r.id)
	}
	go func() {
		cn, err := a.co.u.Character().GetNotification(ctx, r.characterID, r.notificationID)
		if err != nil {
			fyne.Do(func() {
				slog.Error("Failed to load communication", "notificationID", cn.ID, "error", err)
				a.bodyText.SetWithText("ERROR: Failed to load communication: "+a.co.u.ErrorDisplay(err), widget.RichTextStyle{
					ColorName: theme.ColorNameError,
				})

			})
			fyne.Do(func() {
				a.currentNotification = cn
				a.bodyText.Show()
				a.headerWidget.Hide()
				a.subjectLabel.Hide()
				a.toolbar.Hide()
			})
			return
		}
		fyne.Do(func() {
			a.subjectLabel.SetText(cn.TitleDisplay())
			a.headerWidget.Set(cn.Sender, cn.Timestamp, a.notificationRecipient(cn))
			if v, ok := cn.Body.Value(); !ok {
				a.bodyText.SetWithText("[This notification type is not fully supported yet]", widget.RichTextStyle{
					ColorName: theme.ColorNameDisabled,
				})
			} else {
				a.bodyText.ParseMarkdown(v)
				for _, s := range a.bodyText.Segments {
					s2, ok := s.(*widget.HyperlinkSegment)
					if !ok {
						continue
					}
					if s2.URL.Scheme != "showinfo" {
						continue
					}
					typeID, itemID, err := parseIDs(s2.URL.Opaque)
					if err != nil {
						slog.Warn("Failed to parse showinfo link in communication", "error", err)
						continue
					}
					s2.OnTapped = func() {
						a.co.u.InfoViewer().Show2(typeID, itemID, cn.CharacterID)
					}
				}
			}
			if a.co.u.IsDeveloperMode() {
				a.developerAction.ToolbarObject().Show()
			} else {
				a.developerAction.ToolbarObject().Hide()
			}

			a.currentNotification = cn
			a.bodyText.Show()
			a.headerWidget.Show()
			a.subjectLabel.Show()
			a.toolbar.Show()
		})
	}()
}

func (a *communicationsReadingPane) notificationRecipient(cn *app.CharacterNotification) *app.EveEntity {
	o := &app.EveEntity{
		ID:       cn.CharacterID,
		Name:     a.co.character.Load().NameOrZero(),
		Category: app.EveEntityCharacter,
	}
	return o
}

func parseIDs(input string) (int64, int64, error) {
	parts := strings.SplitN(input, "//", 2)
	id1, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid first ID %q: %w", parts[0], err)
	}
	if len(parts) < 2 {
		return id1, 0, nil
	}
	id2, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid second ID %q: %w", parts[1], err)
	}
	return id1, id2, nil
}
