package datanodetree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type simpleNode struct {
	Name string
}

func TestJSONMarshaler(t *testing.T) {
	t.Run("can serialize to and from JSON", func(t *testing.T) {
		f1 := simpleNode{Name: "Crimson Sky"}
		s, err := objectToJSON(f1)
		if assert.NoError(t, err) {
			f2, err := newObjectFromJSON[simpleNode](s)
			if assert.NoError(t, err) {
				assert.Equal(t, f1, f2)
			}
		}
	})
}
