package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Options for industry job select widgets
const (
	industryActivityCopying          = "Copying"
	industryActivityInvention        = "Invention"
	industryActivityManufacturing    = "Manufacturing"
	industryActivityMaterialResearch = "Material efficiency research"
	industryActivityReaction         = "Reactions"
	industryActivityTimeResearch     = "Time efficiency research"
	industryInstallerCorpmates       = "Installed by corpmates"
	industryInstallerMe              = "Installed by me"
	industryOwnerCorp                = "Owned by corp"
	industryOwnerMe                  = "Owned by me"
	industryStatusActive             = "All active jobs"
	industryStatusHalted             = "Halted"
	industryStatusHistory            = "History"
	industryStatusInProgress         = "In progress"
	industryStatusReady              = "Ready for delivery"
)

// Names of the industry job filters, used as labels on desktop and as option names on mobile.
const (
	industryJobsFilterActivity  = "Activity"
	industryJobsFilterInstaller = "Installer"
	industryJobsFilterOwner     = "Owner"
	industryJobsFilterTag       = "Tag"
)

// industryJobsFilter is the selected value of each industry job filter. Empty means not filtered.
type industryJobsFilter struct {
	activity  string
	installer string
	owner     string
	status    string
	tag       string
}

// match reports whether row r passes all filters.
func (f industryJobsFilter) match(r industryJobRow) bool {
	switch {
	case f.installer == industryInstallerMe && !r.isInstallerMe,
		f.installer == industryInstallerCorpmates && r.isInstallerMe,
		f.owner == industryOwnerMe && !r.isOwnerMe,
		f.owner == industryOwnerCorp && r.isOwnerMe,
		f.tag != "" && !r.tags.Contains(f.tag):
		return false
	}
	switch f.activity {
	case industryActivityCopying:
		if r.activity != app.Copying {
			return false
		}
	case industryActivityInvention:
		if r.activity != app.Invention {
			return false
		}
	case industryActivityManufacturing:
		if r.activity != app.Manufacturing {
			return false
		}
	case industryActivityMaterialResearch:
		if r.activity != app.MaterialEfficiencyResearch {
			return false
		}
	case industryActivityReaction:
		if r.activity != app.Reactions1 && r.activity != app.Reactions2 {
			return false
		}
	case industryActivityTimeResearch:
		if r.activity != app.TimeEfficiencyResearch {
			return false
		}
	}
	switch status := r.statusCalculated(); f.status {
	case industryStatusActive:
		return status.IsActive()
	case industryStatusInProgress:
		return status == app.JobActive
	case industryStatusReady:
		return status == app.JobReady
	case industryStatusHalted:
		return status == app.JobPaused
	case industryStatusHistory:
		return !status.IsActive()
	}
	return true
}

// industryJobRow represents a job row in the list widgets.
// It combines character and corporation jobs and has precalculated fields for filters.
type industryJobRow struct {
	activity           app.IndustryActivity
	blueprintID        int64
	blueprintType      *app.EntityShort
	blueprintTypeName  string
	completedCharacter optional.Optional[*app.EveEntity]
	completedDate      optional.Optional[time.Time]
	cost               optional.Optional[float64]
	duration           int
	endDate            time.Time
	installer          *app.EveEntity
	isInstallerMe      bool
	isOwnerMe          bool
	jobID              int64
	licensedRuns       optional.Optional[int]
	location           *app.EveLocationShort
	owner              *app.EveEntity
	pauseDate          optional.Optional[time.Time]
	probability        optional.Optional[float32]
	productType        optional.Optional[*app.EntityShort]
	runs               int
	startDate          time.Time
	status             app.IndustryJobStatus
	successfulRuns     optional.Optional[int64]
	tags               set.Set[string]
}

func (r industryJobRow) remaining() time.Duration {
	return time.Until(r.endDate)
}

// statusCalculated returns the status as ready when the timer has elapsed.
func (r industryJobRow) statusCalculated() app.IndustryJobStatus {
	if r.status == app.JobActive && !r.endDate.IsZero() && r.endDate.Before(time.Now()) {
		return app.JobReady
	}
	return r.status
}

func (r industryJobRow) statusDisplay() []widget.RichTextSegment {
	status := r.statusCalculated()
	if status == app.JobActive {
		return xwidget.RichTextSegmentsFromText(ihumanize.Duration(r.remaining()), widget.RichTextStyle{
			ColorName: theme.ColorNameForeground,
		})
	}
	return xwidget.RichTextSegmentsFromText(status.Display(), widget.RichTextStyle{
		ColorName: status.Color(),
	})
}

