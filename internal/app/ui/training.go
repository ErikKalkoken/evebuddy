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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	trainingStatusActive   = "Active"
	trainingStatusInActive = "Inactive"
)

type trainingRow struct {
	characterID                int32
	characterName              string
	currentRemaining           optional.Optional[time.Duration]
	currentRemainingDisplay    string
	currentSkillDisplay        []widget.RichTextSegment
	currentSkillID             int32
	currentSkillName           string
	isActive                   bool
	tags                       set.Set[string]
	totalRemainingCount        optional.Optional[int]
	totalRemainingCountDisplay string
	totalRemainingTime         optional.Optional[time.Duration]
	totalRemainingTimeDisplay  string
	totalSP                    optional.Optional[int]
	totalSPDisplay             string
	unallocatedSP              optional.Optional[int]
	unallocatedSPDisplay       string
	statusText                 string
	statusImportance           widget.Importance
}

type trainings struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	columnSorter *columnSorter
	rows         []trainingRow
	rowsFiltered []trainingRow
	selectStatus *kxwidget.FilterChipSelect
	selectTag    *kxwidget.FilterChipSelect
	sortButton   *sortButton
	bottom       *widget.Label
	u            *baseUI
}

func newTrainings(u *baseUI) *trainings {
	headers := []headerDef{
		{label: "Name", width: columnWidthCharacter},
		{label: "SP", width: 75},
		{label: "Unall. SP", width: 75},
		{label: "Skill", width: 250},
		{label: "Skill Remaining", width: 0},
		{label: "Total Remaining", width: 0},
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
			return iwidget.RichTextSegmentsFromText(r.characterName)
		case 1:
			return iwidget.RichTextSegmentsFromText(
				r.totalSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 2:
			return iwidget.RichTextSegmentsFromText(
				r.unallocatedSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 3:
			return r.currentSkillDisplay
		case 4:
			return iwidget.RichTextSegmentsFromText(r.currentRemainingDisplay)
		case 5:
			return iwidget.RichTextSegmentsFromText(r.totalRemainingTimeDisplay)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			func(_ int, r trainingRow) {
				a.showDetails(r)
			},
		)
	} else {
		a.body = a.makeDataList()
	}
	a.selectStatus = kxwidget.NewFilterChipSelect(
		"Status",
		[]string{
			trainingStatusActive,
			trainingStatusInActive,
		}, func(string) {
			a.filterRows(-1)
		},
	)
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *trainings) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectStatus, a.selectTag)
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

