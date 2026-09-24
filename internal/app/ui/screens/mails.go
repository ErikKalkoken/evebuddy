package screens

import (
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
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/mailer"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type folderNodeCategory int

const (
	nodeCategoryBranch folderNodeCategory = iota + 1
	nodeCategoryLabel
	nodeCategoryList
)

type folderNodeType uint

const (
	folderNodeUndefined folderNodeType = iota
	folderNodeAll
	folderNodeAlliance
	folderNodeCorp
	folderNodeInbox
	folderNodeLabel
	folderNodeList
	folderNodeSent
	folderNodeTrash
)

// A mailFolderNode in the folder tree, e.g. the inbox
type mailFolderNode struct {
	Category    folderNodeCategory
	CharacterID int64
	IsLeaf      bool
	Type        folderNodeType
	Name        string
	ObjID       int64
	UnreadCount int
}

func (n *mailFolderNode) IsEmpty() bool {
	return n.CharacterID == 0
}

func (n *mailFolderNode) isBranch() bool {
	return n.Category == nodeCategoryBranch
}

func (n *mailFolderNode) icon() fyne.Resource {
	switch n.Type {
	case folderNodeInbox:
		return theme.DownloadIcon()
	case folderNodeSent:
		return theme.UploadIcon()
	case folderNodeTrash:
		return theme.DeleteIcon()
	}
	return theme.FolderIcon()
}

func (n mailFolderNode) UID() widget.TreeNodeID {
	return fmt.Sprintf("%d-%d-%d", n.CharacterID, n.Type, n.ObjID)
}

type Mails struct {
	widget.BaseWidget

	OnUpdate       func(unread, missing int)
	MessagePane    *mailsMessagePane
	NavigationPane *mailsNavigationPane
	ReadingPane    *mailsReadingPane

	character      atomic.Pointer[app.Character]
	missingPercent atomic.Int64
	sig            *singleinstance.Group
	u              baseUI
	unreadCount    atomic.Int64
}

func NewMails(u baseUI) *Mails {
	a := &Mails{
		u:   u,
		sig: singleinstance.NewGroup(),
	}
	a.ExtendBaseWidget(a)
	a.MessagePane = newMailsMessagePane(a)
	a.NavigationPane = newMailsNavigationPane(a)
	a.ReadingPane = newMailsReadingPane(a)

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
		switch arg.Section {
		case
			app.SectionCharacterMailLabels,
			app.SectionCharacterMailLists,
			app.SectionCharacterMailHeaders:
			a.update(ctx)
		}
	})
	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		a.NavigationPane.updateDownloaded(ctx)
	})
	return a
}

func (a *Mails) CreateRenderer() fyne.WidgetRenderer {
	// toolbar is not part of the reading pane, because mobile shows its own actions
	split1 := container.NewHSplit(
		a.MessagePane,
		container.NewBorder(a.ReadingPane.toolbar, nil, nil, nil, a.ReadingPane),
	)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.NavigationPane, split1)
	split2.SetOffset(0.15)
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

