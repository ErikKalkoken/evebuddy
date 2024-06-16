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
)

// mailArea is the UI area showing the mail folders.
type mailArea struct {
	content fyne.CanvasObject
	ui      *ui

	folderSection fyne.CanvasObject
	lastUID       string
	lastFolderAll folderNode
	folders       *widget.Tree
	treeData      binding.StringTree
	currentFolder folderNode

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
		treeData:   binding.NewStringTree(),
		ui:         u,
		headerTop:  widget.NewLabel(""),
		headerData: binding.NewUntypedList(),
		body:       widget.NewLabel(""),
		header:     widget.NewLabel(""),
		subject:    widget.NewLabel(""),
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
	a.folders = a.makeFolderTree()
	newButton := widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		u.showSendMessageWindow(createMessageNew, nil)
	})
	newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), container.NewPadded(newButton), layout.NewSpacer())

	a.folderSection = container.NewBorder(top, nil, nil, nil, a.folders)

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

const (
	folderNodeAllID      = "all"
	folderNodeInboxID    = "inbox"
	folderNodeSentID     = "sent"
	folderNodeCorpID     = "corp"
	folderNodeAllianceID = "alliance"
	folderNodeTrashID    = "trash"
	folderNodeLabelsID   = "labels"
	folderNodeListsID    = "lists"
)

// A folderNode in the folder tree, e.g. the inbox
type folderNode struct {
	Category    folderNodeCategory
	CharacterID int32
	ID          string
	Name        string
	ObjID       int32
	UnreadCount int
}

func (f folderNode) isBranch() bool {
	return f.Category == nodeCategoryBranch
}

func (f folderNode) icon() fyne.Resource {
	switch f.ID {
	case folderNodeInboxID:
		return theme.DownloadIcon()
	case folderNodeSentID:
		return theme.UploadIcon()
	case folderNodeTrashID:
		return theme.DeleteIcon()
	}
	return theme.FolderIcon()
}

