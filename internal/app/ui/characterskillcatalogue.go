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

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
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

type CharacterSkillCatalogue struct {
	widget.BaseWidget

	groups         []skillGroupProgress
	groupsGrid     fyne.CanvasObject
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	skills         []skillTrained
	skillsGrid     fyne.CanvasObject
	total          *widget.Label
	u              *BaseUI
}

func NewCharacterSkillCatalogue(u *BaseUI) *CharacterSkillCatalogue {
	a := &CharacterSkillCatalogue{
		groups:         make([]skillGroupProgress, 0),
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: theme.NewDisabledResource(theme.MediaStopIcon()),
		skills:         make([]skillTrained, 0),
		total:          appwidget.MakeTopLabel(),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.groupsGrid = a.makeGroupsGrid()
	a.skillsGrid = a.makeSkillsGrid()
	return a
}

func (a *CharacterSkillCatalogue) CreateRenderer() fyne.WidgetRenderer {
	s := container.NewVSplit(a.groupsGrid, a.skillsGrid)
	c := container.NewBorder(a.total, nil, nil, nil, s)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterSkillCatalogue) makeGroupsGrid() fyne.CanvasObject {
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
			if !a.u.HasCharacter() {
				unselectAll()
				return
			}
			oo, err := a.u.CharacterService().ListSkillProgress(
				context.TODO(), a.u.CurrentCharacterID(), group.id,
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
	return makeGridOrList(a.u.IsMobile(), length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *CharacterSkillCatalogue) makeSkillsGrid() fyne.CanvasObject {
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
				appwidget.NewSkillLevel(),
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
		level := row[1].(*appwidget.SkillLevel)
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
	return makeGridOrList(a.u.IsMobile(), length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *CharacterSkillCatalogue) Update() {
	switch x := a.groupsGrid.(type) {
	case *widget.GridWrap:
		x.UnselectAll()
	case *widget.List:
		x.UnselectAll()
	}
	a.skills = make([]skillTrained, 0)
	a.Refresh()
}

func (a *CharacterSkillCatalogue) Refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		exists := a.u.StatusCacheService().GeneralSectionExists(app.SectionEveCategories)
		if !exists {
			return "Waiting for universe data to be loaded...", widget.WarningImportance, nil
		}
		if err := a.updateGroups(); err != nil {
			return "", 0, err
		}
		return a.makeTopText()
	}()
	if err != nil {
		slog.Error("Failed to refresh skill catalogue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
}

func (a *CharacterSkillCatalogue) makeTopText() (string, widget.Importance, error) {
	if !a.u.HasCharacter() {
		return "No Character", widget.LowImportance, nil
	}
	c := a.u.CurrentCharacter()
	total := ihumanize.Optional(c.TotalSP, "?")
	unallocated := ihumanize.Optional(c.UnallocatedSP, "?")
	t := fmt.Sprintf("%s Total Skill Points (%s Unallocated)", total, unallocated)
	return t, widget.MediumImportance, nil
}

func (a *CharacterSkillCatalogue) updateGroups() error {
	if !a.u.HasCharacter() {
		return nil
	}
	gg, err := a.u.CharacterService().ListSkillGroupsProgress(context.TODO(), a.u.CurrentCharacterID())
	if err != nil {
		return err
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
	a.groups = groups
	a.groupsGrid.Refresh()
	return nil
}
