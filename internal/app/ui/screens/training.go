package screens

import (
	"bytes"
	"cmp"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
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
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	trainingStatusActive   = "Active"
	trainingStatusInActive = "Inactive"
)

func trainingStatusOptions() []string {
	return []string{trainingStatusActive, trainingStatusInActive}
}

// Names of the training filters, used as labels on desktop and as option names on mobile.
const (
	trainingFilterStatus = "Status"
	trainingFilterTag    = "Tag"
)

// trainingFilter is the selected value of each training filter. Empty means not filtered.
type trainingFilter struct {
	status string
	tag    string
}

// match reports whether row r passes all filters.
func (f trainingFilter) match(r trainingRow) bool {
	switch {
	case f.status == trainingStatusActive && !r.isActive,
		f.status == trainingStatusInActive && r.isActive,
		f.tag != "" && !r.tags.Contains(f.tag):
		return false
	}
	return true
}

type trainingRow struct {
	characterID                int64
	characterName              string
	isActive                   bool
	isWatched                  bool
	searchTarget               string
	skill                      *app.CharacterSkillqueueItem
	skillFinishDate            optional.Optional[time.Time]
	skillID                    int64
	skillName                  string
	skillProgress              optional.Optional[float64]
	tags                       set.Set[string]
	totalFinishDate            optional.Optional[time.Time]
	totalRemainingCount        optional.Optional[int]
	totalRemainingCountDisplay string
	trainedSP                  optional.Optional[int64]
	trainedSPDisplay           string
	unallocatedSP              optional.Optional[int64]
	unallocatedSPDisplay       string
	totalSP                    optional.Optional[int64]
	totalSPDisplay             string
}

func (r trainingRow) CharacterID() int64 {
	return r.characterID
}

func (r trainingRow) CharacterName() string {
	return r.characterName
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
	v, ok := t.Value()
	if !ok {
		return optional.Optional[time.Duration]{}
	}
	d := time.Until(v)
	if d < 0 {
		return optional.New(time.Duration(0))
	}
	return optional.New(time.Duration(d))
}

type Training struct {
	widget.BaseWidget

	OnUpdate func(expired int)

	exportButton *xwidget.ContextMenuButton
	filterChip   *xwidget.FilterChipCompact // only on mobile
	filterRun    latestRun
	footer       *widget.Label
	columnSorter *xwidget.ColumnSorter[trainingRow]
	main         fyne.CanvasObject
	rows         []trainingRow
	rowsFiltered []trainingRow
	searchEntry  *xwidget.SearchEntry
	selectStatus *kxwidget.FilterChipSelect // select chips only on desktop
	selectTag    *kxwidget.FilterChipSelect
	sortChip     *kxwidget.SortChip
	u            baseUI
}

