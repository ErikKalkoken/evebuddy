package gui

import (
	"encoding/json"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type treeItemCategory int

const (
	itemCategoryBranch treeItemCategory = 0
	itemCategoryLabel  treeItemCategory = 1
	itemCategoryList   treeItemCategory = 2
)

const (
	itemAll      = "all"
	itemInbox    = "inbox"
	itemSent     = "sent"
	itemCorp     = "corp"
	itemAlliance = "alliance"
	itemTrash    = "trash"
	itemLabels   = "labels"
	itemLists    = "lists"
)

type treeItem struct {
	Id       int32
	Name     string
	Category treeItemCategory
}

func newTreeItemJSON(s string) treeItem {
	var f treeItem
	err := json.Unmarshal([]byte(s), &f)
	if err != nil {
		panic(err)
	}
	return f
}

func (f treeItem) toJSON() string {
	s, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	return string(s)
}

type folders struct {
	esiApp      *eveApp
	content     fyne.CanvasObject
	boundTree   binding.ExternalStringTree
	boundCharID binding.ExternalInt
	headers     *headers
	tree        *widget.Tree
	btnRefresh  *widget.Button
	btnNew      *widget.Button
}

func (e *eveApp) newFolders(headers *headers) *folders {
	tree, boundTree, boundCharID := makeFolderList(headers)
	f := folders{
		esiApp:      e,
		boundTree:   boundTree,
		boundCharID: boundCharID,
		headers:     headers,
		tree:        tree,
	}
	btnRefresh := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		f.updateMails()
	})
	f.btnRefresh = btnRefresh

	btnNew := widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		d := dialog.NewInformation("New message", "PLACEHOLDER", e.winMain)
		d.Show()
	})
	f.btnNew = btnNew

	top := container.NewHBox(f.btnRefresh, btnNew)
	c := container.NewBorder(top, nil, nil, nil, f.tree)
	f.content = c
	return &f
}

func (f *folders) updateMails() {
	charID, err := f.boundCharID.Get()
	if err != nil {
		slog.Error("Failed to get character ID", "error", err)
		return
	}
	status := f.esiApp.statusBar
	go func() {
		if charID != 0 {
			err = UpdateMails(int32(charID), status)
			if err != nil {
				status.setText("Failed to fetch mail")
				slog.Error("Failed to update mails", "characterID", charID, "error", err)
				return
			}
		}
		f.update(int32(charID))
	}()
}

func (f *folders) update(charID int32) {
	if charID == 0 {
		f.btnRefresh.Disable()
		f.btnNew.Disable()
	} else {
		f.btnRefresh.Enable()
		f.btnNew.Enable()
	}
	if err := f.boundCharID.Set(int(charID)); err != nil {
		slog.Error("Failed to set char ID", "characterID", charID, "error", err)
	}
	folderItemAll := treeItem{Id: model.LabelAll, Name: "All Mails", Category: itemCategoryLabel}
	ids := map[string][]string{
		"":         {itemAll, itemInbox, itemSent, itemCorp, itemAlliance, itemLabels, itemLists},
		itemLabels: {},
		itemLists:  {},
	}
	values := map[string]string{
		itemAll:      folderItemAll.toJSON(),
		itemInbox:    treeItem{Id: model.LabelInbox, Name: "Inbox", Category: itemCategoryLabel}.toJSON(),
		itemSent:     treeItem{Id: model.LabelSent, Name: "Sent", Category: itemCategoryLabel}.toJSON(),
		itemCorp:     treeItem{Id: model.LabelCorp, Name: "Corp", Category: itemCategoryLabel}.toJSON(),
		itemAlliance: treeItem{Id: model.LabelAlliance, Name: "Alliance", Category: itemCategoryLabel}.toJSON(),
		// "trash":    "Trash",
		itemLabels: treeItem{Name: "Labels"}.toJSON(),
		itemLists:  treeItem{Name: "Mailing Lists"}.toJSON(),
	}
	// Add labels to tree
	labels, err := model.FetchAllMailLabels(charID)
	if err != nil {
		slog.Error("Failed to fetch mail labels", "characterID", charID, "error", err)
	} else {
		if len(labels) > 0 {
			for _, l := range labels {
				if l.LabelID > 8 {
					uid := fmt.Sprintf("label%d", l.LabelID)
					ids[itemLabels] = append(ids[itemLabels], uid)
					f := treeItem{Id: l.LabelID, Name: l.Name, Category: itemCategoryLabel}
					values[uid] = f.toJSON()
				}
			}
		}
	}
	// Add mailing lists to tree
	lists, err := model.FetchAllMailLists(charID)
	if err != nil {
		slog.Error("Failed to fetch mail lists", "characterID", charID, "error", err)
	} else {
		for _, l := range lists {
			uid := fmt.Sprintf("list%d", l.EveEntityID)
			ids[itemLists] = append(ids[itemLists], uid)
			f := treeItem{Id: l.EveEntityID, Name: l.EveEntity.Name, Category: itemCategoryList}
			values[uid] = f.toJSON()
		}
	}
	f.boundTree.Set(ids, values)
	f.tree.Select(itemAll)
	f.tree.ScrollToTop()
	f.headers.update(charID, folderItemAll)
}

// func (f *folders) updateMailsWithID(charID int32) {
// 	err := f.boundCharID.Set(int(charID))
// 	if err != nil {
// 		slog.Error("Failed to set char ID", "error", err)
// 	} else {
// 		f.updateMails()
// 	}
// }

func makeFolderList(headers *headers) (*widget.Tree, binding.ExternalStringTree, binding.ExternalInt) {
	ids := make(map[string][]string)
	values := make(map[string]string)
	boundTree := binding.BindStringTree(&ids, &values)

	var charID int
	boundCharID := binding.BindInt(&charID)

	tree := widget.NewTreeWithData(
		boundTree,
		func(branch bool) fyne.CanvasObject {
			if branch {
				return widget.NewLabel("Branch template")
			}
			return widget.NewLabel("Leaf template")
		},
		func(di binding.DataItem, b bool, co fyne.CanvasObject) {
			i := di.(binding.String)
			s, err := i.Get()
			if err != nil {
				slog.Error("Failed to fetch data item for tree")
				return
			}
			item := newTreeItemJSON(s)
			co.(*widget.Label).SetText(item.Name)
		},
	)

	tree.OnSelected = func(uid string) {
		di, err := boundTree.GetItem(uid)
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
		charID, err := boundCharID.Get()
		if err != nil {
			slog.Error("Failed to get char ID", "error", err)
			return
		}
		item := newTreeItemJSON(s)
		if item.Category != itemCategoryBranch {
			headers.update(int32(charID), item)
		}
	}
	return tree, boundTree, boundCharID
}
