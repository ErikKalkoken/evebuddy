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

type notificationRow struct {
	characterID              int64
	characterName            string
	id                       int64
	isRead                   bool
	isRead2                  bool // updated by user, can be newer then isRead
	notificationGroup        app.EveNotificationGroup
	notificationGroupDisplay string
	notificationID           int64
	notificationTypeDisplay  string
	recipient                *app.EveEntity
	searchTarget             string
	sender                   *app.EveEntity
	subject                  string
	timestamp                time.Time
}

func (n notificationRow) isGroup(g app.EveNotificationGroup) bool {
	return g == app.GroupAll || n.notificationGroup == g
}

type Communications struct {
	widget.BaseWidget

	OnUpdate       func(count optional.Optional[int])
	MessagePane    *communicationsMessagePane
	NavigationPane *communicationsNavigationPane
	ReadingPane    *communicationsReadingPane

	character    atomic.Pointer[app.Character]
	forCharacter atomic.Bool
	rows         []notificationRow
	u            baseUI
}

func NewCommunicationsForCharacter(u baseUI) *Communications {
	return newCommunications(u, true)
}

func NewUnifiedCommunications(u baseUI) *Communications {
	return newCommunications(u, false)
}

func newCommunications(u baseUI, forCharacter bool) *Communications {
	a := &Communications{
		u: u,
	}
	a.ExtendBaseWidget(a)
	a.forCharacter.Store(forCharacter)
	a.MessagePane = newCommunicationsMessagePane(a)
	a.NavigationPane = newCommunicationsNavigationPane(a)
	a.ReadingPane = newCommunicationsReadingPane(a)

	// Signals
	if forCharacter {
		a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			fyne.Do(func() {
				a.ReadingPane.clear()
			})
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
	} else {
		a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if arg.Section == app.SectionCharacterNotifications {
				a.update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.update(ctx)
		})
	}
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

func (a *Communications) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.NavigationPane.update()
			a.NavigationPane.folderList.UnselectAll()
			a.NavigationPane.folderList.Select(0)

			if a.OnUpdate != nil {
				a.OnUpdate(optional.Optional[int]{})
			}
		})
	}
	setFooter := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.NavigationPane.footerLabel.Text, a.NavigationPane.footerLabel.Importance = s, i
			a.NavigationPane.footerLabel.Refresh()
		})
	}

	var characterID int64
	if a.forCharacter.Load() {
		characterID = a.character.Load().IDOrZero()
		if characterID == 0 {
			reset()
			setFooter("No character", widget.LowImportance)
			return
		}
		hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterNotifications)
		if err != nil {
			reset()
			setFooter("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
			return
		}
		if !hasData {
			reset()
			setFooter("No data", widget.WarningImportance)
			return
		}
	}
	rows, _, err := a.fetchRows(ctx, characterID)
	if err != nil {
		reset()
		setFooter("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	fyne.Do(func() {
		a.rows = rows
		a.NavigationPane.update()
	})
}

func (a *Communications) fetchRows(ctx context.Context, characterID int64) ([]notificationRow, int, error) {
	characters, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	characterMap := make(map[int64]*app.Character)
	for _, c := range characters {
		characterMap[c.ID] = c
	}
	var oo []*app.CharacterNotification
	if characterID != 0 {
		oo, err = a.u.Character().ListNotifications(ctx, characterID)
	} else {
		oo, err = a.u.Character().ListAllNotifications(ctx)
	}
	if err != nil {
		return nil, 0, err
	}
	var unread int
	rows := make([]notificationRow, len(oo))
	for i, o := range oo {
		// Replace generic corporations && alliances in notifications
		var sender *app.EveEntity
		character, ok := characterMap[o.CharacterID]
		if ok {
			switch o.Sender.ID {
			case app.EveTypeAlliance:
				sender = character.EveCharacter.Alliance.ValueOrFallback(&app.EveEntity{
					ID:       1,
					Name:     "Unknown",
					Category: app.EveEntityCorporation,
				})
			case app.EveTypeCorporation:
				sender = character.EveCharacter.Corporation
			default:
				sender = o.Sender
			}
		} else {
			slog.Warn(
				"Failed to map character to notification in communications UI",
				slog.Int64("characterID", o.CharacterID),
				slog.Int64("notificationID", o.NotificationID),
			)
			sender = o.Sender
		}
		subject := o.TitleDisplay()
		r := notificationRow{
			characterID:              o.CharacterID,
			characterName:            character.NameOrZero(),
			id:                       o.ID,
			isRead:                   o.IsRead,
			notificationGroup:        o.Type.Group(),
			notificationGroupDisplay: o.Type.Group().String(),
			notificationID:           o.NotificationID,
			notificationTypeDisplay:  o.Type.Display(),
			searchTarget:             strings.ToLower(fmt.Sprintf("%s-%s", subject, sender.Name)),
			sender:                   sender,
			subject:                  subject,
			timestamp:                o.Timestamp,
			recipient: o.Recipient.ValueOrFallback(&app.EveEntity{
				ID:       o.CharacterID,
				Name:     character.NameOrZero(),
				Category: app.EveEntityCharacter,
			}),
		}
		r.isRead2 = r.isRead
		rows[i] = r
		if !r.isRead {
			unread++
		}
	}
	return rows, unread, nil
}

