package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	ShowActiveOnly bool
	OnUpdate       func(count int)

	body fyne.CanvasObject
	jobs []*app.CharacterIndustryJob
	top  *widget.Label
	u    *BaseUI
}

func NewIndustryJobs(u *BaseUI) *IndustryJobs {
	a := &IndustryJobs{
		jobs: make([]*app.CharacterIndustryJob, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Blueprint", Width: 250},
		{Text: "Status", Width: 100},
		{Text: "Remain", Width: 100, Refresh: true},
		{Text: "Runs", Width: 50},
		{Text: "Activity", Width: 200},
		{Text: "Facility", Width: columnWidthLocation},
		{Text: "Install date", Width: columnWidthDateTime},
		{Text: "End date", Width: columnWidthDateTime},
		{Text: "Installer", Width: columnWidthCharacter},
	}
	makeCell := func(col int, r *app.CharacterIndustryJob) []widget.RichTextSegment {
		status := r.StatusCorrected()
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.BlueprintType.Name)
		case 1:
			return r.StatusRichText()
		case 2:
			var s string
			if status == app.JobActive {
				s = ihumanize.Duration(time.Until(r.EndDate))
			} else {
				s = ""
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 3:
			return iwidget.NewRichTextSegmentFromText(
				ihumanize.Comma(r.Runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.Activity.Display())
		case 5:
			return r.Facility.DisplayRichText()
		case 6:
			return iwidget.NewRichTextSegmentFromText(r.StartDate.Format(app.DateTimeFormat))
		case 7:
			return iwidget.NewRichTextSegmentFromText(r.EndDate.Format(app.DateTimeFormat))
		case 8:
			return iwidget.NewRichTextSegmentFromText(r.Installer.Name)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	showDetail := func(r *app.CharacterIndustryJob) {
		makeLocationWidget := func(o *app.EveLocationShort) *iwidget.TappableRichText {
			return iwidget.NewTappableRichText(func() {
				a.u.ShowLocationInfoWindow(o.ID)
			},
				o.DisplayRichText()...,
			)
		}
		items := []*widget.FormItem{
			widget.NewFormItem("Blueprint", kxwidget.NewTappableLabel(r.BlueprintType.Name, func() {
				a.u.ShowInfoWindow(app.EveEntityInventoryType, r.BlueprintType.ID)
			})),
			widget.NewFormItem("Activity", widget.NewLabel(r.Activity.Display())),
		}
		if !r.ProductType.IsEmpty() {
			x := r.ProductType.MustValue()
			items = append(items, widget.NewFormItem(
				"Product Type",
				kxwidget.NewTappableLabel(x.Name, func() {
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

		items = slices.Concat(items, []*widget.FormItem{
			widget.NewFormItem("Facility", makeLocationWidget(r.Facility)),
			widget.NewFormItem("Start date", widget.NewLabel(r.StartDate.Format(app.DateTimeFormat))),
			widget.NewFormItem("End date (est.)", widget.NewLabel(r.EndDate.Format(app.DateTimeFormat))),
			widget.NewFormItem("Installer", kxwidget.NewTappableLabel(r.Installer.Name, func() {
				a.u.ShowEveEntityInfoWindow(r.Installer)
			})),
			widget.NewFormItem("Blueprint Location", makeLocationWidget(r.BlueprintLocation)),
			widget.NewFormItem("Output Location", makeLocationWidget(r.OutputLocation)),
			widget.NewFormItem("Station", makeLocationWidget(r.Station)),
		})

		if !r.PauseDate.IsEmpty() {
			items = append(items, widget.NewFormItem(
				"Pause date",
				widget.NewLabel(r.PauseDate.ValueOrZero().Format(app.DateTimeFormat)),
			))
		}
		if !r.CompletedCharacter.IsEmpty() {
			x := r.CompletedCharacter.MustValue()
			items = append(items, widget.NewFormItem("Completed By", kxwidget.NewTappableLabel(x.Name, func() {
				a.u.ShowEveEntityInfoWindow(x)
			})))
		}
		if !r.CompletedDate.IsEmpty() {
			items = append(items, widget.NewFormItem(
				"Completed date",
				widget.NewLabel(r.CompletedDate.ValueOrZero().Format(app.DateTimeFormat))),
			)
		}
		title := fmt.Sprintf("%s - %s - #%d", r.BlueprintType.Name, r.Activity.Display(), r.JobID)
		w := a.u.makeDetailWindow("Industry Job", title, widget.NewForm(items...))
		w.Show()
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop2(headers, &a.jobs, makeCell, func(_ int, r *app.CharacterIndustryJob) {
			showDetail(r)
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.jobs, makeCell, showDetail)
	}
	return a
}

func (a *IndustryJobs) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *IndustryJobs) Update() {
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		a.top.Text = fmt.Sprintf("ERROR: %s", ihumanize.Error(err))
		a.top.Importance = widget.DangerImportance
		a.top.Refresh()
		a.top.Show()
		return
	}
	a.top.Hide()
	a.body.Refresh()
}

func (a *IndustryJobs) updateEntries() error {
	jobs, err := a.u.CharacterService().ListAllCharacterIndustryJob(context.TODO())
	if err != nil {
		return err
	}
	if a.ShowActiveOnly {
		a.jobs = xslices.Filter(jobs, func(o *app.CharacterIndustryJob) bool {
			return o.IsActive()
		})

	} else {
		a.jobs = jobs
	}
	if a.OnUpdate != nil {
		a.OnUpdate(len(a.jobs))
	}
	return nil
}
