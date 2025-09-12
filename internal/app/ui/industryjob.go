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
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
	blueprintType      *app.EntityShort[int32]
	completedCharacter optional.Optional[*app.EveEntity]
	completedDate      optional.Optional[time.Time]
	cost               optional.Optional[float64]
	duration           int
	endDate            time.Time
	installer          *app.EveEntity
	isInstallerMe      bool
	isOwnerMe          bool
	jobID              int32
	licensedRuns       optional.Optional[int]
	location           *app.EveLocationShort
	owner              *app.EveEntity
	pauseDate          optional.Optional[time.Time]
	probability        optional.Optional[float32]
	productType        optional.Optional[*app.EntityShort[int32]]
	remaining          time.Duration
	runs               int
	startDate          time.Time
	status             app.IndustryJobStatus
	statusDisplay      []widget.RichTextSegment
	successfulRuns     optional.Optional[int32]
	tags               set.Set[string]
}

type industryJobs struct {
	widget.BaseWidget

	OnUpdate func(count int)

	body              fyne.CanvasObject
	bottom            *widget.Label
	columnSorter      *iwidget.ColumnSorter
	corporation       *app.Corporation
	isCorporationMode bool
	rows              []industryJobRow
	rowsFiltered      []industryJobRow
	search            *widget.Entry
	selectActivity    *kxwidget.FilterChipSelect
	selectInstaller   *kxwidget.FilterChipSelect
	selectOwner       *kxwidget.FilterChipSelect
	selectStatus      *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *iwidget.SortButton
	u                 *baseUI
}

const (
	industryJobsColBlueprint = 0
	industryJobsColStatus    = 1
	industryJobsColRuns      = 2
	industryJobsColActivity  = 3
	industryJobsColEndDate   = 4
	industryJobsColLocation  = 5
	industryJobsColOwner     = 6
	industryJobsColInstaller = 7
)

func newIndustryJobs(u *baseUI, isCorporationMode bool) *industryJobs {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{
		{Label: "Blueprint", Width: 250},
		{Label: "Status", Width: 100},
		{Label: "Runs", Width: 75},
		{Label: "Activity", Width: 200},
		{Label: "End date", Width: columnWidthDateTime},
		{Label: "Location", Width: columnWidthLocation},
		{Label: "Owner", Width: columnWidthEntity},
		{Label: "Installer", Width: columnWidthEntity},
	})
	a := &industryJobs{
		bottom:            makeTopLabel(),
		columnSorter:      iwidget.NewColumnSorterWithInit(headers, 4, iwidget.SortDesc),
		isCorporationMode: isCorporationMode,
		rows:              make([]industryJobRow, 0),
		rowsFiltered:      make([]industryJobRow, 0),
		u:                 u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, j industryJobRow) []widget.RichTextSegment {
		switch col {
		case industryJobsColBlueprint:
			return iwidget.RichTextSegmentsFromText(j.blueprintType.Name)
		case industryJobsColStatus:
			return j.statusDisplay
		case industryJobsColRuns:
			return iwidget.RichTextSegmentsFromText(
				ihumanize.Comma(j.runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		case industryJobsColActivity:
			return iwidget.RichTextSegmentsFromText(j.activity.Display())
		case industryJobsColEndDate:
			return iwidget.RichTextSegmentsFromText(j.endDate.Format(app.DateTimeFormat))
		case industryJobsColLocation:
			return iwidget.RichTextSegmentsFromText(j.location.Name.ValueOrZero())
		case industryJobsColOwner:
			return iwidget.RichTextSegmentsFromText(j.owner.Name)
		case industryJobsColInstaller:
			return iwidget.RichTextSegmentsFromText(j.installer.Name)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}

	if a.u.isDesktop {
		a.body = iwidget.MakeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, j industryJobRow) {
			a.showIndustryJobWindow(j)
		})
	} else {
		a.body = a.makeDataList()
	}

	a.search = widget.NewEntry()
	a.search.PlaceHolder = "Search Blueprints"
	a.search.OnChanged = func(_ string) {
		a.filterRows(-1)
	}
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{
		industryOwnerMe,
		industryOwnerCorp,
	}, func(_ string) {
		a.filterRows(-1)
	})

	a.selectStatus = kxwidget.NewFilterChipSelect("", []string{
		industryStatusActive,
		industryStatusInProgress,
		industryStatusReady,
		industryStatusHalted,
		industryStatusHistory,
	}, func(_ string) {
		a.filterRows(-1)
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
		a.filterRows(-1)
	})

	a.selectInstaller = kxwidget.NewFilterChipSelect("Installer", []string{
		industryInstallerMe,
		industryInstallerCorpmates,
	}, func(_ string) {
		a.filterRows(-1)
	})

	a.sortButton = a.columnSorter.NewSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window, 6, 7)

	if isCorporationMode {
		a.u.corporationExchanged.AddListener(
			func(_ context.Context, c *app.Corporation) {
				a.corporation = c
			},
		)
		a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
			if corporationIDOrZero(a.corporation) != arg.corporationID {
				return
			}
			if arg.section == app.SectionCorporationIndustryJobs {
				a.update()
			}
		})
	} else {
		a.selectInstaller.Selected = industryInstallerMe
		a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
			if arg.section == app.SectionCharacterIndustryJobs {
				a.update()
			}
		})
		a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
			if arg.section == app.SectionCorporationIndustryJobs {
				a.update()
			}
		})
	}
	return a
}

