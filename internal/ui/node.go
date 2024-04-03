package ui

import (
	"encoding/json"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type nodeCategory int

const (
	nodeCategoryBranch nodeCategory = 0
	nodeCategoryLabel  nodeCategory = 1
	nodeCategoryList   nodeCategory = 2
)

const (
	nodeAllID      = "all"
	nodeInboxID    = "inbox"
	nodeSentID     = "sent"
	nodeCorpID     = "corp"
	nodeAllianceID = "alliance"
	nodeTrashID    = "trash"
	nodeLabelsID   = "labels"
	nodeListsID    = "lists"
)

// A node in the folder tree, e.g. the inbox
type node struct {
	ID          string
	ObjID       int32
	Name        string
	Category    nodeCategory
	UnreadCount int
}

func newNodeFromJSON(s string) node {
	var f node
	err := json.Unmarshal([]byte(s), &f)
	if err != nil {
		panic(err)
	}
	return f
}

func (f node) toJSON() string {
	s, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	return string(s)
}

func (f node) isBranch() bool {
	return f.Category == nodeCategoryBranch
}

func (f node) icon() fyne.Resource {
	switch f.ID {
	case nodeInboxID:
		return theme.DownloadIcon()
	case nodeSentID:
		return theme.UploadIcon()
	case nodeTrashID:
		return theme.DeleteIcon()
	}
	return theme.FolderIcon()
}
