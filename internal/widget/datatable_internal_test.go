package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataColumns(t *testing.T) {
	t.Run("should create data columns", func(t *testing.T) {
		def := NewDataColumns([]DataColumn{{
			Col:   0,
			Label: "Alpha",
		}, {
			Col:   1,
			Label: "Bravo",
		}})
		assert.IsType(t, DataColumns{}, def)
	})
	t.Run("should panic when label is missing", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDataColumns([]DataColumn{{
				Col:   0,
				Label: "",
			}})
		})
	})
	t.Run("should panic when col does not start at 0", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDataColumns([]DataColumn{
				{
					Col:   1,
					Label: "Alpha",
				},
				{
					Col:   2,
					Label: "Bravo",
				},
			})
		})
	})
	t.Run("should panic when col is duplicate", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDataColumns([]DataColumn{
				{
					Col:   0,
					Label: "Alpha",
				},
				{
					Col:   0,
					Label: "Bravo",
				},
			})
		})
	})
	t.Run("should panic when no cols defined", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDataColumns([]DataColumn{})
		})
	})
}

func TestColumnSorter_New(t *testing.T) {
	def := NewDataColumns([]DataColumn{{
		Col:   0,
		Label: "Alpha",
	}, {
		Col:   1,
		Label: "Bravo",
	}})
	t.Run("should create normally", func(t *testing.T) {
		sc := NewColumnSorter(def, 0, SortAsc)
		got := sc.column(0)
		assert.Equal(t, SortAsc, got)
	})
	t.Run("should panic when initialized with invalid colum index", func(t *testing.T) {
		assert.Panics(t, func() {
			NewColumnSorter(def, 3, SortAsc)
		})
	})
}

func TestColumnSorter_Column(t *testing.T) {
	def := NewDataColumns([]DataColumn{{
		Col:   0,
		Label: "Alpha",
	}, {
		Col:   1,
		Label: "Bravo",
	}, {
		Col:   2,
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
	def := NewDataColumns([]DataColumn{{
		Col:   0,
		Label: "Alpha",
	}, {
		Col:    1,
		Label:  "Bravo",
		NoSort: true,
	}, {
		Col:   2,
		Label: "Charlie",
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

// func TestColumnSorterCycleColumn(t *testing.T) {
// 	cases := []struct {
// 		name    string
// 		col     int
// 		dir     sortDir
// 		wantCol int
// 		wantDir sortDir
// 	}{
// 		{"", 2, sortOff, 2, sortAsc},
// 		{"", 2, sortAsc, 2, sortDesc},
// 		{"", 2, sortDesc, -1, sortOff},
// 	}
// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			sc := newColumnSorter(def)
// 			sc.set(tc.col, tc.dir)
// 			got := sc.cycleColumn(tc.col)
// 			assert.Equal(t, tc.wantDir, got)
// 			x, y := sc.current()
// 			assert.Equal(t, tc.wantCol, x)
// 			assert.Equal(t, tc.wantDir, y)
// 		})
// 	}
// }