func (a *Mails) update(ctx context.Context) {
	clearAll := func() {
		fyne.Do(func() {
			a.NavigationPane.clear()
			a.MessagePane.currentFolder.Store(nil)
			a.MessagePane.clear()
			a.ReadingPane.clear()
		})
	}
	setStatus := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.NavigationPane.setStatus(s, i)
		})
	}
	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		clearAll()
		setStatus("No character", widget.LowImportance)
		return
	}
	hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterMailHeaders)
	if err != nil {
		slog.Error("Failed to build mail tree", "character", characterID, "error", err)
		setStatus("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	if !hasData {
		clearAll()
		setStatus("Data not fully loaded yet", widget.WarningImportance)
		return
	}
	td, inbox, err := a.NavigationPane.fetchFolders(ctx, characterID)
	if err != nil {
		slog.Error("Failed to build mail tree", "character", characterID, "error", err)
		setStatus("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	unread, err := a.NavigationPane.updateCountsInTree(ctx, characterID, td)
	if err != nil {
		slog.Error("Failed to update mail counts", "character", characterID, "error", err)
	}
	// keep showing the current folder if it still exists, e.g. after new mail arrived
	current := a.MessagePane.currentFolder.Load()
	folder := inbox
	if current != nil {
		if n, ok := td.Node(current.UID()); ok {
			folder = n
		}
	}
	isSameFolder := current != nil && current.UID() == folder.UID()
	a.MessagePane.currentFolder.Store(folder) // before set, so re-selecting the node is a no-op
	fyne.Do(func() {
		a.NavigationPane.set(td, folder)
	})
	if isSameFolder {
		a.MessagePane.update(ctx)
	} else {
		a.MessagePane.setCurrentFolder(ctx, folder)
	}
	a.unreadCount.Store(int64(unread))
	a.NavigationPane.updateDownloaded(ctx)
	fyne.Do(func() {
		a.callOnUpdate()
	})
}

func (a *Mails) callOnUpdate() {
	if a.OnUpdate == nil {
		return
	}
	a.OnUpdate(int(a.unreadCount.Load()), int(a.missingPercent.Load()))
}

func (a *Mails) showMailerWindow(mode mailer.Mode, mail *app.CharacterMail) {
	c := a.character.Load()
	if c == nil {
		return
	}
	w, err := mailer.NewWindow(a.u, c, mode, mail)
	if err != nil {
		ui.ShowErrorAndLog(
			"Failed to show mailer window",
			err,
			a.u.IsDeveloperMode(),
			a.u.MainWindow(),
		)
		return
	}
	w.Show()
}

func (a *Mails) MakeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.showMailerWindow(mailer.New, nil)
	}
}

type mailsNavigationPane struct {
	widget.BaseWidget

	compose          *widget.Button
	folderDownloaded *ttwidget.Label
	folders          *xwidget.Tree[mailFolderNode]
	folderStatus     *widget.Label
	folderTotal      *widget.Label
	ma               *Mails
}

func newMailsNavigationPane(ma *Mails) *mailsNavigationPane {
	a := &mailsNavigationPane{
		folderDownloaded: ttwidget.NewLabel(""),
		folderStatus:     widget.NewLabel(""),
		folderTotal:      widget.NewLabel("?"),
		ma:               ma,
	}
	a.ExtendBaseWidget(a)
	a.folders = a.makeFolderTree()
	a.folderStatus.Hide()
	r, f := ma.MakeComposeMessageAction()
	a.compose = widget.NewButtonWithIcon("Compose", r, f)
	a.compose.Importance = widget.HighImportance
	a.compose.Disable()
	return a
}

func (a *mailsNavigationPane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(container.NewPadded(a.compose), a.folderStatus),
		container.NewHBox(a.folderTotal, layout.NewSpacer(), a.folderDownloaded),
		nil,
		nil,
		a.folders,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *mailsNavigationPane) makeFolderTree() *xwidget.Tree[mailFolderNode] {
	t := xwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return newMailFolderItemWidget()
		},
		func(n *mailFolderNode, _ bool, co fyne.CanvasObject) {
			co.(*mailFolderItemWidget).set(n)
		},
	)
	t.OnSelectedNode = func(n *mailFolderNode) {
		if n.isBranch() {
			t.UnselectAll()
			t.ToggleBranchNode(n)
			return
		}
		if c := a.ma.MessagePane.currentFolder.Load(); c != nil && c.UID() == n.UID() {
			return // already shown, e.g. re-selected after a refresh
		}
		go a.ma.MessagePane.setCurrentFolder(context.Background(), n)
	}
	return t
}

func (a *mailsNavigationPane) clear() {
	a.folders.Clear()
	a.folderTotal.SetText("?")
	a.folderDownloaded.SetText("")
}

func (a *mailsNavigationPane) set(td *xwidget.TreeData[mailFolderNode], selected *mailFolderNode) {
	a.compose.Enable()
	a.folderStatus.Hide()
	var openBranches []widget.TreeNodeID
	a.folders.Data().Walk(nil, func(n *mailFolderNode) bool {
		if a.folders.IsBranchOpenNode(n) {
			openBranches = append(openBranches, n.UID())
		}
		return true
	})
	a.folders.Set(td)
	for _, uid := range openBranches {
		if n, ok := td.Node(uid); ok {
			a.folders.OpenBranchNode(n)
		}
	}
	a.folders.SelectNode(selected)
}

