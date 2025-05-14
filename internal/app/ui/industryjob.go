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
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// Options for industry job select widgets
const (
	industryActivityAll              = "All activities"
	industryActivityCopying          = "Copying"
	industryActivityInvention        = "Invention"
	industryActivityManufacturing    = "Manufacturing"
	industryActivityMaterialResearch = "Material efficiency research"
	industryActivityReaction         = "Reactions"
	industryActivityTimeResearch     = "Time efficiency research"
	industryInstallerAny             = "Any installer"
	industryInstallerCorpmates       = "Installed by corpmates"
	industryInstallerMe              = "Installed by me"
	industryOwnerAny                 = "Any owner"
	industryOwnerCorp                = "Owned by corp"
	industryOwnerMe                  = "Owned by me"
	industryStatusActive             = "All active jobs"
	industryStatusHalted             = "Halted"
	industryStatusHistory            = "History"
	industryStatusInProgress         = "In progress"
	industryStatusReady              = "Ready for delivery"
)

// industryJob represents a job row in the list widgets.
// It combines character and corporation jobs and has precalcuated fields for filters.
type industryJob struct {
	activity           app.IndustryActivity
	blueprintID        int64
	blueprintType      *app.EntityShort[int32]
	completedCharacter optional.Optional[*app.EveEntity]
	completedDate      optional.Optional[time.Time]
	cost               optional.Optional[float64]
	duration           int
	endDate            time.Time
	installer          *app.EveEntity
	jobID              int32
	licensedRuns       optional.Optional[int]
	location           *app.EveLocationShort
	owner              *app.EveEntity
	pauseDate          optional.Optional[time.Time]
	probability        optional.Optional[float32]
	productType        optional.Optional[*app.EntityShort[int32]]
	runs               int
	startDate          time.Time
	status             app.IndustryJobStatus
	successfulRuns     optional.Optional[int32]
	isInstallerMe      bool
	isOwnerMe          bool
}

func (j industryJob) StatusRichText() []widget.RichTextSegment {
	if j.status == app.JobActive {
		return iwidget.NewRichTextSegmentFromText(ihumanize.Duration(time.Until(j.endDate)), widget.RichTextStyle{
			ColorName: theme.ColorNamePrimary,
		})
	}
	return iwidget.NewRichTextSegmentFromText(j.status.Display(), widget.RichTextStyle{
		ColorName: j.status.Color(),
	})
}

func (j industryJob) IsActive() bool {
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
	colSort         []sortDir
	jobs            []industryJob
	jobsFiltered    []industryJob
	search          *widget.Entry
	selectActivity  *widget.Select
	selectInstaller *widget.Select
	selectOwner     *widget.Select
	selectStatus    *widget.Select
	top             *widget.Label
	u               *BaseUI
}

