package skillui

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type Queue struct {
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
	u                    ui
}

// NewQueue returns a new characterSkillQueue for the current character.
func NewQueue(u ui) *Queue {
	return NewQueueWithCharacter(u, nil)
}

// NewQueueWithCharacter returns a new characterSkillQueue for character c.
// This type of skillqueue is meant to be temporary.
func NewQueueWithCharacter(u ui, c *app.Character) *Queue {
	emptyInfo := widget.NewLabel("Queue is empty")
	emptyInfo.Importance = widget.LowImportance
	emptyInfo.Hide()
	statusResources := theme.MediaRecordIcon()
	a := &Queue{
		emptyInfo:            emptyInfo,
		showCurrentCharacter: c == nil,
		signalKey:            u.Signals().UniqueKey(),
		skillqueue:           app.NewCharacterSkillqueue(),
		statusResource:       statusResources,
		status:               ttwidget.NewIcon(theme.NewDisabledResource(statusResources)),
		top:                  awidget.NewLabelWithWrapping(""),
		u:                    u,
	}
	a.ExtendBaseWidget(a)
	a.character.Store(c)
	a.list = a.makeSkillQueue()

	// Signals
	if a.showCurrentCharacter {
		a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			a.Update(ctx)
		}, a.signalKey)
	}
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		switch arg.Section {
		case app.SectionCharacterSkillqueue:
			a.Update(ctx)
		}
	}, a.signalKey)
	a.u.Signals().CharacterChanged.AddListener(func(ctx context.Context, characterID int64) {
		if a.character.Load().IDOrZero() != characterID {
			return
		}
		c, err := a.u.Character().GetCharacter(ctx, characterID)
		if err != nil {
			slog.Error("characterSkillQueue: update character", "error", err)
			return
		}
		a.character.Store(c)
		a.Update(ctx)
	}, a.signalKey)
	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		fyne.Do(func() {
			a.Update(ctx)
		})
	}, a.signalKey)
	return a
}

func (a *Queue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewBorder(nil, nil, a.status, nil, a.top),
		nil,
		nil,
		nil,
		container.NewStack(a.emptyInfo, a.list),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Queue) makeSkillQueue() *widget.List {
	list := widget.NewList(
		func() int {
			return a.skillqueue.Size()
		},
		func() fyne.CanvasObject {
			level := awidget.NewSkillLevel()
			if a.u.IsMobile() {
				level.Hide()
			}
			return container.NewBorder(nil, nil, level, nil, NewSkillQueueItem(a.u.IsMobile()))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			qi := a.skillqueue.Item(id)
			if qi == nil {
				return
			}
			c := co.(*fyne.Container).Objects
			c[0].(*SkillQueueItem).Set(qi)

			level := c[1].(*awidget.SkillLevel)
			var active, trained, queued int64
			if qi.IsCompleted() {
				active = qi.FinishedLevel
				trained = qi.FinishedLevel
				queued = qi.FinishedLevel
			} else if qi.IsActive() {
				active = qi.FinishedLevel - 1
				trained = qi.FinishedLevel - 1
				queued = 0
			} else {
				active = qi.FinishedLevel - 1
				trained = qi.FinishedLevel - 1
				queued = qi.FinishedLevel
			}
			level.Set(active, trained, queued)
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

// Stop frees resources and removes event listeners.
func (a *Queue) Stop() {
	if a.showCurrentCharacter {
		a.u.Signals().CurrentCharacterExchanged.RemoveListener(a.signalKey)
	}
	a.u.Signals().CharacterSectionChanged.RemoveListener(a.signalKey)
	a.u.Signals().RefreshTickerExpired.RemoveListener(a.signalKey)
}

func (a *Queue) Update(ctx context.Context) {
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
		})
	}
	reset := func() {
		fyne.Do(func() {
			a.list.Hide()
			a.emptyInfo.Hide()
			a.status.Hide()
		})
	}

	c := a.character.Load()
	if c == nil {
		setTop("No character", widget.LowImportance)
		reset()
		return
	}
	hasData := a.u.StatusCache().HasCharacterSection(c.ID, app.SectionCharacterSkillqueue)
	if !hasData {
		setTop("Waiting for character data to be loaded...", widget.WarningImportance)
		reset()
		return
	}
	err := a.skillqueue.Update(ctx, a.u.Character(), c.ID)
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		reset()
		return
	}

	isActive := a.skillqueue.IsActive()
	if isActive {
		s := fmt.Sprintf("Total training time: %s", ihumanize.Optional(a.skillqueue.RemainingTime(), "?"))
		setTop(s, widget.MediumImportance)
	} else if c.IsTrainingWatched {
		setTop("No skill in training", widget.DangerImportance)
	} else {
		setTop("Training not active", widget.MediumImportance)
	}

	fyne.Do(func() {
		var r fyne.Resource
		var s string
		if isActive {
			r = theme.NewSuccessThemedResource(a.statusResource)
			s = "Training is active"
		} else if c.IsTrainingWatched {
			r = theme.NewErrorThemedResource(a.statusResource)
			s = "Training expired"
		} else {
			r = theme.NewDisabledResource(a.statusResource)
			s = "Training is not active"
		}
		a.status.SetResource(r)
		a.status.SetToolTip(s)
		a.status.Show()
		var s1, s2 string
		if isActive {
			if c := a.skillqueue.CompletionP(); c.ValueOrZero() < 1 {
				s1 = fmt.Sprintf("%.0f%%", c.ValueOrZero()*100)
				s2 = fmt.Sprintf("%s (%s)", a.skillqueue.Active(), s1)
			}
		} else if c.IsTrainingWatched {
			s1 = "!"
			s2 = "No skill in training"
		}
		if a.OnUpdate != nil {
			a.OnUpdate(s1, s2)
		}
	})

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

