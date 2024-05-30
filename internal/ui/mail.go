package ui

import (
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	islices "github.com/ErikKalkoken/evebuddy/internal/helper/slices"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type mailArea struct {
	content fyne.CanvasObject
	folder  *folderArea
	header  *headerArea
	detail  *mailDetailArea
	ui      *ui
}

func (u *ui) newMailArea() *mailArea {
	a := &mailArea{
		ui: u,
	}
	a.folder = newFolderArea(a)
	a.header = newHeaderArea(a)
	a.detail = NewDetailMailArea(a)

	split1 := container.NewHSplit(a.header.content, a.detail.content)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.folder.content, split1)
	split2.SetOffset(0.15)
	a.content = split2
	return a
}

func (a *mailArea) redraw() {
	a.folder.redraw()
}

func (a *mailArea) refresh() {
	a.folder.refresh()
}

// folderArea is the UI area showing the mail folders.
type folderArea struct {
	content       fyne.CanvasObject
	lastUID       string
	lastFolderAll folderNode
	mailArea      *mailArea
	newButton     *widget.Button
	errorText     *widget.Label
	tree          *widget.Tree
	treeData      binding.StringTree
}

func newFolderArea(m *mailArea) *folderArea {
	a := &folderArea{
		treeData:  binding.NewStringTree(),
		mailArea:  m,
		errorText: widget.NewLabel("ERROR"),
	}
	a.errorText.Importance = widget.DangerImportance
	a.errorText.Hide()

	a.tree = a.makeFolderTree()
	a.newButton = widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		a.mailArea.ui.showSendMessageWindow(createMessageNew, nil)
	})
	a.newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), container.NewPadded(a.newButton), layout.NewSpacer())

	a.content = container.NewBorder(top, a.errorText, nil, nil, a.tree)
	return a
}

type folderNodeCategory int

const (
	nodeCategoryBranch folderNodeCategory = 0
	nodeCategoryLabel  folderNodeCategory = 1
	nodeCategoryList   folderNodeCategory = 2
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

func (f folderNode) isSent() bool {
	return f.ID == folderNodeSentID
}

func (a *folderArea) makeFolderTree() *widget.Tree {
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
			t := "Failed to select folder"
			slog.Error(t, "err", err)
			a.mailArea.ui.statusBarArea.SetError(t)
			return
		}
		if node.isBranch() {
			tree.ToggleBranch(uid)
			tree.UnselectAll()
			return
		}
		a.lastUID = uid
		a.mailArea.header.setFolder(node)
	}
	return tree
}

func (a *folderArea) redraw() {
	a.lastUID = ""
	a.refresh()
}

func (a *folderArea) refresh() {
	a.errorText.Hide()
	characterID := a.mailArea.ui.currentCharID()
	folderAll, err := func() (folderNode, error) {
		ids, values, folderAll, err := a.buildFolderTree(characterID)
		if err != nil {
			return folderNode{}, err
		}
		if err := a.treeData.Set(ids, values); err != nil {
			return folderNode{}, err
		}
		return folderAll, nil
	}()
	if err != nil {
		slog.Error("Failed to build folder tree", "character", characterID, "error", err)
		a.errorText.Show()
		return
	}
	if a.lastUID == "" {
		a.tree.Select(folderNodeAllID)
		a.tree.ScrollToTop()
		a.mailArea.header.setFolder(folderAll)
	} else {
		a.mailArea.header.refresh()
	}
	a.lastFolderAll = folderAll
	a.updateMailTab(folderAll.UnreadCount)
}

func (a *folderArea) updateMailTab(unreadCount int) {
	s := "Mail"
	if unreadCount > 0 {
		s += fmt.Sprintf(" (%s)", humanize.Comma(int64(unreadCount)))
	}
	a.mailArea.ui.mailTab.Text = s
	a.mailArea.ui.tabs.Refresh()
}

