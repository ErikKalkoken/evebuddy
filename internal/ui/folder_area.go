package ui

import (
	"example/esiapp/internal/logic"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// folderArea is the UI area showing the mail folders.
type folderArea struct {
	content       fyne.CanvasObject
	newButton     *widget.Button
	refreshButton *widget.Button
	tree          *widget.Tree
	treeData      binding.StringTree
	ui            *ui
}

func (u *ui) NewFolderArea() *folderArea {
	f := folderArea{ui: u}
	f.tree, f.treeData = makeFolderTree(u)
	f.refreshButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		f.UpdateMails()
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

func makeFolderTree(u *ui) (*widget.Tree, binding.StringTree) {
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
	lastUID := ""
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
			if lastUID != "" {
				tree.Select(lastUID)
			}
			return
		}
		lastUID = uid
		u.headerArea.Redraw(item)
	}
	return tree, treeData
}

func (f *folderArea) Redraw() {
	characterID := f.ui.CurrentCharID()
	if characterID == 0 {
		f.refreshButton.Disable()
		f.newButton.Disable()
	} else {
		f.refreshButton.Enable()
		f.newButton.Enable()
	}
	ids, values, folderItemAll, err := buildFolderTree(characterID)
	if err != nil {
		slog.Error("Failed to build folder tree", "character", characterID, "error", err)
	}
	f.treeData.Set(ids, values)
	f.tree.Select(nodeAllID)
	f.tree.ScrollToTop()
	f.ui.headerArea.Redraw(folderItemAll)
}

func buildFolderTree(characterID int32) (map[string][]string, map[string]string, node, error) {
	var totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount int
	labelUnreadCounts, err := model.FetchMailLabelUnreadCounts(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	listUnreadCounts, err := model.FetchMailListUnreadCounts(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	ids := map[string][]string{
		"":           {nodeAllID, nodeInboxID, nodeSentID, nodeCorpID, nodeAllianceID, nodeLabelsID, nodeListsID},
		nodeLabelsID: {},
		nodeListsID:  {},
	}
	folders := make(map[string]string)
	defaultFolders := []struct {
		nodeID  string
		labelID int
		name    string
	}{
		{nodeInboxID, model.LabelInbox, "Inbox"},
		{nodeSentID, model.LabelSent, "Sent"},
		{nodeCorpID, model.LabelCorp, "Corp"},
		{nodeAllianceID, model.LabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		u, ok := labelUnreadCounts[o.labelID]
		if !ok {
			u = 0
		}
		totalUnreadCount += u
		if o.labelID > model.LabelAlliance {
			totalLabelsUnreadCount += u
		}
		folders[o.nodeID] = node{
			ID:          o.nodeID,
			ObjID:       int32(o.labelID),
			Name:        o.name,
			Category:    nodeCategoryLabel,
			UnreadCount: u,
		}.toJSON()
	}
	for _, c := range listUnreadCounts {
		totalListUnreadCount += c
	}
	folderItemAll := node{
		ID:          nodeAllID,
		ObjID:       model.LabelAll,
		Name:        "All Mails",
		Category:    nodeCategoryLabel,
		UnreadCount: totalUnreadCount,
	}
	folders[nodeAllID] = folderItemAll.toJSON()
	folders[nodeLabelsID] = node{
		ID:          nodeLabelsID,
		Name:        "Labels",
		UnreadCount: totalLabelsUnreadCount,
	}.toJSON()
	folders[nodeListsID] = node{
		ID:          nodeListsID,
		Name:        "Mailing Lists",
		UnreadCount: totalListUnreadCount,
	}.toJSON()
	lists, err := model.FetchAllMailLists(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	for _, l := range lists {
		uid := fmt.Sprintf("list%d", l.EveEntityID)
		ids[nodeListsID] = append(ids[nodeListsID], uid)
		u, ok := listUnreadCounts[int(l.EveEntityID)]
		if !ok {
			u = 0
		}
		n := node{ObjID: l.EveEntityID, Name: l.EveEntity.Name, Category: nodeCategoryList, UnreadCount: u}
		folders[uid] = n.toJSON()
	}
	labels, err := model.FetchAllMailLabels(characterID)
	if err != nil {
		return nil, nil, node{}, err
	}
	if len(labels) > 0 {
		for _, l := range labels {
			if l.LabelID > 8 {
				uid := fmt.Sprintf("label%d", l.LabelID)
				ids[nodeLabelsID] = append(ids[nodeLabelsID], uid)
				u, ok := labelUnreadCounts[int(l.LabelID)]
				if !ok {
					u = 0
				}
				n := node{ObjID: l.LabelID, Name: l.Name, Category: nodeCategoryLabel, UnreadCount: u}
				folders[uid] = n.toJSON()
			}
		}

	}
	return ids, folders, folderItemAll, nil
}

func (f *folderArea) UpdateMails() {
	status := f.ui.statusArea
	go func() {
		charID := f.ui.CurrentCharID()
		if charID != 0 {
			err := logic.UpdateMails(charID, status.text)
			if err != nil {
				status.setText("Failed to fetch mail")
				slog.Error("Failed to update mails", "characterID", charID, "error", err)
				return
			}
		}
		f.Redraw()
	}()
}
