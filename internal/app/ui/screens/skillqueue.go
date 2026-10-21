package screens

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// skillQueueState is what the screen shows: a message or a loaded queue.
type skillQueueState struct {
	character  *app.Character           // owner of queue
	queue      *app.CharacterSkillqueue // nil when showing a message
	message    string
	importance widget.Importance
	failed     bool // update failed, so the status is unknown
}

type SkillQueue struct {
	widget.BaseWidget

	// OnUpdate reports the training status; "?" when the update failed.
	OnUpdate func(statusShort, statusLong string)

	character            atomic.Pointer[app.Character]
	emptyLabel           *widget.Label
	showCurrentCharacter bool
	signalKey            string
	skillList            *widget.List
	state                skillQueueState // main thread only
	statusIcon           *ttwidget.Icon
	statusResource       fyne.Resource
	topLabel             *widget.Label
	u                    baseUI
	updateRun            xsync.LatestRun
}

// NewSkillQueue returns a new characterSkillQueue for the current character.
func NewSkillQueue(u baseUI) *SkillQueue {
	return NewQueueWithCharacter(u, nil)
}

// NewQueueWithCharacter returns a new characterSkillQueue for character c.
// This type of skillqueue is meant to be temporary.
func NewQueueWithCharacter(u baseUI, c *app.Character) *SkillQueue {
	statusResource := theme.MediaRecordIcon()
	a := &SkillQueue{
		emptyLabel:           widget.NewLabel("Queue is empty"),
		showCurrentCharacter: c == nil,
		signalKey:            u.Signals().UniqueKey(),
		statusIcon:           ttwidget.NewIcon(theme.NewDisabledResource(statusResource)),
		statusResource:       statusResource,
		topLabel:             ui.NewLabelWithWrapping(""),
		u:                    u,
	}
	a.ExtendBaseWidget(a)

	a.character.Store(c)
	a.emptyLabel.Importance = widget.LowImportance
	a.emptyLabel.Hide()
	a.skillList = a.makeSkillList()

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
		old := a.character.Load()
		if old.IDOrZero() != characterID {
			return
		}
		c, err := a.u.Character().GetCharacter(ctx, characterID)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to reload character for skill queue", "characterID", characterID, "error", err)
			return
		}
		if !swapReloadedCharacter(&a.character, old, c) {
			return // character changed meanwhile; that change runs its own update
		}
		a.Update(ctx)
	}, a.signalKey)
	a.u.Signals().RefreshTickerExpired.AddListener(func(_ context.Context, _ struct{}) {
		fyne.Do(func() {
			if a.state.queue == nil {
				return // messages don't change over time
			}
			a.applyState()
		})
	}, a.signalKey)
	return a
}

func (a *SkillQueue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewBorder(nil, nil, a.statusIcon, nil, a.topLabel),
		nil,
		nil,
		nil,
		container.NewStack(a.emptyLabel, a.skillList),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *SkillQueue) makeSkillList() *widget.List {
	list := widget.NewList(
		func() int {
			if a.state.queue == nil {
				return 0
			}
			return a.state.queue.Size()
		},
		func() fyne.CanvasObject {
			level := xwidget.NewSkillLevel()
			if a.u.IsMobile() {
				level.Hide()
			}
			return container.NewBorder(nil, nil, level, nil, NewSkillQueueItem(a.u.IsMobile()))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if a.state.queue == nil {
				return
			}
			qi := a.state.queue.Item(id)
			if qi == nil {
				return
			}
			c := co.(*fyne.Container).Objects
			c[0].(*SkillQueueItem).Set(qi)

			c[1].(*xwidget.SkillLevel).Set(skillQueueItemLevels(qi))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if a.state.queue == nil {
			return
		}
		q := a.state.queue.Item(id)
		if q == nil {
			return
		}
		showSkillInTrainingWindow(a.u, q, a.state.character.NameOrZero())
	}
	return list
}

// Stop frees resources and removes event listeners.
func (a *SkillQueue) Stop() {
	if a.showCurrentCharacter {
		a.u.Signals().CurrentCharacterExchanged.RemoveListener(a.signalKey)
	}
	a.u.Signals().CharacterSectionChanged.RemoveListener(a.signalKey)
	a.u.Signals().CharacterChanged.RemoveListener(a.signalKey)
	a.u.Signals().RefreshTickerExpired.RemoveListener(a.signalKey)
}

func (a *SkillQueue) Update(ctx context.Context) {
	isLatest := a.updateRun.Start()
	setState := func(st skillQueueState) {
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.state = st
			a.applyState()
		})
	}
	showMessage := func(s string, i widget.Importance) {
		setState(skillQueueState{message: s, importance: i})
	}
	showError := func(err error) {
		setState(skillQueueState{
			message:    "ERROR: " + a.u.ErrorDisplay(err),
			importance: widget.DangerImportance,
			failed:     true,
		})
	}

	c := a.character.Load()
	if c == nil {
		showMessage("No character", widget.LowImportance)
		return
	}
	hasData, err := a.u.Character().HasSection(ctx, c.ID, app.SectionCharacterSkillqueue)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh skill queue UI", "err", err)
		showError(err)
		return
	}
	if !hasData {
		showMessage("Waiting for character data to be loaded...", widget.WarningImportance)
		return
	}
	sq := app.NewCharacterSkillqueue()
	err = sq.Update(ctx, a.u.Character(), c.ID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh skill queue UI", "err", err)
		showError(err)
		return
	}
	setState(skillQueueState{character: c, queue: sq})
}

