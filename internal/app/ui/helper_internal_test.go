package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortedColumsColumn(t *testing.T) {
	t.Run("return value", func(t *testing.T) {
		sc := newSortedColumns(3)
		sc.set(1, sortDesc)
		got := sc.column(1)
		assert.Equal(t, sortDesc, got)
	})
	t.Run("out of bounds returns zero value 1", func(t *testing.T) {
		sc := newSortedColumns(3)
		got := sc.column(4)
		assert.Equal(t, sortOff, got)
	})
	t.Run("out of bounds returns zero value 2", func(t *testing.T) {
		sc := newSortedColumns(3)
		got := sc.column(-1)
		assert.Equal(t, sortOff, got)
	})
}

func TestSortedColumsCurrent(t *testing.T) {
	t.Run("return currently sorted column", func(t *testing.T) {
		sc := newSortedColumns(3)
		sc.set(1, sortDesc)
		x, y := sc.current()
		assert.Equal(t, 1, x)
		assert.Equal(t, sortDesc, y)
	})
	t.Run("return -1 if nothing set", func(t *testing.T) {
		sc := newSortedColumns(3)
		x, y := sc.current()
		assert.Equal(t, -1, x)
		assert.Equal(t, sortOff, y)
	})
}

func TestSortedColumsCycleColumn(t *testing.T) {
	cases := []struct {
		name    string
		col     int
		dir     sortDir
		wantCol int
		wantDir sortDir
	}{
		{"", 2, sortOff, 2, sortAsc},
		{"", 2, sortAsc, 2, sortDesc},
		{"", 2, sortDesc, -1, sortOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := newSortedColumns(3)
			sc.set(tc.col, tc.dir)
			got := sc.cycleColumn(tc.col)
			assert.Equal(t, tc.wantDir, got)
			x, y := sc.current()
			assert.Equal(t, tc.wantCol, x)
			assert.Equal(t, tc.wantDir, y)
		})
	}
}