func (a *mailArea) makeFolderTree() *widget.Tree {
	tree := widget.NewTreeWithData(
		a.treeData,
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(&fyne.StaticResource{}), widget.NewLabel("Branch template"))
		},
		func(di binding.DataItem, isBranch bool, co fyne.CanvasObject) {
			label := co.(*fyne.Container).Objects[1].(*widget.Label)
			item, err := treeNodeFromDataItem[folderNode](di)
			if err != nil {
				slog.Error("Failed to fetch data item for tree", "err", err)
				label.SetText("ERROR")
				return
			}
			icon := co.(*fyne.Container).Objects[0].(*widget.Icon)
			icon.SetResource(item.icon())
			var text string
			if item.UnreadCount == 0 {
				text = item.Name
			} else {
				text = fmt.Sprintf("%s (%d)", item.Name, item.UnreadCount)
			}
			label.SetText(text)
		},
	)
	tree.OnSelected = func(uid string) {
		node, err := treeNodeFromBoundTree[folderNode](a.treeData, uid)
		if err != nil {
			slog.Error("Failed to select folder", "err", err)
			tree.UnselectAll()
			return
		}
		if node.isBranch() {
			tree.ToggleBranch(uid)
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
	folderAll, err := func() (folderNode, error) {
		ids, values, folderAll, err := a.makeFolderTreeData(characterID)
		if err != nil {
			return folderNode{}, err
		}
		if err := a.treeData.Set(ids, values); err != nil {
			return folderNode{}, err
		}
		return folderAll, nil
	}()
	if err != nil {
		t := "Failed to build folder tree"
		slog.Error(t, "character", characterID, "error", err)
		a.ui.showErrorDialog(t, err)
		return
	}
	if a.lastUID == "" {
		a.folders.Select(folderNodeAllID)
		a.folders.ScrollToTop()
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

func (a *mailArea) makeFolderTreeData(characterID int32) (map[string][]string, map[string]string, folderNode, error) {
	ctx := context.Background()
	labelUnreadCounts, err := a.ui.CharacterService.GetCharacterMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	listUnreadCounts, err := a.ui.CharacterService.GetCharacterMailListUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)
	ids := map[string][]string{
		"": {folderNodeAllID, folderNodeInboxID, folderNodeSentID, folderNodeCorpID, folderNodeAllianceID},
	}
	folders, err := makeDefaultFolders(characterID, labelUnreadCounts)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	folderAll := folderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		ID:          folderNodeAllID,
		Name:        "All Mails",
		ObjID:       app.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	folders[folderNodeAllID], err = objectToJSON(folderAll)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	labels, err := a.ui.CharacterService.ListCharacterMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	if len(labels) > 0 {
		ids[""] = append(ids[""], folderNodeLabelsID)
		ids[folderNodeLabelsID] = []string{}
		n := folderNode{
			CharacterID: characterID,
			ID:          folderNodeLabelsID,
			Name:        "Labels",
			UnreadCount: totalLabelsUnreadCount,
		}
		x, err := objectToJSON(n)
		if err != nil {
			return nil, nil, folderNode{}, err
		}
		folders[folderNodeLabelsID] = x
		for _, l := range labels {
			uid := fmt.Sprintf("label%d", l.LabelID)
			ids[folderNodeLabelsID] = append(ids[folderNodeLabelsID], uid)
			u, ok := labelUnreadCounts[l.LabelID]
			if !ok {
				u = 0
			}
			n := folderNode{ObjID: l.LabelID, Name: l.Name, Category: nodeCategoryLabel, UnreadCount: u}
			x, err := objectToJSON(n)
			if err != nil {
				return nil, nil, folderNode{}, err
			}
			folders[uid] = x
		}
	}
	lists, err := a.ui.CharacterService.ListCharacterMailLists(ctx, characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	if len(lists) > 0 {
		ids[""] = append(ids[""], folderNodeListsID)
		ids[folderNodeListsID] = []string{}
		n := folderNode{
			CharacterID: characterID,
			ID:          folderNodeListsID,
			Name:        "Mailing Lists",
			UnreadCount: totalListUnreadCount,
		}
		x, err := objectToJSON(n)
		if err != nil {
			return nil, nil, folderNode{}, err
		}
		folders[folderNodeListsID] = x
		for _, l := range lists {
			uid := fmt.Sprintf("list%d", l.ID)
			ids[folderNodeListsID] = append(ids[folderNodeListsID], uid)
			u, ok := listUnreadCounts[l.ID]
			if !ok {
				u = 0
			}
			n := folderNode{ObjID: l.ID, Name: l.Name, Category: nodeCategoryList, UnreadCount: u}
			x, err := objectToJSON(n)
			if err != nil {
				return nil, nil, folderNode{}, err
			}
			folders[uid] = x
		}
	}
	return ids, folders, folderAll, nil
}

func makeDefaultFolders(characterID int32, labelUnreadCounts map[int32]int) (map[string]string, error) {
	folders := make(map[string]string)
	defaultFolders := []struct {
		nodeID  string
		labelID int32
		name    string
	}{
		{folderNodeInboxID, app.MailLabelInbox, "Inbox"},
		{folderNodeSentID, app.MailLabelSent, "Sent"},
		{folderNodeCorpID, app.MailLabelCorp, "Corp"},
		{folderNodeAllianceID, app.MailLabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		u, ok := labelUnreadCounts[o.labelID]
		if !ok {
			u = 0
		}
		n := folderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			ID:          o.nodeID,
			Name:        o.name,
			ObjID:       o.labelID,
			UnreadCount: u,
		}
		x, err := objectToJSON(n)
		if err != nil {
			return nil, err
		}
		folders[o.nodeID] = x
	}
	return folders, nil
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
	a.currentFolder = folder
	a.headerRefresh()
	a.headers.ScrollToTop()
	a.headers.UnselectAll()
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
	ctx := context.Background()
	folder := a.currentFolder
	if folder.CharacterID == 0 {
		return nil
	}
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
					ctx := context.Background()
					if err := a.ui.CharacterService.DeleteCharacterMail(ctx, a.mail.CharacterID, a.mail.MailID); err != nil {
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
	ctx := context.Background()
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
