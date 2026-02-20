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

	"github.com/ErikKalkoken/evebuddy/internal/app"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type skillGroupProgress struct {
	id      int64
	name    string
	trained float64
	total   float64
}

func (g skillGroupProgress) completionP() float64 {
	return g.trained / g.total
}

type skillTrained struct {
	activeLevel  int64
	description  string
	groupID      int64
	groupName    string
	id           int64
	name         string
	trainedLevel int64
}

type characterSkillCatalogue struct {
	widget.BaseWidget

	character      atomic.Pointer[app.Character]
	groups         []skillGroupProgress
	groupsGrid     fyne.CanvasObject
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	skills         []skillTrained
	skillsGrid     fyne.CanvasObject
	top            *widget.Label
	u              *baseUI
}

func newCharacterSkillCatalogue(u *baseUI) *characterSkillCatalogue {
	a := &characterSkillCatalogue{
		groups:         make([]skillGroupProgress, 0),
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: theme.NewDisabledResource(theme.MediaStopIcon()),
		skills:         make([]skillTrained, 0),
		top:            newLabelWithWrapping(),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.groupsGrid = a.makeGroupsGrid()
	a.skillsGrid = a.makeSkillsGrid()

	// signals
	a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	})
	a.u.generalSectionChanged.AddListener(func(ctx context.Context, arg generalSectionUpdated) {
		characterID := characterIDOrZero(a.character.Load())
		if characterID == 0 {
			return
		}
		if arg.section == app.SectionEveTypes {
			a.update(ctx)
		}
	})
	return a
}

func (a *characterSkillCatalogue) CreateRenderer() fyne.WidgetRenderer {
	s := container.NewVSplit(a.groupsGrid, a.skillsGrid)
	c := container.NewBorder(a.top, nil, nil, nil, s)
	return widget.NewSimpleRenderer(c)
}

