package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

const folderUpdateTicker = 10 * time.Second

type mailArea struct {
	content fyne.CanvasObject
	folder  *folderArea
	header  *headerArea
	detail  *mailDetailArea
	ui      *ui
}

func (u *ui) NewMailArea() *mailArea {
	a := &mailArea{
		ui: u,
	}
	a.folder = NewFolderArea(a)
	a.header = NewHeaderArea(a)
	a.detail = NewDetailMailArea(a)

	split1 := container.NewHSplit(a.header.content, a.detail.content)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.folder.content, split1)
	split2.SetOffset(0.15)
	a.content = split2
	return a
}

func (a *mailArea) Redraw() {
	a.folder.Redraw()
}

func (a *mailArea) Refresh() {
	a.folder.Refresh()
}

func (a *mailArea) StartUpdateTicker() {
	ticker := time.NewTicker(folderUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *mailArea) MaybeUpdateAndRefresh(characterID int32) {
	changed1, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionMailLabels)
	if err != nil {
		slog.Error("Failed to update mail labels", "character", characterID, "err", err)
		return
	}
	changed2, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionMailLists)
	if err != nil {
		slog.Error("Failed to update mail lists", "character", characterID, "err", err)
		return
	}
	changed3, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionMails)
	if err != nil {
		slog.Error("Failed to update mail", "character", characterID, "err", err)
		return
	}
	if (changed1 || changed2 || changed3) && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}

// folderArea is the UI area showing the mail folders.
type folderArea struct {
	content       fyne.CanvasObject
	newButton     *widget.Button
	lastUID       string
	lastFolderAll folderNode
	tree          *widget.Tree
	treeData      binding.StringTree
	mailArea      *mailArea
}

func NewFolderArea(m *mailArea) *folderArea {
	a := &folderArea{
		treeData: binding.NewStringTree(),
		mailArea: m,
	}

	a.tree = a.makeFolderTree()
	a.newButton = widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		a.mailArea.ui.ShowSendMessageWindow(CreateMessageNew, nil)
	})
	a.newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), container.NewPadded(a.newButton), layout.NewSpacer())

	a.content = container.NewBorder(top, nil, nil, nil, a.tree)
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

func newFolderTreeNodeFromJSON(s string) folderNode {
	var f folderNode
	err := json.Unmarshal([]byte(s), &f)
	if err != nil {
		panic(err)
	}
	return f
}

func (f folderNode) toJSON() string {
	s, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	return string(s)
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
			s, err := di.(binding.String).Get()
			if err != nil {
				slog.Error("Failed to fetch data item for tree")
				return
			}
			item := newFolderTreeNodeFromJSON(s)
			icon := co.(*fyne.Container).Objects[0].(*widget.Icon)
			icon.SetResource(item.icon())
			var text string
			if item.UnreadCount == 0 {
				text = item.Name
			} else {
				text = fmt.Sprintf("%s (%d)", item.Name, item.UnreadCount)
			}
			label := co.(*fyne.Container).Objects[1].(*widget.Label)
			label.SetText(text)
		},
	)
	tree.OnSelected = func(uid string) {
		di, err := a.treeData.GetItem(uid)
		if err != nil {
			slog.Error("Failed to get tree data item", "error", err)
			return
		}
		s, err := di.(binding.String).Get()
		if err != nil {
			slog.Error("Failed to fetch data item for tree")
			return
		}
		item := newFolderTreeNodeFromJSON(s)
		if item.isBranch() {
			if a.lastUID != "" {
				tree.Select(a.lastUID)
			}
			return
		}
		a.lastUID = uid
		a.mailArea.header.SetFolder(item)
	}
	return tree
}

func (a *folderArea) Redraw() {
	a.lastUID = ""
	a.Refresh()
}