func (a *mailsNavigationPane) setStatus(s string, i widget.Importance) {
	a.folderStatus.Text = s
	a.folderStatus.Importance = i
	a.folderStatus.Refresh()
	a.folderStatus.Show()
	a.compose.Disable()
}

func (a *mailsNavigationPane) updateDownloaded(ctx context.Context) {
	var total2, downloaded, hint string
	var missingPercent int
	func() {
		characterID := a.ma.character.Load().IDOrZero()
		if characterID == 0 {
			return
		}
		total, missing, err := a.ma.u.Character().DownloadedBodiesPercentage(ctx, characterID)
		if err != nil {
			slog.Error("updateDownloaded", "error", err)
			total2 = "ERROR"
			return
		}
		p := message.NewPrinter(language.English)
		total2 = p.Sprintf("%d total", total)
		if total == 0 || missing == 0 {
			return
		}

		missingPercent = int(float64(missing) / float64(total) * 100)
		downloaded = fmt.Sprintf("%d%%", 100-missingPercent)
		hint = p.Sprintf("%d / %d mails downloaded", total-missing, total)
	}()
	a.ma.missingPercent.Store(int64(missingPercent))
	fyne.Do(func() {
		a.folderTotal.SetText(total2)
		if downloaded == "" {
			a.folderDownloaded.Hide()
			return
		}
		a.folderDownloaded.SetText(downloaded)
		a.folderDownloaded.SetToolTip(hint)
		a.folderDownloaded.Show()
		a.ma.callOnUpdate()
	})
}

func (a *mailsNavigationPane) fetchFolders(ctx context.Context, characterID int64) (*xwidget.TreeData[mailFolderNode], *mailFolderNode, error) {
	if characterID == 0 {
		return nil, nil, nil
	}

	td := xwidget.NewTreeData[mailFolderNode]()

	// Add default folders
	var inbox *mailFolderNode
	defaultFolders := []struct {
		nodeType folderNodeType
		labelID  int64
		name     string
	}{
		{folderNodeInbox, app.MailLabelInbox, "Inbox"},
		{folderNodeSent, app.MailLabelSent, "Sent"},
		{folderNodeCorp, app.MailLabelCorp, "Corp"},
		{folderNodeAlliance, app.MailLabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		n := &mailFolderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			Type:        o.nodeType,
			Name:        o.name,
			ObjID:       o.labelID,
		}
		if err := td.Add(nil, n, false); err != nil {
			return td, nil, err
		}
		if o.nodeType == folderNodeInbox {
			inbox = n
		}
	}

	// Add custom labels
	labels, err := a.ma.u.Character().ListMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return td, nil, err
	}
	if len(labels) > 0 {
		n := &mailFolderNode{
			Category:    nodeCategoryBranch,
			CharacterID: characterID,
			Name:        "Labels",
			Type:        folderNodeLabel,
		}
		err := td.Add(nil, n, len(labels) > 0)
		if err != nil {
			return td, nil, err
		}
		for _, l := range labels {
			err := td.Add(n, &mailFolderNode{
				Category:    nodeCategoryLabel,
				CharacterID: characterID,
				Name:        l.Name.ValueOrZero(),
				ObjID:       l.LabelID,
				Type:        folderNodeLabel,
			}, false)
			if err != nil {
				return td, nil, err
			}
		}
	}

	// Add mailing lists
	lists, err := a.ma.u.Character().ListMailLists(ctx, characterID)
	if err != nil {
		return td, nil, err
	}
	if len(lists) > 0 {
		n := &mailFolderNode{
			Category:    nodeCategoryBranch,
			CharacterID: characterID,
			Name:        "Mailing Lists",
			Type:        folderNodeList,
		}
		err := td.Add(nil, n, len(lists) > 0)
		if err != nil {
			return td, nil, err
		}
		for _, l := range lists {
			err := td.Add(n, &mailFolderNode{
				Category:    nodeCategoryList,
				CharacterID: characterID,
				ObjID:       l.ID,
				Name:        l.Name,
				Type:        folderNodeList,
			}, false)
			if err != nil {
				return td, nil, err
			}
		}
	}
	// Add all folder
	err = td.Add(nil, &mailFolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All",
		ObjID:       app.MailLabelAll,
	}, false)
	if err != nil {
		return td, nil, err
	}
	return td, inbox, nil
}