func (a *industryJobs) CreateRenderer() fyne.WidgetRenderer {
	var selections *fyne.Container
	if a.isCorporationMode {
		selections = container.NewHBox(a.selectOwner, a.selectStatus, a.selectActivity, a.selectInstaller)
	} else {
		selections = container.NewHBox(a.selectOwner, a.selectStatus, a.selectActivity, a.selectTag)
	}
	if !a.u.isDesktop {
		selections.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewBorder(
			nil,
			container.NewHScroll(selections),
			nil,
			nil,
			a.search,
		),
		a.bottom,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

// filterRows applies all filters and sorting and freshes the list with the changed rows.
// A new sorting can be applied by providing a sortCol. -1 does not change the current sorting.
func (a *industryJobs) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	rows = xslices.Filter(rows, func(r industryJobRow) bool {
		switch a.selectStatus.Selected {
		case industryStatusActive:
			return r.status.IsActive()
		case industryStatusInProgress:
			return r.status == app.JobActive
		case industryStatusReady:
			return r.status == app.JobReady
		case industryStatusHalted:
			return r.status == app.JobPaused
		case industryStatusHistory:
			return !r.status.IsActive()
		}
		return false
	})
	if x := a.selectInstaller.Selected; x != "" {
		rows = xslices.Filter(rows, func(r industryJobRow) bool {
			switch x {
			case industryInstallerMe:
				return r.isInstallerMe
			case industryInstallerCorpmates:
				return !r.isInstallerMe
			}
			return false
		})
	}
	if x := a.selectActivity.Selected; x != "" {
		rows = xslices.Filter(rows, func(r industryJobRow) bool {
			switch x {
			case industryActivityCopying:
				return r.activity == app.Copying
			case industryActivityInvention:
				return r.activity == app.Invention
			case industryActivityManufacturing:
				return r.activity == app.Manufacturing
			case industryActivityMaterialResearch:
				return r.activity == app.MaterialEfficiencyResearch
			case industryActivityReaction:
				return r.activity == app.Reactions1 || r.activity == app.Reactions2
			case industryActivityTimeResearch:
				return r.activity == app.TimeEfficiencyResearch
			}
			return false
		})
	}
	if x := a.selectOwner.Selected; x != "" {
		rows = xslices.Filter(rows, func(r industryJobRow) bool {
			switch x {
			case industryOwnerCorp:
				return !r.isOwnerMe
			case industryOwnerMe:
				return r.isOwnerMe
			}
			return false
		})
	}
	if s := a.search.Text; len(s) > 1 {
		rows = xslices.Filter(rows, func(r industryJobRow) bool {
			return strings.Contains(strings.ToLower(r.blueprintType.Name), strings.ToLower(s))
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r industryJobRow) bool {
			return r.tags.Contains(x)
		})
	}
	// sort
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(j, k industryJobRow) int {
			var c int
			switch sortCol {
			case industryJobsColBlueprint:
				c = strings.Compare(j.blueprintType.Name, k.blueprintType.Name)
			case industryJobsColStatus:
				c = cmp.Compare(j.remaining, k.remaining)
			case industryJobsColRuns:
				c = cmp.Compare(j.runs, k.runs)
			case industryJobsColActivity:
				c = strings.Compare(j.activity.String(), k.activity.String())
			case industryJobsColEndDate:
				c = j.endDate.Compare(k.endDate)
			case industryJobsColLocation:
				c = strings.Compare(j.location.Name.ValueOrZero(), k.location.Name.ValueOrZero())
			case industryJobsColOwner:
				c = strings.Compare(j.owner.Name, k.owner.Name)
			case industryJobsColInstaller:
				c = strings.Compare(j.installer.Name, k.installer.Name)
			}
			if dir == iwidget.SortAsc {
				return c
			} else {
				return -1 * c
			}
		})
	})
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r industryJobRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.rowsFiltered = rows
	a.body.Refresh()
	switch x := a.body.(type) {
	case *widget.Table:
		x.ScrollToTop()
	}
}

