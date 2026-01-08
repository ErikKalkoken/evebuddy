package ui

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type characterSkillQueue struct {
	widget.BaseWidget

	OnUpdate func(statusShort, statusLong string)

	character            atomic.Pointer[app.Character]
	emptyInfo            *widget.Label
	list                 *widget.List
	showCurrentCharacter bool
	signalKey            string
	skillqueue           *app.CharacterSkillqueue
	status               *ttwidget.Icon
	statusResource       fyne.Resource
	top                  *widget.Label
	u                    *baseUI
}

// newCharacterSkillQueue returns a new characterSkillQueue for the current character.
func newCharacterSkillQueue(u *baseUI) *characterSkillQueue {
	return newCharacterSkillQueueWithCharacter(u, nil)
}

// newCharacterSkillQueue returns a new characterSkillQueue for character c.
func newCharacterSkillQueueWithCharacter(u *baseUI, c *app.Character) *characterSkillQueue {
	emptyInfo := widget.NewLabel("Queue is empty")
	emptyInfo.Importance = widget.LowImportance
	emptyInfo.Hide()
	statusResources := theme.MediaRecordIcon()
	a := &characterSkillQueue{
		emptyInfo:            emptyInfo,
		showCurrentCharacter: c == nil,
		signalKey:            generateUniqueID(),
		skillqueue:           app.NewCharacterSkillqueue(),
		statusResource:       statusResources,
		status:               ttwidget.NewIcon(theme.NewDisabledResource(statusResources)),
		top:                  makeTopLabel(),
		u:                    u,
	}
	a.ExtendBaseWidget(a)
	a.character.Store(c)
	a.list = a.makeSkillQueue()
	return a
}

func (a *characterSkillQueue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewBorder(nil, nil, a.status, nil, a.top),
		nil,
		nil,
		nil,
		container.NewStack(a.emptyInfo, a.list),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterSkillQueue) makeSkillQueue() *widget.List {
	list := widget.NewList(
		func() int {
			return a.skillqueue.Size()
		},
		func() fyne.CanvasObject {
			level := newSkillLevel()
			if a.u.isMobile {
				level.Hide()
			}
			return container.NewBorder(nil, nil, level, nil, newSkillQueueItem(a.u.isMobile))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			qi := a.skillqueue.Item(id)
			if qi == nil {
				return
			}
			c := co.(*fyne.Container).Objects
			c[0].(*skillQueueItem).Set(qi)

			level := c[1].(*skillLevel)
			var active, trained, required int
			if qi.IsCompleted() {
				active = qi.FinishedLevel
				trained = qi.FinishedLevel
				required = qi.FinishedLevel
			} else if qi.IsActive() {
				active = qi.FinishedLevel - 1
				trained = qi.FinishedLevel - 1
				required = 0
			} else {
				active = qi.FinishedLevel - 1
				trained = qi.FinishedLevel - 1
				required = qi.FinishedLevel
			}
			level.Set(active, trained, required)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		q := a.skillqueue.Item(id)
		if q == nil {
			return
		}
		showSkillInTrainingWindow(a.u, q)
	}
	return list
}

func (a *characterSkillQueue) start() {
	if a.showCurrentCharacter {
		a.u.currentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
			a.character.Store(c)
			a.update()
		},
			a.signalKey,
		)
	}
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterSkillqueue {
			a.update()
		}
	}, a.signalKey)
	a.u.refreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			a.update()
		})
	}, a.signalKey)
}

func (a *characterSkillQueue) stop() {
	if a.showCurrentCharacter {
		a.u.currentCharacterExchanged.RemoveListener(a.signalKey)
	}
	a.u.characterSectionChanged.RemoveListener(a.signalKey)
	a.u.refreshTickerExpired.RemoveListener(a.signalKey)
}

func (a *characterSkillQueue) update() {
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
		})
	}
	clear := func() {
		fyne.Do(func() {
			a.list.Hide()
			a.emptyInfo.Hide()
			a.status.Hide()
		})
	}

	characterID := characterIDOrZero(a.character.Load())
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterSkillqueue)
	if !hasData {
		setTop("Waiting for character data to be loaded...", widget.WarningImportance)
		clear()
		return
	}
	err := a.skillqueue.Update(context.Background(), a.u.cs, characterID)
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		setTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		clear()
		return
	}

	isActive := a.skillqueue.IsActive()
	if isActive {
		s := fmt.Sprintf("Total training time: %s", ihumanize.Optional(a.skillqueue.RemainingTime(), "?"))
		setTop(s, widget.MediumImportance)
	} else {
		setTop("Training not active", widget.MediumImportance)
	}
	fyne.Do(func() {
		var r fyne.Resource
		var s string
		if isActive {
			r = theme.NewSuccessThemedResource(a.statusResource)
			s = "Training is active"
		} else {
			r = theme.NewDisabledResource(a.statusResource)
			s = "Training is not active"
		}
		a.status.SetResource(r)
		a.status.SetToolTip(s)
		a.status.Show()
	})
	if a.OnUpdate != nil {
		var s1, s2 string
		if !isActive {
			s1 = "!"
			s2 = "training paused"
		} else {
			if c := a.skillqueue.CompletionP(); c.ValueOrZero() < 1 {
				s1 = fmt.Sprintf("%.0f%%", c.ValueOrZero()*100)
				s2 = fmt.Sprintf("%s (%s)", a.skillqueue.Active(), s1)
			}
		}
		a.OnUpdate(s1, s2)
	}

	fyne.Do(func() {
		if a.skillqueue.Size() == 0 {
			a.emptyInfo.Show()
		} else {
			a.emptyInfo.Hide()
		}
		a.list.Refresh()
		a.list.Show()
	})
}

