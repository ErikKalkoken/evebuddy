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
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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
	folderNodeUnread
)

// A FolderNode in the folder tree, e.g. the inbox
type FolderNode struct {
	Category    folderNodeCategory
	CharacterID int32
	IsLeaf      bool
	Type        folderNodeType
	Name        string
	ObjID       int32
	UnreadCount int
}

func (f FolderNode) IsEmpty() bool {
	return f.CharacterID == 0
}

func (f FolderNode) UID() widget.TreeNodeID {
	if f.CharacterID == 0 || f.Type == folderNodeUndefined {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", f.CharacterID, f.Type, f.ObjID)
}

func (f FolderNode) isBranch() bool {
	return f.Category == nodeCategoryBranch
}

func (f FolderNode) icon() fyne.Resource {
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

// MailArea is the UI area showing the mail folders.
type MailArea struct {
	Content       fyne.CanvasObject
	CurrentFolder optional.Optional[FolderNode]
	Detail        fyne.CanvasObject
	Headers       fyne.CanvasObject
	OnSelected    func()
	OnRefresh     func(count int)
	OnSendMessage func(character *app.Character, mode SendMailMode, mail *app.CharacterMail)

	body          *widget.Label
	folderData    *fynetree.FyneTree[FolderNode]
	folderSection fyne.CanvasObject
	folderTree    *widget.Tree
	header        *widget.Label
	headerList    *widget.List
	headers       []*app.CharacterMailHeader
	headerTop     *widget.Label
	folderDefault FolderNode
	lastSelected  widget.ListItemID
	lastUID       string
	mail          *app.CharacterMail
	subject       *iwidget.Label
	toolbar       *widget.Toolbar
	u             *BaseUI
}

func NewMailArea(u *BaseUI) *MailArea {
	a := &MailArea{
		body:       widget.NewLabel(""),
		folderData: fynetree.New[FolderNode](),
		header:     widget.NewLabel(""),
		headers:    make([]*app.CharacterMailHeader, 0),
		headerTop:  widget.NewLabel(""),
		subject:    iwidget.NewLabelWithSize("", theme.SizeNameSubHeadingText),
		u:          u,
	}

	// Mail
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	a.subject.Truncation = fyne.TextTruncateClip
	a.header.Truncation = fyne.TextTruncateClip
	a.body.Wrapping = fyne.TextWrapWord
	a.Detail = container.NewBorder(container.NewVBox(a.subject, a.header), nil, nil, nil, container.NewVScroll(a.body))
	detailWithToolbar := container.NewBorder(a.toolbar, nil, nil, nil, a.Detail)

	// Headers
	a.headerList = a.makeHeaderList()
	a.Headers = container.NewBorder(a.headerTop, nil, nil, nil, a.headerList)

	// Folders
	a.folderTree = a.makeFolderTree()
	r, f := a.MakeComposeMessageAction()
	newButton := widget.NewButtonWithIcon("Compose", r, f)
	newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), container.NewPadded(newButton), layout.NewSpacer())

	a.folderSection = container.NewBorder(top, nil, nil, nil, a.folderTree)

	// Combine sections
	split1 := container.NewHSplit(a.Headers, detailWithToolbar)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.folderSection, split1)
	split2.SetOffset(0.15)
	a.Content = split2
	return a
}

func (a *MailArea) makeFolderTree() *widget.Tree {
	tree := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.folderData.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.folderData.IsBranch(uid)
		},
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(icon.BlankSvg),
				widget.NewLabel("template"),
				layout.NewSpacer(),
				kwidget.NewBadge("999"),
			)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			node, ok := a.folderData.Value(uid)
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
		node, ok := a.folderData.Value((uid))
		if !ok {
			tree.UnselectAll()
			return
		}
		if node.isBranch() {
			tree.UnselectAll()
			return
		}
		a.lastUID = uid
		a.SetFolder(node)
	}
	return tree
}

