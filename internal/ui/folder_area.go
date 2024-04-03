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
	charID := f.ui.CurrentCharID()
	if charID == 0 {
		f.refreshButton.Disable()
		f.newButton.Disable()
	} else {
		f.refreshButton.Enable()
		f.newButton.Enable()
	}
	ids := map[string][]string{
		"":           {nodeAllID, nodeInboxID, nodeSentID, nodeCorpID, nodeAllianceID, nodeLabelsID, nodeListsID},
		nodeLabelsID: {},
		nodeListsID:  {},
	}
	values, folderItemAll := initialTreeValues(charID)
	addLabelsToTree(charID, ids, values)
	addMailListsToTree(charID, ids, values)
	f.treeData.Set(ids, values)
	f.tree.Select(nodeAllID)
	f.tree.ScrollToTop()
	f.ui.headerArea.Redraw(folderItemAll)
}

func initialTreeValues(characterID int32) (map[string]string, node) {
	unreadCounts, err := model.FetchMailLabelUnreadCounts(characterID)
	if err != nil {
		panic(err)
	}
	values := map[string]string{
		nodeLabelsID: node{ID: nodeLabelsID, Name: "Labels"}.toJSON(),
		nodeListsID:  node{ID: nodeListsID, Name: "Mailing Lists"}.toJSON(),
	}
	var totalUnreadCount int
	f := []struct {
		nodeID  string
		labelID int
		name    string
	}{
		{nodeInboxID, model.LabelInbox, "Inbox"},
		{nodeSentID, model.LabelSent, "Sent"},
		{nodeCorpID, model.LabelCorp, "Corp"},
		{nodeAllianceID, model.LabelAlliance, "Alliance"},
	}
	for _, o := range f {
		u, ok := unreadCounts[o.labelID]
		if !ok {
			u = 0
		}
		totalUnreadCount += u
		values[o.nodeID] = node{ID: o.nodeID, ObjID: int32(o.labelID), Name: o.name, Category: nodeCategoryLabel, UnreadCount: u}.toJSON()
	}
	folderItemAll := node{ID: nodeAllID, ObjID: model.LabelAll, Name: "All Mails", Category: nodeCategoryLabel, UnreadCount: totalUnreadCount}
	values[nodeAllID] = folderItemAll.toJSON()
	return values, folderItemAll
}

func addMailListsToTree(charID int32, ids map[string][]string, values map[string]string) {
	lists, err := model.FetchAllMailLists(charID)
	if err != nil {
		slog.Error("Failed to fetch mail lists", "characterID", charID, "error", err)
	} else {
		for _, l := range lists {
			uid := fmt.Sprintf("list%d", l.EveEntityID)
			ids[nodeListsID] = append(ids[nodeListsID], uid)
			n := node{ObjID: l.EveEntityID, Name: l.EveEntity.Name, Category: nodeCategoryList}
			values[uid] = n.toJSON()
		}
	}
}

func addLabelsToTree(charID int32, ids map[string][]string, values map[string]string) {
	labels, err := model.FetchAllMailLabels(charID)
	if err != nil {
		slog.Error("Failed to fetch mail labels", "characterID", charID, "error", err)
	} else {
		if len(labels) > 0 {
			for _, l := range labels {
				if l.LabelID > 8 {
					uid := fmt.Sprintf("label%d", l.LabelID)
					ids[nodeLabelsID] = append(ids[nodeLabelsID], uid)
					n := node{ObjID: l.LabelID, Name: l.Name, Category: nodeCategoryLabel}
					values[uid] = n.toJSON()
				}
			}
		}
	}
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
