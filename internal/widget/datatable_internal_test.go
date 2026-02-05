package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColumnSorter_New(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		ID:    0,
		Label: "Alpha",
	}, {
		ID:    1,
		Label: "Bravo",
	}})
	t.Run("should create normally", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortAsc)
		got := sc.column(0)
		assert.Equal(t, SortAsc, got)
	})
}

func TestColumnSorter_Column(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		ID:    0,
		Label: "Alpha",
	}, {
		ID:    1,
		Label: "Bravo",
	}, {
		ID:    2,
		Label: "Charlie",
	}})
	t.Run("return value", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortOff)
		sc.Set(1, SortDesc)
		got := sc.column(1)
		assert.Equal(t, SortDesc, got)
	})
	t.Run("out of bounds returns zero value 1", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortOff)
		got := sc.column(4)
		assert.Equal(t, SortOff, got)
	})
	t.Run("out of bounds returns zero value 2", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortOff)
		got := sc.column(-1)
		assert.Equal(t, SortOff, got)
	})
}

func TestColumnSorter_Current(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		ID:    0,
		Label: "Alpha",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		ID:    1,
		Label: "Bravo",
	}, {
		ID:    2,
		Label: "Charlie",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}})
	t.Run("return currently sorted column", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortOff)
		sc.Set(0, SortDesc)
		x, y := sc.current()
		assert.Equal(t, 0, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return currently sorted column 2", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortOff)
		sc.Set(2, SortDesc)
		x, y := sc.current()
		assert.Equal(t, 2, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return -1 if nothing set", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortOff)
		x, y := sc.current()
		assert.Equal(t, -1, x)
		assert.Equal(t, SortOff, y)
	})
}