func (a *MailArea) Redraw() {
	a.lastUID = ""
	a.Refresh()
}

func (a *MailArea) Folders() []FolderNode {
	return a.folderData.Flat()
}

func (a *MailArea) Refresh() {
	characterID := a.u.CharacterID()
	folderAll, err := a.updateFolderData(characterID)
	if err != nil {
		t := "Failed to build folder tree"
		slog.Error(t, "character", characterID, "error", err)
		d := NewErrorDialog(t, err, a.u.Window)
		d.Show()
		return
	}
	if a.OnRefresh != nil {
		a.OnRefresh(folderAll.UnreadCount)
	}
	a.folderTree.Refresh()
	if folderAll.IsEmpty() {
		a.clearFolder()
		return
	}
	if a.lastUID == "" {
		a.folderTree.UnselectAll()
		a.folderTree.ScrollToTop()
		a.folderTree.Select(folderAll.UID())
		a.SetFolder(folderAll)
	} else {
		a.headerRefresh()
	}
	a.folderDefault = folderAll
}

func (a *MailArea) updateFolderData(characterID int32) (FolderNode, error) {
	tree := fynetree.New[FolderNode]()
	if characterID == 0 {
		a.folderData = tree
		return FolderNode{}, nil
	}
	ctx := context.TODO()
	labelUnreadCounts, err := a.u.CharacterService.GetCharacterMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		a.folderData = tree
		return FolderNode{}, err
	}
	listUnreadCounts, err := a.u.CharacterService.GetCharacterMailListUnreadCounts(ctx, characterID)
	if err != nil {
		a.folderData = tree
		return FolderNode{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)

	// Add unread folder
	folderUnread := FolderNode{
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
		n := FolderNode{
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
		return FolderNode{}, err
	}
	if len(labels) > 0 {
		n := FolderNode{
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
			n := FolderNode{
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
		return FolderNode{}, err
	}
	if len(lists) > 0 {
		n := FolderNode{
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
			n := FolderNode{
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
	folderAll := FolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All",
		ObjID:       app.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	tree.MustAdd("", folderAll.UID(), folderAll)

	a.folderData = tree
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

func (a *MailArea) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.headers)
		},
		func() fyne.CanvasObject {
			return appwidget.NewMailHeaderItem(a.u.EveImageService, app.TimeDefaultFormat)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.headers) {
				return
			}
			m := a.headers[id]
			if !a.u.HasCharacter() {
				return
			}
			item := co.(*appwidget.MailHeaderItem)
			item.Set(m.From, m.Subject, m.Timestamp, m.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.headers) {
			return
		}
		r := a.headers[id]
		a.setMail(r.MailID)
		a.lastSelected = id
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *MailArea) ResetFolders() {
	a.SetFolder(a.folderDefault)
}

func (a *MailArea) SetFolder(folder FolderNode) {
	a.CurrentFolder = optional.New(folder)
	a.headerRefresh()
	a.headerList.ScrollToTop()
	a.headerList.UnselectAll()
	a.clearMail()
}

func (a *MailArea) clearFolder() {
	a.CurrentFolder = optional.Optional[FolderNode]{}
	a.headers = make([]*app.CharacterMailHeader, 0)
	a.headerList.Refresh()
	a.headerTop.SetText("")
	a.clearMail()
}

func (a *MailArea) headerRefresh() {
	var t string
	var i widget.Importance
	f, err := a.updateHeaders()
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeFolderTopText(f)
	}
	a.headerTop.Text = t
	a.headerTop.Importance = i
	a.headerTop.Refresh()
}

func (a *MailArea) updateHeaders() (FolderNode, error) {
	ctx := context.TODO()
	folderOption := a.CurrentFolder
	if folderOption.IsEmpty() {
		return FolderNode{}, nil
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
		return FolderNode{}, err
	}
	a.headers = headers
	a.headerList.Refresh()
	if len(headers) == 0 {
		a.clearMail()
	}
	return folder, nil
}

func (a *MailArea) makeFolderTopText(f FolderNode) (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No Character", widget.LowImportance
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	p := message.NewPrinter(language.English)
	s := p.Sprintf("%s • %d mails", f.Name, len(a.headers))
	return s, widget.MediumImportance
}

func (a *MailArea) onSendMessage(mode SendMailMode, mail *app.CharacterMail) {
	if a.OnSendMessage == nil {
		return
	}
	character := a.u.CurrentCharacter()
	if character == nil {
		return
	}
	a.OnSendMessage(character, mode, mail)
}

func (a *MailArea) MakeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.onSendMessage(SendMailNew, nil)
	}
}

func (a *MailArea) MakeDeleteAction(onSuccess func()) (fyne.Resource, func()) {
	return theme.DeleteIcon(), func() {
		d := NewConfirmDialog(
			"Delete mail",
			fmt.Sprintf("Are you sure you want to permanently delete this mail?\n\n%s", a.mail.Header()),
			"Delete",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				m := kxmodal.NewProgressInfinite(
					"Deleting mail...",
					"",
					func() error {
						return a.u.CharacterService.DeleteCharacterMail(context.TODO(), a.mail.CharacterID, a.mail.MailID)
					},
					a.u.Window,
				)
				m.OnSuccess = func() {
					a.headerRefresh()
					if onSuccess != nil {
						onSuccess()
					}
					a.u.Snackbar.Show(fmt.Sprintf("Mail \"%s\" deleted", a.mail.Subject))
				}
				m.OnError = func(err error) {
					slog.Error("Failed to delete mail", "characterID", a.mail.CharacterID, "mailID", a.mail.MailID, "err", err)
					a.u.Snackbar.Show(fmt.Sprintf("Failed to delete mail: %s", humanize.Error(err)))
				}
				m.Start()
			}, a.u.Window)
		d.Show()
	}
}

