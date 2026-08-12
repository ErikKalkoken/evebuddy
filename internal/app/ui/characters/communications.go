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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type notificationFolder struct {
	group  app.EveNotificationGroup
	name   string
	unread optional.Optional[int]
	total  optional.Optional[int]
}

type notificationHeaderRow struct {
	characterID    int64
	isRead         bool
	notificationID int64
	searchTarget   string
	sender         *app.EveEntity
	subject        string
	timestamp      time.Time
}

type Communications struct {
	widget.BaseWidget

	Detail     *fyne.Container
	Headers    *fyne.Container
	OnSelected func()
	OnUpdate   func(count optional.Optional[int])

	character              atomic.Pointer[app.Character]
	columnSorter           *xwidget.ColumnSorter[notificationHeaderRow]
	currentFolder          app.EveNotificationGroup
	currentNotification    *app.CharacterNotification
	developerToolbarAction *widget.ToolbarAction
	folderList             *widget.List
	folders                []notificationFolder
	foldersFooter          *widget.Label
	headerList             *widget.List
	headerRows             []notificationHeaderRow
	headerRowsFiltered     []notificationHeaderRow
	headersFooter          *widget.Label
	headersTop             *widget.Label
	notificationDetail     *notificationDetailWidget
	searchEntry            *widget.Entry
	sortButton             *xwidget.SortButton
	toolbar                *widget.Toolbar
	u                      baseUI
}

func (n notificationHeaderRow) IsZero() bool {
	return n.notificationID == 0
}

const (
	communicationsColTimestamp = iota + 1
	communicationsColSender
)