func NewIndustryJobs(u *BaseUI) *industryJobs {
	headers := []iwidget.HeaderDef{
		{Text: "Blueprint", Width: 250},
		{Text: "Status", Width: 100, Refresh: true},
		{Text: "Runs", Width: 75},
		{Text: "Activity", Width: 200},
		{Text: "End date", Width: columnWidthDateTime},
		{Text: "Location", Width: columnWidthLocation},
		{Text: "Owner", Width: columnWidthCharacter},
		{Text: "Installer", Width: columnWidthCharacter},
	}
	a := &industryJobs{
		colSort:      make([]sortDir, len(headers)),
		jobs:         make([]industryJob, 0),
		jobsFiltered: make([]industryJob, 0),
		top:          appwidget.MakeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, j industryJob) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(j.blueprintType.Name)
		case 1:
			return j.StatusRichText()
		case 2:
			return iwidget.NewRichTextSegmentFromText(
				ihumanize.Comma(j.runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		case 3:
			return iwidget.NewRichTextSegmentFromText(j.activity.Display())
		case 4:
			return iwidget.NewRichTextSegmentFromText(j.endDate.Format(app.DateTimeFormat))
		case 5:
			return iwidget.NewRichTextSegmentFromText(j.location.Name.ValueOrZero())
		case 6:
			return iwidget.NewRichTextSegmentFromText(j.owner.Name)
		case 7:
			return iwidget.NewRichTextSegmentFromText(j.installer.Name)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}

	if a.u.isDesktop() {
		t := iwidget.MakeDataTableForDesktop(headers, &a.jobsFiltered, makeCell, func(_ int, r industryJob) {
			a.showJob(r)
		})
		iconSortAsc := theme.NewPrimaryThemedResource(icons.SortAscendingSvg)
		iconSortDesc := theme.NewPrimaryThemedResource(icons.SortDescendingSvg)
		iconSortOff := theme.NewThemedResource(icons.SortSvg)
		t.CreateHeader = func() fyne.CanvasObject {
			icon := widget.NewIcon(iconSortOff)
			label := kxwidget.NewTappableLabel("XXX", nil)
			return container.NewBorder(nil, nil, nil, icon, label)
		}
		t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
			h := headers[tci.Col]
			row := co.(*fyne.Container).Objects
			label := row[0].(*kxwidget.TappableLabel)
			label.OnTapped = func() {
				a.processJobs(tci.Col)
			}
			label.SetText(h.Text)
			icon := row[1].(*widget.Icon)
			switch a.colSort[tci.Col] {
			case sortOff:
				icon.SetResource(iconSortOff)
			case sortAsc:
				icon.SetResource(iconSortAsc)
			case sortDesc:
				icon.SetResource(iconSortDesc)
			}
		}
		a.body = t
	} else {
		a.body = a.makeListForMobile()
	}

	a.colSort[4] = sortDesc // default sorting

	a.search = widget.NewEntry()
	a.search.PlaceHolder = "Search Blueprints"
	a.search.OnChanged = func(_ string) {
		a.processJobs(-1)
	}
	a.search.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
	})

	a.selectOwner = widget.NewSelect([]string{
		industryOwnerAny,
		industryOwnerMe,
		industryOwnerCorp,
	}, func(_ string) {
		a.processJobs(-1)
	})
	a.selectOwner.Selected = industryOwnerAny

	a.selectStatus = widget.NewSelect([]string{
		industryStatusActive,
		industryStatusInProgress,
		industryStatusReady,
		industryStatusHalted,
		industryStatusHistory,
	}, func(_ string) {
		a.processJobs(-1)
	})
	a.selectStatus.Selected = industryStatusActive

	a.selectActivity = widget.NewSelect([]string{
		industryActivityAll,
		industryActivityManufacturing,
		industryActivityMaterialResearch,
		industryActivityTimeResearch,
		industryActivityCopying,
		industryActivityInvention,
		industryActivityReaction,
	}, func(_ string) {
		a.processJobs(-1)
	})
	a.selectActivity.Selected = industryActivityAll

	a.selectInstaller = widget.NewSelect([]string{
		industryInstallerAny,
		industryInstallerMe,
		industryInstallerCorpmates,
	}, func(_ string) {
		a.processJobs(-1)
	})
	a.selectInstaller.Selected = industryInstallerMe

	return a
}

