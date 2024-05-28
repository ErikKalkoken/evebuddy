package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderItem(t *testing.T) {
	t.Run("can serialize to and from JSON", func(t *testing.T) {
		f1 := folderNode{ObjID: 7, Name: "Crimson Sky", Category: nodeCategoryLabel}
		s, err := f1.toJSON()
		if assert.NoError(t, err) {
			f2, err := newFolderTreeNodeFromJSON(s)
			if assert.NoError(t, err) {
				assert.Equal(t, f1, f2)
			}
		}
	})
}
