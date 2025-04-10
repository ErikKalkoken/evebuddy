package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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
			return iwidget.NewRichTextSegmentFromText(status.Display(), widget.RichTextStyle{
				ColorName: status.Color(),
			})
		case 2:
			var s string
			if status == app.JobActive {
				s = humanize.Duration(time.Until(r.EndDate))
			} else {
				s = ""
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 3:
			return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.Runs),
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.Activity.Display())
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.Facility.Name)
		case 6:
			return iwidget.NewRichTextSegmentFromText(r.StartDate.Format(app.DateTimeFormat))
		case 7:
			return iwidget.NewRichTextSegmentFromText(r.EndDate.Format(app.DateTimeFormat))
		case 8:
			return iwidget.NewRichTextSegmentFromText(r.Installer.Name)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop2(headers, &a.jobs, makeCell, func(col int, r *app.CharacterIndustryJob) {
			switch col {
			case 0:
				a.u.ShowInfoWindow(app.EveEntityInventoryType, r.BlueprintType.ID)
			case 5:
				a.u.ShowLocationInfoWindow(r.Facility.ID)
			case 8:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.Installer.ID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.jobs, makeCell, nil)
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
		a.top.Text = fmt.Sprintf("ERROR: %s", humanize.Error(err))
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