func showSkillInTrainingWindow(u *baseUI, r *app.CharacterSkillqueueItem) {
	characterName := u.scs.CharacterName(r.CharacterID)
	w, created := u.getOrCreateWindow(
		fmt.Sprintf("skill-%d-%d", r.CharacterID, r.SkillID),
		"Skill: Information",
		characterName,
	)
	if !created {
		w.Show()
		return
	}
	description := widget.NewLabel(r.SkillDescription)
	description.Wrapping = fyne.TextWrapWord
	var isActive *widget.Label
	if r.IsActive() {
		isActive = widget.NewLabel("active")
		isActive.Importance = widget.SuccessImportance
	} else {
		isActive = widget.NewLabel("inactive")
		isActive.Importance = widget.LowImportance
	}
	items := []*widget.FormItem{
		widget.NewFormItem(
			"Owner",
			makeCharacterActionLabel(r.CharacterID, characterName, u.ShowEveEntityInfoWindow),
		),
		widget.NewFormItem("Skill", makeLinkLabel(app.SkillDisplayName(r.SkillName, r.FinishedLevel), func() {
			u.ShowTypeInfoWindowWithCharacter(r.SkillID, r.CharacterID)
		})),
		widget.NewFormItem("Group", widget.NewLabel(r.GroupName)),
		widget.NewFormItem("Description", description),
		widget.NewFormItem("Active?", isActive),
		widget.NewFormItem("Completed", widget.NewLabel(fmt.Sprintf("%.0f%%", r.CompletionP()*100))),
		widget.NewFormItem("Remaining", widget.NewLabel(ihumanize.Optional(r.Remaining(), "?"))),
		widget.NewFormItem("Duration", widget.NewLabel(ihumanize.Optional(r.Duration(), "?"))),
		widget.NewFormItem("Start date", widget.NewLabel(timeFormattedOrFallback(r.StartDate, app.DateTimeFormat, "?"))),
		widget.NewFormItem("End date", widget.NewLabel(timeFormattedOrFallback(r.FinishDate, app.DateTimeFormat, "?"))),
		widget.NewFormItem("SP at start", widget.NewLabel(humanize.Comma(int64(r.TrainingStartSP-r.LevelStartSP)))),
		widget.NewFormItem("Total SP", widget.NewLabel(humanize.Comma(int64(r.LevelEndSP-r.LevelStartSP)))),
	}
	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem(
			"Queue Position",
			widget.NewLabel(fmt.Sprint(r.QueuePosition)),
		))
	}

	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("%s by %s", app.SkillDisplayName(r.SkillName, r.FinishedLevel), characterName)
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindowWithCharacter(r.SkillID, r.CharacterID)
		},
		imageLoader: func() (fyne.Resource, error) {
			return u.eis.InventoryTypeIcon(r.SkillID, 256)
		},
		minSize: fyne.NewSize(500, 450),
		title:   subTitle,
		window:  w,
	})
	w.Show()
}

type skillQueueItem struct {
	widget.BaseWidget

	Placeholder string

	duration *widget.Label
	isMobile bool
	name     *widget.Label
	progress *widget.ProgressBar
}

func newSkillQueueItem(isMobile bool) *skillQueueItem {
	pb := widget.NewProgressBar()
	w := &skillQueueItem{
		Placeholder: "N/A",
		duration:    widget.NewLabel(""),
		progress:    pb,
		isMobile:    isMobile,
	}
	w.ExtendBaseWidget(w)
	w.name = widget.NewLabel(w.Placeholder)
	w.name.Truncation = fyne.TextTruncateEllipsis
	pb.Hide()
	if w.isMobile {
		pb.TextFormatter = func() string {
			return ""
		}
	}
	return w
}

func (w *skillQueueItem) Set(qi *app.CharacterSkillqueueItem) {
	var (
		completionP float64
		importance  widget.Importance
		isActive    bool
		s           string
		name        string
	)
	if qi == nil {
		name = w.Placeholder
	} else {
		isActive = qi.IsActive()
		completionP = qi.CompletionP()
		isCompleted := qi.IsCompleted()
		if isCompleted {
			importance = widget.LowImportance
			s = "Completed"
		} else if isActive {
			importance = widget.MediumImportance
			s = ihumanize.Optional(qi.Remaining(), "?")
		} else {
			importance = widget.MediumImportance
			s = ihumanize.Optional(qi.Duration(), "?")
		}
		if w.isMobile {
			name = qi.StringShortened()
		} else {
			name = qi.String()
		}
	}
	w.name.Importance = importance
	w.name.Text = name
	w.name.Refresh()
	w.duration.Text = s
	w.duration.Importance = importance
	w.duration.Refresh()
	if isActive {
		w.progress.SetValue(completionP)
		w.progress.Show()
	} else {
		w.progress.Hide()
	}
}

func (w *skillQueueItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(
		w.progress,
		container.NewBorder(nil, nil, nil, w.duration, w.name),
	)
	return widget.NewSimpleRenderer(c)
}
