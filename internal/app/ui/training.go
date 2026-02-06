package ui

import (
	"cmp"
	"context"
	"errors"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
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
	isWatched                  bool
	searchTarget               string
	skill                      *app.CharacterSkillqueueItem
	skillFinishDate            optional.Optional[time.Time]
	skillID                    int32
	skillName                  string
	skillProgress              optional.Optional[float64]
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
	return timeUntil(r.skillFinishDate)
}

func (r trainingRow) currentRemainingTimeString() string {
	return r.remainingTimeString(r.currentRemainingTime())
}

func (r trainingRow) totalRemainingTime() optional.Optional[time.Duration] {
	return timeUntil(r.totalFinishDate)
}

func (r trainingRow) totalRemainingTimeString() string {
	return r.remainingTimeString(r.totalRemainingTime())
}

func (r trainingRow) remainingTimeString(d optional.Optional[time.Duration]) string {
	if !r.isActive {
		return "N/A"
	}
	return d.StringFunc("?", func(v time.Duration) string {
		return ihumanize.Duration(v)
	})
}

func (r trainingRow) status() (string, widget.Importance) {
	if r.isActive {
		return "Active", widget.SuccessImportance
	}
	if r.isWatched {
		return "Expired", widget.DangerImportance
	}
	return "Inactive", widget.LowImportance
}

func timeUntil(t optional.Optional[time.Time]) optional.Optional[time.Duration] {
	if t.IsEmpty() {
		return optional.Optional[time.Duration]{}
	}
	d := time.Until(t.MustValue())
	if d < 0 {
		return optional.New(time.Duration(0))
	}
	return optional.New(time.Duration(d))
}

type training struct {
	widget.BaseWidget

	onUpdate func(expired int)

	bottom       *widget.Label
	columnSorter *iwidget.ColumnSorter[trainingRow]
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
	trainingColName = iota + 1
	trainingColTags
	trainingColCurrentSkill
	trainingColCurrentRemaining
	trainingColQueuedCount
	trainingColQueuedRemaining
	trainingColSkillpoints
	trainingColUnallocatedSP
)

func newTraining(u *baseUI) *training {
	columns := iwidget.NewDataColumns([]iwidget.DataColumn[trainingRow]{{
		ID:    trainingColName,
		Label: "Character",
		Width: 220,
		Create: func() fyne.CanvasObject {
			avatar := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			avatar.CornerRadius = app.IconUnitSize / 2
			name := widget.NewLabel("Template")
			return container.NewHBox(avatar, name)
		},
		Update: func(r trainingRow, co fyne.CanvasObject) {
			hbox := co.(*fyne.Container).Objects
			u.eis.CharacterPortraitAsync(r.characterID, app.IconPixelSize, func(r fyne.Resource) {
				x := hbox[0].(*canvas.Image)
				x.Resource = r
				x.Refresh()
			})
			hbox[1].(*widget.Label).SetText(r.characterName)
		},
		Sort: func(a, b trainingRow) int {
			return xstrings.CompareIgnoreCase(a.characterName, b.characterName)
		},
	}, {
		ID:    trainingColTags,
		Label: "Tags",
		Width: 150,
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			x.Text = strings.Join(slices.Sorted(r.tags.All()), ", ")
			x.Alignment = fyne.TextAlignLeading
			x.Refresh()

		},
	}, {
		ID:    trainingColCurrentSkill,
		Label: "Current Skill",
		Width: 250,
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			var s string
			var i widget.Importance
			if r.isActive {
				s = r.skillName
			} else if r.isWatched {
				s = "Expired"
				i = widget.DangerImportance
			} else {
				s = "Inactive"
				i = widget.LowImportance
			}
			x.Text = s
			x.Importance = i
			x.Alignment = fyne.TextAlignLeading
			x.Refresh()
		},
		Sort: func(a, b trainingRow) int {
			return strings.Compare(a.skillName, b.skillName)
		},
	}, {
		ID:    trainingColCurrentRemaining,
		Label: "Current Time",
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			x.Text = r.currentRemainingTimeString()
			x.Alignment = fyne.TextAlignLeading
			x.Refresh()
		},
		Sort: func(a, b trainingRow) int {
			return cmp.Compare(a.currentRemainingTime().ValueOrZero(), b.currentRemainingTime().ValueOrZero())
		},
	}, {
		ID:    trainingColQueuedCount,
		Label: "Queued",
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			x.Text = r.totalRemainingCountDisplay
			x.Alignment = fyne.TextAlignLeading
			x.Refresh()
		},
		Sort: func(a, b trainingRow) int {
			return cmp.Compare(a.totalRemainingCount.ValueOrZero(), b.totalRemainingCount.ValueOrZero())
		},
	}, {
		ID:    trainingColQueuedRemaining,
		Label: "Queue Time",
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			x.Text = r.totalRemainingTimeString()
			x.Alignment = fyne.TextAlignLeading
			x.Refresh()
		},
		Sort: func(a, b trainingRow) int {
			return cmp.Compare(a.totalRemainingTime().ValueOrZero(), b.totalRemainingTime().ValueOrZero())
		},
	}, {
		ID:    trainingColSkillpoints,
		Label: "SP",
		Width: 100,
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			x.Text = r.totalSPDisplay
			x.Alignment = fyne.TextAlignTrailing
			x.Refresh()
		},
		Sort: func(a, b trainingRow) int {
			return cmp.Compare(a.totalSP.ValueOrZero(), b.totalSP.ValueOrZero())
		},
	}, {
		ID:    trainingColUnallocatedSP,
		Label: "Unall.",
		Width: 100,
		Update: func(r trainingRow, co fyne.CanvasObject) {
			x := co.(*widget.Label)
			x.Text = r.unallocatedSPDisplay
			x.Alignment = fyne.TextAlignTrailing
			x.Refresh()
		},
		Sort: func(a, b trainingRow) int {
			return cmp.Compare(a.unallocatedSP.ValueOrZero(), b.unallocatedSP.ValueOrZero())
		},
	}})
	a := &training{
		bottom:       widget.NewLabel(""),
		columnSorter: iwidget.NewColumnSorter(columns, trainingColName, iwidget.SortAsc),
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
	if a.u.isMobile {
		a.main = a.makeDataList()
	} else {
		a.main = iwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				return widget.NewLabel("Template")
			},
			a.columnSorter,
			a.filterRows,
			func(_ int, r trainingRow) {
				a.showTrainingQueueWindow(r)
			},
		)
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

	// Signals
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterSkills, app.SectionCharacterSkillqueue:
			a.updateItem(ctx, arg.characterID)
		}
	})
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update()
	})
	a.u.characterChanged.AddListener(func(ctx context.Context, characterID int32) {
		a.updateItem(ctx, characterID)
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
	if a.u.isMobile {
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
				newSkillQueueItem(a.u.isMobile),
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
			status.Text, status.Importance = r.status()
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
	selectStatus := a.selectStatus.Selected
	selectTag := a.selectTag.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if selectStatus != "" {
			rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
				switch selectStatus {
				case trainingStatusActive:
					return !r.isActive
				case trainingStatusInActive:
					return r.isActive
				}
				return true
			})
		}
		if selectTag != "" {
			rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
				return !r.tags.Contains(selectTag)
			})
		}
		// search filter
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r trainingRow) set.Set[string] {
			return r.tags
		})...).All())

		// Queue UI changes
		fyne.Do(func() {
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	}()
}

