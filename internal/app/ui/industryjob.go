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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type IndustryJobs struct {
	widget.BaseWidget

	OnUpdate func(count int)

	body           fyne.CanvasObject
	jobs           []*app.CharacterIndustryJob
	jobsFiltered   []*app.CharacterIndustryJob
	showActiveOnly atomic.Bool
	top            *widget.Label
	u              *BaseUI
	search         *widget.Entry
}

func NewIndustryJobs(u *BaseUI, showActiveOnly bool) *IndustryJobs {
	a := &IndustryJobs{
		jobs:         make([]*app.CharacterIndustryJob, 0),
		jobsFiltered: make([]*app.CharacterIndustryJob, 0),
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
		{Text: "Facility", Width: columnWidthLocation},
		{Text: "Installer", Width: columnWidthCharacter},
	}
	makeCell := func(col int, r *app.CharacterIndustryJob) []widget.RichTextSegment {
		status := r.StatusCorrected()
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.BlueprintType.Name)
		case 1:
			if status == app.JobActive {
				return iwidget.NewRichTextSegmentFromText(ihumanize.Duration(time.Until(r.EndDate)), widget.RichTextStyle{
					ColorName: theme.ColorNamePrimary,
				})
			}
			return r.StatusRichText()
		case 2:
			return iwidget.NewRichTextSegmentFromText(
				ihumanize.Comma(r.Runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		case 3:
			return iwidget.NewRichTextSegmentFromText(r.Activity.Display())
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.EndDate.Format(app.DateTimeFormat))
		case 5:
			return r.Facility.DisplayRichText()
		case 6:
			return iwidget.NewRichTextSegmentFromText(r.Installer.Name)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}

	if a.u.isDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.jobsFiltered, makeCell, func(_ int, r *app.CharacterIndustryJob) {
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
		a.jobsFiltered = xslices.Filter(a.jobs, func(x *app.CharacterIndustryJob) bool {
			return strings.Contains(strings.ToLower(x.BlueprintType.Name), strings.ToLower(s))
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
	jobs, err := a.u.cs.ListAllCharacterIndustryJob(context.Background())
	if err != nil {
		slog.Error("Failed to refresh industry jobs UI", "err", err)
		fyne.Do(func() {
			a.top.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}
	if a.showActiveOnly.Load() {
		jobs = xslices.Filter(jobs, func(o *app.CharacterIndustryJob) bool {
			return o.IsActive()
		})
	}
	var readyCount int
	for _, j := range jobs {
		if j.StatusCorrected() == app.JobReady {
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

func (a *IndustryJobs) showJob(r *app.CharacterIndustryJob) {
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
		widget.NewFormItem("Blueprint", newTappableLabelWithWrap(r.BlueprintType.Name, func() {
			a.u.ShowInfoWindow(app.EveEntityInventoryType, r.BlueprintType.ID)
		})),
		widget.NewFormItem("Activity", widget.NewLabel(r.Activity.Display())),
	}
	if !r.ProductType.IsEmpty() {
		x := r.ProductType.MustValue()
		items = append(items, widget.NewFormItem(
			"Product Type",
			newTappableLabelWithWrap(x.Name, func() {
				a.u.ShowInfoWindow(app.EveEntityInventoryType, x.ID)
			}),
		))
	}
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Status", widget.NewRichText(r.StatusRichText()...)),
		widget.NewFormItem("Runs", widget.NewLabel(ihumanize.Comma(r.Runs))),
	})

	if !r.LicensedRuns.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Licensed Runs",
			widget.NewLabel(ihumanize.Comma(r.LicensedRuns.ValueOrZero())),
		))
	}
	if !r.SuccessfulRuns.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Successful Runs",
			widget.NewLabel(ihumanize.Comma(r.SuccessfulRuns.ValueOrZero())),
		))
	}
	if !r.Probability.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Probability",
			widget.NewLabel(fmt.Sprintf("%.0f%%", r.Probability.ValueOrZero()*100)),
		))
	}
	items = append(items, widget.NewFormItem("Start date", widget.NewLabel(r.StartDate.Format(app.DateTimeFormat))))
	if !r.PauseDate.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Pause date",
			widget.NewLabel(r.PauseDate.ValueOrZero().Format(app.DateTimeFormat)),
		))
	}
	items = append(items, widget.NewFormItem("End date", widget.NewLabel(r.EndDate.Format(app.DateTimeFormat))))
	if !r.CompletedDate.IsEmpty() {
		items = append(items, widget.NewFormItem(
			"Completed date",
			widget.NewLabel(r.CompletedDate.ValueOrZero().Format(app.DateTimeFormat))),
		)
	}

	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Facility", makeLocationWidget(r.Facility)),
		widget.NewFormItem("Blueprint Location", makeLocationWidget(r.BlueprintLocation)),
		widget.NewFormItem("Output Location", makeLocationWidget(r.OutputLocation)),
		widget.NewFormItem("Station", makeLocationWidget(r.Station)),
		widget.NewFormItem("Installer", newTappableLabelWithWrap(r.Installer.Name, func() {
			a.u.ShowEveEntityInfoWindow(r.Installer)
		})),
		widget.NewFormItem("Owner", newTappableLabelWithWrap(
			a.u.scs.CharacterName(r.CharacterID),
			func() {
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.CharacterID)
			},
		)),
	})
	if !r.CompletedCharacter.IsEmpty() {
		x := r.CompletedCharacter.MustValue()
		items = append(items, widget.NewFormItem("Completed By", newTappableLabelWithWrap(x.Name, func() {
			a.u.ShowEveEntityInfoWindow(x)
		})))
	}
	if a.u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Job ID", a.u.makeCopyToClipboardLabel(fmt.Sprint(r.JobID))))
	}
	title := fmt.Sprintf("%s - %s - #%d", r.BlueprintType.Name, r.Activity.Display(), r.JobID)
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	w := a.u.makeDetailWindow("Industry Job", title, f)
	w.Show()
}
