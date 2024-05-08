package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// StaticTable is a modification of Fyne's table widget, which does not allow the user to select or hover.
// It is useful for displaying data in a table format, when the user is not supposed to interact with it.
type StaticTable struct {
	widget.Table
}

func NewStaticTable(length func() (rows int, cols int), create func() fyne.CanvasObject, update func(widget.TableCellID, fyne.CanvasObject)) *StaticTable {
	t := &StaticTable{
		Table: widget.Table{Length: length, CreateCell: create, UpdateCell: update},
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *StaticTable) Tapped(e *fyne.PointEvent) {}

func (t *StaticTable) Enable() {}

func (t *StaticTable) Disable() {}

func (t *StaticTable) Disabled() bool {
	return true
}
