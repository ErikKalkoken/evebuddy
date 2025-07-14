package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/dustin/go-humanize"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type skillGroupProgress struct {
	id      int32
	name    string
	trained float64
	total   float64
}

func (g skillGroupProgress) completionP() float64 {
	return g.trained / g.total
}

type skillTrained struct {
	activeLevel  int
	description  string
	groupName    string
	id           int32
	name         string
	trainedLevel int
}

type characterSkillCatalogue struct {
	widget.BaseWidget

	groups         []skillGroupProgress
	groupsGrid     fyne.CanvasObject
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	skills         []skillTrained
	skillsGrid     fyne.CanvasObject
	total          *widget.Label
	u              *baseUI
}

func newCharacterSkillCatalogue(u *baseUI) *characterSkillCatalogue {
	a := &characterSkillCatalogue{
		groups:         make([]skillGroupProgress, 0),
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: theme.NewDisabledResource(theme.MediaStopIcon()),
		skills:         make([]skillTrained, 0),
		total:          makeTopLabel(),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.groupsGrid = a.makeGroupsGrid()
	a.skillsGrid = a.makeSkillsGrid()
	return a
}

func (a *characterSkillCatalogue) CreateRenderer() fyne.WidgetRenderer {
	s := container.NewVSplit(a.groupsGrid, a.skillsGrid)
	c := container.NewBorder(a.total, nil, nil, nil, s)
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
			if !a.u.hasCharacter() {
				unselectAll()
				return
			}
			oo, err := a.u.cs.ListSkillProgress(
				context.TODO(), a.u.currentCharacterID(), group.id,
			)
			if err != nil {
				slog.Error("Failed to fetch skill group data", "err", err)
				unselectAll()
				return
			}
			skills := make([]skillTrained, len(oo))
			for i, o := range oo {
				skills[i] = skillTrained{
					activeLevel:  o.ActiveSkillLevel,
					description:  o.TypeDescription,
					groupName:    group.name,
					id:           o.TypeID,
					name:         o.TypeName,
					trainedLevel: o.TrainedSkillLevel,
				}
			}
			a.skills = skills
			a.skillsGrid.Refresh()
		}
	}
	return makeGridOrList(!a.u.isDesktop, length, makeCreateItem, updateItem, makeOnSelected)
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
			a.u.ShowTypeInfoWindow(skill.id)
		}
	}
	return makeGridOrList(!a.u.isDesktop, length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *characterSkillCatalogue) update() {
	var err error
	groups := make([]skillGroupProgress, 0)
	characterID := a.u.currentCharacterID()
	hasData := a.u.scs.HasGeneralSection(app.SectionEveTypes) && a.u.scs.HasCharacterSection(characterID, app.SectionCharacterSkills)
	if hasData {
		groups2, err2 := a.updateGroups(characterID, a.u.services())
		if err2 != nil {
			slog.Error("Failed to refresh skill catalogue UI", "err", err)
			err = err2
		} else {
			groups = groups2
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		character := a.u.currentCharacter()
		total := ihumanize.Optional(character.TotalSP, "?")
		unallocated := ihumanize.Optional(character.UnallocatedSP, "?")
		return fmt.Sprintf("%s Total Skill Points (%s Unallocated)", total, unallocated), widget.MediumImportance
	})
	fyne.Do(func() {
		a.total.Text = t
		a.total.Importance = i
		a.total.Refresh()
	})
	fyne.Do(func() {
		a.skills = make([]skillTrained, 0)
		a.groups = groups
		a.groupsGrid.Refresh()
		switch x := a.groupsGrid.(type) {
		case *widget.GridWrap:
			x.UnselectAll()
		case *widget.List:
			x.UnselectAll()
		}
	})
}

func (*characterSkillCatalogue) updateGroups(characterID int32, s services) ([]skillGroupProgress, error) {
	gg, err := s.cs.ListSkillGroupsProgress(context.TODO(), characterID)
	if err != nil {
		return nil, err
	}
	groups := make([]skillGroupProgress, len(gg))
	for i, g := range gg {
		groups[i] = skillGroupProgress{
			trained: g.Trained,
			id:      g.GroupID,
			name:    g.GroupName,
			total:   g.Total,
		}
	}
	return groups, nil
}