type IndustryJobs struct {
	widget.BaseWidget

	OnUpdate func(count int)

	body            fyne.CanvasObject
	filterChip      *xwidget.FilterChipCompact // only on mobile
	filterRun       latestRun
	footer          *widget.Label
	columnSorter    *xwidget.ColumnSorter[industryJobRow]
	corporation     atomic.Pointer[app.Corporation]
	forCorporation  bool
	rows            []industryJobRow
	rowsFiltered    []industryJobRow
	searchEntry     *xwidget.SearchEntry
	selectActivity  *kxwidget.FilterChipSelect // select chips only on desktop
	selectInstaller *kxwidget.FilterChipSelect
	selectOwner     *kxwidget.FilterChipSelect
	selectStatus    *kxwidget.FilterChipSelect // mode chip on all platforms
	selectTag       *kxwidget.FilterChipSelect
	sortChip        *kxwidget.SortChip
	u               baseUI
}

func NewJobsForOverview(u baseUI) *IndustryJobs {
	return newIndustryJobs(u, false)
}

func NewJobsForCorporation(u baseUI) *IndustryJobs {
	return newIndustryJobs(u, true)
}

func newIndustryJobs(u baseUI, forCorporation bool) *IndustryJobs {
	corporationIcon := theme.NewThemedResource(icons.StarCircleOutlineSvg)
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[industryJobRow]{{
		Label: "Blueprint",
		Width: 250,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.blueprintType.Name, b.blueprintType.Name)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(ui.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(r.blueprintTypeName)
			x := border[1].(*canvas.Image)
			u.EVEImage().InventoryTypeBPOAsync(r.blueprintType.ID, ui.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		Label: "Status",
		Width: 100,
		Sort: func(a, b industryJobRow) int {
			return cmp.Compare(a.remaining(), b.remaining())
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.statusDisplay())
		},
	}, {
		Label: "Runs",
		Width: 75,
		Sort: func(a, b industryJobRow) int {
			return cmp.Compare(a.runs, b.runs)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(
				ihumanize.Comma(r.runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		},
	}, {
		Label: "Activity",
		Width: 200,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.activity.String(), b.activity.String())
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.activity.Display())
		},
	}, {
		Label: "End date",
		Width: ui.ColumnWidthDateTime,
		Sort: func(a, b industryJobRow) int {
			return a.endDate.Compare(b.endDate)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.endDate.Format(app.DateTimeFormat))
		},
	}, {
		Label: "Location",
		Width: ui.ColumnWidthLocation,
		Sort: func(a, b industryJobRow) int {
			return optional.Compare(a.location.Name, b.location.Name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.location.Name.ValueOrZero())
		},
	}, {
		Label: "Owner",
		Width: 250,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.owner.Name, b.owner.Name)
		},
		Create: func() fyne.CanvasObject {
			icon := widget.NewIcon(icons.BlankSvg)
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(r.owner.Name)
			icon := border[1].(*widget.Icon)
			if r.owner.IsCharacter() {
				icon.SetResource(theme.AccountIcon())
			} else {
				icon.SetResource(corporationIcon)
			}
		},
	}, {
		Label: "Installer",
		Width: ui.ColumnWidthEntity,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.installer.Name, b.installer.Name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.installer.Name)
		},
	}})
	a := &IndustryJobs{
		footer:         ui.NewLabelWithWrapping(""),
		columnSorter:   xwidget.NewColumnSorter(columns, "End date", xwidget.SortDesc),
		forCorporation: forCorporation,
		u:              u,
	}
	a.ExtendBaseWidget(a)

	if a.u.IsMobile() {
		a.body = a.makeDataList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync, func(_ int, r industryJobRow) {
				showIndustryJobWindow(a.u, r)
			})
	}

	placeholder := "Search blueprints"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	a.selectStatus = kxwidget.NewFilterChipSelect("", []string{
		industryStatusActive,
		industryStatusInProgress,
		industryStatusReady,
		industryStatusHalted,
		industryStatusHistory,
	}, func(_ string) {
		a.filterRowsAsync("")
	})
	a.selectStatus.Selected = industryStatusActive
	a.selectStatus.SortDisabled = true

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		makeSelect := func(label string, options ...string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelect(label, options, func(string) {
				a.filterRowsAsync("")
			})
		}
		a.selectTag = makeSelect(industryJobsFilterTag)
		a.selectOwner = makeSelect(industryJobsFilterOwner, industryJobsOwnerOptions()...)
		a.selectActivity = makeSelect(industryJobsFilterActivity, industryJobsActivityOptions()...)
		a.selectInstaller = makeSelect(industryJobsFilterInstaller, industryJobsInstallerOptions()...)
		if !forCorporation {
			a.selectInstaller.Selected = industryInstallerMe // hidden filter outside corporation mode
		}
	}

	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	}, "Owner", "Installer")

	// signals
	if forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			fyne.Do(func() {
				a.searchEntry.ClearSilent()
				a.selectStatus.Selected = industryStatusActive
				a.selectStatus.Refresh()
				if a.filterChip != nil {
					a.filterChip.ResetSilent()
					return
				}
				clearSelectsSilent(a.selectActivity, a.selectInstaller, a.selectOwner, a.selectTag)
			})
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section == app.SectionCorporationIndustryJobs {
				a.update(ctx)
			}
		})
	} else {
		a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if arg.Section == app.SectionCharacterIndustryJobs {
				a.update(ctx)
			}
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if arg.Section == app.SectionCorporationIndustryJobs {
				a.update(ctx)
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
	}
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.body.Refresh()
		})
	})
	return a
}