func (a *mailsNavigationPane) updateCountsInTree(ctx context.Context, characterID int64, td *xwidget.TreeData[mailFolderNode]) (int, error) {
	if td.IsEmpty() {
		return 0, nil
	}
	labelUnreadCounts, err := a.ma.u.Character().GetMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		return 0, err
	}
	listUnreadCounts, err := a.ma.u.Character().GetMailListUnreadCounts(ctx, characterID)
	if err != nil {
		return 0, err
	}

	var totalCount, labelCount, listCount int
	for id, c := range labelUnreadCounts {
		totalCount += c
		if id > app.MailLabelAlliance {
			labelCount += c
		}
	}
	for _, c := range listUnreadCounts {
		totalCount += c
		listCount += c
	}

	td.Walk(nil, func(n *mailFolderNode) bool {
		var c int
		switch n.Type {
		case folderNodeAll:
			c = totalCount
		case folderNodeInbox, folderNodeAlliance, folderNodeCorp:
			c = labelUnreadCounts[n.ObjID]
		case folderNodeLabel:
			if n.ObjID == 0 {
				c = labelCount
				break
			}
			c = labelUnreadCounts[n.ObjID]
		case folderNodeList:
			if n.ObjID == 0 {
				c = listCount
				break
			}
			c = listUnreadCounts[n.ObjID]
		}
		if n.UnreadCount != c {
			n.UnreadCount = c
		}
		return true
	})
	return totalCount, nil
}

func (a *mailsNavigationPane) updateUnreadCounts(ctx context.Context) {
	td := a.folders.Data()
	characterID := a.ma.character.Load().IDOrZero()
	unread, err := a.updateCountsInTree(ctx, characterID, td)
	if err != nil {
		slog.Error("Failed to update unread counts", "characterID", characterID, "error", err)
		return
	}
	a.ma.unreadCount.Store(int64(unread))
	fyne.Do(func() {
		a.folders.Refresh() // counts changed in place; Set would reset selection and branches
		a.ma.callOnUpdate()
	})
}

func (a *mailsNavigationPane) MakeFolderMenu() []*fyne.MenuItem {
	var items1 []*fyne.MenuItem
	a.folders.Data().Walk(nil, func(f *mailFolderNode) bool {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			go a.ma.MessagePane.setCurrentFolder(context.Background(), f)
		})
		items1 = append(items1, it)
		return true
	})
	return items1
}

type mailFolderItemWidget struct {
	widget.BaseWidget

	icon   *widget.Icon
	name   *widget.Label
	unread *widget.Label
}

func newMailFolderItemWidget() *mailFolderItemWidget {
	w := &mailFolderItemWidget{
		name:   widget.NewLabel(""),
		icon:   widget.NewIcon(icons.BlankSvg),
		unread: widget.NewLabel("999"),
	}
	w.ExtendBaseWidget(w)
	w.name.Truncation = fyne.TextTruncateClip
	return w
}

func (w *mailFolderItemWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, w.icon, w.unread, w.name)
	return widget.NewSimpleRenderer(c)
}

func (w *mailFolderItemWidget) set(n *mailFolderNode) {
	w.icon.SetResource(n.icon())
	if n.UnreadCount == 0 {
		w.name.TextStyle.Bold = false
		w.unread.Hide()
	} else {
		w.name.TextStyle.Bold = true
		w.unread.SetText(strconv.Itoa(n.UnreadCount))
		w.unread.Show()
	}
	w.name.Text = n.Name
	w.name.Refresh()

}

type mailRow struct {
	characterID  int64
	from         *app.EveEntity
	id           int64
	isRead       bool // snapshot used for filtering, so rows don't vanish from an active Unread filter
	isRead2      bool // current state, used for display
	mailID       int64
	searchTarget string
	subject      string
	timestamp    time.Time
}

// Filter options
const (
	mailsFilterFrom         = "From"
	mailsFilterStatus       = "Status"
	mailsFilterStatusRead   = "Read"
	mailsFilterStatusUnread = "Unread"
)

