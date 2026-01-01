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
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	trainingStatusActive   = "Active"
	trainingStatusInActive = "Inactive"
)

type trainingRow struct {
	characterID                int32
	characterName              string
	isActive                   bool
	searchTarget               string
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
		return optional.New(time.Duration(0))
	}
	return optional.New(time.Duration(d))
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

	bottom       *widget.Label
	columnSorter *iwidget.ColumnSorter
	main         fyne.CanvasObject
	rows         []trainingRow
	rowsFiltered []trainingRow
	search       *widget.Entry
	selectStatus *kxwidget.FilterChipSelect
	selectTag    *kxwidget.FilterChipSelect
	sortButton   *iwidget.SortButton
	u            *baseUI
}

const (
	trainingColName             = 0
	trainingColTags             = 1
	trainingColCurrentSkill     = 2
	trainingColCurrentRemaining = 3
	trainingColQueuedCount      = 4
	trainingColQueuedRemaining  = 5
	trainingColSkillpoints      = 6
	trainingColUnallocatedSP    = 7
)

func newTraining(u *baseUI) *training {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   trainingColName,
		Label: "Name",
		Width: columnWidthEntity,
	}, {
		Col:    trainingColTags,
		Label:  "Tags",
		Width:  150,
		NoSort: true,
	}, {
		Col:   trainingColCurrentSkill,
		Label: "Current Skill",
		Width: 250,
	}, {
		Col:   trainingColCurrentRemaining,
		Label: "Current Time",
	}, {
		Col:   trainingColQueuedCount,
		Label: "Queued",
	}, {
		Col:   trainingColQueuedRemaining,
		Label: "Queue Time",
	}, {
		Col:   trainingColSkillpoints,
		Label: "SP",
		Width: 100,
	}, {
		Col:   trainingColUnallocatedSP,
		Label: "Unall.",
		Width: 100,
	}})
	a := &training{
		bottom:       widget.NewLabel(""),
		columnSorter: headers.NewColumnSorter(trainingColName, iwidget.SortAsc),
		rows:         make([]trainingRow, 0),
		rowsFiltered: make([]trainingRow, 0),
		search:       widget.NewEntry(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRows(-1)
	})
	a.search.OnChanged = func(s string) {
		a.filterRows(-1)
	}
	a.search.PlaceHolder = "Search characters"
	makeCell := func(col int, r trainingRow) []widget.RichTextSegment {
		switch col {
		case trainingColName:
			return iwidget.RichTextSegmentsFromText(r.characterName)
		case trainingColTags:
			s := strings.Join(slices.Sorted(r.tags.All()), ", ")
			return iwidget.RichTextSegmentsFromText(s)
		case trainingColCurrentSkill:
			return r.skillDisplay
		case trainingColCurrentRemaining:
			return iwidget.RichTextSegmentsFromText(r.currentRemainingTimeString())
		case trainingColQueuedCount:
			return iwidget.RichTextSegmentsFromText(r.totalRemainingCountDisplay)
		case trainingColQueuedRemaining:
			return iwidget.RichTextSegmentsFromText(r.totalRemainingTimeString())
		case trainingColSkillpoints:
			return iwidget.RichTextSegmentsFromText(
				r.totalSPDisplay,
				widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				},
			)
		case trainingColUnallocatedSP:
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
		a.main = iwidget.MakeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			func(_ int, r trainingRow) {
				a.showTrainingQueueWindow(r)
			},
		)
	} else {
		a.main = a.makeDataList()
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
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterSkills, app.SectionCharacterSkillqueue:
			a.update()
		}
	})
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.main.Refresh()
		})
	})
	return a
}

