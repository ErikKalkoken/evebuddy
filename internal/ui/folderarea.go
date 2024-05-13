package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
)

// folderArea is the UI area showing the mail folders.
type folderArea struct {
	content       fyne.CanvasObject
	newButton     *widget.Button
	refreshButton *widget.Button
	lastUID       string
	lastFolderAll node
	tree          *widget.Tree
	treeData      binding.StringTree
	ui            *ui
}

// TODO: Replace date driven tree with direct refresh

func (u *ui) NewFolderArea() *folderArea {
	f := folderArea{ui: u}
	f.tree, f.treeData = u.makeFolderTree()
	f.refreshButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		go f.UpdateMails(true)
	})
	f.newButton = widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		f.ui.ShowSendMessageWindow(CreateMessageNew, nil)
	})
	f.newButton.Importance = widget.HighImportance
	top := container.NewHBox(f.refreshButton, f.newButton)
	c := container.NewBorder(top, nil, nil, nil, f.tree)
	f.content = c
	return &f
}

func (u *ui) makeFolderTree() (*widget.Tree, binding.StringTree) {
	treeData := binding.NewStringTree()
	tree := widget.NewTreeWithData(
		treeData,
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(&fyne.StaticResource{}), widget.NewLabel("Branch template"))
		},
		func(di binding.DataItem, isBranch bool, co fyne.CanvasObject) {
			i := di.(binding.String)
			s, err := i.Get()
			if err != nil {
				slog.Error("Failed to fetch data item for tree")
				return
			}
			item := newNodeFromJSON(s)
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
		di, err := treeData.GetItem(uid)
		if err != nil {
			slog.Error("Failed to get char ID item", "error", err)
			return
		}
		i := di.(binding.String)
		s, err := i.Get()
		if err != nil {
			slog.Error("Failed to fetch data item for tree")
			return
		}
		item := newNodeFromJSON(s)
		if item.isBranch() {
			if u.folderArea.lastUID != "" {
				tree.Select(u.folderArea.lastUID)
			}
			return
		}
		u.folderArea.lastUID = uid
		u.headerArea.SetFolder(item)
	}
	return tree, treeData
}

func (a *folderArea) Refresh() {
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		a.refreshButton.Disable()
		a.newButton.Disable()
	} else {
		a.refreshButton.Enable()
		a.newButton.Enable()
	}
	ids, values, folderAll, err := a.buildFolderTree(characterID)
	if err != nil {
		slog.Error("Failed to build folder tree", "character", characterID, "error", err)
	}
	a.treeData.Set(ids, values)
	if a.lastUID == "" || a.lastFolderAll != folderAll {
		a.tree.Select(nodeAllID)
		a.tree.ScrollToTop()
		a.ui.headerArea.SetFolder(folderAll)
	} else {
		a.ui.headerArea.Refresh()
	}
	a.lastFolderAll = folderAll
	var s string
	if folderAll.UnreadCount > 0 {
		s = fmt.Sprintf("Mail (%d)", folderAll.UnreadCount)
	} else {
		s = "Mail"
	}
	a.ui.mailTab.Text = s
	a.ui.tabs.Refresh()
}

func (a *folderArea) buildFolderTree(characterID int32) (map[string][]string, map[string]string, node, error) {
	labelUnreadCounts, err := a.ui.service.GetMailLabelUnreadCounts(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	listUnreadCounts, err := a.ui.service.GetMailListUnreadCounts(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)
	ids := map[string][]string{
		"": {nodeAllID, nodeInboxID, nodeSentID, nodeCorpID, nodeAllianceID},
	}
	folders := makeDefaultFolders(characterID, labelUnreadCounts)
	folderAll := node{
		Category:      nodeCategoryLabel,
		MyCharacterID: characterID,
		ID:            nodeAllID,
		Name:          "All Mails",
		ObjID:         model.MailLabelAll,
		UnreadCount:   totalUnreadCount,
	}
	folders[nodeAllID] = folderAll.toJSON()
	labels, err := a.ui.service.ListMailLabelsOrdered(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	if len(labels) > 0 {
		ids[""] = append(ids[""], nodeLabelsID)
		ids[nodeLabelsID] = []string{}
		folders[nodeLabelsID] = node{
			MyCharacterID: characterID,
			ID:            nodeLabelsID,
			Name:          "Labels",
			UnreadCount:   totalLabelsUnreadCount,
		}.toJSON()
		for _, l := range labels {
			uid := fmt.Sprintf("label%d", l.LabelID)
			ids[nodeLabelsID] = append(ids[nodeLabelsID], uid)
			u, ok := labelUnreadCounts[l.LabelID]
			if !ok {
				u = 0
			}
			n := node{ObjID: l.LabelID, Name: l.Name, Category: nodeCategoryLabel, UnreadCount: u}
			folders[uid] = n.toJSON()
		}
	}
	lists, err := a.ui.service.ListMailLists(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	if len(lists) > 0 {
		ids[""] = append(ids[""], nodeListsID)
		ids[nodeListsID] = []string{}
		folders[nodeListsID] = node{
			MyCharacterID: characterID,
			ID:            nodeListsID,
			Name:          "Mailing Lists",
			UnreadCount:   totalListUnreadCount,
		}.toJSON()
		for _, l := range lists {
			uid := fmt.Sprintf("list%d", l.ID)
			ids[nodeListsID] = append(ids[nodeListsID], uid)
			u, ok := listUnreadCounts[l.ID]
			if !ok {
				u = 0
			}
			n := node{ObjID: l.ID, Name: l.Name, Category: nodeCategoryList, UnreadCount: u}
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
		{nodeInboxID, model.MailLabelInbox, "Inbox"},
		{nodeSentID, model.MailLabelSent, "Sent"},
		{nodeCorpID, model.MailLabelCorp, "Corp"},
		{nodeAllianceID, model.MailLabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		u, ok := labelUnreadCounts[o.labelID]
		if !ok {
			u = 0
		}
		folders[o.nodeID] = node{
			MyCharacterID: characterID,
			Category:      nodeCategoryLabel,
			ID:            o.nodeID,
			Name:          o.name,
			ObjID:         o.labelID,
			UnreadCount:   u,
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

func (a *folderArea) UpdateMails(respondToUser bool) {
	character := a.ui.CurrentChar()
	if character == nil {
		return
	}
	status := a.ui.statusArea
	if respondToUser {
		status.SetInfoWithProgress(fmt.Sprintf("Checking mail for %s", character.Character.Name))
	}
	unreadCount, err := a.ui.service.UpdateMail(character.ID)
	if err != nil {
		status.SetError("Failed to fetch mail")
		slog.Error("Failed to fetch mails", "characterID", character.ID, "error", err)
		return
	}
	if respondToUser {
		if unreadCount > 0 {
			status.SetInfo(fmt.Sprintf("%s has %d new mail", character.Character.Name, unreadCount))
		} else if respondToUser {
			status.SetInfo(fmt.Sprintf("No new mail for %s", character.Character.Name))
		} else {
			status.ClearInfo()
		}
	}
	a.Refresh()
}

func (a *folderArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := a.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				isExpired, err := a.ui.service.SectionIsUpdateExpired(characterID, service.UpdateSectionMail)
				if err != nil {
					slog.Error(err.Error())
					return
				}
				if !isExpired {
					return
				}
				a.UpdateMails(false)
			}()
			<-ticker.C
		}
	}()
}