type mailsMessagePane struct {
	widget.BaseWidget

	OnSelected func()

	columnSorter  *xwidget.ColumnSorter[mailRow]
	currentFolder atomic.Pointer[mailFolderNode]
	filterChip    *xwidget.FilterChipCompact
	filterRun     latestRun
	footerLabel   *widget.Label
	headerList    *widget.List
	headerStatus  *widget.Label
	ma            *Mails
	reselecting   bool // suppresses OnSelected while restoring the selection
	rows          []mailRow
	rowsFolderUID widget.TreeNodeID // folder the rows were loaded for
	rowsFiltered  []mailRow
	searchEntry   *xwidget.SearchEntry
	sortButton    *kxwidget.SortChip
	topLabel      *widget.Label
}

func newMailsMessagePane(ma *Mails) *mailsMessagePane {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[mailRow]{{
		Label: "Date",
		Sort: func(a, b mailRow) int {
			return a.timestamp.Compare(b.timestamp)
		},
	}, {
		Label: "From",
		Sort: func(a, b mailRow) int {
			return strings.Compare(a.from.NameOrZero(), b.from.NameOrZero())
		},
	}, {
		Label: "Subject",
		Sort: func(a, b mailRow) int {
			return strings.Compare(a.subject, b.subject)
		},
	}}),
		"Date",
		xwidget.SortDesc,
	)
	a := &mailsMessagePane{
		columnSorter: columnSorter,
		footerLabel:  widget.NewLabel(""),
		headerStatus: widget.NewLabel(""),
		ma:           ma,
		topLabel:     widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.headerStatus.Hide()
	a.headerList = a.makeHeaderList()
	a.topLabel.SizeName = theme.SizeNameSubHeadingText
	a.topLabel.Truncation = fyne.TextTruncateEllipsis
	a.searchEntry = xwidget.NewSearchEntry("Search mails", func(_ string) {
		a.filterRowsAsync()
	})
	a.sortButton = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync()
	})
	a.filterChip = xwidget.NewFilterChipCompact(nil, func(state map[string]string) {
		if state[mailsFilterStatus] != "" {
			a.updateIsRead()
		}
		a.filterRowsAsync()
	})
	return a
}

func (a *mailsMessagePane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.topLabel,
			a.headerStatus,
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
		a.headerList,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *mailsMessagePane) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItemWidget(a.ma.u.EVEImage().EveEntityLogoAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			if a.ma.character.Load() == nil {
				return
			}
			item := co.(*MailHeaderItemWidget)
			item.Set(r.from, r.subject, r.timestamp, r.isRead2)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if a.reselecting || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.ma.ReadingPane.showMail(r.mailID)
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *mailsMessagePane) clear() {
	a.filterRun.start() // discards pending filter results
	xslices.Clear(&a.rows)
	xslices.Clear(&a.rowsFiltered)
	a.rowsFolderUID = ""
	a.headerList.Refresh()
	a.topLabel.SetText("")
	a.footerLabel.SetText("")
}

func (a *mailsMessagePane) setCurrentFolder(ctx context.Context, folder *mailFolderNode) {
	a.currentFolder.Store(folder)
	fyne.Do(func() {
		a.headerList.ScrollToOffset(0) // ScrollToTop panics on an unrendered list
		a.headerList.UnselectAll()
		a.ma.ReadingPane.clear()
		a.searchEntry.ClearSilent()
		a.filterChip.SetSelected(map[string]string{}) // silent, the following update filters again
	})
	a.update(ctx)
}

func (a *mailsMessagePane) updateIsRead() {
	for i := range a.rows {
		a.rows[i].isRead = a.rows[i].isRead2
	}
}

