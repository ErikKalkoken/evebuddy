package ui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	isActive                   bool
	skill                      *app.CharacterSkillqueueItem
	skillDisplay               []widget.RichTextSegment
	skillFinishDate            optional.Optional[time.Time]
	skillID                    int32
	skillName                  string
	skillProgress              optional.Optional[float64]
	statusImportance           widget.Importance
	statusText                 string
	tags                       set.Set[string]
	totalFinishDate            optional.Optional[time.Time]
	totalRemainingCount        optional.Optional[int]
	totalRemainingCountDisplay string
	totalSP                    optional.Optional[int]
	totalSPDisplay             string
	unallocatedSP              optional.Optional[int]
	unallocatedSPDisplay       string
}

func (r trainingRow) currentRemainingTime() optional.Optional[time.Duration] {
	return r.remainingTime(r.skillFinishDate)
}

func (r trainingRow) currentRemainingTimeString() string {
	return r.remainingTimeString(r.currentRemainingTime())
}

func (r trainingRow) totalRemainingTime() optional.Optional[time.Duration] {
	return r.remainingTime(r.totalFinishDate)
}

func (r trainingRow) totalRemainingTimeString() string {
	return r.remainingTimeString(r.totalRemainingTime())
}

func (trainingRow) remainingTime(t optional.Optional[time.Time]) optional.Optional[time.Duration] {
	if t.IsEmpty() {
		return optional.Optional[time.Duration]{}
	}
	d := time.Until(t.MustValue())
	if d < 0 {
		return optional.From(time.Duration(0))
	}
	return optional.From(time.Duration(d))
}

func (r trainingRow) remainingTimeString(d optional.Optional[time.Duration]) string {
	if !r.isActive {
		return "N/A"
	}
	return d.StringFunc("?", func(v time.Duration) string {
		return ihumanize.Duration(v)
	})
}

