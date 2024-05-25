package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

const skillsUpdateTicker = 10 * time.Second

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
	content    *fyne.Container
	groupsGrid *widget.GridWrap
	groups     binding.UntypedList
	// list           *widget.List
	skillsGrid     *widget.GridWrap
	skills         binding.UntypedList
	total          *widget.Label
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	ui             *ui
}

func (u *ui) NewSkillCatalogueArea() *skillCatalogueArea {
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

	// skill groups
	a.groupsGrid = widget.NewGridWrap(
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
			group, err := getFromBoundUntypedList[skillGroupProgress](a.groups, id)
			if err != nil {
				panic(err)
			}
			row := co.(*fyne.Container)
			pb := row.Objects[0].(*fyne.Container).Objects[0].(*widget.ProgressBar)
			pb.SetValue(group.completionP())
			c := row.Objects[0].(*fyne.Container).Objects[1].(*fyne.Container)
			name := c.Objects[0].(*widget.Label)
			name.SetText(group.name)
			total := c.Objects[2].(*widget.Label)
			total.SetText(humanize.Comma(int64(group.total)))
		},
	)
	a.groupsGrid.OnSelected = func(id widget.ListItemID) {
		group, err := getFromBoundUntypedList[skillGroupProgress](a.groups, id)
		if err != nil {
			panic(err)
		}
		c := a.ui.CurrentChar()
		if c == nil {
			return
		}
		oo, err := a.ui.service.ListCharacterSkillProgress(c.ID, group.id)
		if err != nil {
			panic(err)
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

	// details
	a.skillsGrid = widget.NewGridWrap(
		func() int {
			return a.skills.Length()
		},
		func() fyne.CanvasObject {
			x := container.NewHBox(
				widget.NewIcon(theme.MediaStopIcon()),
				widget.NewIcon(theme.MediaStopIcon()),
				widget.NewIcon(theme.MediaStopIcon()),
				widget.NewIcon(theme.MediaStopIcon()),
				widget.NewIcon(theme.MediaStopIcon()),
				widget.NewLabel("Capital Shipboard Compression Technology"))
			return x
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			skill, err := getFromBoundUntypedList[skillTrained](a.skills, id)
			if err != nil {
				panic(err)
			}
			row := co.(*fyne.Container)
			row.Objects[5].(*widget.Label).SetText(skill.name)
			for i := range 5 {
				y := row.Objects[i].(*widget.Icon)
				if skill.activeLevel > i {
					y.SetResource(a.levelTrained)
				} else if skill.trainedLevel > i {
					y.SetResource(a.levelBlocked)
				} else {
					y.SetResource(a.levelUnTrained)
				}
			}
		},
	)
	a.skillsGrid.OnSelected = func(id widget.GridWrapItemID) {
		o, err := getFromBoundUntypedList[skillTrained](a.skills, id)
		if err != nil {
			slog.Error("failed to access skill item", "err", err)
			return
		}
		var data = []struct {
			label string
			value string
			wrap  bool
		}{
			{"Name", o.name, false},
			{"Group", o.groupName, false},
			{"Description", o.description, true},
			{"Trained level", fmt.Sprintf("%d", o.trainedLevel), false},
			{"Active level", fmt.Sprintf("%d", o.activeLevel), false},
		}
		form := widget.NewForm()
		for _, row := range data {
			c := widget.NewLabel(row.value)
			if row.wrap {
				c.Wrapping = fyne.TextWrapWord
			}
			form.Append(row.label, c)
		}
		s := container.NewScroll(form)
		dlg := dialog.NewCustom("Skill Details", "OK", s, u.window)
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.ui.window.Canvas().Size().Width,
			Height: 0.8 * a.ui.window.Canvas().Size().Height,
		})
		a.skillsGrid.UnselectAll()
	}

	s := container.NewVSplit(a.groupsGrid, a.skillsGrid)
	a.content = container.NewBorder(a.total, nil, nil, nil, s)
	return a
}

func (a *skillCatalogueArea) Redraw() {
	a.groupsGrid.UnselectAll()
	x := make([]skillTrained, 0)
	a.skills.Set(copyToUntypedSlice(x))
	a.Refresh()
}

func (a *skillCatalogueArea) Refresh() {
	c, err := a.updateGroups()
	if err != nil {
		panic(err)
	}
	total := "?"
	unallocated := "?"
	if c != nil {
		total = humanizedNullInt64(c.TotalSP, "?")
		unallocated = humanizedNullInt64(c.UnallocatedSP, "?")
	}
	a.total.SetText(fmt.Sprintf("%s Total Skill Points (%s Unallocated)", total, unallocated))
}

func (a *skillCatalogueArea) updateGroups() (*model.Character, error) {
	c := a.ui.CurrentChar()
	if c == nil {
		return nil, nil
	}
	gg, err := a.ui.service.ListCharacterSkillGroupsProgress(c.ID)
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
	a.groups.Set(copyToUntypedSlice(groups))
	return c, nil
}

func (a *skillCatalogueArea) StartUpdateTicker() {
	ticker := time.NewTicker(skillsUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *skillCatalogueArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionSkills)
	if err != nil {
		slog.Error("Failed to update skillqueue", "character", characterID, "err", err)
		return
	}
	if characterID == a.ui.CurrentCharID() && changed {
		a.Refresh()
	}
}