// processJobs applies all filters and sorting and freshes the list with the changed rows.
// A new sorting can be applied by providing a sortCol. -1 does not change the current sorting.
func (a *industryJobs) processJobs(sortCol int) {
	jobs := slices.Clone(a.jobs)
	// filter
	jobs = xslices.Filter(jobs, func(o industryJob) bool {
		switch a.selectStatus.Selected {
		case industryStatusActive:
			return o.IsActive()
		case industryStatusInProgress:
			return o.status == app.JobActive
		case industryStatusReady:
			return o.status == app.JobReady
		case industryStatusHalted:
			return o.status == app.JobPaused
		case industryStatusHistory:
			return o.status == app.JobDelivered
		}
		return false
	})
	if a.selectInstaller.Selected != industryInstallerAny {
		jobs = xslices.Filter(jobs, func(o industryJob) bool {
			switch a.selectInstaller.Selected {
			case industryInstallerMe:
				return o.isInstallerMe
			case industryInstallerCorpmates:
				return !o.isInstallerMe
			}
			return false
		})
	}
	if a.selectActivity.Selected != industryActivityAll {
		jobs = xslices.Filter(jobs, func(o industryJob) bool {
			switch a.selectActivity.Selected {
			case industryActivityCopying:
				return o.activity == app.Copying
			case industryActivityInvention:
				return o.activity == app.Invention
			case industryActivityManufacturing:
				return o.activity == app.Manufacturing
			case industryActivityMaterialResearch:
				return o.activity == app.MaterialEfficiencyResearch
			case industryActivityReaction:
				return o.activity == app.Reactions
			case industryActivityTimeResearch:
				return o.activity == app.TimeEfficiencyResearch
			}
			return false
		})
	}
	if a.selectOwner.Selected != industryOwnerAny {
		jobs = xslices.Filter(jobs, func(o industryJob) bool {
			switch a.selectOwner.Selected {
			case industryOwnerCorp:
				return !o.isOwnerMe
			case industryOwnerMe:
				return o.isOwnerMe
			}
			return false
		})
	}
	if s := a.search.Text; len(s) > 1 {
		jobs = xslices.Filter(jobs, func(x industryJob) bool {
			return strings.Contains(strings.ToLower(x.blueprintType.Name), strings.ToLower(s))
		})
	}
	// sort
	var order sortDir
	if sortCol >= 0 {
		order = a.colSort[sortCol]
		order++
		if order > sortDesc {
			order = sortOff
		}
		for i := range a.colSort {
			a.colSort[i] = sortOff
		}
		a.colSort[sortCol] = order
	} else {
		for i := range a.colSort {
			if a.colSort[i] != sortOff {
				order = a.colSort[i]
				sortCol = i
				break
			}
		}
	}
	if sortCol >= 0 && order != sortOff {
		slices.SortFunc(jobs, func(j, k industryJob) int {
			var c int
			switch sortCol {
			case 0:
				c = strings.Compare(j.blueprintType.Name, k.blueprintType.Name)
			case 1:
				c = strings.Compare(j.status.String(), k.status.String())
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
			if order == sortAsc {
				return c
			} else {
				return -1 * c
			}
		})
	}
	// set data & refresh
	a.jobsFiltered = jobs
	a.body.Refresh()
	switch x := a.body.(type) {
	case *widget.Table:
		x.ScrollToTop()
	}
}

func (a *industryJobs) CreateRenderer() fyne.WidgetRenderer {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(180, 1))
	c := container.NewBorder(
		container.NewVBox(
			container.NewBorder(
				nil,
				container.NewHScroll(container.NewHBox(
					container.NewStack(spacer, a.selectOwner),
					container.NewStack(spacer, a.selectStatus),
					container.NewStack(spacer, a.selectActivity),
					container.NewStack(spacer, a.selectInstaller),
				)),
				nil,
				nil,
				a.search,
			),
			a.top,
		),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *industryJobs) makeListForMobile() *widget.List {
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(a.jobsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("Template")
			title.TextStyle.Bold = true
			title.Wrapping = fyne.TextWrapWord
			status := widget.NewRichText()
			activity := widget.NewLabel("Template")
			location := widget.NewLabel("Template")
			location.Wrapping = fyne.TextWrapWord
			completed := widget.NewLabel("Template")
			p := theme.Padding()
			return container.NewBorder(
				nil,
				nil,
				nil,
				status,
				container.New(layout.NewCustomPaddedVBoxLayout(-p),
					title,
					activity,
					location,
					completed,
				),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.jobsFiltered) || id < 0 {
				return
			}
			j := a.jobsFiltered[id]
			c1 := co.(*fyne.Container).Objects
			c2 := c1[0].(*fyne.Container).Objects
			c2[0].(*widget.Label).SetText(j.blueprintType.Name)
			c2[1].(*widget.Label).SetText(fmt.Sprintf("%s x %s", j.activity.Display(), ihumanize.Comma(j.runs)))
			c2[2].(*widget.Label).SetText(j.location.Name.ValueOrFallback("?"))
			iwidget.SetRichText(c1[1].(*widget.RichText), j.StatusRichText()...)
			completed := c2[3].(*widget.Label)
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
		if id >= len(a.jobsFiltered) || id < 0 {
			return
		}
		a.showJob(a.jobsFiltered[id])
	}
	return l
}

func (a *industryJobs) update() {
	reportError := func(err error) {
		slog.Error("Failed to refresh industry jobs UI", "err", err)
		fyne.Do(func() {
			a.top.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
	}
	fixStatus := func(s app.IndustryJobStatus, endDate time.Time) app.IndustryJobStatus {
		if s == app.JobActive && !endDate.IsZero() && endDate.Before(time.Now()) {
			// Workaround for known bug: https://github.com/esi/esi-issues/issues/752
			return app.JobReady
		}
		return s
	}
	cj, err := a.u.cs.ListAllCharacterIndustryJob(context.Background())
	if err != nil {
		reportError(err)
		return
	}
	rj, err := a.u.rs.ListAllCorporationIndustryJobs(context.Background())
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
	eeMap, err := a.u.eus.ToEntities(context.Background(), ids)
	if err != nil {
		reportError(err)
		return
	}
	cc, err := a.u.cs.ListCharactersShort(context.Background())
	if err != nil {
		reportError(err)
		return
	}
	myCharacters := set.Of(xslices.Map(cc, func(c *app.EntityShort[int32]) int32 {
		return c.ID
	})...)
	characterJobs := xslices.Map(cj, func(cj *app.CharacterIndustryJob) industryJob {
		j := industryJob{
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
			status:             fixStatus(cj.Status, cj.EndDate),
			successfulRuns:     cj.SuccessfulRuns,
			isInstallerMe:      true,
			isOwnerMe:          true,
		}
		return j
	})
	corporationJobs := xslices.Map(rj, func(rj *app.CorporationIndustryJob) industryJob {
		j := industryJob{
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
			status:             fixStatus(rj.Status, rj.EndDate),
			successfulRuns:     rj.SuccessfulRuns,
			isInstallerMe:      myCharacters.Contains(rj.Installer.ID),
			isOwnerMe:          false,
		}
		return j
	})
	jobs := slices.Concat(characterJobs, corporationJobs)
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
		a.top.Hide()
		a.jobs = jobs
		a.processJobs(-1)
	})
}

func (a *industryJobs) showJob(r industryJob) {
	makeLocationWidget := func(o *app.EveLocationShort) *iwidget.TappableRichText {
		x := iwidget.NewTappableRichText(func() {
			a.u.ShowLocationInfoWindow(o.ID)
		},
			o.DisplayRichText()...,
		)
		x.Wrapping = fyne.TextWrapWord
		return x
	}
	newTappableLabelWithWrap := func(text string, f func()) *kxwidget.TappableLabel {
		x := kxwidget.NewTappableLabel(text, f)
		x.Wrapping = fyne.TextWrapWord
		return x
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Blueprint", newTappableLabelWithWrap(r.blueprintType.Name, func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, r.blueprintType.ID)
		})),
		widget.NewFormItem("Activity", widget.NewLabel(r.activity.Display())),
	}
	if !r.productType.IsEmpty() {
		x := r.productType.MustValue()
		items = append(items, widget.NewFormItem(
			"Product Type",
			newTappableLabelWithWrap(x.Name, func() {
				a.u.ShowInfoWindow(app.EveEntityInventoryType, x.ID)
			}),
		))
	}
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Status", widget.NewRichText(r.StatusRichText()...)),
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
		widget.NewFormItem("Location", makeLocationWidget(r.location)),
		widget.NewFormItem("Installer", newTappableLabelWithWrap(r.installer.Name, func() {
			a.u.ShowEveEntityInfoWindow(r.installer)
		})),
		widget.NewFormItem("Owner", newTappableLabelWithWrap(r.owner.Name, func() {
			a.u.ShowEveEntityInfoWindow(r.owner)
		})),
		widget.NewFormItem("Type", widget.NewLabel(r.owner.CategoryDisplay())),
	})
	if !r.completedCharacter.IsEmpty() {
		x := r.completedCharacter.MustValue()
		items = append(items, widget.NewFormItem("Completed By", newTappableLabelWithWrap(x.Name, func() {
			a.u.ShowEveEntityInfoWindow(x)
		})))
	}
	if a.u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Job ID", a.u.makeCopyToClipboardLabel(fmt.Sprint(r.jobID))))
	}
	title := fmt.Sprintf("%s - %s - #%d", r.blueprintType.Name, r.activity.Display(), r.jobID)
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	w := a.u.makeDetailWindow("Industry Job", title, f)
	w.Show()
}
