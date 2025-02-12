package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
)

type headerDef struct {
	text  string
	width float32
}

func maxHeaderWidth(headers []headerDef) float32 {
	var m float32
	for _, h := range headers {
		l := widget.NewLabel(h.text)
		m = max(l.MinSize().Width, m)
	}
	return m
}

// makeTopLabel returns a new empty label meant for the top bar on a screen.
func makeTopLabel() *widget.Label {
	l := widget.NewLabel("")
	l.TextStyle.Bold = true
	l.Wrapping = fyne.TextWrapWord
	return l
}

func makeDataTableForDesktop[S ~[]E, E any](
	headers []headerDef,
	data *S,
	makeLabel func(int, E) (string, fyne.TextAlign, widget.Importance),
	onSelected func(E),
) *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(*data), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*widget.Label)
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			cell.Text, cell.Alignment, cell.Importance = makeLabel(tci.Col, r)
			cell.Truncation = fyne.TextTruncateClip
			cell.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h := headers[tci.Col]
		label := co.(*widget.Label)
		label.SetText(h.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if onSelected != nil {
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			onSelected(r)
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	return t
}

func makeDataTableForMobile[S ~[]E, E any](
	headers []headerDef,
	data *S,
	makeLabel func(int, E) (string, fyne.TextAlign, widget.Importance),
	onSelected func(E),
) *widget.List {
	l := widget.NewList(
		func() int {
			return len(*data)
		},
		func() fyne.CanvasObject {
			p := theme.Padding()
			rowLayout := kxlayout.NewColumns(maxHeaderWidth(headers) + theme.Padding())
			c := container.New(layout.NewCustomPaddedVBoxLayout(0))
			for _, h := range headers {
				row := container.New(rowLayout, widget.NewLabel(h.text), widget.NewLabel(""))
				bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
				bg.Hide()
				c.Add(container.NewStack(bg, row))
				c.Add(container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, 2*p), widget.NewSeparator()))
			}
			return c
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			f := co.(*fyne.Container).Objects
			if id >= len(*data) || id < 0 {
				return
			}
			r := (*data)[id]
			for col := range len(headers) {
				row := f[col*2].(*fyne.Container).Objects[1].(*fyne.Container).Objects
				data := row[1].(*widget.Label)
				data.Text, _, data.Importance = makeLabel(col, r)
				data.Truncation = fyne.TextTruncateEllipsis
				bg := f[col*2].(*fyne.Container).Objects[0]
				if col == 0 {
					bg.Show()
					data.TextStyle.Bold = true
					label := row[0].(*widget.Label)
					label.TextStyle.Bold = true
					label.Refresh()
				} else {
					bg.Hide()
				}
				data.Refresh()
				divider := f[col*2+1]
				if col > 0 && col < len(headers)-1 {
					divider.Show()
				} else {
					divider.Hide()
				}
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if onSelected != nil {
			if id >= len(*data) || id < 0 {
				return
			}
			r := (*data)[id]
			onSelected(r)
		}
	}
	return l
}
