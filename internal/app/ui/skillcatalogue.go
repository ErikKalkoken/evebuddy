package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
	"github.com/dustin/go-humanize"
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

// skillCatalogueArea is the UI area that shows the skill catalogue
type skillCatalogueArea struct {
	content        *fyne.Container
	groupsGrid     *widget.GridWrap
	groups         binding.UntypedList
	skillsGrid     *widget.GridWrap
	skills         binding.UntypedList
	total          *widget.Label
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	ui             *ui
}

func (u *ui) newSkillCatalogueArea() *skillCatalogueArea {
	a := &skillCatalogueArea{
		groups:         binding.NewUntypedList(),
		skills:         binding.NewUntypedList(),
		total:          widget.NewLabel(""),
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: theme.NewDisabledResource(theme.MediaStopIcon()),
		ui:             u,
	}
	a.total.TextStyle.Bold = true
	a.groupsGrid = a.makeSkillGroups()
	a.skillsGrid = a.makeSkillsGrid()
	s := container.NewVSplit(a.groupsGrid, a.skillsGrid)
	a.content = container.NewBorder(a.total, nil, nil, nil, s)
	return a
}

func (a *skillCatalogueArea) makeSkillGroups() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return a.groups.Length()
		},
		func() fyne.CanvasObject {
			pb := widget.NewProgressBar()
			pb.TextFormatter = func() string {
				return ""
			}
			row := container.NewPadded(container.NewStack(
				pb,
				container.NewHBox(
					widget.NewLabel("Corporation Management"), layout.NewSpacer(), widget.NewLabel("99"),
				)))
			return row
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			c := row.Objects[0].(*fyne.Container).Objects[1].(*fyne.Container)
			name := c.Objects[0].(*widget.Label)
			total := c.Objects[2].(*widget.Label)
			group, err := getItemUntypedList[skillGroupProgress](a.groups, id)
			if err != nil {
				slog.Error("Failed to render group item in skill catalogue UI", "err", err)
				name.SetText("ERROR")
				return
			}
			pb := row.Objects[0].(*fyne.Container).Objects[0].(*widget.ProgressBar)
			pb.SetValue(group.completionP())
			name.SetText(group.name)
			total.SetText(humanize.Comma(int64(group.total)))
		},
	)
	g.OnSelected = func(id widget.ListItemID) {
		group, err := getItemUntypedList[skillGroupProgress](a.groups, id)
		if err != nil {
			slog.Error("Failed to select skill group", "err", err)
			g.UnselectAll()
			return
		}
		if !a.ui.hasCharacter() {
			g.UnselectAll()
			return
		}
		oo, err := a.ui.sv.Character.ListCharacterSkillProgress(
			context.Background(), a.ui.characterID(), group.id)
		if err != nil {
			slog.Error("Failed to fetch skill group data", "err", err)
			g.UnselectAll()
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
		a.skills.Set(copyToUntypedSlice(skills))
		a.skillsGrid.Refresh()
	}
	return g
}

func (a *skillCatalogueArea) makeSkillsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return a.skills.Length()
		},
		func() fyne.CanvasObject {
			c := container.NewHBox(
				widgets.NewSkillLevel(),
				widget.NewLabel("Capital Shipboard Compression Technology"))
			return c
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			level := row.Objects[0].(*widgets.SkillLevel)
			label := row.Objects[1].(*widget.Label)
			skill, err := getItemUntypedList[skillTrained](a.skills, id)
			if err != nil {
				slog.Error("Failed to render skill item in skill catalogue UI", "err", err)
				label.SetText("ERROR")
				return
			}
			label.SetText(skill.name)
			level.Set(skill.activeLevel, skill.trainedLevel, 0)
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		o, err := getItemUntypedList[skillTrained](a.skills, id)
		if err != nil {
			slog.Error("Failed to access skill item", "err", err)
			return
		}
		a.ui.showTypeInfoWindow(o.id, a.ui.characterID())
	}
	return g
}

func (a *skillCatalogueArea) redraw() {
	a.groupsGrid.UnselectAll()
	x := make([]skillTrained, 0)
	a.skills.Set(copyToUntypedSlice(x))
	a.refresh()
}

func (a *skillCatalogueArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		exists := a.ui.sv.StatusCache.GeneralSectionExists(app.SectionEveCategories)
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

func (a *skillCatalogueArea) makeTopText() (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No Character", widget.LowImportance, nil
	}
	c := a.ui.currentCharacter()
	total := humanizedNullInt64(c.TotalSP, "?")
	unallocated := humanizedNullInt64(c.UnallocatedSP, "?")
	t := fmt.Sprintf("%s Total Skill Points (%s Unallocated)", total, unallocated)
	return t, widget.MediumImportance, nil
}

func (a *skillCatalogueArea) updateGroups() error {
	if !a.ui.hasCharacter() {
		return nil
	}
	gg, err := a.ui.sv.Character.ListCharacterSkillGroupsProgress(context.Background(), a.ui.characterID())
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
	if err := a.groups.Set(copyToUntypedSlice(groups)); err != nil {
		return err
	}
	return nil
}
