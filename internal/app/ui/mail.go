package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// mailArea is the UI area showing the mail folders.
type mailArea struct {
	content fyne.CanvasObject
	u       *UI

	currentFolder optional.Optional[folderNode]
	foldersData   *fynetree.FyneTree[folderNode]
	folderSection fyne.CanvasObject
	foldersWidget *widget.Tree
	lastFolderAll folderNode
	lastUID       string

	headerList    *widget.List
	headers       []*app.CharacterMailHeader
	headerSection fyne.CanvasObject
	headerTop     *widget.Label
	lastSelected  widget.ListItemID

	body        *widget.Label
	header      *widget.Label
	mail        *app.CharacterMail
	mailSection fyne.CanvasObject
	subject     *widget.Label
	toolbar     *widget.Toolbar
}

func (u *UI) newMailArea() *mailArea {
	a := &mailArea{
		body:        widget.NewLabel(""),
		foldersData: fynetree.New[folderNode](),
		header:      widget.NewLabel(""),
		headers:     make([]*app.CharacterMailHeader, 0),
		headerTop:   widget.NewLabel(""),
		subject:     widget.NewLabel(""),
		u:           u,
	}

	// Mail
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	a.subject.TextStyle = fyne.TextStyle{Bold: true}
	a.subject.Truncation = fyne.TextTruncateClip
	a.header.Truncation = fyne.TextTruncateClip
	wrapper := container.NewVBox(a.toolbar, a.subject, a.header)
	a.body.Wrapping = fyne.TextWrapWord
	a.mailSection = container.NewBorder(wrapper, nil, nil, nil, container.NewVScroll(a.body))

	// Headers
	a.headerList = a.makeHeaderList()
	a.headerSection = container.NewBorder(a.headerTop, nil, nil, nil, a.headerList)

	// Folders
	a.foldersWidget = a.makeFolderTree()
	newButton := widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		u.showSendMessageWindow(createMessageNew, nil)
	})
	newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), container.NewPadded(newButton), layout.NewSpacer())

	a.folderSection = container.NewBorder(top, nil, nil, nil, a.foldersWidget)

	// Combine sections
	split1 := container.NewHSplit(a.headerSection, a.mailSection)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.folderSection, split1)
	split2.SetOffset(0.15)
	a.content = split2
	return a
}

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
	folderNodeUnread
)

// A folderNode in the folder tree, e.g. the inbox
type folderNode struct {
	Category    folderNodeCategory
	CharacterID int32
	IsLeaf      bool
	Type        folderNodeType
	Name        string
	ObjID       int32
	UnreadCount int
}

func (f folderNode) IsEmpty() bool {
	return f.CharacterID == 0
}

func (f folderNode) UID() widget.TreeNodeID {
	if f.CharacterID == 0 || f.Type == folderNodeUndefined {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", f.CharacterID, f.Type, f.ObjID)
}

func (f folderNode) isBranch() bool {
	return f.Category == nodeCategoryBranch
}

func (f folderNode) icon() fyne.Resource {
	switch f.Type {
	case folderNodeInbox:
		return theme.DownloadIcon()
	case folderNodeSent:
		return theme.UploadIcon()
	case folderNodeTrash:
		return theme.DeleteIcon()
	}
	return theme.FolderIcon()
}

func (a *mailArea) makeFolderTree() *widget.Tree {
	tree := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.foldersData.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.foldersData.IsBranch(uid)
		},
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(resourceBlankSvg),
				widget.NewLabel("template"),
				layout.NewSpacer(),
				kwidget.NewBadge("999"),
			)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			node, ok := a.foldersData.Value(uid)
			if !ok {
				return
			}
			hbox := co.(*fyne.Container).Objects
			icon := hbox[0].(*widget.Icon)
			icon.SetResource(node.icon())
			label := hbox[1].(*widget.Label)
			badge := hbox[3].(*kwidget.Badge)
			if node.UnreadCount == 0 {
				label.TextStyle.Bold = false
				badge.Hide()
			} else {
				label.TextStyle.Bold = true
				badge.SetText(strconv.Itoa(node.UnreadCount))
				badge.Show()
			}
			label.Text = node.Name
			label.Refresh()
		},
	)
	tree.OnSelected = func(uid string) {
		node, ok := a.foldersData.Value((uid))
		if !ok {
			tree.UnselectAll()
			return
		}
		if node.isBranch() {
			tree.UnselectAll()
			return
		}
		a.lastUID = uid
		a.setFolder(node)
	}
	return tree
}

func (a *mailArea) redraw() {
	a.lastUID = ""
	a.refresh()
}