func (a *characterSkillCatalogue) makeGroupsGrid() fyne.CanvasObject {
	length := func() int {
		return len(a.groups)
	}
	makeCreateItem := func(trunc fyne.TextTruncation) func() fyne.CanvasObject {
		return func() fyne.CanvasObject {
			pb := widget.NewProgressBar()
			pb.TextFormatter = func() string {
				return ""
			}
			title := widget.NewLabel("Corporation Management")
			title.Truncation = trunc
			row := container.NewPadded(container.NewStack(
				pb,
				container.NewBorder(
					nil,
					nil,
					nil,
					widget.NewLabel("99"),
					title,
				)))
			return row
		}
	}
	updateItem := func(id int, co fyne.CanvasObject) {
		if id >= len(a.groups) {
			return
		}
		group := a.groups[id]
		row := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects
		c := row[1].(*fyne.Container).Objects
		name := c[0].(*widget.Label)
		total := c[1].(*widget.Label)
		pb := row[0].(*widget.ProgressBar)
		pb.SetValue(group.completionP())
		name.SetText(group.name)
		total.SetText(humanize.Comma(int64(group.total)))
	}
	makeOnSelected := func(unselectAll func()) func(int) {
		return func(id int) {
			if id >= len(a.groups) {
				unselectAll()
				return
			}
			group := a.groups[id]
			characterID := characterIDOrZero(a.character.Load())
			if characterID == 0 {
				unselectAll()
				return
			}
			ctx := context.Background()
			go func() {
				oo, err := a.u.cs.ListSkillProgress(ctx, characterID, group.id)
				if err != nil {
					slog.Error("Failed to fetch skill group data", "err", err)
					fyne.Do(func() {
						unselectAll()
					})
					return
				}

				var skills []skillTrained
				for _, o := range oo {
					skills = append(skills, skillTrained{
						activeLevel:  o.ActiveSkillLevel,
						description:  o.TypeDescription,
						groupID:      group.id,
						groupName:    group.name,
						id:           o.TypeID,
						name:         o.TypeName,
						trainedLevel: o.TrainedSkillLevel,
					})
				}

				fyne.Do(func() {
					a.skills = skills
					a.skillsGrid.Refresh()
				})
			}()
		}
	}
	return makeGridOrList(a.u.isMobile, length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *characterSkillCatalogue) makeSkillsGrid() fyne.CanvasObject {
	length := func() int {
		return len(a.skills)
	}
	makeCreateItem := func(trunc fyne.TextTruncation) func() fyne.CanvasObject {
		return func() fyne.CanvasObject {
			title := widget.NewLabel("Capital Shipboard Compression Technology")
			title.Truncation = trunc
			c := container.NewBorder(
				nil,
				nil,
				newSkillLevel(),
				nil,
				title,
			)
			return c
		}
	}
	updateItem := func(id int, co fyne.CanvasObject) {
		if id >= len(a.skills) {
			return
		}
		skill := a.skills[id]
		row := co.(*fyne.Container).Objects
		label := row[0].(*widget.Label)
		label.SetText(skill.name)
		level := row[1].(*skillLevel)
		level.Set(skill.activeLevel, skill.trainedLevel, 0)
	}
	makeOnSelected := func(unselectAll func()) func(int) {
		unselectAll()
		return func(id int) {
			if id >= len(a.skills) {
				return
			}
			skill := a.skills[id]
			a.u.ShowTypeInfoWindowWithCharacter(skill.id, characterIDOrZero(a.character.Load()))
		}
	}
	return makeGridOrList(a.u.isMobile, length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *characterSkillCatalogue) update(ctx context.Context) {
	setTop := func(t string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = t
			a.top.Importance = i
			a.top.Refresh()
		})
	}
	unselectTasks := func() {
		fyne.Do(func() {
			switch x := a.groupsGrid.(type) {
			case *widget.GridWrap:
				x.UnselectAll()
			case *widget.List:
				x.UnselectAll()
			}
		})
	}

	clear := func() {
		fyne.Do(func() {
			a.skills = make([]skillTrained, 0)
			a.groups = make([]skillGroupProgress, 0)
			a.groupsGrid.Refresh()
		})
		unselectTasks()
	}
	c := a.character.Load()
	if c == nil {
		clear()
		setTop("No character", widget.LowImportance)
		return
	}
	characterID := characterIDOrZero(c)
	hasData := a.u.scs.HasGeneralSection(app.SectionEveTypes) && a.u.scs.HasCharacterSection(characterID, app.SectionCharacterSkills)
	if !hasData {
		clear()
		setTop("No data yet", widget.WarningImportance)
		return
	}
	groups, err := a.updateGroups(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh skill catalogue UI", "err", err)
		clear()
		setTop("No data yet", widget.WarningImportance)
		return
	}
	top := fmt.Sprintf("%s Total SP (%s Unallocated)",
		c.TotalSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}), c.UnallocatedSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}),
	)
	setTop(top, widget.MediumImportance)
	fyne.Do(func() {
		a.skills = make([]skillTrained, 0)
		a.groups = groups
		a.groupsGrid.Refresh()
	})
	unselectTasks()
}

func (a *characterSkillCatalogue) updateGroups(ctx context.Context, characterID int64) ([]skillGroupProgress, error) {
	gg, err := a.u.cs.ListSkillGroupsProgress(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var groups []skillGroupProgress
	for _, g := range gg {
		groups = append(groups, skillGroupProgress{
			trained: g.Trained,
			id:      g.GroupID,
			name:    g.GroupName,
			total:   g.Total,
		})
	}
	return groups, nil
}

// makeGridOrList makes and returns a GridWrap on desktop and a List on mobile.
//
// This allows the grid items to render nicely as list on mobile and also enable truncation.
func makeGridOrList(isMobile bool, length func() int, makeCreateItem func(trunc fyne.TextTruncation) func() fyne.CanvasObject, updateItem func(id int, co fyne.CanvasObject), makeOnSelected func(unselectAll func()) func(int)) fyne.CanvasObject {
	var w fyne.CanvasObject
	if isMobile {
		w = widget.NewList(length, makeCreateItem(fyne.TextTruncateEllipsis), updateItem)
		l := w.(*widget.List)
		l.OnSelected = makeOnSelected(func() {
			l.UnselectAll()
		})
	} else {
		w = widget.NewGridWrap(length, makeCreateItem(fyne.TextTruncateOff), updateItem)
		g := w.(*widget.GridWrap)
		g.OnSelected = makeOnSelected(func() {
			g.UnselectAll()
		})
	}
	return w
}
