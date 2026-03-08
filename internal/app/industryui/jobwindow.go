package industryui

import (
	"context"
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// showIndustryJobWindow shows the details of a industry job in a window.
func showIndustryJobWindow(u ui, r industryJobRow) {
	title := fmt.Sprintf("Industry Job #%d", r.jobID)
	key := fmt.Sprintf("industryjob-%d-%d", r.owner.ID, r.jobID)
	w, ok, onClosed := u.GetOrCreateWindowWithOnClosed(key, title, r.owner.Name)
	if !ok {
		w.Show()
		return
	}

	activity := fmt.Sprintf("%s (%s)", r.activity.Display(), r.activity.JobType().Display())
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.owner.ID,
			r.owner.Name,
			u.InfoWindow().ShowEntity,
		)),
		widget.NewFormItem("Blueprint", makeLinkLabelWithWrap(r.blueprintType.Name, func() {
			u.InfoWindow().Show(app.EveEntityInventoryType, r.blueprintType.ID)
		})),
		widget.NewFormItem("Activity", widget.NewLabel(activity)),
	}
	if v, ok := r.productType.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Product Type",
			makeLinkLabelWithWrap(v.Name, func() {
				u.InfoWindow().Show(app.EveEntityInventoryType, v.ID)
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
	items = append(items, widget.NewFormItem(
		"End date",
		widget.NewLabel(r.endDate.Format(app.DateTimeFormat)),
	))
	if v, ok := r.completedDate.Value(); ok {
		items = append(items, widget.NewFormItem(
			"Completed date",
			widget.NewLabel(v.Format(app.DateTimeFormat))),
		)
	}
	items = slices.Concat(items, []*widget.FormItem{
		widget.NewFormItem("Location", makeLocationLabel(r.location, u.InfoWindow().ShowLocation)),
		widget.NewFormItem("Installer", makeLinkLabelWithWrap(r.installer.Name, func() {
			u.InfoWindow().ShowEntity(r.installer)
		})),
		widget.NewFormItem("Type", widget.NewLabel(r.owner.CategoryDisplay())),
	})
	if v, ok := r.completedCharacter.Value(); ok {
		items = append(items, widget.NewFormItem("Completed By", makeLinkLabelWithWrap(v.Name, func() {
			u.InfoWindow().ShowEntity(v)
		})))
	}
	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem(
			"Job ID",
			xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(r.jobID)),
		))
	}
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			status.Set(r.statusDisplay())
		})
	}, key)
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		u.Signals().RefreshTickerExpired.RemoveListener(key)
	})
	xwindow.Set(xwindow.Params{
		Content: f,
		ImageAction: func() {
			u.InfoWindow().ShowType(r.blueprintType.ID)
		},
		ImageLoader: func(setter func(r fyne.Resource)) {
			u.EVEImage().InventoryTypeBPOAsync(r.blueprintType.ID, 256, setter)
		},
		Title:  title,
		Window: w,
	})
	w.Show()
}
