package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

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
}

func (j industryJob) StatusRichText() []widget.RichTextSegment {
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

type IndustryJobs struct {
	widget.BaseWidget

	OnUpdate func(count int)

	body           fyne.CanvasObject
	jobs           []industryJob
	jobsFiltered   []industryJob
	showActiveOnly atomic.Bool
	top            *widget.Label
	u              *BaseUI
	search         *widget.Entry
}

func NewIndustryJobs(u *BaseUI, showActiveOnly bool) *IndustryJobs {
	a := &IndustryJobs{
		jobs:         make([]industryJob, 0),
		jobsFiltered: make([]industryJob, 0),
		top:          appwidget.MakeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.showActiveOnly.Store(showActiveOnly)
	headers := []iwidget.HeaderDef{
		{Text: "Blueprint", Width: 250},
		{Text: "Status", Width: 100, Refresh: true},
		{Text: "Runs", Width: 50},
		{Text: "Activity", Width: 200},
		{Text: "End date", Width: columnWidthDateTime},
		{Text: "Location", Width: columnWidthLocation},
		{Text: "Owner", Width: columnWidthCharacter},
		{Text: "Installer", Width: columnWidthCharacter},
	}
	makeCell := func(col int, j industryJob) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(j.blueprintType.Name)
		case 1:
			if j.status == app.JobActive {
				return iwidget.NewRichTextSegmentFromText(ihumanize.Duration(time.Until(j.endDate)), widget.RichTextStyle{
					ColorName: theme.ColorNamePrimary,
				})
			}
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
			return j.location.DisplayRichText()
		case 6:
			return iwidget.NewRichTextSegmentFromText(j.owner.Name)
		case 7:
			return iwidget.NewRichTextSegmentFromText(j.installer.Name)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}

	if a.u.isDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.jobsFiltered, makeCell, func(_ int, r industryJob) {
			a.showJob(r)
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.jobsFiltered, makeCell, a.showJob)
	}
	a.search = widget.NewEntry()
	a.search.PlaceHolder = "Search blueprints"
	a.search.OnChanged = func(s string) {
		if len(s) < 2 {
			a.jobsFiltered = slices.Clone(a.jobs)
			a.body.Refresh()
			return
		}
		a.jobsFiltered = xslices.Filter(a.jobs, func(x industryJob) bool {
			return strings.Contains(strings.ToLower(x.blueprintType.Name), strings.ToLower(s))
		})
		a.body.Refresh()
	}
	a.search.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
	})
	return a
}

func (a *IndustryJobs) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(container.NewVBox(a.search, a.top), nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *IndustryJobs) update() {
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
		if s == app.JobActive && endDate.Before(time.Now()) {
			// Workaroud for known bug: https://github.com/esi/esi-issues/issues/752
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
	eeMap, err := a.u.eus.ToEntities(context.Background(), ids.Slice())
	if err != nil {
		reportError(err)
		return
	}
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
		}
		return j
	})
	jobs := slices.Concat(characterJobs, corporationJobs)
	if a.showActiveOnly.Load() {
		jobs = xslices.Filter(jobs, func(o industryJob) bool {
			return o.IsActive()
		})
	}
	var readyCount int
	for _, j := range jobs {
		if j.status == app.JobReady {
			readyCount++
		}
	}
	if a.OnUpdate != nil {
		a.OnUpdate(readyCount)
	}
	fyne.Do(func() {
		a.top.Hide()
		a.jobs = jobs
		a.jobsFiltered = slices.Clone(jobs)
		a.search.SetText("")
		a.body.Refresh()
	})
}

func (a *IndustryJobs) showJob(r industryJob) {
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
