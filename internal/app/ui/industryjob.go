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

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type industryJobs struct {
	widget.BaseWidget

	body fyne.CanvasObject
	jobs []*app.CharacterIndustryJob
	top  *widget.Label
	u    *BaseUI
}

func newIndustryJobs(u *BaseUI) *industryJobs {
	a := &industryJobs{
		jobs: make([]*app.CharacterIndustryJob, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Blueprint", Width: 200},
		{Text: "Status", Width: 100},
		{Text: "Remain", Width: 100, Refresh: true},
		{Text: "Runs", Width: 50},
		{Text: "Activity", Width: 150},
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
			return iwidget.NewRichTextSegmentFromText(status.String())
		case 2:
			var s string
			if status == app.JobActive {
				s = humanize.Duration(time.Until(r.EndDate))
			} else {
				s = ""
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 3:
			return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.Runs))
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.Activity.String())
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

func (a *industryJobs) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *industryJobs) Update() {
	var s string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		s = "ERROR"
		i = widget.DangerImportance
	} else {
		s = fmt.Sprintf("%d jobs", len(a.jobs))
	}
	a.top.Text = s
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
}

func (a *industryJobs) updateEntries() error {
	jobs, err := a.u.CharacterService().ListAllCharacterIndustryJob(context.TODO())
	if err != nil {
		return err
	}
	a.jobs = jobs
	return nil
}