type training struct {
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

func newTraining(u *baseUI) *training {
	headers := []headerDef{
		{label: "Name", width: columnWidthCharacter},
		{label: "Current Skill", width: 250},
		{label: "Current Remaining", width: 0},
		{label: "Queued", width: 0},
		{label: "Queue Remaining", width: 0},
		{label: "SP", width: 50},
		{label: "Unall.", width: 50},
	}
	a := &training{
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
			return r.skillDisplay
		case 2:
			return iwidget.RichTextSegmentsFromText(r.currentRemainingTimeString())
		case 3:
			return iwidget.RichTextSegmentsFromText(r.totalRemainingCountDisplay)
		case 4:
			return iwidget.RichTextSegmentsFromText(r.totalRemainingTimeString())
		case 5:
			return iwidget.RichTextSegmentsFromText(
				r.totalSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case 6:
			return iwidget.RichTextSegmentsFromText(
				r.unallocatedSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
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

func (a *training) CreateRenderer() fyne.WidgetRenderer {
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

func (a *training) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			character := widget.NewLabel("Template")
			character.Truncation = fyne.TextTruncateClip
			character.SizeName = theme.SizeNameSubHeadingText
			status := widget.NewLabel("Template")
			queueRemaining := widget.NewLabel("Template")
			queueCount := widget.NewLabel("Template")
			queueCount.Truncation = fyne.TextTruncateClip
			totalSP := widget.NewLabel("Template")
			totalSP.Truncation = fyne.TextTruncateClip
			unallocatedSP := widget.NewLabel("Template")
			spacer1 := canvas.NewRectangle(color.Transparent)
			spacer1.SetMinSize(fyne.NewSize(1, 2*p))
			spacer2 := canvas.NewRectangle(color.Transparent)
			spacer2.SetMinSize(fyne.NewSize(1, 4*p))
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, status, character),
				spacer1,
				newSkillQueueItem(),
				container.NewBorder(nil, nil, nil, queueRemaining, queueCount),
				container.NewBorder(nil, nil, nil, unallocatedSP, totalSP),
				spacer2,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects

			c0 := c[0].(*fyne.Container).Objects
			c0[0].(*widget.Label).SetText(r.characterName)
			status := c0[1].(*widget.Label)
			status.Text = r.statusText
			status.Importance = r.statusImportance
			status.Refresh()

			c1 := c[2].(*skillQueueItem)
			c1.Set(r.skill)

			c2 := c[3].(*fyne.Container).Objects
			queueCount := c2[0].(*widget.Label)
			queueRemaining := c2[1].(*widget.Label)
			if r.totalRemainingCount.IsEmpty() {
				queueCount.Text = "N/A"
				queueRemaining.Text = ""
			} else {
				queueCount.Text = r.totalRemainingCountDisplay + " skills queued"
				queueRemaining.Text = r.totalRemainingTimeString()
			}
			queueCount.Refresh()
			queueRemaining.Refresh()

			c3 := c[4].(*fyne.Container).Objects
			c3[0].(*widget.Label).SetText(r.totalSPDisplay + " total SP")
			unallocated := c3[1].(*widget.Label)
			if r.unallocatedSP.ValueOrZero() == 0 {
				unallocated.Text = ""
			} else {
				unallocated.Text = r.unallocatedSPDisplay + " unallocated SP"
			}
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

func (a *training) filterRows(sortCol int) {
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
				x = strings.Compare(a.skillName, b.skillName)
			case 2:
				x = cmp.Compare(a.currentRemainingTime().ValueOrZero(), b.currentRemainingTime().ValueOrZero())
			case 3:
				x = cmp.Compare(a.totalRemainingCount.ValueOrZero(), b.totalRemainingCount.ValueOrZero())
			case 4:
				x = cmp.Compare(a.totalRemainingTime().ValueOrZero(), b.totalRemainingTime().ValueOrZero())
			case 5:
				x = cmp.Compare(a.totalSP.ValueOrZero(), b.totalSP.ValueOrZero())
			case 6:
				x = cmp.Compare(a.unallocatedSP.ValueOrZero(), b.unallocatedSP.ValueOrZero())
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

func (a *training) update() {
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

func (*training) fetchRows(s services) ([]trainingRow, error) {
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
		r.skill = queue.Current()
		if r.skill != nil {
			r.isActive = true
			r.statusText = "Active"
			r.statusImportance = widget.SuccessImportance
			r.skillID = r.skill.SkillID
			r.skillName = app.SkillDisplayName(r.skill.SkillName, r.skill.FinishedLevel)
			r.skillDisplay = iwidget.RichTextSegmentsFromText(r.skillName)
			r.skillFinishDate = r.skill.FinishDateEstimate()
			r.skillProgress.Set(r.skill.CompletionP())
		} else {
			r.statusText = "Inactive"
			r.statusImportance = widget.WarningImportance
			r.skillName = "N/A"
			r.skillDisplay = iwidget.RichTextSegmentsFromText(
				"Inactive",
				widget.RichTextStyle{
					ColorName: theme.ColorNameWarning,
				},
			)
		}
		r.totalFinishDate = queue.FinishDateEstimate()
		r.totalRemainingCount = queue.RemainingCount()
		r.totalRemainingCountDisplay = r.totalRemainingCount.StringFunc("N/A", func(v int) string {
			return ihumanize.Comma(v)
		})
		rows[i] = r
	}
	return rows, nil
}

func (a *training) startUpdateTicker() {
	ticker := time.NewTicker(time.Second * 60)
	go func() {
		for {
			<-ticker.C
			fyne.DoAndWait(func() {
				a.body.Refresh()
			})
		}
	}()
}

func (a *training) showDetails(r trainingRow) {
	status := widget.NewLabel(r.statusText)
	status.Importance = r.statusImportance
	var skill fyne.CanvasObject
	if r.isActive {
		skill = makeLinkLabelWithWrap(r.skillName, func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, r.skillID)
		})
	} else {
		skill = widget.NewLabel(r.skillName)
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			r.characterID,
			r.characterName,
			a.u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Status", status),
		widget.NewFormItem("Current skill", skill),
		widget.NewFormItem("Current skill progress", widget.NewLabel(r.skillProgress.StringFunc("N/A", func(v float64) string {
			return fmt.Sprintf("%.0f%%", v*100)
		}))),
		widget.NewFormItem("Current skill finished", widget.NewLabel(r.skillFinishDate.StringFunc("N/A", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("Current skill remaining", widget.NewLabel(r.currentRemainingTimeString())),
		widget.NewFormItem("Skills queued", widget.NewLabel(r.totalRemainingCountDisplay)),
		widget.NewFormItem("Queue est. finished", widget.NewLabel(r.totalFinishDate.StringFunc("?", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("Queue time remaining", widget.NewLabel(r.totalRemainingTimeString())),
		widget.NewFormItem("Skillpoints", widget.NewLabel(r.totalSPDisplay)),
		widget.NewFormItem("Unalloc. SP", widget.NewLabel(r.unallocatedSPDisplay)),
	}

	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	title := fmt.Sprintf("Training info for %s", r.characterName)
	w := a.u.makeDetailWindowWithSize("Training info", title, fyne.NewSize(500, 450), f)
	w.Show()
}