// update refreshes the headers for the current folder.
func (a *mailsMessagePane) update(ctx context.Context) {
	folder := a.currentFolder.Load()
	reset := func() {
		fyne.Do(func() {
			a.clear()
			a.ma.ReadingPane.clear()
		})
	}
	setStatus := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.headerStatus.Text = s
			a.headerStatus.Importance = i
			a.headerStatus.Refresh()
			a.headerStatus.Show()
		})
	}
	if folder == nil {
		reset()
		return
	}
	hasData, err := a.ma.u.Character().HasSection(ctx, folder.CharacterID, app.SectionCharacterMailHeaders)
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "characterID", folder.CharacterID, "folder", folder.Name, "err", err)
		setStatus("Failed to load: "+a.ma.u.ErrorDisplay(err), widget.DangerImportance)
		reset()
		return
	}
	if !hasData {
		setStatus("Data not yet loaded", widget.WarningImportance)
		reset()
		return
	}

	rows, err := a.fetchRows(ctx, folder)
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "characterID", folder.CharacterID, "folder", folder.Name, "err", err)
		setStatus("Failed to load: "+a.ma.u.ErrorDisplay(err), widget.DangerImportance)
		reset()
		return
	}

	fyne.Do(func() {
		if a.rowsFolderUID == folder.UID() {
			snapshot := make(map[int64]bool, len(a.rows))
			for _, r := range a.rows {
				snapshot[r.id] = r.isRead
			}
			for i, r := range rows {
				if v, ok := snapshot[r.id]; ok {
					rows[i].isRead = v
				}
			}
		}
		a.headerStatus.Hide()
		a.topLabel.SetText(folder.Name)
		a.rows = rows
		a.rowsFolderUID = folder.UID()
		a.filterRowsAsync()
	})
}

