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

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
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

func (u *ui) NewFolderArea() *folderArea {
	f := folderArea{ui: u}
	f.tree, f.treeData = u.makeFolderTree()
	f.refreshButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		go f.UpdateMails(true)
	})
	f.newButton = widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		f.ui.ShowCreateMessageWindow(CreateMessageNew, nil)
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
		u.headerArea.DrawFolder(item)
	}
	return tree, treeData
}

func (f *folderArea) Refresh() {
	characterID := f.ui.CurrentCharID()
	if characterID == 0 {
		f.refreshButton.Disable()
		f.newButton.Disable()
	} else {
		f.refreshButton.Enable()
		f.newButton.Enable()
	}
	ids, values, folderAll, err := f.buildFolderTree(characterID)
	if err != nil {
		slog.Error("Failed to build folder tree", "character", characterID, "error", err)
	}
	f.treeData.Set(ids, values)
	if f.lastUID == "" || f.lastFolderAll != folderAll {
		f.tree.Select(nodeAllID)
		f.tree.ScrollToTop()
		f.ui.headerArea.DrawFolder(folderAll)
	} else {
		f.ui.headerArea.Refresh()
	}
	f.lastFolderAll = folderAll
}

func (f *folderArea) buildFolderTree(characterID int32) (map[string][]string, map[string]string, node, error) {
	labelUnreadCounts, err := f.ui.service.GetMailLabelUnreadCounts(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	listUnreadCounts, err := f.ui.service.GetMailListUnreadCounts(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)
	ids := map[string][]string{
		"": {nodeAllID, nodeInboxID, nodeSentID, nodeCorpID, nodeAllianceID},
	}
	folders := makeDefaultFolders(characterID, labelUnreadCounts)
	folderAll := node{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		ID:          nodeAllID,
		Name:        "All Mails",
		ObjID:       model.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	folders[nodeAllID] = folderAll.toJSON()
	labels, err := f.ui.service.ListMailLabelsOrdered(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	if len(labels) > 0 {
		ids[""] = append(ids[""], nodeLabelsID)
		ids[nodeLabelsID] = []string{}
		folders[nodeLabelsID] = node{
			CharacterID: characterID,
			ID:          nodeLabelsID,
			Name:        "Labels",
			UnreadCount: totalLabelsUnreadCount,
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
	lists, err := f.ui.service.ListMailLists(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	if len(lists) > 0 {
		ids[""] = append(ids[""], nodeListsID)
		ids[nodeListsID] = []string{}
		folders[nodeListsID] = node{
			CharacterID: characterID,
			ID:          nodeListsID,
			Name:        "Mailing Lists",
			UnreadCount: totalListUnreadCount,
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

func (f *folderArea) UpdateMails(respondToUser bool) {
	character := f.ui.CurrentChar()
	if character == nil {
		return
	}
	status := f.ui.statusArea
	if respondToUser {
		status.Info.SetWithProgress(fmt.Sprintf("Checking mail for %s", character.Name))
	}
	unreadCount, err := f.ui.service.UpdateMail(character.ID)
	if err != nil {
		status.Info.SetError("Failed to fetch mail")
		slog.Error("Failed to fetch mails", "characterID", character.ID, "error", err)
		return
	}
	if unreadCount > 0 {
		status.Info.Set(fmt.Sprintf("%s has %d new mail", character.Name, unreadCount))
	} else if respondToUser {
		status.Info.Set(fmt.Sprintf("No new mail for %s", character.Name))
	} else {
		status.Info.Clear()
	}
	f.Refresh()
}

func (f *folderArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := f.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				if !f.ui.service.SectionUpdatedExpired(characterID, service.UpdateSectionMail) {
					return
				}
				f.UpdateMails(false)
			}()
			<-ticker.C
		}
	}()
}