func (a *trainings) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			character := widget.NewLabel("Template")
			character.Truncation = fyne.TextTruncateClip
			status := widget.NewLabel("Template")
			totalRemaining := widget.NewLabel("Template")
			totalCount := widget.NewLabel("Template")
			totalCount.Truncation = fyne.TextTruncateClip
			currentName := widget.NewLabel("Template")
			currentName.Truncation = fyne.TextTruncateClip
			currentRemaining := widget.NewLabel("Template")
			totalSP := widget.NewLabel("Template")
			totalSP.Truncation = fyne.TextTruncateClip
			unallocatedSP := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, status, character),
				container.NewBorder(nil, nil, nil, totalRemaining, totalCount),
				container.NewBorder(nil, nil, nil, currentRemaining, currentName),
				container.NewBorder(nil, nil, nil, unallocatedSP, totalSP),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects

			b0 := c[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.characterName)
			status := b0[1].(*widget.Label)
			status.Text = r.statusText
			status.Importance = r.statusImportance
			status.Refresh()

			b1 := c[1].(*fyne.Container).Objects
			b1[0].(*widget.Label).SetText(r.totalRemainingCountDisplay + " skills queued")
			b1[1].(*widget.Label).SetText(r.totalRemainingTimeDisplay)

			b2 := c[2].(*fyne.Container).Objects
			b2[0].(*widget.Label).SetText(r.currentSkillName)
			b2[1].(*widget.Label).SetText(r.currentRemainingDisplay)

			b3 := c[3].(*fyne.Container).Objects
			b3[0].(*widget.Label).SetText(r.totalSPDisplay + " total SP")
			unallocated := b3[1].(*widget.Label)
			unallocated.Text = r.unallocatedSPDisplay + " unallocated SP"
			unallocated.TextStyle.Bold = r.unallocatedSP.ValueOrZero() > 0
			unallocated.Refresh()
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		a.showDetails(a.rowsFiltered[id])
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
				return r.isActive
			case trainingStatusInActive:
				return !r.isActive
			}
			return false
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r trainingRow) bool {
			return r.tags.Contains(x)
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
				x = strings.Compare(a.currentSkillName, b.currentSkillName)
			case 4:
				x = cmp.Compare(a.currentRemaining.ValueOrZero(), b.currentRemaining.ValueOrZero())
			case 5:
				x = cmp.Compare(a.totalRemainingTime.ValueOrZero(), b.totalRemainingTime.ValueOrZero())
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r trainingRow) set.Set[string] {
		return r.tags
	})...).All()))
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
		tags, err := s.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		r.tags = set.Collect(xiter.MapSlice(tags, func(x *app.CharacterTag) string {
			return x.Name
		}))
		queue := app.NewCharacterSkillqueue()
		if err := queue.Update(ctx, s.cs, c.ID); err != nil {
			return nil, err
		}
		current := queue.Current()
		if current != nil {
			r.isActive = true
			r.statusText = "Active"
			r.statusImportance = widget.SuccessImportance
			r.currentSkillID = current.SkillID
			r.currentSkillName = app.SkillDisplayName(current.SkillName, current.FinishedLevel)
			r.currentSkillDisplay = iwidget.RichTextSegmentsFromText(r.currentSkillName)
			r.currentRemaining = current.Remaining()
			r.currentRemainingDisplay = ihumanize.Duration(current.Remaining().ValueOrZero())

		} else {
			r.statusText = "Inactive"
			r.statusImportance = widget.WarningImportance
			r.currentSkillName = "N/A"
			r.currentSkillDisplay = iwidget.RichTextSegmentsFromText(
				"Inactive",
				widget.RichTextStyle{
					ColorName: theme.ColorNameWarning,
				},
			)
			r.currentRemainingDisplay = "N/A"
		}
		r.totalRemainingTime = queue.RemainingTime()
		if !r.isActive {
			r.totalRemainingTimeDisplay = "N/A"
		} else {
			r.totalRemainingTimeDisplay = r.totalRemainingTime.StringFunc("?", func(v time.Duration) string {
				return ihumanize.Duration(v)
			})
		}
		r.totalRemainingCount = queue.RemainingCount()
		r.totalRemainingCountDisplay = r.totalRemainingCount.StringFunc("N/A", func(v int) string {
			return ihumanize.Comma(v)
		})
		rows[i] = r
	}
	return rows, nil
}

func (a *trainings) showDetails(r trainingRow) {
	status := widget.NewLabel(r.statusText)
	status.Importance = r.statusImportance
	var skill fyne.CanvasObject
	if r.isActive {
		skill = makeLinkLabelWithWrap(r.currentSkillName, func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, r.currentSkillID)
		})
	} else {
		skill = widget.NewLabel(r.currentSkillName)
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			r.characterID,
			r.characterName,
			a.u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Skillpoints", widget.NewLabel(r.totalSPDisplay)),
		widget.NewFormItem("Unalloc. SP", widget.NewLabel(r.unallocatedSPDisplay)),
		widget.NewFormItem("Status", status),
		widget.NewFormItem("Current skill", skill),
		widget.NewFormItem("Current skill remaining", widget.NewLabel(r.currentRemainingDisplay)),
		widget.NewFormItem("Total skills queued", widget.NewLabel(r.totalRemainingCountDisplay)),
		widget.NewFormItem("Total time remaining", widget.NewLabel(r.totalRemainingTimeDisplay)),
	}

	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	title := fmt.Sprintf("Training info for %s", r.characterName)
	w := a.u.makeDetailWindowWithSize("Training info", title, fyne.NewSize(500, 350), f)
	w.Show()
}