func (a *training) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectStatus, a.selectTag)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(
			a.search,
			container.NewHScroll(filter),
		),
		nil,
		nil,
		nil,
		a.main,
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
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(1, 4*p))
			tags := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, status, character),
				tags,
				newSkillQueueItem(),
				container.NewBorder(nil, nil, nil, queueRemaining, queueCount),
				container.NewBorder(nil, nil, nil, unallocatedSP, totalSP),
				spacer,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			vbox := co.(*fyne.Container).Objects

			b0 := vbox[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.characterName)
			status := b0[1].(*widget.Label)
			status.Text = r.statusText
			status.Importance = r.statusImportance
			status.Refresh()

			s := strings.Join(slices.Sorted(r.tags.All()), ", ")
			if s == "" {
				s = "-"
			}
			vbox[1].(*widget.Label).SetText(s)

			b1 := vbox[2].(*skillQueueItem)
			b1.Set(r.skill)

			b2 := vbox[3].(*fyne.Container).Objects
			queueCount := b2[0].(*widget.Label)
			queueRemaining := b2[1].(*widget.Label)
			if r.totalRemainingCount.IsEmpty() {
				queueCount.Text = "N/A"
				queueRemaining.Text = ""
			} else {
				queueCount.Text = r.totalRemainingCountDisplay + " skills queued"
				queueRemaining.Text = r.totalRemainingTimeString()
			}
			queueCount.Refresh()
			queueRemaining.Refresh()

			b3 := vbox[4].(*fyne.Container).Objects
			b3[0].(*widget.Label).SetText(r.totalSPDisplay + " SP")
			unallocated := b3[1].(*widget.Label)
			if r.unallocatedSP.ValueOrZero() == 0 {
				unallocated.Text = ""
			} else {
				unallocated.Text = fmt.Sprintf("unalloc: %s SP", r.unallocatedSPDisplay)
			}
			unallocated.Refresh()
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.showTrainingQueueWindow(r)
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
	// search filter
	if search := strings.ToLower(a.search.Text); search != "" {
		rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
			return !strings.Contains(r.searchTarget, search)
		})
	}
	// sort
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b trainingRow) int {
			var x int
			switch sortCol {
			case trainingColName:
				x = xstrings.CompareIgnoreCase(a.characterName, b.characterName)
			case trainingColCurrentRemaining:
				x = cmp.Compare(a.currentRemainingTime().ValueOrZero(), b.currentRemainingTime().ValueOrZero())
			case trainingColCurrentSkill:
				x = strings.Compare(a.skillName, b.skillName)
			case trainingColQueuedCount:
				x = cmp.Compare(a.totalRemainingCount.ValueOrZero(), b.totalRemainingCount.ValueOrZero())
			case trainingColQueuedRemaining:
				x = cmp.Compare(a.totalRemainingTime().ValueOrZero(), b.totalRemainingTime().ValueOrZero())
			case trainingColSkillpoints:
				x = cmp.Compare(a.totalSP.ValueOrZero(), b.totalSP.ValueOrZero())
			case trainingColUnallocatedSP:
				x = cmp.Compare(a.unallocatedSP.ValueOrZero(), b.unallocatedSP.ValueOrZero())
			}
			if dir == iwidget.SortAsc {
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
	a.main.Refresh()
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
			characterID:   c.ID,
			characterName: c.EveCharacter.Name,
			searchTarget:  strings.ToLower(c.EveCharacter.Name),
			totalSP:       c.TotalSP,
			totalSPDisplay: c.TotalSP.StringFunc("?", func(v int) string {
				return humanize.Comma(int64(v))
			}),
			unallocatedSP: c.UnallocatedSP,
			unallocatedSPDisplay: c.UnallocatedSP.StringFunc("?", func(v int) string {
				return humanize.Comma(int64(v))
			}),
		}
		tags, err := s.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		queue := app.NewCharacterSkillqueue()
		if err := queue.Update(ctx, s.cs, c.ID); err != nil {
			return nil, err
		}
		r.skill = queue.Active()
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
		r.totalFinishDate = queue.FinishDate()
		r.totalRemainingCount = queue.RemainingCount()
		r.totalRemainingCountDisplay = r.totalRemainingCount.StringFunc("N/A", func(v int) string {
			return ihumanize.Comma(v)
		})
		rows[i] = r
	}
	return rows, nil
}

func (a *training) showTrainingQueueWindow(r trainingRow) {
	w, ok, onClosed := a.u.getOrCreateWindowWithOnClosed(fmt.Sprintf("skillqueue-%d", r.characterID), "Skill Queue", r.characterName)
	if !ok {
		w.Show()
		return
	}
	c, err := a.u.cs.GetCharacter(context.Background(), r.characterID)
	if err != nil {
		a.u.showErrorDialog("Failed to fetch character", err, a.u.MainWindow())
		return
	}
	sq := newCharacterSkillQueueWithCharacter(a.u, c)
	sq.update()
	sq.start()
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		sq.stop()
	})
	subTitle := fmt.Sprintf("Skill Queue for %s", r.characterName)
	setDetailWindow(detailWindowParams{
		content: sq,
		minSize: fyne.NewSize(800, 450),
		title:   subTitle,
		window:  w,
	})
	w.Show()
}