func NewTraining(u baseUI) *Training {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[trainingRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[trainingRow]{
			EIS: u.EVEImage(),
			GetEntity: func(r trainingRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			Label: "Tags",
			Width: 150,
			Update: func(r trainingRow, co fyne.CanvasObject) {
				s := strings.Join(slices.Sorted(r.tags.All()), ", ")
				co.(*xwidget.RichText).SetWithText(s)
			},
		}, {
			Label: "Current Skill",
			Width: 250,
			Update: func(r trainingRow, co fyne.CanvasObject) {
				var s string
				var c fyne.ThemeColorName
				if r.isActive {
					s = r.skillName
					c = theme.ColorNameForeground
				} else if r.isWatched {
					s = "Expired"
					c = theme.ColorNameError
				} else {
					s = "Inactive"
					c = theme.ColorNameDisabled
				}
				co.(*xwidget.RichText).SetWithText(s, widget.RichTextStyle{
					ColorName: c,
				})
			},
			Sort: func(a, b trainingRow) int {
				return strings.Compare(a.skillName, b.skillName)
			},
		}, {
			Label: "Current Time",
			Update: func(r trainingRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.currentRemainingTimeString())
			},
			Sort: func(a, b trainingRow) int {
				return cmp.Compare(
					a.currentRemainingTime().ValueOrZero(),
					b.currentRemainingTime().ValueOrZero(),
				)
			},
		}, {
			Label: "Queued",
			Update: func(r trainingRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.totalRemainingCountDisplay)
			},
			Sort: func(a, b trainingRow) int {
				return optional.Compare(a.totalRemainingCount, b.totalRemainingCount)
			},
		}, {
			Label: "Queue Time",
			Update: func(r trainingRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.totalRemainingTimeString())
			},
			Sort: func(a, b trainingRow) int {
				return optional.Compare(a.totalRemainingTime(), b.totalRemainingTime())
			},
		}, {
			Label: "Trained SP",
			Width: 100,
			Update: func(r trainingRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.trainedSPDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
			},
			Sort: func(a, b trainingRow) int {
				return optional.Compare(a.trainedSP, b.trainedSP)
			},
		}, {
			Label: "Unall. SP",
			Width: 100,
			Update: func(r trainingRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.unallocatedSPDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
			},
			Sort: func(a, b trainingRow) int {
				return optional.Compare(a.unallocatedSP, b.unallocatedSP)
			},
		}, {
			Label: "Total SP",
			Width: 100,
			Update: func(r trainingRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.totalSPDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
			},
			Sort: func(a, b trainingRow) int {
				return optional.Compare(a.totalSP, b.totalSP)
			},
		}})
	a := &Training{
		columnSorter: xwidget.NewColumnSorter(columns, "Character", xwidget.SortAsc),
		footer:       widget.NewLabel(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	placeholder := "Search characters"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	if a.u.IsMobile() {
		a.main = a.makeDataList()
	} else {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r trainingRow) {
				a.showTrainingQueueWindow(r)
			},
		)
	}
	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		a.selectStatus = kxwidget.NewFilterChipSelect(trainingFilterStatus, trainingStatusOptions(), func(string) {
			a.filterRowsAsync("")
		})
		a.selectStatus.SortDisabled = true
		a.selectTag = kxwidget.NewFilterChipSelect(trainingFilterTag, []string{}, func(string) {
			a.filterRowsAsync("")
		})
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case app.SectionCharacterSkills, app.SectionCharacterSkillqueue:
			a.updateItem(ctx, arg.CharacterID)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterChanged.AddListener(func(ctx context.Context, characterID int64) {
		a.updateItem(ctx, characterID)
	})
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.main.Refresh()
		})
	})
	return a
}

func (a *Training) CreateRenderer() fyne.WidgetRenderer {
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry)
	} else {
		filter := container.NewHBox(a.selectStatus, a.selectTag)
		topBox = container.NewBorder(nil, nil, filter, nil, a.searchEntry)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.main,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Training) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
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
			spacer := xwidget.NewSpacer(fyne.NewSize(1, 4*p))
			tags := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, status, character),
				tags,
				NewSkillQueueItem(a.u.IsMobile()),
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

			b1 := vbox[2].(*SkillQueueItem)
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
			total := b3[0].(*widget.Label)
			total.SetText(r.totalSPDisplay + " SP")
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

// MoreItems returns the list of menu items for the overflow menu.
func (a *Training) MoreItems() []*fyne.MenuItem {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Copy training to clipboard", a.copyTrainingToClipboard),
		fyne.NewMenuItem("Export training as CSV", a.saveTrainingAsCSV),
	}
	return items
}

func (a *Training) copyTrainingToClipboard() {
	copyRowsToClipboard(a.u, "training", a.rowsFiltered, func(rows []trainingRow) (string, error) {
		var buf bytes.Buffer
		if err := writeTrainingRowsToCSV(&buf, rows); err != nil {
			return "", err
		}
		return buf.String(), nil
	})
}

func (a *Training) saveTrainingAsCSV() {
	exportRowsAsCSV(a.u, "training", "training.csv", a.rowsFiltered, writeTrainingRowsToCSV)
}

func writeTrainingRowsToCSV(w io.Writer, rows []trainingRow) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"Character", "Tags", "Current Skill", "Current Remaining",
		"Queued", "Queue Time", "Trained SP", "Unallocated SP", "Total SP",
	}); err != nil {
		return err
	}
	for _, r := range rows {
		tags := strings.Join(slices.Sorted(r.tags.All()), ";")
		if err := cw.Write([]string{
			r.characterName,
			tags,
			r.skillName,
			r.currentRemainingTimeString(),
			r.totalRemainingCountDisplay,
			r.totalRemainingTimeString(),
			r.trainedSPDisplay,
			r.unallocatedSPDisplay,
			r.totalSPDisplay,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *Training) currentFilter() trainingFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return trainingFilter{
			status: s[trainingFilterStatus],
			tag:    s[trainingFilterTag],
		}
	}
	return trainingFilter{
		status: a.selectStatus.Selected,
		tag:    a.selectTag.Selected,
	}
}

