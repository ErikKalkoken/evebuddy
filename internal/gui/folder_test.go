package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFolderItem(t *testing.T) {
	t.Run("can serialize to and from JSON", func(t *testing.T) {
		f1 := treeItem{Id: 7, Name: "Crimson Sky", Category: itemCategoryLabel}
		s := f1.toJSON()
		f2 := newTreeItemJSON(s)
		assert.Equal(t, f1, f2)

	})
}