func (a *folderArea) Refresh() {
	characterID := a.mailArea.ui.CurrentCharID()
	ids, values, folderAll, err := a.buildFolderTree(characterID)
	if err != nil {
		slog.Error("Failed to build folder tree", "character", characterID, "error", err)
	}
	if err := a.treeData.Set(ids, values); err != nil {
		panic(err)
	}
	if a.lastUID == "" {
		a.tree.Select(folderNodeAllID)
		a.tree.ScrollToTop()
		a.mailArea.header.SetFolder(folderAll)
	} else {
		a.mailArea.header.Refresh()
	}
	a.lastFolderAll = folderAll
	s := "Mail"
	if folderAll.UnreadCount > 0 {
		s += fmt.Sprintf(" (%s)", humanize.Comma(int64(folderAll.UnreadCount)))
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
	folders := makeDefaultFolders(characterID, labelUnreadCounts)
	folderAll := folderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		ID:          folderNodeAllID,
		Name:        "All Mails",
		ObjID:       model.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	folders[folderNodeAllID] = folderAll.toJSON()
	labels, err := a.mailArea.ui.service.ListCharacterMailLabelsOrdered(characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	if len(labels) > 0 {
		ids[""] = append(ids[""], folderNodeLabelsID)
		ids[folderNodeLabelsID] = []string{}
		folders[folderNodeLabelsID] = folderNode{
			CharacterID: characterID,
			ID:          folderNodeLabelsID,
			Name:        "Labels",
			UnreadCount: totalLabelsUnreadCount,
		}.toJSON()
		for _, l := range labels {
			uid := fmt.Sprintf("label%d", l.LabelID)
			ids[folderNodeLabelsID] = append(ids[folderNodeLabelsID], uid)
			u, ok := labelUnreadCounts[l.LabelID]
			if !ok {
				u = 0
			}
			n := folderNode{ObjID: l.LabelID, Name: l.Name, Category: nodeCategoryLabel, UnreadCount: u}
			folders[uid] = n.toJSON()
		}
	}
	lists, err := a.mailArea.ui.service.ListCharacterMailLists(characterID)
	if err != nil {
		return nil, nil, folderNode{}, err
	}
	if len(lists) > 0 {
		ids[""] = append(ids[""], folderNodeListsID)
		ids[folderNodeListsID] = []string{}
		folders[folderNodeListsID] = folderNode{
			CharacterID: characterID,
			ID:          folderNodeListsID,
			Name:        "Mailing Lists",
			UnreadCount: totalListUnreadCount,
		}.toJSON()
		for _, l := range lists {
			uid := fmt.Sprintf("list%d", l.ID)
			ids[folderNodeListsID] = append(ids[folderNodeListsID], uid)
			u, ok := listUnreadCounts[l.ID]
			if !ok {
				u = 0
			}
			n := folderNode{ObjID: l.ID, Name: l.Name, Category: nodeCategoryList, UnreadCount: u}
			folders[uid] = n.toJSON()
		}
	}
	return ids, folders, folderAll, nil
}

func makeDefaultFolders(characterID int32, labelUnreadCounts map[int32]int) map[string]string {
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
		folders[o.nodeID] = folderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			ID:          o.nodeID,
			Name:        o.name,
			ObjID:       o.labelID,
			UnreadCount: u,
		}.toJSON()
	}
	return folders
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
	infoText      *widget.Label
	list          *widget.List
	lastSelected  widget.ListItemID
	mailIDs       binding.IntList
	mailArea      *mailArea
}

func NewHeaderArea(m *mailArea) *headerArea {
	a := headerArea{
		infoText: widget.NewLabel(""),
		mailIDs:  binding.NewIntList(),
		mailArea: m,
	}

	a.list = a.makeHeaderTree()

	a.content = container.NewBorder(a.infoText, nil, nil, nil, a.list)
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
			characterID := a.mailArea.ui.CurrentCharID()
			if characterID == 0 {
				return
			}
			mailID, err := di.(binding.Int).Get()
			if err != nil {
				panic(err)
			}
			m, err := a.mailArea.ui.service.GetCharacterMail(characterID, int32(mailID))
			if err != nil {
				if !errors.Is(err, storage.ErrNotFound) {
					slog.Error("Failed to get mail", "error", err)
				}
				return
			}
			parent := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			fg := theme.ForegroundColor()

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

			subject := parent.Objects[1].(*canvas.Text)
			subject.Text = m.Subject
			subject.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			subject.Color = fg
			subject.Refresh()
		})
	list.OnSelected = func(id widget.ListItemID) {
		di, err := a.mailIDs.GetItem(id)
		if err != nil {
			panic((err))
		}
		mailID, err := di.(binding.Int).Get()
		if err != nil {
			panic(err)
		}
		a.mailArea.detail.SetMail(int32(mailID), id)
		a.lastSelected = id
	}
	return list
}