func NewCommunications(u baseUI) *Communications {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[notificationHeaderRow]{{
		ID:    communicationsColTimestamp,
		Label: "Date",
		Sort: func(a, b notificationHeaderRow) int {
			return a.timestamp.Compare(b.timestamp)
		},
	}, {
		ID:    communicationsColSender,
		Label: "Sender",
		Sort: func(a, b notificationHeaderRow) int {
			return strings.Compare(a.sender.Name, b.sender.Name)
		},
	}}),
		communicationsColTimestamp,
		xwidget.SortDesc,
	)
	a := &Communications{
		columnSorter:  columnSorter,
		foldersFooter: widget.NewLabel(""),
		headersFooter: widget.NewLabel(""),
		headersTop:    widget.NewLabel(""),
		searchEntry:   widget.NewEntry(),
		u:             u,
	}
	a.ExtendBaseWidget(a)
	a.headersTop.SizeName = theme.SizeNameSubHeadingText

	a.searchEntry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.searchEntry.SetText("")
		a.filterRowsAsync()
	})
	a.searchEntry.OnChanged = func(_ string) {
		a.filterRowsAsync()
	}
	a.searchEntry.PlaceHolder = "Search notifications"
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.u.MainWindow())

	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	a.folderList = a.makeFolderList()
	a.notificationDetail = newCommunicationDetailWidget(
		u.EVEImage().EveEntityLogoAsync,
		u.InfoViewer().Show, func(typeID, itemID int64) {
			u.InfoViewer().Show2(typeID, itemID, a.character.Load().IDOrZero())
		},
	)
	a.Detail = container.NewBorder(
		a.toolbar,
		nil,
		nil,
		nil,
		a.notificationDetail,
	)
	a.headerList = a.makeNotificationList()
	a.Headers = container.NewBorder(
		container.NewVBox(
			a.headersTop,
			container.NewBorder(nil, nil, nil, a.sortButton, a.searchEntry),
		),
		a.headersFooter,
		nil,
		nil,
		a.headerList,
	)
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
		a.Headers,
		a.Detail,
	)
	split1.Offset = 0.35
	split2 := container.NewHSplit(
		container.NewBorder(nil, a.foldersFooter, nil, nil, a.folderList),
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
		s := f.name
		if f.unread.ValueOrZero() > 0 {
			s += fmt.Sprintf(" (%s)", ihumanize.OptionalWithComma(f.unread, "?"))
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
		a.headerList.UnselectAll()
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
			return len(a.headerRowsFiltered)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItemWidget(a.u.EVEImage().EveEntityLogoAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.headerRowsFiltered) {
				return
			}
			r := a.headerRowsFiltered[id]
			item := co.(*MailHeaderItemWidget)
			item.Set(
				r.sender,
				r.subject,
				r.timestamp,
				r.isRead,
			)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		a.clearDetail()
		if id >= len(a.headerRowsFiltered) {
			l.UnselectAll()
			return
		}
		a.setDetail(a.headerRowsFiltered[id])
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *Communications) setDetail(r notificationHeaderRow) {
	ctx := context.Background()
	if !r.isRead {
		r.isRead = true
		go a.setNotificationRead(ctx, r.characterID, r.notificationID)
	}
	go func() {
		cn, err := a.u.Character().GetNotification(ctx, r.characterID, r.notificationID)
		if err != nil {
			fyne.Do(func() {
				ui.ShowErrorAndLog("Failed to load communication", err, a.u.IsDeveloperMode(), a.u.MainWindow())
			})
			return
		}
		fyne.Do(func() {
			err := a.notificationDetail.set(cn, a.notificationRecipient(cn))
			if err != nil {
				slog.Warn("Failed to set notification detail", "err", err)
				a.notificationDetail.setError(a.u.ErrorDisplay(err))
				return
			}
			if a.u.IsDeveloperMode() {
				a.developerToolbarAction.ToolbarObject().Show()
			} else {
				a.developerToolbarAction.ToolbarObject().Hide()
			}
			a.currentNotification = cn
			a.toolbar.Show()
			a.notificationDetail.Show()
		})
	}()
}

func (a *Communications) setNotificationRead(ctx context.Context, characterID, notificationID int64) {
	err := a.u.Character().SetNotificationRead(ctx, characterID, notificationID)
	if err != nil {
		slog.Error("Failed to set notification as read", "characterID", characterID, "notificationID", notificationID)
		return
	}
	fyne.Do(func() {
		var totalUnread optional.Optional[int]
		unreadIdx, currentIdx, allIdx := -1, -1, -1
		for i, f := range a.folders {
			if f.group == app.GroupUnread {
				unreadIdx = i
				totalUnread = f.unread
			}
			if f.group == a.currentFolder {
				currentIdx = i
			}
			if f.group == app.GroupAll {
				allIdx = i
			}
		}
		a.folders[currentIdx].unread.Set(a.folders[currentIdx].unread.ValueOrZero() - 1)
		a.folders[unreadIdx].unread.Set(totalUnread.ValueOrZero() - 1)
		a.folders[allIdx].unread = a.folders[unreadIdx].unread
		a.folderList.Refresh()
		if a.OnUpdate != nil {
			a.OnUpdate(a.folders[unreadIdx].unread)
		}
	})
	fyne.Do(func() {
		for i, r := range a.headerRows {
			if characterID == r.characterID && notificationID == r.notificationID {
				a.headerRows[i].isRead = true
			}
		}
		a.filterRowsAsync()
	})
}

func (a *Communications) notificationRecipient(cn *app.CharacterNotification) *app.EveEntity {
	o := &app.EveEntity{
		ID:       cn.CharacterID,
		Name:     a.character.Load().NameOrZero(),
		Category: app.EveEntityCharacter,
	}
	return o
}

func (a *Communications) makeToolbar() *widget.Toolbar {
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
					a.u.IsDeveloperMode(),
					a.u.MainWindow(),
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
			a.u.ShowSnackbar("Communication copied to clipboard")
		}),
	)
	items := []*fyne.MenuItem{
		fyne.NewMenuItem(
			"Send test notification",
			func() {
				if a.currentNotification == nil {
					return
				}
				if a.character.Load() == nil {
					return
				}
				// go a.u.Character().SendDesktopNotification(context.Background(), a.currentNotification)
			},
		),
		fyne.NewMenuItem(
			"Copy notification object to clipboard",
			func() {
				b, err := a.currentNotification.ToJSON()
				if err != nil {
					slog.Error("Failed to convert notification to JSON", "characterID", a.currentNotification.CharacterID, "notificationID", a.currentNotification.NotificationID, "error", err)
					a.u.ShowSnackbar("ERROR: Failed to convert data: " + err.Error())
					return
				}
				if len(b) == 0 {
					return
				}
				fyne.CurrentApp().Clipboard().SetContent(string(b))
				a.u.ShowSnackbar("Notification object copied to clipboard")
			},
		),
	}
	a.developerToolbarAction = kxwidget.NewToolbarActionMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("", items...))
	a.developerToolbarAction.ToolbarObject().Hide()
	toolbar.Append(widget.NewToolbarSpacer())
	toolbar.Append(a.developerToolbarAction)
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
			a.folderList.Select(0)
			if a.OnUpdate != nil {
				a.OnUpdate(optional.Optional[int]{})
			}
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.foldersFooter.Text, a.foldersFooter.Importance = s, i
			a.foldersFooter.Refresh()
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

	top := fmt.Sprintf("%s total", ihumanize.OptionalWithComma(totalCount, "?"))
	setTop(top, widget.MediumImportance)
	a.ResetCurrentFolder(ctx)

	fyne.Do(func() {
		a.clearDetail()
		a.folders = folders
		a.folderList.Refresh()
		a.folderList.UnselectAll()
		a.folderList.Select(0)
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
			name:  g.String(),
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
	if unreadCount.ValueOrZero() > 0 {
		folders = slices.Insert(folders, 0, notificationFolder{
			group:  app.GroupUnread,
			name:   "Unread",
			unread: unreadCount,
		})
	}
	folders = append(folders, notificationFolder{
		group:  app.GroupAll,
		name:   "All",
		unread: unreadCount,
	})
	return folders, unreadCount, totalCount, err
}

func (a *Communications) ResetCurrentFolder(ctx context.Context) {
	a.setCurrentFolder(ctx, app.GroupUnread)
	fyne.Do(func() {
		a.headerList.UnselectAll()
	})
}

func (a *Communications) filterRowsAsync() {
	totalRows := len(a.headerRows)
	rows := slices.Clone(a.headerRows)
	// selectStatus := a.selectStatus.Selected
	// selectTag := a.selectTag.Selected
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		// filter
		// if selectStatus != "" {
		// 	rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
		// 		switch selectStatus {
		// 		case trainingStatusActive:
		// 			return !r.isActive
		// 		case trainingStatusInActive:
		// 			return r.isActive
		// 		}
		// 		return true
		// 	})
		// }
		// if selectTag != "" {
		// 	rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
		// 		return !r.tags.Contains(selectTag)
		// 	})
		// }
		// search filter
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r notificationHeaderRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh

		footer := fmt.Sprintf("Showing %d / %d messages", len(rows), totalRows)

		fyne.Do(func() {
			a.headersFooter.Text = footer
			a.headersFooter.Importance = widget.MediumImportance
			a.headersFooter.Refresh()
			a.headerRowsFiltered = rows
			a.headerList.Refresh()
		})
	}()
}

