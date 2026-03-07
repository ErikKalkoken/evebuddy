package ui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
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
	footer          *widget.Label
	columnSorter    *xwidget.ColumnSorter[industryJobRow]
	corporation     atomic.Pointer[app.Corporation]
	forCorporation  bool
	rows            []industryJobRow
	rowsFiltered    []industryJobRow
	search          *widget.Entry
	selectActivity  *kxwidget.FilterChipSelect
	selectInstaller *kxwidget.FilterChipSelect
	selectOwner     *kxwidget.FilterChipSelect
	selectStatus    *kxwidget.FilterChipSelect
	selectTag       *kxwidget.FilterChipSelect
	sortButton      *xwidget.SortButton
	u         uiservices.UIServices
}

const (
	industryJobsColBlueprint = iota + 1
	industryJobsColStatus
	industryJobsColRuns
	industryJobsColActivity
	industryJobsColEndDate
	industryJobsColLocation
	industryJobsColOwner
	industryJobsColInstaller
)

func NewIndustryJobsForOverview(u         uiservices.UIServices) *IndustryJobs {
	return newIndustryJobs(u, false)
}

func NewIndustryJobsForCorporation(u         uiservices.UIServices) *IndustryJobs {
	return newIndustryJobs(u, true)
}

func newIndustryJobs(u         uiservices.UIServices, forCorporation bool) *IndustryJobs {
	corporationIcon := theme.NewThemedResource(icons.StarCircleOutlineSvg)
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[industryJobRow]{{
		ID:    industryJobsColBlueprint,
		Label: "Blueprint",
		Width: 250,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.blueprintType.Name, b.blueprintType.Name)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(r.blueprintTypeName)
			x := border[1].(*canvas.Image)
			u.EVEImage().InventoryTypeBPOAsync(r.blueprintType.ID, app.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		ID:    industryJobsColStatus,
		Label: "Status",
		Width: 100,
		Sort: func(a, b industryJobRow) int {
			return cmp.Compare(a.remaining(), b.remaining())
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.statusDisplay())
		},
	}, {
		ID:    industryJobsColRuns,
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
		ID:    industryJobsColActivity,
		Label: "Activity",
		Width: 200,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.activity.String(), b.activity.String())
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.activity.Display())
		},
	}, {
		ID:    industryJobsColEndDate,
		Label: "End date",
		Width: columnWidthDateTime,
		Sort: func(a, b industryJobRow) int {
			return a.endDate.Compare(b.endDate)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.endDate.Format(app.DateTimeFormat))
		},
	}, {
		ID:    industryJobsColLocation,
		Label: "Location",
		Width: columnWidthLocation,
		Sort: func(a, b industryJobRow) int {
			return optional.Compare(a.location.Name, b.location.Name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.location.Name.ValueOrZero())
		},
	}, {
		ID:    industryJobsColOwner,
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
		ID:    industryJobsColInstaller,
		Label: "Installer",
		Width: columnWidthEntity,
		Sort: func(a, b industryJobRow) int {
			return strings.Compare(a.installer.Name, b.installer.Name)
		},
		Update: func(r industryJobRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.installer.Name)
		},
	}})
	a := &IndustryJobs{
		footer:         newLabelWithWrapping(),
		columnSorter:   xwidget.NewColumnSorter(columns, industryJobsColEndDate, xwidget.SortDesc),
		forCorporation: forCorporation,
		u:              u,
	}
	a.ExtendBaseWidget(a)

	if app.IsMobile() {
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
				a.showIndustryJobWindow(r)
			})
	}

	a.search = widget.NewEntry()
	a.search.PlaceHolder = "Search Blueprints"
	a.search.OnChanged = func(_ string) {
		a.filterRowsAsync(-1)
	}
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{
		industryOwnerMe,
		industryOwnerCorp,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})

	a.selectStatus = kxwidget.NewFilterChipSelect("", []string{
		industryStatusActive,
		industryStatusInProgress,
		industryStatusReady,
		industryStatusHalted,
		industryStatusHistory,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})
	a.selectStatus.Selected = industryStatusActive
	a.selectStatus.SortDisabled = true

	a.selectActivity = kxwidget.NewFilterChipSelect("Activity", []string{
		industryActivityManufacturing,
		industryActivityMaterialResearch,
		industryActivityTimeResearch,
		industryActivityCopying,
		industryActivityInvention,
		industryActivityReaction,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})

	a.selectInstaller = kxwidget.NewFilterChipSelect("Installer", []string{
		industryInstallerMe,
		industryInstallerCorpmates,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})

	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow(), 6, 7)

	if forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if corporationIDOrZero(a.corporation.Load()) != arg.CorporationID {
				return
			}
			if arg.Section == app.SectionCorporationIndustryJobs {
				a.update(ctx)
			}
		})
	} else {
		a.selectInstaller.Selected = industryInstallerMe
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
		a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
			a.update(ctx)
		})
	}
	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			a.body.Refresh()
		})
	})
	return a
}

