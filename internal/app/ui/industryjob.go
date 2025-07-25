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

func (j industryJobRow) IsActive() bool {
	switch j.status {
	case app.JobActive, app.JobReady, app.JobPaused:
		return true
	}
	return false
}

type industryJobs struct {
	widget.BaseWidget

	OnUpdate func(count int)

	body            fyne.CanvasObject
	columnSorter    *columnSorter
	rows            []industryJobRow
	rowsFiltered    []industryJobRow
	search          *widget.Entry
	selectActivity  *kxwidget.FilterChipSelect
	selectInstaller *kxwidget.FilterChipSelect
	selectOwner     *kxwidget.FilterChipSelect
	selectStatus    *kxwidget.FilterChipSelect
	selectTag       *kxwidget.FilterChipSelect
	sortButton      *sortButton
	bottom          *widget.Label
	u               *baseUI
}

func newIndustryJobs(u *baseUI) *industryJobs {
	headers := []headerDef{
		{label: "Blueprint", width: 250},
		{label: "Status", width: 100, refresh: true},
		{label: "Runs", width: 75},
		{label: "Activity", width: 200},
		{label: "End date", width: columnWidthDateTime},
		{label: "Location", width: columnWidthLocation},
		{label: "Owner", width: columnWidthEntity},
		{label: "Installer", width: columnWidthEntity},
	}
	a := &industryJobs{
		columnSorter: newColumnSorterWithInit(headers, 4, sortDesc),
		rows:         make([]industryJobRow, 0),
		rowsFiltered: make([]industryJobRow, 0),
		bottom:       makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, j industryJobRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.RichTextSegmentsFromText(j.blueprintType.Name)
		case 1:
			return j.statusDisplay
		case 2:
			return iwidget.RichTextSegmentsFromText(
				ihumanize.Comma(j.runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		case 3:
			return iwidget.RichTextSegmentsFromText(j.activity.Display())
		case 4:
			return iwidget.RichTextSegmentsFromText(j.endDate.Format(app.DateTimeFormat))
		case 5:
			return iwidget.RichTextSegmentsFromText(j.location.Name.ValueOrZero())
		case 6:
			return iwidget.RichTextSegmentsFromText(j.owner.Name)
		case 7:
			return iwidget.RichTextSegmentsFromText(j.installer.Name)
		}
		return iwidget.RichTextSegmentsFromText("?")
	}

	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, j industryJobRow) {
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
	a.selectInstaller.Selected = industryInstallerMe

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window, 6, 7)
	return a
}

func (a *industryJobs) CreateRenderer() fyne.WidgetRenderer {
	selections := container.NewHBox(a.selectOwner, a.selectStatus, a.selectActivity, a.selectInstaller, a.selectTag)
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
			return r.IsActive()
		case industryStatusInProgress:
			return r.status == app.JobActive
		case industryStatusReady:
			return r.status == app.JobReady
		case industryStatusHalted:
			return r.status == app.JobPaused
		case industryStatusHistory:
			return r.status == app.JobDelivered
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
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(j, k industryJobRow) int {
			var c int
			switch sortCol {
			case 0:
				c = strings.Compare(j.blueprintType.Name, k.blueprintType.Name)
			case 1:
				c = cmp.Compare(j.remaining, k.remaining)
			case 2:
				c = cmp.Compare(j.runs, k.runs)
			case 3:
				c = strings.Compare(j.activity.String(), k.activity.String())
			case 4:
				c = j.endDate.Compare(j.endDate)
			case 5:
				c = strings.Compare(j.location.Name.ValueOrZero(), k.location.Name.ValueOrZero())
			case 6:
				c = strings.Compare(j.owner.Name, k.owner.Name)
			case 7:
				c = strings.Compare(j.installer.Name, k.installer.Name)
			}
			if dir == sortAsc {
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
	reportError := func(err error) {
		slog.Error("Failed to refresh industry jobs UI", "err", err)
		fyne.Do(func() {
			a.bottom.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.bottom.Importance = widget.DangerImportance
			a.bottom.Refresh()
			a.bottom.Show()
		})
	}
	ctx := context.Background()
	cj, err := a.u.cs.ListAllCharacterIndustryJob(ctx)
	if err != nil {
		reportError(err)
		return
	}
	rj, err := a.u.rs.ListAllCorporationIndustryJobs(ctx)
	if err != nil {
		reportError(err)
		return
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
		reportError(err)
		return
	}
	cc, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		reportError(err)
		return
	}
	tagsPerCharacter := make(map[int32]set.Set[string])
	for _, c := range cc {
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			reportError(err)
			return
		}
		tagsPerCharacter[c.ID] = set.Collect(xiter.MapSlice(tags, func(x *app.CharacterTag) string {
			return x.Name
		}))
	}
	myCharacters := set.Of(xslices.Map(cc, func(c *app.EntityShort[int32]) int32 {
		return c.ID
	})...)
	characterJobs := xslices.Map(cj, func(cj *app.CharacterIndustryJob) industryJobRow {
		j := industryJobRow{
			activity:           cj.Activity,
			blueprintID:        cj.BlueprintID,
			blueprintType:      cj.BlueprintType,
			completedCharacter: cj.CompletedCharacter,
			completedDate:      cj.CompletedDate,
			cost:               cj.Cost,
			duration:           cj.Duration,
			endDate:            cj.EndDate,
			installer:          cj.Installer,
			jobID:              cj.JobID,
			licensedRuns:       cj.LicensedRuns,
			location:           cj.Station,
			owner:              eeMap[cj.CharacterID],
			pauseDate:          cj.PauseDate,
			probability:        cj.Probability,
			productType:        cj.ProductType,
			runs:               cj.Runs,
			startDate:          cj.StartDate,
			status:             cj.Status,
			successfulRuns:     cj.SuccessfulRuns,
			isInstallerMe:      true,
			isOwnerMe:          true,
			tags:               tagsPerCharacter[cj.Installer.ID],
		}
		return j
	})
	corporationJobs := xslices.Map(rj, func(rj *app.CorporationIndustryJob) industryJobRow {
		j := industryJobRow{
			activity:           rj.Activity,
			blueprintID:        rj.BlueprintID,
			blueprintType:      rj.BlueprintType,
			completedCharacter: rj.CompletedCharacter,
			completedDate:      rj.CompletedDate,
			cost:               rj.Cost,
			duration:           rj.Duration,
			endDate:            rj.EndDate,
			installer:          rj.Installer,
			jobID:              rj.JobID,
			licensedRuns:       rj.LicensedRuns,
			location:           rj.Location,
			owner:              eeMap[rj.CorporationID],
			pauseDate:          rj.PauseDate,
			probability:        rj.Probability,
			productType:        rj.ProductType,
			runs:               rj.Runs,
			startDate:          rj.StartDate,
			status:             rj.Status,
			successfulRuns:     rj.SuccessfulRuns,
			isInstallerMe:      myCharacters.Contains(rj.Installer.ID),
			isOwnerMe:          false,
			tags:               tagsPerCharacter[rj.Installer.ID],
		}
		return j
	})
	jobs := slices.Concat(characterJobs, corporationJobs)
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
		widget.NewFormItem("Owner", makeOwnerActionLabel(
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
