package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type characterSkillQueue struct {
	widget.BaseWidget

	OnUpdate func(statusShort, statusLong string)

	character          *app.Character
	emptyInfo          *widget.Label
	isCharacterUpdated bool
	list               *widget.List
	signalKey          string
	sq                 *app.CharacterSkillqueue
	top                *widget.Label
	u                  *baseUI
}

// newCharacterSkillQueue returns a new characterSkillQueue object with dynamic character.
func newCharacterSkillQueue(u *baseUI) *characterSkillQueue {
	return newCharacterSkillQueueWithCharacter(u, nil)
}

// newCharacterSkillQueue returns a new characterSkillQueue object with static character.
func newCharacterSkillQueueWithCharacter(u *baseUI, c *app.Character) *characterSkillQueue {
	emptyInfo := widget.NewLabel("Queue is empty")
	emptyInfo.Importance = widget.LowImportance
	emptyInfo.Hide()
	a := &characterSkillQueue{
		character:          c,
		emptyInfo:          emptyInfo,
		isCharacterUpdated: c == nil,
		signalKey:          generateUniqueID(),
		sq:                 app.NewCharacterSkillqueue(),
		top:                makeTopLabel(),
		u:                  u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeSkillQueue()
	return a
}

func (a *characterSkillQueue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.top,
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
			return a.sq.Size()
		},
		func() fyne.CanvasObject {
			level := newSkillLevel()
			if !a.u.isDesktop {
				level.Hide()
			}
			return container.NewBorder(nil, nil, level, nil, newSkillQueueItem())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			qi := a.sq.Item(id)
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
		q := a.sq.Item(id)
		if q == nil {
			list.UnselectAll()
			return
		}
		showSkillInTrainingWindow(a.u, q)
	}
	return list
}

func (a *characterSkillQueue) start() {
	if a.isCharacterUpdated {
		a.u.currentCharacterExchanged.AddListener(
			func(_ context.Context, c *app.Character) {
				a.character = c
			},
			a.signalKey,
		)
	}
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character) != arg.characterID {
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
	if a.isCharacterUpdated {
		a.u.currentCharacterExchanged.RemoveListener(a.signalKey)
	}
	a.u.characterSectionChanged.RemoveListener(a.signalKey)
	a.u.refreshTickerExpired.RemoveListener(a.signalKey)
}

func (a *characterSkillQueue) update() {
	var t string
	var i widget.Importance
	err := a.sq.Update(context.Background(), a.u.cs, characterIDOrZero(a.character))
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		var s1, s2 string
		isActive := a.sq.IsActive()
		if !isActive {
			s1 = "!"
			s2 = "training paused"
		} else if c := a.sq.CompletionP(); c.ValueOrZero() < 1 {
			s1 = fmt.Sprintf("%.0f%%", c.ValueOrZero()*100)
			s2 = fmt.Sprintf("%s (%s)", a.sq.Active(), s1)
		}
		if a.OnUpdate != nil {
			a.OnUpdate(s1, s2)
		}
		var total optional.Optional[time.Duration]
		if isActive {
			total = a.sq.RemainingTime()
		}
		t, i = a.makeTopText(total)
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		if a.sq.Size() == 0 {
			a.emptyInfo.Show()
		} else {
			a.emptyInfo.Hide()
		}
		a.list.Refresh()
	})
}

func (a *characterSkillQueue) makeTopText(total optional.Optional[time.Duration]) (string, widget.Importance) {
	hasData := a.u.scs.HasCharacterSection(characterIDOrZero(a.character), app.SectionCharacterSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	if !a.sq.IsActive() {
		return "Training not active", widget.WarningImportance
	}
	t := fmt.Sprintf("Total training time: %s", ihumanize.Optional(total, "?"))
	return t, widget.MediumImportance
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
		isActive.Importance = widget.DangerImportance
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
	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("%s by %s", app.SkillDisplayName(r.SkillName, r.FinishedLevel), characterName)
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindow(r.SkillID)
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

func newSkillQueueItem() *skillQueueItem {
	pb := widget.NewProgressBar()
	w := &skillQueueItem{
		Placeholder: "N/A",
		duration:    widget.NewLabel(""),
		progress:    pb,
		isMobile:    fyne.CurrentDevice().IsMobile(),
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
