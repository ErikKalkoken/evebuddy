package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// StaticList is a modification of Fyne's list widget, which does not allow the user to select anything.
// It is useful for displaying data in a list format, when the user is not supposed to interact with it.
type StaticList struct {
	widget.List
}

func NewStaticList(length func() int, createItem func() fyne.CanvasObject, updateItem func(widget.ListItemID, fyne.CanvasObject)) *StaticList {
	t := &StaticList{
		List: widget.List{Length: length, CreateItem: createItem, UpdateItem: updateItem},
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *StaticList) Tapped(e *fyne.PointEvent) {}

func (t *StaticList) Enable() {}

func (t *StaticList) Disable() {}

func (t *StaticList) Disabled() bool {
	return true
}