func (a *folderArea) buildFolderTree(characterID int32) (map[string][]string, map[string]string, folderNode, error) {
	labelUnreadCounts, err := a.mailArea.ui.service.GetCharacterMailLabelUnreadCounts(characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	listUnreadCounts, err := a.mailArea.ui.service.GetCharacterMailListUnreadCounts(characterID)
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
		ObjID:       model.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	folders[folderNodeAllID], err = objectToJSON(folderAll)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	labels, err := a.mailArea.ui.service.ListCharacterMailLabelsOrdered(characterID)
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
	lists, err := a.mailArea.ui.service.ListCharacterMailLists(characterID)
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
		{folderNodeInboxID, model.MailLabelInbox, "Inbox"},
		{folderNodeSentID, model.MailLabelSent, "Sent"},
		{folderNodeCorpID, model.MailLabelCorp, "Corp"},
		{folderNodeAllianceID, model.MailLabelAlliance, "Alliance"},
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
		if id > model.MailLabelAlliance {
			labels += c
		}
	}
	for _, c := range listCounts {
		total += c
		lists += c
	}
	return total, labels, lists
}

// headerArea is the UI area showing the list of mail headers.
type headerArea struct {
	content       fyne.CanvasObject
	currentFolder folderNode
	top           *widget.Label
	list          *widget.List
	lastSelected  widget.ListItemID
	mailIDs       binding.IntList
	mailArea      *mailArea
}

func newHeaderArea(m *mailArea) *headerArea {
	a := headerArea{
		top:      widget.NewLabel(""),
		mailIDs:  binding.NewIntList(),
		mailArea: m,
	}

	a.list = a.makeHeaderTree()

	a.content = container.NewBorder(a.top, nil, nil, nil, a.list)
	return &a
}

func (a *headerArea) makeHeaderTree() *widget.List {
	foregroundColor := theme.ForegroundColor()
	subjectSize := theme.TextSize() * 1.15
	list := widget.NewListWithData(
		a.mailIDs,
		func() fyne.CanvasObject {
			from := canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor)
			timestamp := canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor)
			subject := canvas.NewText("subject", foregroundColor)
			subject.TextSize = subjectSize
			return container.NewPadded(container.NewPadded(container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp),
				subject,
			)))
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			parent := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
			subject := parent.Objects[1].(*canvas.Text)

			if !a.mailArea.ui.hasCharacter() {
				return
			}
			m, err := func() (*model.CharacterMail, error) {
				mailID, err := di.(binding.Int).Get()
				if err != nil {
					return nil, err
				}
				m, err := a.mailArea.ui.service.GetCharacterMail(a.mailArea.ui.currentCharID(), int32(mailID))
				if err != nil {
					return nil, err
				}
				return m, nil
			}()
			if err != nil {
				slog.Error("Failed to get mail", "error", err)
				subject.Text = "ERROR"
				subject.Color = theme.ErrorColor()
				subject.Refresh()
				return
			}

			fg := theme.ForegroundColor()
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*canvas.Text)
			var t string
			if a.currentFolder.isSent() {
				t = strings.Join(m.RecipientNames(), ", ")
			} else {
				t = m.From.Name
			}
			from.Text = t
			from.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			from.Color = fg
			from.Refresh()

			timestamp := top.Objects[2].(*canvas.Text)
			timestamp.Text = m.Timestamp.Format(myDateTime)
			timestamp.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			timestamp.Color = fg
			timestamp.Refresh()

			subject.Text = m.Subject
			subject.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			subject.Color = fg
			subject.Refresh()
		})
	list.OnSelected = func(id widget.ListItemID) {
		mailID, err := func() (int32, error) {
			di, err := a.mailIDs.GetItem(id)
			if err != nil {
				return 0, err
			}
			mailID, err := di.(binding.Int).Get()
			if err != nil {
				return 0, err
			}
			return int32(mailID), nil
		}()
		if err != nil {
			t := "Failed to select mail header"
			slog.Error(t, "err", err)
			a.mailArea.ui.statusBarArea.SetError(t)
			return
		}
		a.mailArea.detail.setMail(mailID)
		a.lastSelected = id
	}
	return list
}

func (a *headerArea) setFolder(folder folderNode) {
	a.currentFolder = folder
	a.refresh()
	a.list.ScrollToTop()
	a.list.UnselectAll()
	a.mailArea.detail.clear()
}