func (a *industryJobs) makeDataList() *iwidget.StripedList {
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
	var l *iwidget.StripedList
	l = iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("Template")
			title.TextStyle.Bold = true
			title.Wrapping = fyne.TextWrapWord
			status := iwidget.NewRichText()
			location := widget.NewLabel("Template")
			location.Wrapping = fyne.TextWrapWord
			completed := widget.NewLabel("Template")
			p := theme.Padding()
			activityIcon := widget.NewIcon(icons.BlankSvg)
			statusIcon := widget.NewIcon(icons.BlankSvg)
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(1, p))
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
			if j.status == app.JobActive {
				statusStack[0].(*iwidget.RichText).Set(j.statusDisplay)
				statusStack[0].Show()
				statusStack[1].Hide()
			} else {
				r, ok := statusMap[j.status]
				if !ok {
					r = theme.NewThemedResource(icons.QuestionmarkSvg)
				}
				statusStack[1].(*fyne.Container).Objects[1].(*widget.Icon).SetResource(r)
				statusStack[0].Hide()
				statusStack[1].Show()
			}

			completed := c2[2].(*widget.Label)
			if j.status == app.JobDelivered {
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

func (a *industryJobs) update() {
	var jobs []industryJobRow
	var err error
	if a.isCorporationMode {
		jobs, err = a.fetchCorporationJobs()
	} else {
		jobs, err = a.fetchCombinedJobs()
	}
	if err != nil {
		slog.Error("Failed to refresh industry jobs UI", "err", err)
		fyne.Do(func() {
			a.bottom.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.bottom.Importance = widget.DangerImportance
			a.bottom.Refresh()
			a.bottom.Show()
		})
	}
	for i, j := range jobs {
		remaining := time.Until(j.endDate)
		var segs []widget.RichTextSegment
		if j.status == app.JobActive {
			segs = iwidget.RichTextSegmentsFromText(ihumanize.Duration(remaining), widget.RichTextStyle{
				ColorName: theme.ColorNameForeground,
			})
		} else {
			segs = iwidget.RichTextSegmentsFromText(j.status.Display(), widget.RichTextStyle{
				ColorName: j.status.Color(),
			})
		}
		jobs[i].statusDisplay = segs
		jobs[i].remaining = remaining
	}
	var readyCount int
	for _, j := range jobs {
		if j.status == app.JobReady && j.isInstallerMe {
			readyCount++
		}
	}
	if a.OnUpdate != nil {
		a.OnUpdate(readyCount)
	}
	fyne.Do(func() {
		a.search.SetText("")
		a.bottom.Hide()
		a.rows = jobs
		a.filterRows(-1)
	})
}

func (a *industryJobs) fetchCombinedJobs() ([]industryJobRow, error) {
	ctx := context.Background()
	cj, err := a.u.cs.ListAllCharacterIndustryJob(ctx)
	if err != nil {
		return nil, err
	}
	rj, err := a.u.rs.ListAllCorporationIndustryJobs(ctx)
	if err != nil {
		return nil, err
	}
	ids1 := set.Collect(xiter.MapSlice(cj, func(x *app.CharacterIndustryJob) int32 {
		return x.CharacterID
	}))
	ids2 := set.Collect(xiter.MapSlice(rj, func(x *app.CorporationIndustryJob) int32 {
		return x.CorporationID
	}))
	ids := set.Union(ids1, ids2)
	eeMap, err := a.u.eus.ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	cc, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	tagsPerCharacter := make(map[int32]set.Set[string])
	for _, c := range cc {
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		tagsPerCharacter[c.ID] = tags
	}
	myCharacters := set.Of(xslices.Map(cc, func(c *app.EntityShort[int32]) int32 {
		return c.ID
	})...)
	characterJobs := make([]industryJobRow, 0)
	for _, j := range cj {
		characterJobs = append(characterJobs, industryJobRow{
			activity:           j.Activity,
			blueprintID:        j.BlueprintID,
			blueprintType:      j.BlueprintType,
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

	corporationJobs := make([]industryJobRow, 0)
	for _, j := range rj {
		if !myCharacters.Contains(j.Installer.ID) {
			continue
		}
		corporationJobs = append(corporationJobs, industryJobRow{
			activity:           j.Activity,
			blueprintID:        j.BlueprintID,
			blueprintType:      j.BlueprintType,
			completedCharacter: j.CompletedCharacter,
			completedDate:      j.CompletedDate,
			cost:               j.Cost,
			duration:           j.Duration,
			endDate:            j.EndDate,
			installer:          j.Installer,
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
			isInstallerMe:      myCharacters.Contains(j.Installer.ID),
			isOwnerMe:          false,
			tags:               tagsPerCharacter[j.Installer.ID],
		})
	}
	jobs := slices.Concat(characterJobs, corporationJobs)
	return jobs, nil
}

func (a *industryJobs) fetchCorporationJobs() ([]industryJobRow, error) {
	corporationID := corporationIDOrZero(a.corporation)
	if corporationID == 0 {
		return []industryJobRow{}, nil
	}
	ctx := context.Background()
	rj, err := a.u.rs.ListCorporationIndustryJobs(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	ids := set.Collect(xiter.MapSlice(rj, func(x *app.CorporationIndustryJob) int32 {
		return x.CorporationID
	}))
	eeMap, err := a.u.eus.ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	cc, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	myCharacters := set.Of(xslices.Map(cc, func(c *app.EntityShort[int32]) int32 {
		return c.ID
	})...)
	jobs := make([]industryJobRow, 0)
	for _, j := range rj {
		jobs = append(jobs, industryJobRow{
			activity:           j.Activity,
			blueprintID:        j.BlueprintID,
			blueprintType:      j.BlueprintType,
			completedCharacter: j.CompletedCharacter,
			completedDate:      j.CompletedDate,
			cost:               j.Cost,
			duration:           j.Duration,
			endDate:            j.EndDate,
			installer:          j.Installer,
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
			isInstallerMe:      myCharacters.Contains(j.Installer.ID),
			isOwnerMe:          false,
		})
	}
	return jobs, nil
}

// showIndustryJobWindow shows the details of a industry job in a window.
func (a *industryJobs) showIndustryJobWindow(r industryJobRow) {
	title := fmt.Sprintf("Industry Job #%d", r.jobID)
	w, ok := a.u.getOrCreateWindow(fmt.Sprintf("industryjob-%d-%d", r.owner.ID, r.jobID), title, r.owner.Name)
	if !ok {
		w.Show()
		return
	}
	activity := fmt.Sprintf("%s (%s)", r.activity.Display(), r.activity.JobType().Display())
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.owner.ID,
			r.owner.Name,
			a.u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Blueprint", makeLinkLabelWithWrap(r.blueprintType.Name, func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, r.blueprintType.ID)
		})),
		widget.NewFormItem("Activity", widget.NewLabel(activity)),
	}
	if !r.productType.IsEmpty() {
		x := r.productType.MustValue()
		items = append(items, widget.NewFormItem(
			"Product Type",
			makeLinkLabelWithWrap(x.Name, func() {
				a.u.ShowInfoWindow(app.EveEntityInventoryType, x.ID)
			}),
		))
	}
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Status", iwidget.NewRichText(r.statusDisplay...)),
		widget.NewFormItem("Runs", widget.NewLabel(ihumanize.Comma(r.runs))),
	})

	if !r.licensedRuns.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Licensed Runs",
			widget.NewLabel(ihumanize.Comma(r.licensedRuns.ValueOrZero())),
		))
	}
	if !r.successfulRuns.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Successful Runs",
			widget.NewLabel(ihumanize.Comma(r.successfulRuns.ValueOrZero())),
		))
	}
	if !r.probability.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Probability",
			widget.NewLabel(fmt.Sprintf("%.0f%%", r.probability.ValueOrZero()*100)),
		))
	}
	items = append(items, widget.NewFormItem("Start date", widget.NewLabel(r.startDate.Format(app.DateTimeFormat))))
	if !r.pauseDate.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Pause date",
			widget.NewLabel(r.pauseDate.ValueOrZero().Format(app.DateTimeFormat)),
		))
	}
	items = append(items, widget.NewFormItem("End date", widget.NewLabel(r.endDate.Format(app.DateTimeFormat))))
	if !r.completedDate.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Completed date",
			widget.NewLabel(r.completedDate.ValueOrZero().Format(app.DateTimeFormat))),
		)
	}
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Location", makeLocationLabel(r.location, a.u.ShowLocationInfoWindow)),
		widget.NewFormItem("Installer", makeLinkLabelWithWrap(r.installer.Name, func() {
			a.u.ShowEveEntityInfoWindow(r.installer)
		})),
		widget.NewFormItem("Type", widget.NewLabel(r.owner.CategoryDisplay())),
	})
	if !r.completedCharacter.IsEmpty() {
		x := r.completedCharacter.MustValue()
		items = append(items, widget.NewFormItem("Completed By", makeLinkLabelWithWrap(x.Name, func() {
			a.u.ShowEveEntityInfoWindow(x)
		})))
	}
	if a.u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Job ID", a.u.makeCopyToClipboardLabel(fmt.Sprint(r.jobID))))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			a.u.ShowTypeInfoWindow(r.blueprintType.ID)
		},
		imageLoader: func() (fyne.Resource, error) {
			return a.u.eis.InventoryTypeBPO(r.blueprintType.ID, 256)
		},
		title:  title,
		window: w,
	})
	w.Show()
}