func (a *mailArea) refresh() {
	characterID := a.u.characterID()
	folderAll, err := a.updateFolderData(characterID)
	if err != nil {
		t := "Failed to build folder tree"
		slog.Error(t, "character", characterID, "error", err)
		d := NewErrorDialog(t, err, a.u.window)
		d.Show()
		return
	}
	a.foldersWidget.Refresh()
	if folderAll.IsEmpty() {
		a.updateMailTab(0)
		a.clearFolder()
		return
	}
	if a.lastUID == "" {
		a.foldersWidget.UnselectAll()
		a.foldersWidget.ScrollToTop()
		a.foldersWidget.Select(folderAll.UID())
		a.setFolder(folderAll)
	} else {
		a.headerRefresh()
	}
	a.lastFolderAll = folderAll
	a.updateMailTab(folderAll.UnreadCount)
}

func (a *mailArea) updateMailTab(unreadCount int) {
	s := "Comm."
	if unreadCount > 0 {
		s += fmt.Sprintf(" (%s)", humanize.Comma(int64(unreadCount)))
	}
	a.u.mailTab.Text = s
	a.u.tabs.Refresh()
}

func (a *mailArea) updateFolderData(characterID int32) (folderNode, error) {
	tree := fynetree.New[folderNode]()
	if characterID == 0 {
		a.foldersData = tree
		return folderNode{}, nil
	}
	ctx := context.TODO()
	labelUnreadCounts, err := a.u.CharacterService.GetCharacterMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		a.foldersData = tree
		return folderNode{}, err
	}
	listUnreadCounts, err := a.u.CharacterService.GetCharacterMailListUnreadCounts(ctx, characterID)
	if err != nil {
		a.foldersData = tree
		return folderNode{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)

	// Add unread folder
	folderUnread := folderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeUnread,
		Name:        "Unread",
		ObjID:       app.MailLabelUnread,
		UnreadCount: totalUnreadCount,
	}
	tree.MustAdd("", folderUnread.UID(), folderUnread)

	// Add default folders
	defaultFolders := []struct {
		nodeType folderNodeType
		labelID  int32
		name     string
	}{
		{folderNodeInbox, app.MailLabelInbox, "Inbox"},
		{folderNodeSent, app.MailLabelSent, "Sent"},
		{folderNodeCorp, app.MailLabelCorp, "Corp"},
		{folderNodeAlliance, app.MailLabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		u, ok := labelUnreadCounts[o.labelID]
		if !ok {
			u = 0
		}
		n := folderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			Type:        o.nodeType,
			Name:        o.name,
			ObjID:       o.labelID,
			UnreadCount: u,
		}
		tree.MustAdd("", n.UID(), n)
	}

	// Add custom labels
	labels, err := a.u.CharacterService.ListCharacterMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return folderNode{}, err
	}
	if len(labels) > 0 {
		n := folderNode{
			CharacterID: characterID,
			Type:        folderNodeLabel,
			Name:        "Labels",
			UnreadCount: totalLabelsUnreadCount,
		}
		uid := tree.MustAdd("", n.UID(), n)
		for _, l := range labels {
			u, ok := labelUnreadCounts[l.LabelID]
			if !ok {
				u = 0
			}
			n := folderNode{
				Category:    nodeCategoryLabel,
				CharacterID: characterID,
				Name:        l.Name,
				ObjID:       l.LabelID,
				UnreadCount: u,
				Type:        folderNodeLabel,
			}
			tree.MustAdd(uid, n.UID(), n)
		}
	}

	// Add mailing lists
	lists, err := a.u.CharacterService.ListCharacterMailLists(ctx, characterID)
	if err != nil {
		return folderNode{}, err
	}
	if len(lists) > 0 {
		n := folderNode{
			CharacterID: characterID,
			Type:        folderNodeList,
			Name:        "Mailing Lists",
			UnreadCount: totalListUnreadCount,
		}
		uid := tree.MustAdd("", n.UID(), n)
		for _, l := range lists {
			u, ok := listUnreadCounts[l.ID]
			if !ok {
				u = 0
			}
			n := folderNode{
				Category:    nodeCategoryList,
				CharacterID: characterID,
				ObjID:       l.ID,
				Name:        l.Name,
				UnreadCount: u,
				Type:        folderNodeList,
			}
			tree.MustAdd(uid, n.UID(), n)
		}
	}
	// Add all folder
	folderAll := folderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All Mails",
		ObjID:       app.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	tree.MustAdd("", folderAll.UID(), folderAll)

	a.foldersData = tree
	return folderAll, nil
}

func calcUnreadTotals(labelCounts, listCounts map[int32]int) (int, int, int) {
	var total, labels, lists int
	for id, c := range labelCounts {
		total += c
		if id > app.MailLabelAlliance {
			labels += c
		}
	}
	for _, c := range listCounts {
		total += c
		lists += c
	}
	return total, labels, lists
}

