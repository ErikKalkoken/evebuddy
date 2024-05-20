package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const folderUpdateTicker = 10 * time.Second

// folderArea is the UI area showing the mail folders.
type folderArea struct {
	content       fyne.CanvasObject
	newButton     *widget.Button
	lastUID       string
	lastFolderAll node
	tree          *widget.Tree
	treeData      binding.StringTree
	ui            *ui
}

func (u *ui) NewFolderArea() *folderArea {
	a := &folderArea{
		treeData: binding.NewStringTree(),
		ui:       u,
	}
	a.tree = widget.NewTreeWithData(
		a.treeData,
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
	a.tree.OnSelected = func(uid string) {
		di, err := a.treeData.GetItem(uid)
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
				a.tree.Select(u.folderArea.lastUID)
			}
			return
		}
		u.folderArea.lastUID = uid
		u.headerArea.SetFolder(item)
	}
	a.newButton = widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		a.ui.ShowSendMessageWindow(CreateMessageNew, nil)
	})
	a.newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), a.newButton, layout.NewSpacer())
	a.content = container.NewBorder(top, nil, nil, nil, a.tree)
	return a
}

func (a *folderArea) Refresh() {
	characterID := a.ui.CurrentCharID()
	ids, values, folderAll, err := a.buildFolderTree(characterID)
	if err != nil {
		slog.Error("Failed to build folder tree", "character", characterID, "error", err)
	}
	a.treeData.Set(ids, values)
	if a.lastUID == "" {
		a.tree.Select(nodeAllID)
		a.tree.ScrollToTop()
		a.ui.headerArea.SetFolder(folderAll)
	} else {
		a.ui.headerArea.Refresh()
	}
	a.lastFolderAll = folderAll
	s := "Mail"
	if folderAll.UnreadCount > 0 {
		s += fmt.Sprintf(" (%s)", humanize.Comma(int64(folderAll.UnreadCount)))
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

func (a *folderArea) StartUpdateTicker() {
	ticker := time.NewTicker(folderUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListMyCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of my characters", "err", err)
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

func (a *folderArea) MaybeUpdateAndRefresh(characterID int32) {
	changed1, err := a.ui.service.UpdateSectionIfExpired(characterID, model.UpdateSectionMailLabels)
	if err != nil {
		slog.Error("Failed to update mail labels", "character", characterID, "err", err)
		return
	}
	changed2, err := a.ui.service.UpdateSectionIfExpired(characterID, model.UpdateSectionMailLists)
	if err != nil {
		slog.Error("Failed to update mail lists", "character", characterID, "err", err)
		return
	}
	changed3, err := a.ui.service.UpdateSectionIfExpired(characterID, model.UpdateSectionMails)
	if err != nil {
		slog.Error("Failed to update mail", "character", characterID, "err", err)
		return
	}
	if (changed1 || changed2 || changed3) && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}
