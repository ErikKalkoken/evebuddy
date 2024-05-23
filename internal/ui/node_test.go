package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderItem(t *testing.T) {
	t.Run("can serialize to and from JSON", func(t *testing.T) {
		f1 := folderNode{ObjID: 7, Name: "Crimson Sky", Category: nodeCategoryLabel}
		s := f1.toJSON()
		f2 := newFolderTreeNodeFromJSON(s)
		assert.Equal(t, f1, f2)

	})
}
