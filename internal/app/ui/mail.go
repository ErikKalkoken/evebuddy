package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// mailArea is the UI area showing the mail folders.
type mailArea struct {
	content fyne.CanvasObject
	ui      *ui

	folderSection fyne.CanvasObject
	lastUID       string
	lastFolderAll folderNode
	foldersWidget *widget.Tree
	foldersData   *fynetree.FyneTree[folderNode]
	currentFolder optional.Optional[folderNode]

	headerSection fyne.CanvasObject
	headerTop     *widget.Label
	headers       *widget.List
	lastSelected  widget.ListItemID
	headerData    binding.UntypedList

	body        *widget.Label
	mailSection fyne.CanvasObject
	header      *widget.Label
	mail        *app.CharacterMail
	subject     *widget.Label
	toolbar     *widget.Toolbar
}

func (u *ui) newMailArea() *mailArea {
	a := &mailArea{
		foldersData: fynetree.New[folderNode](),
		ui:          u,
		headerTop:   widget.NewLabel(""),
		headerData:  binding.NewUntypedList(),
		body:        widget.NewLabel(""),
		header:      widget.NewLabel(""),
		subject:     widget.NewLabel(""),
	}

	// Mail
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	a.subject.TextStyle = fyne.TextStyle{Bold: true}
	a.subject.Truncation = fyne.TextTruncateEllipsis
	a.header.Truncation = fyne.TextTruncateEllipsis
	wrapper := container.NewVBox(a.toolbar, a.subject, a.header)
	a.body.Wrapping = fyne.TextWrapWord
	a.mailSection = container.NewBorder(wrapper, nil, nil, nil, container.NewVScroll(a.body))

	// Headers
	a.headers = a.makeHeaderList()
	a.headerSection = container.NewBorder(a.headerTop, nil, nil, nil, a.headers)

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
	nodeCategoryBranch folderNodeCategory = iota
	nodeCategoryLabel
	nodeCategoryList
)

type folderNodeType string

