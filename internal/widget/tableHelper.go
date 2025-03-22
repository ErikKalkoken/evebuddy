package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
)

type HeaderDef struct {
	Text  string
	Width float32
}

func maxHeaderWidth(headers []HeaderDef) float32 {
	var m float32
	for _, h := range headers {
		l := widget.NewLabel(h.Text)
		m = max(l.MinSize().Width, m)
	}
	return m
}

func MakeDataTableForDesktop[S ~[]E, E any](
	headers []HeaderDef,
	data *S,
	makeCell func(int, E) (string, fyne.TextAlign, widget.Importance),
	onSelected func(int, E),
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
			cell.Text, cell.Alignment, cell.Importance = makeCell(tci.Col, r)
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
		label.SetText(h.Text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if onSelected != nil {
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			onSelected(tci.Col, r)
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.Width)
	}
	return t
}

func MakeDataTableForMobile[S ~[]E, E any](
	headers []HeaderDef,
	data *S,
	makeCell func(int, E) (string, fyne.TextAlign, widget.Importance),
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
				row := container.New(rowLayout, widget.NewLabel(h.Text), widget.NewLabel(""))
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
				data.Text, _, data.Importance = makeCell(col, r)
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

func MakeDataTableForDesktop2[S ~[]E, E any](
	headers []HeaderDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	onSelected func(int, E),
) *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(*data), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*widget.RichText)
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			cell.Segments = makeCell(tci.Col, r)
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
		label.SetText(h.Text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if onSelected != nil {
			if tci.Row >= len(*data) || tci.Row < 0 {
				return
			}
			r := (*data)[tci.Row]
			onSelected(tci.Col, r)
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.Width)
	}
	return t
}

func MakeDataTableForMobile2[S ~[]E, E any](
	headers []HeaderDef,
	data *S,
	makeCell func(int, E) []widget.RichTextSegment,
	onSelected func(E),
) *widget.List {
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(*data)
		},
		func() fyne.CanvasObject {
			p := theme.Padding()
			rowLayout := kxlayout.NewColumns(maxHeaderWidth(headers) + theme.Padding())
			c := container.New(layout.NewCustomPaddedVBoxLayout(0))
			for _, h := range headers {
				row := container.New(rowLayout, widget.NewLabel(h.Text), widget.NewRichText())
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
				data := row[1].(*widget.RichText)
				data.Segments = makeCell(col, r)
				data.Wrapping = fyne.TextWrapWord
				bg := f[col*2].(*fyne.Container).Objects[0]
				if col == 0 {
					bg.Show()
					for _, s := range data.Segments {
						x, ok := s.(*widget.TextSegment)
						if !ok {
							continue
						}
						x.Style.TextStyle.Bold = true
					}
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
			l.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
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
