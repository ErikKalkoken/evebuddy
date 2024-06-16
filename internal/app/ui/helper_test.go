package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONMarshaler(t *testing.T) {
	t.Run("can serialize to and from JSON", func(t *testing.T) {
		f1 := folderNode{ObjID: 7, Name: "Crimson Sky", Category: nodeCategoryLabel}
		s, err := objectToJSON(f1)
		if assert.NoError(t, err) {
			f2, err := newObjectFromJSON[folderNode](s)
			if assert.NoError(t, err) {
				assert.Equal(t, f1, f2)
			}
		}
	})
}