const (
	folderNodeAll      folderNodeType = "all"
	folderNodeInbox    folderNodeType = "inbox"
	folderNodeSent     folderNodeType = "sent"
	folderNodeCorp     folderNodeType = "corp"
	folderNodeAlliance folderNodeType = "alliance"
	folderNodeTrash    folderNodeType = "trash"
	folderNodeLabel    folderNodeType = "label"
	folderNodeList     folderNodeType = "list"
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
	if f.CharacterID == 0 || f.Type == "" {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%s-%d", f.CharacterID, f.Type, f.ObjID)
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
				widget.NewIcon(&fyne.StaticResource{}), widget.NewLabel("Branch template"))
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			label := co.(*fyne.Container).Objects[1].(*widget.Label)
			node, err := a.foldersData.Value(uid)
			if err != nil {
				return
			}
			icon := co.(*fyne.Container).Objects[0].(*widget.Icon)
			icon.SetResource(node.icon())
			var text string
			if node.UnreadCount == 0 {
				text = node.Name
			} else {
				text = fmt.Sprintf("%s (%d)", node.Name, node.UnreadCount)
			}
			label.SetText(text)
		},
	)
	tree.OnSelected = func(uid string) {
		node, err := a.foldersData.Value((uid))
		if err != nil {
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
	characterID := a.ui.characterID()
	folderAll, err := a.updateFolderData(characterID)
	if err != nil {
		t := "Failed to build folder tree"
		slog.Error(t, "character", characterID, "error", err)
		a.ui.showErrorDialog(t, err)
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
	s := "Mail"
	if unreadCount > 0 {
		s += fmt.Sprintf(" (%s)", humanize.Comma(int64(unreadCount)))
	}
	a.ui.mailTab.Text = s
	a.ui.tabs.Refresh()
}

func (a *mailArea) updateFolderData(characterID int32) (folderNode, error) {
	tree := fynetree.New[folderNode]()
	if characterID == 0 {
		a.foldersData = tree
		return folderNode{}, nil
	}
	ctx := context.TODO()
	labelUnreadCounts, err := a.ui.CharacterService.GetCharacterMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		a.foldersData = tree
		return folderNode{}, err
	}
	listUnreadCounts, err := a.ui.CharacterService.GetCharacterMailListUnreadCounts(ctx, characterID)
	if err != nil {
		a.foldersData = tree
		return folderNode{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)

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
	labels, err := a.ui.CharacterService.ListCharacterMailLabelsOrdered(ctx, characterID)
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
	lists, err := a.ui.CharacterService.ListCharacterMailLists(ctx, characterID)
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
	l := widget.NewListWithData(
		a.headerData,
		func() fyne.CanvasObject {
			return widgets.NewMailHeaderItem(myDateTime)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			if !a.ui.hasCharacter() {
				return
			}
			item := co.(*widgets.MailHeaderItem)
			m, err := convertDataItem[*app.CharacterMailHeader](di)
			if err != nil {
				slog.Error("Failed to get mail", "error", err)
				item.SetError("Failed to get mail")
				return
			}
			item.Set(m.From, m.Subject, m.Timestamp, m.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		r, err := getItemUntypedList[*app.CharacterMailHeader](a.headerData, id)
		if err != nil {
			slog.Error("Failed to select mail header", "err", err)
			l.UnselectAll()
			return
		}
		a.setMail(r.MailID)
		a.lastSelected = id
	}
	return l
}

func (a *mailArea) setFolder(folder folderNode) {
	a.currentFolder = optional.New(folder)
	a.headerRefresh()
	a.headers.ScrollToTop()
	a.headers.UnselectAll()
	a.clearMail()
}
func (a *mailArea) clearFolder() {
	a.currentFolder = optional.Optional[folderNode]{}
	a.headerData.Set(nil)
	a.headerTop.SetText("")
	a.clearMail()
}

func (a *mailArea) headerRefresh() {
	var t string
	var i widget.Importance
	if err := a.updateHeaderData(); err != nil {
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

func (a *mailArea) updateHeaderData() error {
	ctx := context.TODO()
	folderOption := a.currentFolder
	if folderOption.IsEmpty() {
		return nil
	}
	folder := folderOption.ValueOrZero()
	var oo []*app.CharacterMailHeader
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		oo, err = a.ui.CharacterService.ListCharacterMailHeadersForLabelOrdered(
			ctx, folder.CharacterID, folder.ObjID)
	case nodeCategoryList:
		oo, err = a.ui.CharacterService.ListCharacterMailHeadersForListOrdered(
			ctx, folder.CharacterID, folder.ObjID)
	}
	if err != nil {
		return err
	}
	if err := a.headerData.Set(copyToUntypedSlice(oo)); err != nil {
		return err
	}
	if len(oo) == 0 {
		a.clearMail()
	}
	return nil
}

func (a *mailArea) makeFolderTopText() (string, widget.Importance) {
	if !a.ui.hasCharacter() {
		return "No Character", widget.LowImportance
	}
	hasData := a.ui.StatusCacheService.CharacterSectionExists(a.ui.characterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	s := fmt.Sprintf("%d mails", a.headerData.Length())
	return s, widget.MediumImportance
}

func (a *mailArea) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MailReplyIcon(), func() {
			a.ui.showSendMessageWindow(createMessageReply, a.mail)
		}),
		widget.NewToolbarAction(theme.MailReplyAllIcon(), func() {
			a.ui.showSendMessageWindow(createMessageReplyAll, a.mail)
		}),
		widget.NewToolbarAction(theme.MailForwardIcon(), func() {
			a.ui.showSendMessageWindow(createMessageForward, a.mail)
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			t := fmt.Sprintf("Are you sure you want to delete this mail?\n\n%s", a.mail.Subject)
			d := dialog.NewConfirm("Delete mail", t, func(confirmed bool) {
				if confirmed {
					if err := a.ui.CharacterService.DeleteCharacterMail(context.TODO(), a.mail.CharacterID, a.mail.MailID); err != nil {
						t := "Failed to delete mail"
						slog.Error(t, "characterID", a.mail.CharacterID, "mailID", a.mail.MailID, "err", err)
						a.ui.showErrorDialog(t, err)
					} else {
						a.headerRefresh()
					}
				}
			}, a.ui.window)
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
	characterID := a.ui.characterID()
	var err error
	a.mail, err = a.ui.CharacterService.GetCharacterMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.setErrorText()
		return
	}
	if !a.mail.IsRead {
		go func() {
			err = a.ui.CharacterService.UpdateMailRead(ctx, characterID, a.mail.MailID)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
				a.setErrorText()
				return
			}
			a.refresh()
		}()
	}

	header := a.mail.MakeHeaderText(myDateTime)
	a.updateContent(a.mail.Subject, header, a.mail.BodyPlain())
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