func (a *mailArea) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.headers)
		},
		func() fyne.CanvasObject {
			return widgets.NewMailHeaderItem(app.TimeDefaultFormat)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.headers) {
				return
			}
			m := a.headers[id]
			if !a.u.hasCharacter() {
				return
			}
			item := co.(*widgets.MailHeaderItem)
			item.Set(m.From, m.Subject, m.Timestamp, m.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.headers) {
			return
		}
		r := a.headers[id]
		a.setMail(r.MailID)
		a.lastSelected = id
	}
	return l
}

func (a *mailArea) setFolder(folder folderNode) {
	a.currentFolder = optional.New(folder)
	a.headerRefresh()
	a.headerList.ScrollToTop()
	a.headerList.UnselectAll()
	a.clearMail()
}
func (a *mailArea) clearFolder() {
	a.currentFolder = optional.Optional[folderNode]{}
	a.headers = make([]*app.CharacterMailHeader, 0)
	a.headerList.Refresh()
	a.headerTop.SetText("")
	a.clearMail()
}

func (a *mailArea) headerRefresh() {
	var t string
	var i widget.Importance
	if err := a.updateHeaders(); err != nil {
		slog.Error("Failed to refresh mail headers UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeFolderTopText()
	}
	a.headerTop.Text = t
	a.headerTop.Importance = i
	a.headerTop.Refresh()
}

func (a *mailArea) updateHeaders() error {
	ctx := context.TODO()
	folderOption := a.currentFolder
	if folderOption.IsEmpty() {
		return nil
	}
	folder := folderOption.ValueOrZero()
	var headers []*app.CharacterMailHeader
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		headers, err = a.u.CharacterService.ListCharacterMailHeadersForLabelOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	case nodeCategoryList:
		headers, err = a.u.CharacterService.ListCharacterMailHeadersForListOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	}
	if err != nil {
		return err
	}
	a.headers = headers
	a.headerList.Refresh()
	if len(headers) == 0 {
		a.clearMail()
	}
	return nil
}

func (a *mailArea) makeFolderTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No Character", widget.LowImportance
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.characterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	p := message.NewPrinter(language.English)
	s := p.Sprintf("%d mails", len(a.headers))
	return s, widget.MediumImportance
}

func (a *mailArea) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MailReplyIcon(), func() {
			a.u.showSendMessageWindow(createMessageReply, a.mail)
		}),
		widget.NewToolbarAction(theme.MailReplyAllIcon(), func() {
			a.u.showSendMessageWindow(createMessageReplyAll, a.mail)
		}),
		widget.NewToolbarAction(theme.MailForwardIcon(), func() {
			a.u.showSendMessageWindow(createMessageForward, a.mail)
		}),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			a.u.window.Clipboard().SetContent(a.mail.String())
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			t := fmt.Sprintf("Are you sure you want to delete this mail?\n\n%s", a.mail.Header())
			d := NewConfirmDialog("Delete mail", t, "Delete", func(confirmed bool) {
				if confirmed {
					if err := a.u.CharacterService.DeleteCharacterMail(context.TODO(), a.mail.CharacterID, a.mail.MailID); err != nil {
						t := "Failed to delete mail"
						slog.Error(t, "characterID", a.mail.CharacterID, "mailID", a.mail.MailID, "err", err)
						d2 := NewErrorDialog(t, err, a.u.window)
						d2.Show()
					} else {
						a.headerRefresh()
					}
				}
			}, a.u.window)
			d.Show()
		}),
	)
	return toolbar
}

func (a *mailArea) clearMail() {
	a.updateContent("", "", "")
	a.toolbar.Hide()
}

func (a *mailArea) setMail(mailID int32) {
	ctx := context.TODO()
	characterID := a.u.characterID()
	var err error
	a.mail, err = a.u.CharacterService.GetCharacterMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.setErrorText()
		return
	}
	if !a.u.IsOffline && !a.mail.IsRead {
		go func() {
			err = a.u.CharacterService.UpdateMailRead(ctx, characterID, a.mail.MailID)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
				a.setErrorText()
				return
			}
			a.refresh()
			a.u.overviewArea.refresh()
		}()
	}

	a.updateContent(a.mail.Subject, a.mail.Header(), a.mail.BodyPlain())
	a.toolbar.Show()
}

func (a *mailArea) updateContent(s string, h string, b string) {
	a.subject.SetText(s)
	a.header.SetText(h)
	a.body.SetText(b)
}

func (a *mailArea) setErrorText() {
	a.clearMail()
	a.subject.Text = "ERROR"
	a.subject.Importance = widget.DangerImportance
	a.subject.Refresh()
}
