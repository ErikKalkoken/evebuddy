package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	trainingStatusActive   = "Active"
	trainingStatusInActive = "Inactive"
)

type trainingRow struct {
	characterID          int32
	characterName        string
	totalSP              optional.Optional[int]
	totalSPDisplay       string
	training             optional.Optional[time.Duration]
	trainingDisplay      []widget.RichTextSegment
	unallocatedSP        optional.Optional[int]
	unallocatedSPDisplay string
}

func (r trainingRow) isActive() bool {
	return !r.training.IsEmpty()
}

type trainings struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	columnSorter *columnSorter
	rows         []trainingRow
	rowsFiltered []trainingRow
	selectStatus *iwidget.FilterChipSelect
	sortButton   *sortButton
	bottom       *widget.Label
	u            *baseUI
}

func newTrainings(u *baseUI) *trainings {
	headers := []headerDef{
		{Text: "Name", Width: 250},
		{Text: "SP", Width: 100},
		{Text: "Unall. SP", Width: 100},
		{Text: "Training", Width: 100},
	}
	a := &trainings{
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		rows:         make([]trainingRow, 0),
		rowsFiltered: make([]trainingRow, 0),
		bottom:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r trainingRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.characterName)
		case 1:
			return iwidget.NewRichTextSegmentFromText(
				r.totalSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 2:
			return iwidget.NewRichTextSegmentFromText(
				r.unallocatedSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 3:
			return r.trainingDisplay
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			nil,
		)
	} else {
		// a.body = makeDataList(headers, &a.rowsFiltered, makeCell, nil)
		a.body = a.makeDataList()
	}
	a.selectStatus = iwidget.NewFilterChipSelect(
		"Status",
		[]string{
			trainingStatusActive,
			trainingStatusInActive,
		}, func(string) {
			a.filterRows(-1)
		},
	)

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *trainings) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectStatus)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewHScroll(filter),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *trainings) makeDataList() *widget.List {
	p := theme.Padding()
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			title.Wrapping = fyne.TextWrapWord
			training := widget.NewRichTextWithText("Template")
			sp := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				title,
				training,
				sp,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects
			c[0].(*widget.Label).SetText(r.characterName)
			iwidget.SetRichText(c[1].(*widget.RichText), r.trainingDisplay...)
			c[2].(*widget.Label).SetText(fmt.Sprintf("%s (%s) SP", r.totalSPDisplay, r.unallocatedSPDisplay))
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *trainings) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectStatus.Selected; x != "" {
		rows = xslices.Filter(rows, func(r trainingRow) bool {
			switch x {
			case trainingStatusActive:
				return r.isActive()
			case trainingStatusInActive:
				return !r.isActive()
			}
			return false
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b trainingRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.characterName, b.characterName)
			case 1:
				x = cmp.Compare(a.totalSP.ValueOrZero(), b.totalSP.ValueOrZero())
			case 2:
				x = cmp.Compare(a.unallocatedSP.ValueOrZero(), b.unallocatedSP.ValueOrZero())
			case 3:
				x = cmp.Compare(a.training.ValueOrZero(), b.training.ValueOrZero())
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *trainings) update() {
	rows := make([]trainingRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchRows(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		return "", widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh training UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		if t != "" {
			a.bottom.Text = t
			a.bottom.Importance = i
			a.bottom.Refresh()
			a.bottom.Show()
		} else {
			a.bottom.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (*trainings) fetchRows(s services) ([]trainingRow, error) {
	ctx := context.Background()
	characters, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]trainingRow, len(characters))
	for i, c := range characters {
		if c.EveCharacter == nil {
			continue
		}
		r := trainingRow{
			characterID:          c.ID,
			characterName:        c.EveCharacter.Name,
			totalSP:              c.TotalSP,
			totalSPDisplay:       ihumanize.Optional(c.TotalSP, "?"),
			unallocatedSP:        c.UnallocatedSP,
			unallocatedSPDisplay: ihumanize.Optional(c.UnallocatedSP, "?"),
		}
		x, err := s.cs.GetTotalTrainingTime(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		r.training = x
		if x := r.training; x.IsEmpty() {
			r.trainingDisplay = iwidget.NewRichTextSegmentFromText(
				"Inactive",
				widget.RichTextStyle{
					ColorName: theme.ColorNameWarning,
				},
			)
		} else {
			r.trainingDisplay = iwidget.NewRichTextSegmentFromText(
				ihumanize.Duration(x.ValueOrZero()),
				widget.RichTextStyle{
					ColorName: theme.ColorNameSuccess,
				},
			)
		}
		rows[i] = r
	}
	return rows, nil
}