func (a *MailArea) MakeForwardAction() (fyne.Resource, func()) {
	return theme.MailForwardIcon(), func() {
		a.onSendMessage(SendMailForward, a.mail)
	}
}

func (a *MailArea) MakeReplyAction() (fyne.Resource, func()) {
	return theme.MailReplyIcon(), func() {
		a.onSendMessage(SendMailReply, a.mail)
	}
}

func (a *MailArea) MakeReplyAllAction() (fyne.Resource, func()) {
	return theme.MailReplyAllIcon(), func() {
		a.onSendMessage(SendMailReplyAll, a.mail)
	}
}

func (a *MailArea) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(a.MakeReplyAction()),
		widget.NewToolbarAction(a.MakeReplyAllAction()),
		widget.NewToolbarAction(a.MakeForwardAction()),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			a.u.Window.Clipboard().SetContent(a.mail.String())
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(a.MakeDeleteAction(nil)),
	)
	return toolbar
}

func (a *MailArea) clearMail() {
	a.setMailContent("", "", "")
	a.toolbar.Hide()
}

func (a *MailArea) setMail(mailID int32) {
	ctx := context.TODO()
	characterID := a.u.CharacterID()
	var err error
	a.mail, err = a.u.CharacterService.GetCharacterMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.u.Snackbar.Show("ERROR: Failed to fetch mail")
		return
	}
	if !a.u.IsOffline && !a.mail.IsRead {
		go func() {
			err = a.u.CharacterService.UpdateMailRead(ctx, characterID, a.mail.MailID)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
				a.u.Snackbar.Show("ERROR: Failed to mark mail as read")
				return
			}
			a.Refresh()
			a.u.OverviewArea.Refresh()
			a.u.UpdateMailIndicator()
		}()
	}
	a.setMailContent(a.mail.Subject, a.mail.Header(), a.mail.BodyPlain())
	a.toolbar.Show()
}

func (a *MailArea) setMailContent(s string, h string, b string) {
	a.subject.SetText(s)
	a.header.SetText(h)
	a.body.SetText(b)
}