func (a *training) update() {
	ctx := context.Background()
	rows := make([]trainingRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, err := a.fetchRows(ctx)
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
		a.updateOnUpdate()
	})
}

func (a *training) updateItem(ctx context.Context, characterID int32) {
	logErr := func(err error) {
		slog.Error("Training: Failed to update item", "characterID", characterID, "error", err)
	}
	c, err := a.u.cs.GetCharacter(ctx, characterID)
	if err != nil {
		logErr(err)
		return
	}
	r, err := a.fetchRow(ctx, c)
	if err != nil {
		logErr(err)
		return
	}
	fyne.Do(func() {
		id := slices.IndexFunc(a.rows, func(x trainingRow) bool {
			return x.characterID == characterID
		})
		if id == -1 {
			return
		}
		a.rows[id] = r
		a.updateOnUpdate()
		a.filterRows(-1)
	})
}

func (a *training) updateOnUpdate() {
	var expired int
	for _, r := range a.rows {
		if !r.isActive && r.isWatched {
			expired++
		}
	}
	if a.onUpdate != nil {
		a.onUpdate(expired)
	}
}

func (a *training) fetchRows(ctx context.Context) ([]trainingRow, error) {
	characters, err := a.u.cs.ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]trainingRow, 0)
	for _, c := range characters {
		r, err := a.fetchRow(ctx, c)
		if errors.Is(err, app.ErrInvalid) {
			continue
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *training) fetchRow(ctx context.Context, c *app.Character) (trainingRow, error) {
	var z trainingRow
	if c == nil || c.EveCharacter == nil {
		return z, app.ErrInvalid
	}
	tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
	if err != nil {
		return z, err
	}
	r := trainingRow{
		characterID:   c.ID,
		characterName: c.EveCharacter.Name,
		searchTarget:  strings.ToLower(c.EveCharacter.Name),
		isWatched:     c.IsTrainingWatched,
		tags:          tags,
		totalSP:       c.TotalSP,
		totalSPDisplay: c.TotalSP.StringFunc("?", func(v int) string {
			return humanize.Comma(int64(v))
		}),
		unallocatedSP: c.UnallocatedSP,
		unallocatedSPDisplay: c.UnallocatedSP.StringFunc("?", func(v int) string {
			return humanize.Comma(int64(v))
		}),
	}
	queue := app.NewCharacterSkillqueue()
	if err := queue.Update(ctx, a.u.cs, c.ID); err != nil {
		return z, err
	}
	r.skill = queue.Active()
	if r.skill != nil {
		r.isActive = true
		r.skillID = r.skill.SkillID
		r.skillName = app.SkillDisplayName(r.skill.SkillName, r.skill.FinishedLevel)
		r.skillFinishDate.Set(r.skill.FinishDate)
		r.skillProgress.Set(r.skill.CompletionP())
	} else {
		r.skillName = "N/A"
	}
	r.totalFinishDate = queue.FinishDate()
	r.totalRemainingCount = queue.RemainingCount()
	r.totalRemainingCountDisplay = r.totalRemainingCount.StringFunc("N/A", func(v int) string {
		return ihumanize.Comma(v)
	})
	return r, nil
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
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		sq.stop()
	})
	subTitle := fmt.Sprintf("Skill Queue for %s", r.characterName)
	setDetailWindow(detailWindowParams{
		content:        sq,
		enableTooltips: true,
		minSize:        fyne.NewSize(800, 450),
		title:          subTitle,
		window:         w,
	})
	w.Show()
}