func (a *headerArea) SetFolder(folder folderNode) {
	a.currentFolder = folder
	a.Refresh()
	a.list.ScrollToTop()
	a.list.UnselectAll()
	a.mailArea.detail.Clear()
}

func (a *headerArea) Refresh() {
	a.updateMails()
	a.list.Refresh()
}

func (a *headerArea) updateMails() {
	folder := a.currentFolder
	if folder.CharacterID == 0 {
		return
	}
	var mailIDs []int32
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		mailIDs, err = a.mailArea.ui.service.ListCharacterMailIDsForLabelOrdered(folder.CharacterID, folder.ObjID)
	case nodeCategoryList:
		mailIDs, err = a.mailArea.ui.service.ListCharacterMailIDsForListOrdered(folder.CharacterID, folder.ObjID)
	}
	x := islices.ConvertNumeric[int32, int](mailIDs)
	if err := a.mailIDs.Set(x); err != nil {
		panic(err)
	}
	var s string
	var i widget.Importance
	if err != nil {
		slog.Error("Failed to fetch mail", "characterID", folder.CharacterID, "error", err)
		s = "ERROR"
		i = widget.DangerImportance
	}
	s, i = a.makeTopText()
	a.infoText.Text = s
	a.infoText.Importance = i
	a.infoText.Refresh()

	if len(mailIDs) == 0 {
		a.mailArea.detail.Clear()
		return
	}
}

func (a *headerArea) makeTopText() (string, widget.Importance) {
	hasData, err := a.mailArea.ui.service.CharacterSectionWasUpdated(a.mailArea.ui.CurrentCharID(), model.CharacterSectionSkillqueue)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data yet...", widget.LowImportance
	}
	s := fmt.Sprintf("%d mails", a.mailIDs.Length())
	return s, widget.MediumImportance

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
			a.mailArea.ui.ShowSendMessageWindow(CreateMessageReply, a.mail)
		}),
		widget.NewToolbarAction(theme.MailReplyAllIcon(), func() {
			a.mailArea.ui.ShowSendMessageWindow(CreateMessageReplyAll, a.mail)
		}),
		widget.NewToolbarAction(theme.MailForwardIcon(), func() {
			a.mailArea.ui.ShowSendMessageWindow(CreateMessageForward, a.mail)
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			t := fmt.Sprintf("Are you sure you want to delete this mail?\n\n%s", a.mail.Subject)
			d := dialog.NewConfirm("Delete mail", t, func(confirmed bool) {
				if confirmed {
					err := a.mailArea.ui.service.DeleteCharacterMail(a.mail.CharacterID, a.mail.MailID)
					if err != nil {
						errorDialog := dialog.NewError(err, a.mailArea.ui.window)
						errorDialog.Show()
					} else {
						a.mailArea.header.Refresh()
					}
				}
			}, a.mailArea.ui.window)
			d.Show()
		}),
	)
	return toolbar
}

func (a *mailDetailArea) Clear() {
	a.updateContent("", "", "")
	a.toolbar.Hide()
}

func (a *mailDetailArea) SetMail(mailID int32, listItemID widget.ListItemID) {
	characterID := a.mailArea.ui.CurrentCharID()
	var err error
	a.mail, err = a.mailArea.ui.service.GetCharacterMail(characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		return
	}
	if !a.mail.IsRead {
		go func() {
			err := func() error {
				err = a.mailArea.ui.service.UpdateMailRead(characterID, a.mail.MailID)
				if err != nil {
					return err
				}
				a.mailArea.folder.Refresh()
				return nil
			}()
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
			}
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