func (a *Communications) updateIsRead(ng app.EveNotificationGroup) {
	for i, r := range a.rows {
		if r.isGroup(ng) {
			a.rows[i].isRead = a.rows[i].isRead2
		}
	}
}

type notificationFolder struct {
	folder app.EveNotificationGroup
	name   string
	unread optional.Optional[int]
	total  optional.Optional[int]
}
type communicationsNavigationPane struct {
	widget.BaseWidget

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
			a.co.MessagePane.set(f.folder)
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
		a.co.MessagePane.set(o.folder)
	}
	return l
}

func (a *communicationsNavigationPane) update() {
	folders, totalCount, unreadCount := a.makeFolders()
	a.footerLabel.Text = fmt.Sprintf("%s total", ihumanize.OptionalWithComma(totalCount, "?"))
	a.footerLabel.Refresh()
	a.folders = folders
	a.folderList.Refresh()
	if a.co.MessagePane.currentFolder == app.GroupUndefined {
		a.folderList.UnselectAll()
		a.folderList.Select(0)
	} else {
		a.co.MessagePane.update()
	}
	if a.co.OnUpdate != nil {
		a.co.OnUpdate(unreadCount)
	}
}

func (a *communicationsNavigationPane) makeFolders() ([]notificationFolder, optional.Optional[int], optional.Optional[int]) {
	groupCounts := make(map[app.EveNotificationGroup]struct {
		total  int
		unread int
	})
	for _, r := range a.co.rows {
		gc := groupCounts[r.notificationGroup]
		gc.total++
		if !r.isRead2 {
			gc.unread++
		}
		groupCounts[r.notificationGroup] = gc
	}

	var folders []notificationFolder
	var unreadCount optional.Optional[int]
	for _, g := range app.NotificationGroups() {
		nf := notificationFolder{
			folder: g,
			name:   g.String(),
		}
		gc, ok := groupCounts[g]
		if ok {
			nf.total.Set(gc.total)
			nf.unread.Set(gc.unread)
			unreadCount.Set(unreadCount.ValueOrZero() + gc.unread)
		}
		folders = append(folders, nf)
	}
	slices.SortFunc(folders, func(a, b notificationFolder) int {
		return cmp.Compare(a.name, b.name)
	})
	folders = slices.Insert(folders, 0, notificationFolder{
		folder: app.GroupAll,
		name:   "All",
		unread: unreadCount,
	})
	totalCount := optional.New(len(a.co.rows))
	return folders, totalCount, unreadCount
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

// Columns for sorting
const (
	communicationsColTimestamp = iota + 1
	communicationsColSender
	communicationsColType
)

// Filter options
const (
	communicationsFilterCharacter = "Character"
	communicationsFilterGroup     = "Group"
	communicationsFilterRecipient = "Recipient"
	// communicationsFilterType      = "Type"
	communicationsFilterStatus       = "Status"
	communicationsFilterStatusRead   = "Read"
	communicationsFilterStatusUnread = "Unread"
)

type communicationsMessagePane struct {
	widget.BaseWidget

	OnSelected func()

	co            *Communications
	columnSorter  *xwidget.ColumnSorter[notificationRow]
	currentFolder app.EveNotificationGroup
	filterChip    *xwidget.FilterChipCompact
	footerLabel   *widget.Label
	messageList   *widget.List
	moreButton    *kxwidget.IconButton
	rowsFiltered  []notificationRow
	searchEntry   *xwidget.SearchEntry
	sortButton    *xwidget.SortButton
	topLabel      *widget.Label
}

func newCommunicationsMessagePane(co *Communications) *communicationsMessagePane {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[notificationRow]{{
		ID:    communicationsColTimestamp,
		Label: "Date",
		Sort: func(a, b notificationRow) int {
			return a.timestamp.Compare(b.timestamp)
		},
	}, {
		ID:    communicationsColSender,
		Label: "Sender",
		Sort: func(a, b notificationRow) int {
			return strings.Compare(a.sender.Name, b.sender.Name)
		},
	}, {
		ID:    communicationsColType,
		Label: "Type",
		Sort: func(a, b notificationRow) int {
			return strings.Compare(a.notificationTypeDisplay, b.notificationTypeDisplay)
		},
	}}),
		communicationsColTimestamp,
		xwidget.SortDesc,
	)
	a := &communicationsMessagePane{
		columnSorter: columnSorter,
		footerLabel:  widget.NewLabel(""),
		topLabel:     widget.NewLabel(""),
		co:           co,
	}
	a.ExtendBaseWidget(a)
	a.messageList = a.makeMessageList()
	a.topLabel.SizeName = theme.SizeNameSubHeadingText
	a.searchEntry = xwidget.NewSearchEntry("Search communications", func(s string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.co.u.MainWindow())
	a.filterChip = xwidget.NewFilterChipCompact(nil, func(state map[string]string) {
		if state[communicationsFilterStatus] != "" {
			a.co.updateIsRead(a.currentFolder)
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
				container.NewHBox(a.filterChip, a.sortButton),
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

func (a *communicationsMessagePane) makeMessageList() *widget.List {
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

func (a *communicationsMessagePane) ResetHeaders() {
	a.set(app.GroupAll)
}

func (a *communicationsMessagePane) markCurrentFolderRead() {
	var ids set.Set[int64]
	for _, r := range a.co.rows {
		if r.isGroup(a.currentFolder) && !r.isRead2 {
			ids.Add(r.id)
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
		fyne.Do(func() {
			for i, r := range a.co.rows {
				if ids.Contains(r.id) && !r.isRead2 {
					a.co.rows[i].isRead2 = true
				}
			}
			a.co.NavigationPane.update()
			a.filterRowsAsync()
		})
	}()
}

func (a *communicationsMessagePane) setNotificationRead(ctx context.Context, id int64) {
	err := a.co.u.Character().SetNotificationsAsRead(ctx, set.Of(id))
	if err != nil {
		slog.Error("Failed to set notification as read", "ID", id)
		return
	}
	fyne.Do(func() {
		for i, r := range a.co.rows {
			if id == r.id {
				a.co.rows[i].isRead2 = true
			}
		}
		a.co.NavigationPane.update()
		a.filterRowsAsync()
	})
}

func (a *communicationsMessagePane) filterRowsAsync() {
	var rows []notificationRow
	if a.currentFolder == app.GroupAll {
		rows = slices.Clone(a.co.rows)
	} else {
		rows = xslices.Filter(a.co.rows, func(x notificationRow) bool {
			return x.notificationGroup == a.currentFolder
		})
	}
	totalRows := len(rows)
	filter := a.filterChip.Selected()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)
	go func() {
		// filter
		if x := filter[communicationsFilterStatus]; x != "" {
			switch x {
			case communicationsFilterStatusUnread:
				rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
					return r.isRead
				})
			case communicationsFilterStatusRead:
				rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
					return !r.isRead
				})
			}
		}
		if x := filter[communicationsFilterCharacter]; x != "" {
			rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
				return r.characterName != x
			})
		}
		if x := filter[communicationsFilterGroup]; x != "" {
			rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
				return r.notificationGroupDisplay != x
			})
		}
		if x := filter[communicationsFilterRecipient]; x != "" {
			rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
				return r.recipient.Name != x
			})
		}
		// if x := filter[communicationsFilterType]; x != "" {
		// 	rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
		// 		return r.notificationTypeDisplay != x
		// 	})
		// }
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r notificationRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		// sort
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// collect options
		// typesOptions := xslices.Map(rows, func(r notificationRow) string {
		// 	return r.notificationTypeDisplay
		// })
		characterOptions := xslices.Map(rows, func(r notificationRow) string {
			return r.characterName
		})
		groupOptions := xslices.Map(rows, func(r notificationRow) string {
			return r.notificationGroupDisplay
		})
		recipientOptions := xslices.Map(rows, func(r notificationRow) string {
			return r.recipient.Name
		})
		unreadOptions := xslices.Map(rows, func(r notificationRow) string {
			if r.isRead2 {
				return communicationsFilterStatusRead
			}
			return communicationsFilterStatusUnread
		})

		// refresh
		id2idx := make(map[int64]int)
		for i, r := range rows {
			id2idx[r.id] = i
		}

		footer := fmt.Sprintf(
			"Showing %s / %s messages",
			ihumanize.Comma(len(rows)),
			ihumanize.Comma(totalRows),
		)
		fyne.Do(func() {
			options := []xwidget.FilterOption{
				xwidget.NewFilterOptionMultiChoice(
					communicationsFilterStatus,
					unreadOptions,
				),
			}
			if !a.co.forCharacter.Load() {
				options = append(options, xwidget.NewFilterOptionMultiChoice(
					communicationsFilterCharacter,
					characterOptions,
				))
			}
			if a.currentFolder.IsContainer() {
				options = append(options, xwidget.NewFilterOptionMultiChoice(
					communicationsFilterGroup,
					groupOptions,
				))
			}
			options = append(options, xwidget.NewFilterOptionMultiChoice(
				communicationsFilterRecipient,
				recipientOptions,
			))
			a.footerLabel.Text = footer
			a.footerLabel.Importance = widget.MediumImportance
			a.footerLabel.Refresh()
			a.filterChip.SetOptions(options...)
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

func (a *communicationsMessagePane) set(ng app.EveNotificationGroup) {
	a.co.ReadingPane.clear()
	a.currentFolder = ng
	a.topLabel.SetText(ng.String())
	a.searchEntry.SetText("")
	a.filterChip.Reset()
	a.filterRowsAsync()
}

func (a *communicationsMessagePane) update() {
	a.filterRowsAsync()
}

type communicationsReadingPane struct {
	widget.BaseWidget

	bodyText            *xwidget.RichText
	co                  *Communications
	currentNotification *app.CharacterNotification
	developerAction     *widget.ToolbarAction
	copyAction          *widget.ToolbarAction
	headerWidget        *MailHeaderWidget
	subjectLabel        *widget.Label
	toolbar             *widget.Toolbar
}

func newCommunicationsReadingPane(co *Communications) *communicationsReadingPane {
	header := NewMailHeaderWidget(co.u.EVEImage().EveEntityLogoAsync, co.u.InfoViewer().Show)
	a := &communicationsReadingPane{
		bodyText:     xwidget.NewRichText(),
		co:           co,
		headerWidget: header,
		subjectLabel: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.bodyText.Wrapping = fyne.TextWrapWord
	a.subjectLabel.SizeName = theme.SizeNameSubHeadingText
	a.subjectLabel.Wrapping = fyne.TextWrapWord

	a.copyAction = widget.NewToolbarAction(theme.ContentCopyIcon(), nil)
	a.developerAction = xwidget.NewToolbarActionMenu(theme.MoreHorizontalIcon(), fyne.NewMenu(""))
	a.developerAction.ToolbarObject().Hide()
	a.toolbar = widget.NewToolbar(a.copyAction, widget.NewToolbarSpacer(), a.developerAction)
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

func (a *communicationsReadingPane) clear() {
	a.currentNotification = nil
	a.bodyText.Hide()
	a.headerWidget.Hide()
	a.subjectLabel.Hide()
	a.toolbar.Hide()
}

func (a *communicationsReadingPane) set(r notificationRow) {
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
			a.headerWidget.Set(cn.Sender, cn.Timestamp, r.recipient)
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
				items := a.makeMenuItems(cn)
				xwidget.SetToolbarActionMenu(a.developerAction, fyne.NewMenu("", items...))
				a.developerAction.ToolbarObject().Show()
			} else {
				a.developerAction.ToolbarObject().Hide()
			}

			a.copyAction.OnActivated = a.makeCopyAction(cn, r.recipient.NameOrZero())

			a.currentNotification = cn
			a.bodyText.Show()
			a.headerWidget.Show()
			a.subjectLabel.Show()
			a.toolbar.Show()
		})
	}()
}

func (a *communicationsReadingPane) makeCopyAction(cn *app.CharacterNotification, recipientName string) func() {
	f := func() {
		if cn == nil {
			return
		}

		header := fmt.Sprintf(
			"From: %s\nSent: %s\nTo: %s",
			cn.Sender.Name,
			cn.Timestamp.Format(app.DateTimeFormat),
			recipientName,
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
	}
	return f
}

func (a *communicationsReadingPane) makeMenuItems(cn *app.CharacterNotification) []*fyne.MenuItem {
	typeItem := fyne.NewMenuItem(cn.Type.Display(), nil)
	typeItem.Disabled = true
	items := []*fyne.MenuItem{
		typeItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(
			"Send test notification",
			func() {
				go a.co.u.Character().SendDesktopNotification(context.Background(), cn)
			},
		),
		fyne.NewMenuItem(
			"Copy notification object to clipboard",
			func() {
				b, err := cn.ToJSON()
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
	return items
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