func (a *IndustryJobs) CreateRenderer() fyne.WidgetRenderer {
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(
			container.NewHBox(a.selectStatus),
			container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry),
		)
	} else {
		filter := container.NewHBox(a.selectStatus, a.selectOwner, a.selectActivity)
		if a.forCorporation {
			filter.Add(a.selectInstaller)
		} else {
			filter.Add(a.selectTag)
		}
		topBox = container.NewBorder(
			nil,
			nil,
			filter,
			nil,
			a.searchEntry,
		)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func industryJobsActivityOptions() []string {
	return []string{
		industryActivityManufacturing,
		industryActivityMaterialResearch,
		industryActivityTimeResearch,
		industryActivityCopying,
		industryActivityInvention,
		industryActivityReaction,
	}
}

func industryJobsInstallerOptions() []string {
	return []string{industryInstallerMe, industryInstallerCorpmates}
}

func industryJobsOwnerOptions() []string {
	return []string{industryOwnerMe, industryOwnerCorp}
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop. Status comes from its mode chip on both.
func (a *IndustryJobs) currentFilter() industryJobsFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		installer := s[industryJobsFilterInstaller]
		if !a.forCorporation {
			installer = industryInstallerMe // hidden filter outside corporation mode, as on desktop
		}
		return industryJobsFilter{
			activity:  s[industryJobsFilterActivity],
			installer: installer,
			owner:     s[industryJobsFilterOwner],
			status:    a.selectStatus.Selected,
			tag:       s[industryJobsFilterTag],
		}
	}
	return industryJobsFilter{
		activity:  a.selectActivity.Selected,
		installer: a.selectInstaller.Selected,
		owner:     a.selectOwner.Selected,
		status:    a.selectStatus.Selected,
		tag:       a.selectTag.Selected,
	}
}

func (a *IndustryJobs) makeDataList() *xwidget.StripedList {
	statusMap := map[app.IndustryJobStatus]fyne.Resource{
		app.JobDelivered: theme.NewThemedResource(icons.IndydeliveredSvg),
		app.JobPaused:    theme.NewWarningThemedResource(icons.IndyhaltedSvg),
		app.JobReady:     theme.NewSuccessThemedResource(icons.IndyreadySvg),
		app.JobCancelled: theme.NewErrorThemedResource(icons.IndycanceledSvg),
	}
	activityMap := map[app.IndustryActivity]fyne.Resource{
		app.Manufacturing:              theme.NewThemedResource(icons.IndymanufacturingSvg),
		app.MaterialEfficiencyResearch: theme.NewThemedResource(icons.IndymaterialresearchSvg),
		app.TimeEfficiencyResearch:     theme.NewThemedResource(icons.IndytimeresearchSvg),
		app.Copying:                    theme.NewThemedResource(icons.IndycopyingSvg),
		app.Invention:                  theme.NewThemedResource(icons.IndyinventionSvg),
		app.Reactions2:                 theme.NewThemedResource(icons.IndyreactionsSvg),
	}
	var l *xwidget.StripedList
	l = xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("Template")
			title.TextStyle.Bold = true
			title.Wrapping = fyne.TextWrapWord
			status := xwidget.NewRichText()
			location := widget.NewLabel("Template")
			location.Wrapping = fyne.TextWrapWord
			completed := widget.NewLabel("Template")
			p := theme.Padding()
			activityIcon := widget.NewIcon(icons.BlankSvg)
			statusIcon := widget.NewIcon(icons.BlankSvg)
			spacer := xwidget.NewSpacer(fyne.NewSize(1, p))
			return container.NewBorder(
				nil,
				nil,
				container.NewVBox(spacer, activityIcon),
				container.NewStack(status, container.NewVBox(spacer, statusIcon)),
				container.New(layout.NewCustomPaddedVBoxLayout(-p),
					title,
					location,
					completed,
				),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) || id < 0 {
				return
			}
			j := a.rowsFiltered[id]
			c1 := co.(*fyne.Container).Objects
			c2 := c1[0].(*fyne.Container).Objects
			title := fmt.Sprintf("%s x%s", j.blueprintType.Name, ihumanize.Comma(j.runs))
			c2[0].(*widget.Label).SetText(title)
			c2[1].(*widget.Label).SetText(j.location.Name.ValueOrFallback("?"))

			r, ok := activityMap[j.activity]
			if !ok {
				r = theme.NewThemedResource(icons.QuestionmarkSvg)
			}
			c1[1].(*fyne.Container).Objects[1].(*widget.Icon).SetResource(r)

			statusStack := c1[2].(*fyne.Container).Objects
			status := j.statusCalculated()
			if status == app.JobActive {
				statusStack[0].(*xwidget.RichText).Set(j.statusDisplay())
				statusStack[0].Show()
				statusStack[1].Hide()
			} else {
				r, ok := statusMap[status]
				if !ok {
					r = theme.NewThemedResource(icons.QuestionmarkSvg)
				}
				statusStack[1].(*fyne.Container).Objects[1].(*widget.Icon).SetResource(r)
				statusStack[0].Hide()
				statusStack[1].Show()
			}

			completed := c2[2].(*widget.Label)
			if status == app.JobDelivered {
				completed.SetText(humanize.Time(j.endDate))
				completed.Show()
			} else {
				completed.Hide()
			}

			l.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) || id < 0 {
			return
		}
		showIndustryJobWindow(a.u, a.rowsFiltered[id])
	}
	return l
}