func (a *mailsMessagePane) filterRowsAsync() {
	isLatest := a.filterRun.start()
	rows := slices.Clone(a.rows)
	totalRows := len(rows)
	filter := a.filterChip.Selected()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort("")
	runAsync(func() {
		// filter
		if x := filter[mailsFilterStatus]; x != "" {
			switch x {
			case mailsFilterStatusUnread:
				rows = slices.DeleteFunc(rows, func(r mailRow) bool {
					return r.isRead
				})
			case mailsFilterStatusRead:
				rows = slices.DeleteFunc(rows, func(r mailRow) bool {
					return !r.isRead
				})
			}
		}
		if x := filter[mailsFilterFrom]; x != "" {
			rows = slices.DeleteFunc(rows, func(r mailRow) bool {
				return r.from.NameOrZero() != x
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r mailRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		// sort
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// collect options
		fromOptions := xslices.Map(rows, func(r mailRow) string {
			return r.from.NameOrZero()
		})
		statusOptions := xslices.Map(rows, func(r mailRow) string {
			if r.isRead2 {
				return mailsFilterStatusRead
			}
			return mailsFilterStatusUnread
		})

		footer := fmt.Sprintf(
			"Showing %s / %s messages",
			ihumanize.Comma(len(rows)),
			ihumanize.Comma(totalRows),
		)
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.SetText(footer)
			a.filterChip.SetOptions(
				xwidget.NewFilterOptionMultiChoice(mailsFilterStatus, statusOptions),
				xwidget.NewFilterOptionMultiChoice(mailsFilterFrom, fromOptions),
			)
			a.rowsFiltered = rows
			a.headerList.Refresh()
			a.syncSelection()
		})
	})
}

// syncSelection keeps the displayed mail selected after the rows changed
// and clears it when it is no longer shown.
func (a *mailsMessagePane) syncSelection() {
	mailID := a.ma.ReadingPane.requested.mailID
	if mailID == 0 {
		return
	}
	idx := slices.IndexFunc(a.rowsFiltered, func(r mailRow) bool {
		return r.mailID == mailID
	})
	if idx == -1 {
		a.headerList.UnselectAll()
		a.ma.ReadingPane.clear()
		return
	}
	if a.OnSelected != nil {
		return // mobile does not keep a selection
	}
	a.reselecting = true
	a.headerList.Select(idx)
	a.reselecting = false
}

func (a *mailsMessagePane) fetchRows(ctx context.Context, f *mailFolderNode) ([]mailRow, error) {
	var hh []*app.CharacterMailHeader
	var err error
	switch f.Category {
	case nodeCategoryLabel:
		hh, err = a.ma.u.Character().ListMailHeadersForLabelOrdered(ctx, f.CharacterID, f.ObjID)
	case nodeCategoryList:
		hh, err = a.ma.u.Character().ListMailHeadersForListOrdered(ctx, f.CharacterID, f.ObjID)
	}
	if err != nil {
		return nil, err
	}
	rows := make([]mailRow, len(hh))
	for i, h := range hh {
		rows[i] = mailRow{
			characterID:  h.CharacterID,
			from:         h.From,
			id:           h.ID,
			isRead:       h.IsRead,
			isRead2:      h.IsRead,
			mailID:       h.MailID,
			searchTarget: strings.ToLower(h.Subject + "-" + h.From.NameOrZero()),
			subject:      h.Subject,
			timestamp:    h.Timestamp,
		}
	}
	return rows, nil
}

type mailsReadingPane struct {
	widget.BaseWidget

	body      *widget.Label
	header    *MailHeaderWidget
	ma        *Mails
	mail      *app.CharacterMail
	requested struct{ characterID, mailID int64 } // latest mail requested for display
	subject   *widget.Label
	toolbar   *widget.Toolbar
}

func newMailsReadingPane(ma *Mails) *mailsReadingPane {
	a := &mailsReadingPane{
		body:    widget.NewLabel(""),
		header:  NewMailHeaderWidget(ma.u.EVEImage().EveEntityLogoAsync, ma.u.InfoViewer().Show),
		ma:      ma,
		subject: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.subject.SizeName = theme.SizeNameSubHeadingText
	a.subject.Truncation = fyne.TextTruncateClip
	a.subject.Selectable = true
	a.body.Wrapping = fyne.TextWrapWord
	a.body.Selectable = true
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	return a
}

func (a *mailsReadingPane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(a.subject, a.header),
		nil,
		nil,
		nil,
		container.NewVScroll(a.body),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *mailsReadingPane) MakeDeleteAction(onSuccess func()) (fyne.Resource, func()) {
	return theme.DeleteIcon(), func() {
		// callback runs off the main thread, so it must not read a.mail
		m := a.mail
		if m == nil {
			return
		}
		subject := xstrings.TruncateWithSuffix(m.Subject.ValueOrFallback("?"), 50, 0)
		ui.ShowProgressConfirm(
			"Delete mail?",
			fmt.Sprintf(
				"You are about to permanently delete \"%s\" from %s. "+
					"This action cannot be undone and the mail cannot be recovered.",
				subject,
				m.From.Name,
			),
			"Delete",
			widget.DangerImportance,
			func() {
				ctx := context.Background()
				err := a.ma.u.Character().DeleteMail(ctx, m.CharacterID, m.MailID)
				if err != nil {
					slog.Error("Failed to delete mail",
						slog.Int64("characterID", m.CharacterID),
						slog.Int64("mailID", m.MailID),
						slog.Any("err", err),
					)
					a.ma.u.DisplaySnackbar(fmt.Sprintf("Failed to delete mail \"%s\": %s", subject, a.ma.u.ErrorDisplay(err)))
					return
				}
				a.ma.MessagePane.update(ctx)
				if onSuccess != nil {
					onSuccess()
				}
				a.ma.u.DisplaySnackbar(fmt.Sprintf("Mail \"%s\" deleted", subject))
			}, a.ma.u.MainWindow(),
		)
	}
}

func (a *mailsReadingPane) MakeForwardAction() (fyne.Resource, func()) {
	return theme.MailForwardIcon(), func() {
		if a.mail == nil {
			return
		}
		a.ma.showMailerWindow(mailer.Forward, a.mail)
	}
}

func (a *mailsReadingPane) MakeReplyAction() (fyne.Resource, func()) {
	return theme.MailReplyIcon(), func() {
		if a.mail == nil {
			return
		}
		a.ma.showMailerWindow(mailer.Reply, a.mail)
	}
}

func (a *mailsReadingPane) MakeReplyAllAction() (fyne.Resource, func()) {
	return theme.MailReplyAllIcon(), func() {
		if a.mail == nil {
			return
		}
		a.ma.showMailerWindow(mailer.ReplyAll, a.mail)
	}
}

func (a *mailsReadingPane) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(a.MakeReplyAction()),
		widget.NewToolbarAction(a.MakeReplyAllAction()),
		widget.NewToolbarAction(a.MakeForwardAction()),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.mail == nil {
				return
			}
			fyne.CurrentApp().Clipboard().SetContent(a.mail.String())
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(a.MakeDeleteAction(nil)),
	)
	return toolbar
}

