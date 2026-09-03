package xwidget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColumnSorter_New(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
	}, {
		Label: "Bravo",
	}})
	t.Run("should create normally", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		got := sc.column(0)
		assert.Equal(t, SortAsc, got)
	})
}

func TestColumnSorter_Column(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
	}, {
		Label: "Bravo",
	}, {
		Label: "Charlie",
	}})
	t.Run("return value", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Bravo", SortDesc)
		got := sc.column(1)
		assert.Equal(t, SortDesc, got)
	})
	t.Run("out of bounds returns zero value 1", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		got := sc.column(4)
		assert.Equal(t, SortOff, got)
	})
	t.Run("out of bounds returns zero value 2", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		got := sc.column(-1)
		assert.Equal(t, SortOff, got)
	})
}

func TestColumnSorter_Set(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		Label: "Bravo",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		Label: "Charlie",
	}})
	t.Run("sets the sort direction for the named column", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Bravo", SortDesc)
		assert.Equal(t, SortDesc, sc.column(1))
	})
	t.Run("resets other columns when a new one is set", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		sc.Set("Bravo", SortDesc)
		assert.Equal(t, SortOff, sc.column(0))
		assert.Equal(t, SortDesc, sc.column(1))
	})
	t.Run("does nothing for an unknown label", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		sc.Set("Zulu", SortDesc)
		assert.Equal(t, SortAsc, sc.column(0))
		assert.Equal(t, SortOff, sc.column(1))
	})
	t.Run("sets the direction even for a column without a Sort func", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortAsc)
		sc.Set("Charlie", SortDesc)
		assert.Equal(t, SortDesc, sc.column(2))
	})
}

func TestColumnSorter_Current(t *testing.T) {
	def := NewDataColumns([]DataColumn[struct{}]{{
		Label: "Alpha",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}, {
		Label: "Bravo",
	}, {
		Label: "Charlie",
		Sort: func(a, b struct{}) int {
			return 0
		},
	}})
	t.Run("return currently sorted column", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Alpha", SortDesc)
		x, y := sc.current()
		assert.Equal(t, 0, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return currently sorted column 2", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		sc.Set("Charlie", SortDesc)
		x, y := sc.current()
		assert.Equal(t, 2, x)
		assert.Equal(t, SortDesc, y)
	})
	t.Run("return -1 if nothing set", func(t *testing.T) {
		sc := NewColumnSorter(def, "Alpha", SortOff)
		x, y := sc.current()
		assert.Equal(t, -1, x)
		assert.Equal(t, SortOff, y)
	})
}