func (a *Communications) setCurrentFolder(ctx context.Context, ng app.EveNotificationGroup) {
	reset := func() {
		fyne.Do(func() {
			a.headerRows = xslices.Reset(a.headerRows)
			a.headerList.Refresh()
			a.headerList.ScrollToTop()
		})

	}

	character := a.character.Load()
	if character == nil {
		reset()
		fyne.Do(func() {
			a.headersTop.SetText("ERROR: " + "No character")
		})
		return
	}

	hasData, err := a.u.Character().HasSection(ctx, character.ID, app.SectionCharacterNotifications)
	if err != nil {
		reset()
		fyne.Do(func() {
			a.headersTop.SetText(a.u.ErrorDisplay(err))
		})
		return
	}

	if !hasData {
		reset()
		fyne.Do(func() {
			a.headersTop.SetText("ERROR: " + "No data")
		})
	}

	notificationRows, err := a.fetchNotifications(ctx, ng, character)
	if err != nil {
		reset()
		fyne.Do(func() {
			a.headersTop.SetText("ERROR" + a.u.ErrorDisplay(err))
		})
		return
	}

	fyne.Do(func() {
		a.currentFolder = ng
		a.headersTop.SetText(ng.String())
		a.headerRows = notificationRows
		a.filterRowsAsync()
		a.headerList.ScrollToTop()
	})
}

func (a *Communications) fetchNotifications(ctx context.Context, nc app.EveNotificationGroup, character *app.Character) ([]notificationHeaderRow, error) {
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

	var rows []notificationHeaderRow

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
		r := notificationHeaderRow{
			characterID:    n.CharacterID,
			notificationID: n.NotificationID,
			sender:         sender,
			subject:        subject,
			timestamp:      n.Timestamp,
			isRead:         n.IsRead.ValueOrZero(),
			searchTarget:   strings.ToLower(fmt.Sprintf("%s-%s", subject, sender.Name)),
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *Communications) clearDetail() {
	a.notificationDetail.Hide()
	a.toolbar.Hide()
	a.currentNotification = nil
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

// notificationDetailWidget shows the complete communication for a character.
type notificationDetailWidget struct {
	widget.BaseWidget

	body    *xwidget.RichText
	header  *MailHeaderWidget
	subject *widget.Label
	show2   func(int64, int64)
}

func newCommunicationDetailWidget(loadIcon ui.EveEntityIconLoader, show func(*app.EveEntity), show2 func(int64, int64)) *notificationDetailWidget {
	subject := widget.NewLabel("")
	subject.SizeName = theme.SizeNameSubHeadingText
	subject.Wrapping = fyne.TextWrapWord
	body := xwidget.NewRichText()
	body.Wrapping = fyne.TextWrapWord
	w := &notificationDetailWidget{
		body:    body,
		header:  NewMailHeaderWidget(loadIcon, show),
		subject: subject,
		show2:   show2,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *notificationDetailWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(w.subject, w.header, w.body)
	return widget.NewSimpleRenderer(c)
}

func (w *notificationDetailWidget) set(n *app.CharacterNotification, recipient *app.EveEntity) error {
	w.subject.SetText(n.TitleDisplay())
	w.header.Set(n.Sender, n.Timestamp, recipient)
	v, ok := n.Body.Value()
	if !ok {
		w.body.SetWithText("[This notification type is not fully supported yet]", widget.RichTextStyle{
			ColorName: theme.ColorNameDisabled,
		})
		return nil
	}
	w.body.ParseMarkdown(v)
	for _, s := range w.body.Segments {
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
			w.show2(typeID, itemID)
		}
	}
	return nil
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

func (w *notificationDetailWidget) setError(text string) {
	w.subject.SetText("ERROR")
	w.header.Clear()
	w.body.SetWithText(text, widget.RichTextStyle{
		ColorName: theme.ColorNameError,
	})
}