// applyState updates all widgets from the current state.
// Time dependent values are recalculated, so refresh ticks can call it.
// Must be called on the main thread.
func (a *SkillQueue) applyState() {
	st := a.state
	if st.queue == nil {
		a.topLabel.Text = st.message
		a.topLabel.Importance = st.importance
		a.topLabel.Refresh()
		a.skillList.Hide()
		a.emptyLabel.Hide()
		a.statusIcon.Hide()
		if a.OnUpdate != nil {
			if st.failed {
				a.OnUpdate("?", "?")
			} else {
				a.OnUpdate("", "")
			}
		}
		return
	}

	sq := st.queue
	var icon fyne.Resource
	var tooltip, statusShort, statusLong string
	switch {
	case sq.IsActive():
		a.topLabel.Text = fmt.Sprintf("Total training time: %s", ihumanize.Optional(sq.RemainingTime(), "?"))
		a.topLabel.Importance = widget.MediumImportance
		icon = theme.NewSuccessThemedResource(a.statusResource)
		tooltip = "Training is active"
		if p := sq.CompletionP(); p.ValueOrZero() < 1 {
			statusShort = fmt.Sprintf("%.0f%%", p.ValueOrZero()*100)
			statusLong = fmt.Sprintf("%s (%s)", sq.Active(), statusShort)
		}
	case st.character.IsTrainingWatched:
		a.topLabel.Text = "No skill in training"
		a.topLabel.Importance = widget.DangerImportance
		icon = theme.NewErrorThemedResource(a.statusResource)
		tooltip = "Training expired"
		statusShort = "!"
		statusLong = "No skill in training"
	default:
		a.topLabel.Text = "Training not active"
		a.topLabel.Importance = widget.MediumImportance
		icon = theme.NewDisabledResource(a.statusResource)
		tooltip = "Training is not active"
	}
	a.topLabel.Refresh()
	a.statusIcon.SetResource(icon)
	a.statusIcon.SetToolTip(tooltip)
	a.statusIcon.Show()
	if a.OnUpdate != nil {
		a.OnUpdate(statusShort, statusLong)
	}
	if sq.Size() == 0 {
		a.emptyLabel.Show()
	} else {
		a.emptyLabel.Hide()
	}
	a.skillList.Refresh()
	a.skillList.Show()
}

// skillQueueItemLevels returns the levels to show for a skill in the queue.
func skillQueueItemLevels(qi *app.CharacterSkillqueueItem) (active, trained, queued int64) {
	if qi.IsCompleted() {
		return qi.FinishedLevel, qi.FinishedLevel, qi.FinishedLevel
	}
	return qi.FinishedLevel - 1, qi.FinishedLevel - 1, qi.FinishedLevel
}

func showSkillInTrainingWindow(u baseUI, r *app.CharacterSkillqueueItem, characterName string) {
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
			ui.MakeCharacterActionLabel(r.CharacterID, characterName, u.InfoViewer().Show),
		),
		widget.NewFormItem("Skill", ui.MakeLinkLabel(ui.SkillDisplayName(r.SkillName, r.FinishedLevel), func() {
			u.InfoViewer().ShowType(r.SkillID, r.CharacterID)
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
	subTitle := fmt.Sprintf("%s by %s", ui.SkillDisplayName(r.SkillName, r.FinishedLevel), characterName)
	ui.MakeDetailWindow(ui.MakeDetailWindowParams{
		Content: f,
		ImageAction: func() {
			u.InfoViewer().ShowType(r.SkillID, r.CharacterID)
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