func showSkillInTrainingWindow(u ui, r *app.CharacterSkillqueueItem) {
	characterName := u.StatusCache().CharacterName(r.CharacterID)
	w, created := u.GetOrCreateWindow(
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
			makeCharacterActionLabel(r.CharacterID, characterName, u.InfoWindow().ShowEntity),
		),
		widget.NewFormItem("Skill", makeLinkLabel(app.SkillDisplayName(r.SkillName, r.FinishedLevel), func() {
			u.InfoWindow().ShowTypeWithCharacter(r.SkillID, r.CharacterID)
		})),
		widget.NewFormItem("Group", widget.NewLabel(r.GroupName)),
		widget.NewFormItem("Description", description),
		widget.NewFormItem("Active?", isActive),
		widget.NewFormItem("Completed", widget.NewLabel(fmt.Sprintf("%.0f%%", r.CompletionP()*100))),
		widget.NewFormItem("Remaining", widget.NewLabel(r.Remaining().StringFunc("?", func(v time.Duration) string {
			return ihumanize.DurationRoundedUp(v)
		}))),
		widget.NewFormItem("Duration", widget.NewLabel(r.Duration().StringFunc("?", func(v time.Duration) string {
			return ihumanize.DurationRoundedUp(v)
		}))),
		widget.NewFormItem("Start date", widget.NewLabel(r.StartDate.StringFunc("?", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("End date", widget.NewLabel(r.FinishDate.StringFunc("?", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))),
		widget.NewFormItem("SP at start", widget.NewLabel(humanize.Comma(r.TrainingStartSP.ValueOrZero()-r.LevelStartSP.ValueOrZero()))),
		widget.NewFormItem("Total SP", widget.NewLabel(humanize.Comma(r.LevelEndSP.ValueOrZero()-r.LevelStartSP.ValueOrZero()))),
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
	xwindow.Set(xwindow.Params{
		Content: f,
		ImageAction: func() {
			u.InfoWindow().ShowTypeWithCharacter(r.SkillID, r.CharacterID)
		},
		ImageLoader: func(setter func(r fyne.Resource)) {
			u.EVEImage().InventoryTypeIconAsync(r.SkillID, 256, setter)
		},
		MinSize: fyne.NewSize(500, 450),
		Title:   subTitle,
		Window:  w,
	})
	w.Show()
}

type SkillQueueItem struct {
	widget.BaseWidget

	Placeholder string

	duration *widget.Label
	isMobile bool
	name     *ttwidget.Label
	progress *widget.ProgressBar
}

func NewSkillQueueItem(isMobile bool) *SkillQueueItem {
	pb := widget.NewProgressBar()
	w := &SkillQueueItem{
		Placeholder: "N/A",
		duration:    widget.NewLabel(""),
		progress:    pb,
		isMobile:    isMobile,
	}
	w.ExtendBaseWidget(w)
	w.name = ttwidget.NewLabel(w.Placeholder)
	w.name.Truncation = fyne.TextTruncateEllipsis
	pb.Hide()
	if w.isMobile {
		pb.TextFormatter = func() string {
			return ""
		}
	}
	return w
}

func (w *SkillQueueItem) Set(qi *app.CharacterSkillqueueItem) {
	var (
		completionP float64
		importance  widget.Importance
		isActive    bool
		duration    string
		name        string
		description string
	)
	if qi == nil {
		name = w.Placeholder
	} else {
		isActive = qi.IsActive()
		completionP = qi.CompletionP()
		isCompleted := qi.IsCompleted()
		if isCompleted {
			importance = widget.LowImportance
			duration = "Completed"
		} else if isActive {
			importance = widget.MediumImportance
			duration = ihumanize.Optional(qi.Remaining(), "?")
		} else {
			importance = widget.MediumImportance
			duration = ihumanize.Optional(qi.Duration(), "?")
		}
		if w.isMobile {
			name = qi.StringShortened()
		} else {
			name = qi.String()
		}
		description = qi.SkillDescription
	}
	w.name.Importance = importance
	w.name.Text = name
	w.name.Refresh()
	w.name.SetToolTip(description)
	w.duration.Text = duration
	w.duration.Importance = importance
	w.duration.Refresh()
	if isActive {
		w.progress.SetValue(completionP)
		w.progress.Show()
	} else {
		w.progress.Hide()
	}
}

func (w *SkillQueueItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(
		w.progress,
		container.NewBorder(nil, nil, nil, w.duration, w.name),
	)
	return widget.NewSimpleRenderer(c)
}