func (a *IndustryJobs) CreateRenderer() fyne.WidgetRenderer {
	var filter *fyne.Container
	if a.forCorporation {
		filter = container.NewHBox(a.selectOwner, a.selectStatus, a.selectActivity, a.selectInstaller)
	} else {
		filter = container.NewHBox(a.selectOwner, a.selectStatus, a.selectActivity, a.selectTag)
	}
	if app.IsMobile() {
		filter.Add(a.sortButton)
	}
	var topBox *fyne.Container
	if app.IsMobile() {
		topBox = container.NewVBox(
			a.search,
			container.NewHScroll(filter),
		)
	} else {
		topBox = container.NewBorder(
			nil,
			nil,
			filter,
			nil,
			a.search,
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
			spacer := newSpacer(fyne.NewSize(1, p))
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
		a.showIndustryJobWindow(a.rowsFiltered[id])
	}
	return l
}

// filterRowsAsync applies all filters and sorting and freshes the list with the changed rows.
// A new sorting can be applied by providing a sortCol. -1 does not change the current sorting.
func (a *IndustryJobs) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	installer := a.selectInstaller.Selected
	activity := a.selectActivity.Selected
	owner := a.selectOwner.Selected
	tag := a.selectTag.Selected
	search := a.search.Text
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		rows := slices.DeleteFunc(rows, func(r industryJobRow) bool {
			status := r.statusCalculated()
			switch a.selectStatus.Selected {
			case industryStatusActive:
				return !status.IsActive()
			case industryStatusInProgress:
				return status != app.JobActive
			case industryStatusReady:
				return status != app.JobReady
			case industryStatusHalted:
				return status != app.JobPaused
			case industryStatusHistory:
				return status.IsActive()
			}
			return true
		})
		if installer != "" {
			rows = slices.DeleteFunc(rows, func(r industryJobRow) bool {
				switch installer {
				case industryInstallerMe:
					return !r.isInstallerMe
				case industryInstallerCorpmates:
					return r.isInstallerMe
				}
				return true
			})
		}
		if activity != "" {
			rows = slices.DeleteFunc(rows, func(r industryJobRow) bool {
				switch activity {
				case industryActivityCopying:
					return r.activity != app.Copying
				case industryActivityInvention:
					return r.activity != app.Invention
				case industryActivityManufacturing:
					return r.activity != app.Manufacturing
				case industryActivityMaterialResearch:
					return r.activity != app.MaterialEfficiencyResearch
				case industryActivityReaction:
					return r.activity != app.Reactions1 || r.activity == app.Reactions2
				case industryActivityTimeResearch:
					return r.activity != app.TimeEfficiencyResearch
				}
				return true
			})
		}
		if owner != "" {
			rows = slices.DeleteFunc(rows, func(r industryJobRow) bool {
				switch owner {
				case industryOwnerCorp:
					return r.isOwnerMe
				case industryOwnerMe:
					return !r.isOwnerMe
				}
				return true
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r industryJobRow) bool {
				return !r.tags.Contains(tag)
			})
		}
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
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
			switch x := a.body.(type) {
			case *widget.Table:
				x.ScrollToTop()
			}
		})
	}()
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
		slog.Error("Failed to refresh industry jobs UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = fmt.Sprintf("ERROR: %s", app.ErrorDisplay(err))
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
		a.filterRowsAsync(-1)
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
	corporationID := corporationIDOrZero(a.corporation.Load())
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

// showIndustryJobWindow shows the details of a industry job in a window.
func (a *IndustryJobs) showIndustryJobWindow(r industryJobRow) {
	title := fmt.Sprintf("Industry Job #%d", r.jobID)
	key := fmt.Sprintf("industryjob-%d-%d", r.owner.ID, r.jobID)
	w, ok, onClosed := a.u.GetOrCreateWindowWithOnClosed(key, title, r.owner.Name)
	if !ok {
		w.Show()
		return
	}
	activity := fmt.Sprintf("%s (%s)", r.activity.Display(), r.activity.JobType().Display())
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.owner.ID,
			r.owner.Name,
			a.u.InfoWindow().ShowEntity,
		)),
		widget.NewFormItem("Blueprint", makeLinkLabelWithWrap(r.blueprintType.Name, func() {
			a.u.InfoWindow().Show(app.EveEntityInventoryType, r.blueprintType.ID)
		})),
		widget.NewFormItem("Activity", widget.NewLabel(activity)),
	}
	if v, ok := r.productType.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Product Type",
			makeLinkLabelWithWrap(v.Name, func() {
				a.u.InfoWindow().Show(app.EveEntityInventoryType, v.ID)
			}),
		))
	}
	status := xwidget.NewRichText(r.statusDisplay()...)
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Status", status),
		widget.NewFormItem("Runs", widget.NewLabel(ihumanize.Comma(r.runs))),
	})

	if v, ok := r.licensedRuns.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Licensed Runs",
			widget.NewLabel(ihumanize.Comma(v)),
		))
	}
	if v, ok := r.successfulRuns.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Successful Runs",
			widget.NewLabel(ihumanize.Comma(v)),
		))
	}
	if v, ok := r.probability.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Probability",
			widget.NewLabel(fmt.Sprintf("%.0f%%", v*100)),
		))
	}
	items = append(items, widget.NewFormItem("Start date", widget.NewLabel(r.startDate.Format(app.DateTimeFormat))))
	if v, ok := r.pauseDate.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Pause date",
			widget.NewLabel(v.Format(app.DateTimeFormat)),
		))
	}
	items = append(items, widget.NewFormItem("End date", widget.NewLabel(r.endDate.Format(app.DateTimeFormat))))
	if v, ok := r.completedDate.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Completed date",
			widget.NewLabel(v.Format(app.DateTimeFormat))),
		)
	}
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Location", makeLocationLabel(r.location, a.u.InfoWindow().ShowLocation)),
		widget.NewFormItem("Installer", makeLinkLabelWithWrap(r.installer.Name, func() {
			a.u.InfoWindow().ShowEntity(r.installer)
		})),
		widget.NewFormItem("Type", widget.NewLabel(r.owner.CategoryDisplay())),
	})
	if v, ok := r.completedCharacter.Value(); ok {
		items = append(items, widget.NewFormItem("Completed By", makeLinkLabelWithWrap(v.Name, func() {
			a.u.InfoWindow().ShowEntity(v)
		})))
	}
	if app.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Job ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(r.jobID))))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			status.Set(r.statusDisplay())
		})
	}, key)
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		a.u.Signals().RefreshTickerExpired.RemoveListener(key)
	})
	xwindow.Set(xwindow.Params{
		Content: f,
		ImageAction: func() {
			a.u.InfoWindow().ShowType(r.blueprintType.ID)
		},
		ImageLoader: func(setter func(r fyne.Resource)) {
			a.u.EVEImage().InventoryTypeBPOAsync(r.blueprintType.ID, 256, setter)
		},
		Title:  title,
		Window: w,
	})
	w.Show()
}