func (a *Training) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r trainingRow) bool {
			return !filter.match(r)
		})
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

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoiceOrdered(trainingFilterStatus, trainingStatusOptions()),
					xwidget.NewFilterOptionMultiChoice(trainingFilterTag, tagOptions),
				)
			} else {
				a.selectTag.SetOptions(tagOptions)
			}
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	})
}

func (a *Training) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh training UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync("")
		a.refreshOnUpdate()
	})
}

func (a *Training) updateItem(ctx context.Context, characterID int64) {
	logErr := func(err error) {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Training: Failed to update item", "characterID", characterID, "error", err)
	}
	c, err := a.u.Character().GetCharacter(ctx, characterID)
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
		a.filterRowsAsync("")
		a.refreshOnUpdate()
	})
}

func (a *Training) refreshOnUpdate() {
	var expired int
	for _, r := range a.rows {
		if !r.isActive && r.isWatched {
			expired++
		}
	}
	if a.OnUpdate != nil {
		a.OnUpdate(expired)
	}
}

func (a *Training) fetchRows(ctx context.Context) ([]trainingRow, error) {
	characters, err := a.u.Character().ListCharacters(ctx)
	if err != nil {
		return nil, err
	}
	var rows []trainingRow
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

func (a *Training) fetchRow(ctx context.Context, c *app.Character) (trainingRow, error) {
	var z trainingRow
	if c == nil || c.EveCharacter == nil {
		return z, app.ErrInvalid
	}
	tags, err := a.u.Character().ListTagsForCharacter(ctx, c.ID)
	if err != nil {
		return z, err
	}
	r := trainingRow{
		characterID:   c.ID,
		characterName: c.EveCharacter.Name,
		searchTarget:  strings.ToLower(c.EveCharacter.Name),
		isWatched:     c.IsTrainingWatched,
		tags:          tags,
		trainedSP:     c.TrainedSP,
		trainedSPDisplay: c.TrainedSP.StringFunc("?", func(v int64) string {
			return humanize.Comma(v)
		}),
		unallocatedSP: c.UnallocatedSP,
		unallocatedSPDisplay: c.UnallocatedSP.StringFunc("?", func(v int64) string {
			return humanize.Comma(v)
		}),
		totalSP: optional.SumNonEmpty(c.TrainedSP, c.UnallocatedSP),
	}
	r.totalSPDisplay = r.totalSP.StringFunc("?", func(v int64) string {
		return humanize.Comma(v)
	})
	queue := app.NewCharacterSkillqueue()
	if err := queue.Update(ctx, a.u.Character(), c.ID); err != nil {
		return z, err
	}
	r.skill = queue.Active()
	if r.skill != nil {
		r.isActive = true
		r.skillID = r.skill.SkillID
		r.skillName = ui.SkillDisplayName(r.skill.SkillName, r.skill.FinishedLevel)
		r.skillFinishDate = r.skill.FinishDate
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

func (a *Training) showTrainingQueueWindow(r trainingRow) {
	windowID := fmt.Sprintf("skillqueue-%d", r.characterID)
	w, ok, onClosed := a.u.GetOrCreateWindowWithOnClosed(windowID, "Skill Queue", r.characterName)
	if !ok {
		w.Show()
		return
	}
	go func() {
		ctx := context.Background()
		c, err := a.u.Character().GetCharacter(ctx, r.characterID)
		if err != nil {
			fyne.Do(func() {
				a.u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to fetch character", err, a.u.IsDeveloperMode(), a.u.MainWindow())
			})
			return
		}
		fyne.Do(func() {
			sq := NewQueueWithCharacter(a.u, c)
			go sq.Update(ctx)
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Content:        sq,
				EnableTooltips: true,
				MinSize:        fyne.NewSize(800, 450),
				Title:          fmt.Sprintf("Skill Queue for %s", r.characterName),
				Window:         w,
			})
			w.SetOnClosed(func() {
				if onClosed != nil {
					onClosed()
				}
				sq.Stop()
			})
			w.Show()
		})
	}()
}
