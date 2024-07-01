package fynetree_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	"github.com/stretchr/testify/assert"
)

func TestFyneTree(t *testing.T) {
	t.Run("can create a new tree", func(t *testing.T) {
		tree := fynetree.New[string]()
		tree.MustAdd("", "1", "Alpha")
		assert.Equal(t, "Alpha", tree.Value("1"))
	})
}