// filterRowsAsync applies all filters and sorting and freshes the list with the changed rows.
// A new sorting can be applied by providing a sortCol. -1 does not change the current sorting.
func (a *IndustryJobs) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := a.searchEntry.Text
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows := slices.DeleteFunc(rows, func(r industryJobRow) bool {
			return !filter.match(r)
		})
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r industryJobRow) bool {
				return !strings.Contains(strings.ToLower(r.blueprintType.Name), strings.ToLower(search))
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r industryJobRow) set.Set[string] {
			return r.tags
		})...).All())

		footer := fmt.Sprintf("Showing %s / %s jobs", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				options := []xwidget.FilterOption{
					xwidget.NewFilterOptionMultiChoice(industryJobsFilterOwner, industryJobsOwnerOptions()),
					xwidget.NewFilterOptionMultiChoice(industryJobsFilterActivity, industryJobsActivityOptions()),
				}
				if a.forCorporation {
					options = append(options, xwidget.NewFilterOptionMultiChoice(industryJobsFilterInstaller, industryJobsInstallerOptions()))
				} else {
					options = append(options, xwidget.NewFilterOptionMultiChoice(industryJobsFilterTag, tagOptions))
				}
				a.filterChip.SetOptions(options...)
			} else {
				a.selectTag.SetOptions(tagOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
			switch x := a.body.(type) {
			case *widget.Table:
				x.ScrollToTop()
			}
		})
	})
}

func (a *IndustryJobs) update(ctx context.Context) {
	var jobs []industryJobRow
	var err error
	if a.forCorporation {
		jobs, err = a.fetchCorporationJobs(ctx)
	} else {
		jobs, err = a.fetchCombinedJobs(ctx)
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh industry jobs UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = fmt.Sprintf("ERROR: %s", a.u.ErrorDisplay(err))
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
	}
	var readyCount int
	for _, j := range jobs {
		if j.status == app.JobReady && j.isInstallerMe {
			readyCount++
		}
	}
	fyne.Do(func() {
		a.rows = jobs
		a.filterRowsAsync("")
		if a.OnUpdate != nil {
			a.OnUpdate(readyCount)
		}
	})
}