// showMail displays a mail and discards results from earlier requests.
func (a *mailsReadingPane) showMail(mailID int64) {
	a.clear()
	characterID := a.ma.character.Load().IDOrZero()
	if characterID == 0 {
		return
	}
	a.requested.characterID, a.requested.mailID = characterID, mailID
	go a.loadMail(context.Background(), characterID, mailID)
}

func (a *mailsReadingPane) isRequested(characterID, mailID int64) bool {
	return a.requested.characterID == characterID && a.requested.mailID == mailID
}

func (a *mailsReadingPane) clear() {
	a.mail = nil
	a.requested.characterID, a.requested.mailID = 0, 0
	a.subject.SetText("")
	a.header.Clear()
	a.body.SetText("")
	a.toolbar.Hide()
}

func (a *mailsReadingPane) setMail(m *app.CharacterMail) {
	a.subject.SetText(m.Subject.ValueOrZero())
	a.setBody(m.BodyPlain())
	a.header.Set(m.From, m.Timestamp, m.Recipients...)
}

func (a *mailsReadingPane) setBody(s string) {
	var i widget.Importance
	if s == "" {
		i = widget.LowImportance
		s = "Loading..."
	}
	a.body.Importance = i
	a.body.Text = s
	a.body.Refresh()
}

func (a *mailsReadingPane) loadMail(ctx context.Context, characterID, mailID int64) {
	mail, err := a.ma.u.Character().GetMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		fyne.Do(func() {
			if !a.isRequested(characterID, mailID) {
				return
			}
			a.setBody("ERROR: Failed to load: " + a.ma.u.ErrorDisplay(err))
		})
		return
	}
	fyne.Do(func() {
		if !a.isRequested(characterID, mailID) {
			return
		}
		a.mail = mail
		a.setMail(mail)
		a.toolbar.Show()
	})

	if a.ma.u.IsOffline() || a.ma.u.IsUpdateDisabled() {
		return
	}

	// try to fetch mail body if missing
	if mail.Body.IsEmpty() {
		go func() {
			a.ma.sig.Do(fmt.Sprintf("charactermails-load-mail-%d-%d", characterID, mailID), func() (any, error) {
				body, err := a.ma.u.Character().UpdateMailBodyESI(ctx, characterID, mail.MailID)
				if err != nil {
					slog.Error("Failed to update mail body", "characterID", characterID, "mailID", mail.MailID, "error", err)
					fyne.Do(func() {
						if a.mail == nil || a.mail.CharacterID != characterID || a.mail.MailID != mailID {
							return
						}
						a.setBody("ERROR: Failed to load: " + a.ma.u.ErrorDisplay(err))
					})
					return nil, nil
				}
				fyne.Do(func() {
					if a.mail == nil || a.mail.CharacterID != characterID || a.mail.MailID != mailID {
						return
					}
					a.mail.Body.Set(body)
					a.setBody(a.mail.BodyPlain())
				})
				return nil, nil
			})
		}()
	}

	// try to update mail as read if unread
	if !mail.IsRead.ValueOrZero() {
		go func() {
			a.ma.sig.Do(fmt.Sprintf("charactermails-set-read-%d-%d", characterID, mailID), func() (any, error) {
				err := a.ma.u.Character().UpdateMailRead(ctx, characterID, mail.MailID, true)
				if err != nil {
					slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", mail.MailID, "error", err)
					a.ma.u.DisplaySnackbar("ERROR: Failed to mark mail as read: " + mail.Subject.ValueOrZero())
					return nil, nil
				}
				a.ma.NavigationPane.updateUnreadCounts(ctx)
				a.ma.MessagePane.update(ctx)
				go a.ma.u.Signals().CharacterChanged.Emit(ctx, characterID) // update character overview
				a.ma.u.UpdateMailIndicator(ctx)
				fyne.Do(func() {
					if a.mail == nil || a.mail.CharacterID != characterID || a.mail.MailID != mailID {
						return
					}
					a.mail.IsRead.Set(true)
				})
				return nil, nil
			})
		}()
	}
}
