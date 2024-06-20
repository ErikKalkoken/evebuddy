package stringdatatree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type treeNode struct {
	Name string
}

func TestJSONMarshaler(t *testing.T) {
	t.Run("can serialize to and from JSON", func(t *testing.T) {
		f1 := treeNode{Name: "Crimson Sky"}
		s, err := objectToJSON(f1)
		if assert.NoError(t, err) {
			f2, err := newObjectFromJSON[treeNode](s)
			if assert.NoError(t, err) {
				assert.Equal(t, f1, f2)
			}
		}
	})
}