func (a *IndustryJobs) fetchCombinedJobs(ctx context.Context) ([]industryJobRow, error) {
	cj, err := a.u.Character().ListAllCharacterIndustryJob(ctx)
	if err != nil {
		return nil, err
	}
	rj, err := a.u.Corporation().ListAllCorporationIndustryJobs(ctx)
	if err != nil {
		return nil, err
	}
	ids1 := set.Collect(xiter.MapSlice(cj, func(x *app.CharacterIndustryJob) int64 {
		return x.CharacterID
	}))
	ids2 := set.Collect(xiter.MapSlice(rj, func(x *app.CorporationIndustryJob) int64 {
		return x.CorporationID
	}))
	ids := set.Union(ids1, ids2)
	eeMap, err := a.u.EVEUniverse().ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	myCharacters, err := a.u.Character().ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	tagsPerCharacter := make(map[int64]set.Set[string])
	for id := range myCharacters.All() {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, id)
		if err != nil {
			return nil, err
		}
		tagsPerCharacter[id] = tags
	}

	var characterJobs []industryJobRow
	for _, j := range cj {
		characterJobs = append(characterJobs, industryJobRow{
			activity:           j.Activity,
			blueprintID:        j.BlueprintID,
			blueprintType:      j.BlueprintType,
			blueprintTypeName:  shortenBlueprintName(j.BlueprintType),
			completedCharacter: j.CompletedCharacter,
			completedDate:      j.CompletedDate,
			cost:               j.Cost,
			duration:           j.Duration,
			endDate:            j.EndDate,
			installer:          j.Installer,
			jobID:              j.JobID,
			licensedRuns:       j.LicensedRuns,
			location:           j.Station,
			owner:              eeMap[j.CharacterID],
			pauseDate:          j.PauseDate,
			probability:        j.Probability,
			productType:        j.ProductType,
			runs:               j.Runs,
			startDate:          j.StartDate,
			status:             j.Status,
			successfulRuns:     j.SuccessfulRuns,
			isInstallerMe:      true,
			isOwnerMe:          true,
			tags:               tagsPerCharacter[j.Installer.ID],
		})
	}

	var corporationJobs []industryJobRow
	for _, j := range rj {
		if !myCharacters.Contains(j.Installer.ID) {
			continue
		}
		corporationJobs = append(corporationJobs, industryJobRow{
			activity:           j.Activity,
			blueprintID:        j.BlueprintID,
			blueprintType:      j.BlueprintType,
			blueprintTypeName:  shortenBlueprintName(j.BlueprintType),
			completedCharacter: j.CompletedCharacter,
			completedDate:      j.CompletedDate,
			cost:               j.Cost,
			duration:           j.Duration,
			endDate:            j.EndDate,
			installer:          j.Installer,
			isInstallerMe:      true,
			isOwnerMe:          false,
			jobID:              j.JobID,
			licensedRuns:       j.LicensedRuns,
			location:           j.Location,
			owner:              eeMap[j.CorporationID],
			pauseDate:          j.PauseDate,
			probability:        j.Probability,
			productType:        j.ProductType,
			runs:               j.Runs,
			startDate:          j.StartDate,
			status:             j.Status,
			successfulRuns:     j.SuccessfulRuns,
			tags:               tagsPerCharacter[j.Installer.ID],
		})
	}
	jobs := slices.Concat(characterJobs, corporationJobs)
	return jobs, nil
}

func shortenBlueprintName(typ *app.EntityShort) string {
	s, _ := strings.CutSuffix(typ.Name, " Blueprint")
	return s
}

func (a *IndustryJobs) fetchCorporationJobs(ctx context.Context) ([]industryJobRow, error) {
	corporationID := a.corporation.Load().IDOrZero()
	if corporationID == 0 {
		return []industryJobRow{}, nil
	}
	rj, err := a.u.Corporation().ListCorporationIndustryJobs(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	ids := set.Collect(xiter.MapSlice(rj, func(x *app.CorporationIndustryJob) int64 {
		return x.CorporationID
	}))
	eeMap, err := a.u.EVEUniverse().ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	myCharacters, err := a.u.Character().ListCharacterIDs(ctx)
	if err != nil {
		return nil, err
	}
	var jobs []industryJobRow
	for _, j := range rj {
		jobs = append(jobs, industryJobRow{
			activity:           j.Activity,
			blueprintID:        j.BlueprintID,
			blueprintType:      j.BlueprintType,
			blueprintTypeName:  shortenBlueprintName(j.BlueprintType),
			completedCharacter: j.CompletedCharacter,
			completedDate:      j.CompletedDate,
			cost:               j.Cost,
			duration:           j.Duration,
			endDate:            j.EndDate,
			installer:          j.Installer,
			isInstallerMe:      myCharacters.Contains(j.Installer.ID),
			isOwnerMe:          false,
			jobID:              j.JobID,
			licensedRuns:       j.LicensedRuns,
			location:           j.Location,
			owner:              eeMap[j.CorporationID],
			pauseDate:          j.PauseDate,
			probability:        j.Probability,
			productType:        j.ProductType,
			runs:               j.Runs,
			startDate:          j.StartDate,
			status:             j.Status,
			successfulRuns:     j.SuccessfulRuns,
		})
	}
	return jobs, nil
}