func (a *headerArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.updateMails(); err != nil {
			return "", 0, err
		}
		return a.makeTopText()
	}()
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *headerArea) updateMails() error {
	folder := a.currentFolder
	if folder.CharacterID == 0 {
		return nil
	}
	var mailIDs []int32
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		mailIDs, err = a.mailArea.ui.service.ListCharacterMailIDsForLabelOrdered(folder.CharacterID, folder.ObjID)
	case nodeCategoryList:
		mailIDs, err = a.mailArea.ui.service.ListCharacterMailIDsForListOrdered(folder.CharacterID, folder.ObjID)
	}
	if err != nil {
		return err
	}
	x := islices.ConvertNumeric[int32, int](mailIDs)
	if err := a.mailIDs.Set(x); err != nil {
		return err
	}
	if len(mailIDs) == 0 {
		a.mailArea.detail.clear()
	}
	return nil
}

func (a *headerArea) makeTopText() (string, widget.Importance, error) {
	if !a.mailArea.ui.hasCharacter() {
		return "No Character", widget.LowImportance, nil
	}
	hasData, err := a.mailArea.ui.service.CharacterSectionWasUpdated(
		a.mailArea.ui.currentCharID(), model.CharacterSectionSkillqueue)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	s := fmt.Sprintf("%d mails", a.mailIDs.Length())
	return s, widget.MediumImportance, nil

}

// mailDetailArea is the UI area showing the current mail.
type mailDetailArea struct {
	body     *widget.Label
	content  fyne.CanvasObject
	header   *widget.Label
	mail     *model.CharacterMail
	subject  *widget.Label
	toolbar  *widget.Toolbar
	mailArea *mailArea
}

func NewDetailMailArea(m *mailArea) *mailDetailArea {
	a := mailDetailArea{
		body:     widget.NewLabel(""),
		header:   widget.NewLabel(""),
		subject:  widget.NewLabel(""),
		mailArea: m,
	}

	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()

	a.subject.TextStyle = fyne.TextStyle{Bold: true}
	a.subject.Truncation = fyne.TextTruncateEllipsis
	a.header.Truncation = fyne.TextTruncateEllipsis
	wrapper := container.NewVBox(a.toolbar, a.subject, a.header)

	a.body.Wrapping = fyne.TextWrapWord

	a.content = container.NewBorder(wrapper, nil, nil, nil, container.NewVScroll(a.body))
	return &a
}

func (a *mailDetailArea) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MailReplyIcon(), func() {
			a.mailArea.ui.showSendMessageWindow(createMessageReply, a.mail)
		}),
		widget.NewToolbarAction(theme.MailReplyAllIcon(), func() {
			a.mailArea.ui.showSendMessageWindow(createMessageReplyAll, a.mail)
		}),
		widget.NewToolbarAction(theme.MailForwardIcon(), func() {
			a.mailArea.ui.showSendMessageWindow(createMessageForward, a.mail)
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			t := fmt.Sprintf("Are you sure you want to delete this mail?\n\n%s", a.mail.Subject)
			d := dialog.NewConfirm("Delete mail", t, func(confirmed bool) {
				if confirmed {
					if err := a.mailArea.ui.service.DeleteCharacterMail(a.mail.CharacterID, a.mail.MailID); err != nil {
						t := "Failed to delete mail"
						slog.Error(t, "characterID", a.mail.CharacterID, "mailID", a.mail.MailID, "err", err)
						a.mailArea.ui.showErrorDialog(t, err)
					} else {
						a.mailArea.header.refresh()
					}
				}
			}, a.mailArea.ui.window)
			d.Show()
		}),
	)
	return toolbar
}

func (a *mailDetailArea) clear() {
	a.updateContent("", "", "")
	a.toolbar.Hide()
}

func (a *mailDetailArea) setMail(mailID int32) {
	characterID := a.mailArea.ui.currentCharID()
	var err error
	a.mail, err = a.mailArea.ui.service.GetCharacterMail(characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.setErrorText()
		return
	}
	if !a.mail.IsRead {
		go func() {
			err = a.mailArea.ui.service.UpdateMailRead(characterID, a.mail.MailID)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
				a.setErrorText()
				return
			}
			a.mailArea.folder.refresh()
		}()
	}

	header := a.mail.MakeHeaderText(myDateTime)
	a.updateContent(a.mail.Subject, header, a.mail.BodyPlain())
	a.toolbar.Show()
}

func (a *mailDetailArea) updateContent(s string, h string, b string) {
	a.subject.SetText(s)
	a.header.SetText(h)
	a.body.SetText(b)
}

func (a *mailDetailArea) setErrorText() {
	a.clear()
	a.subject.Text = "ERROR"
	a.subject.Importance = widget.DangerImportance
	a.subject.Refresh()
}
