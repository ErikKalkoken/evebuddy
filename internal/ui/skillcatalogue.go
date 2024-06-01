package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
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
		groupName, oo, err := func() (string, []model.ListCharacterSkillProgress, error) {
			group, err := getItemUntypedList[skillGroupProgress](a.groups, id)
			if err != nil {
				return "", nil, err
			}
			if !a.ui.hasCharacter() {
				return "", nil, nil
			}
			oo, err := a.ui.service.ListCharacterSkillProgress(a.ui.currentCharID(), group.id)
			if err != nil {
				return "", nil, err
			}
			return group.name, oo, nil
		}()
		if err != nil {
			t := "Failed to select skill group"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
			return
		}
		if oo == nil {
			return
		}
		skills := make([]skillTrained, len(oo))
		for i, o := range oo {
			skills[i] = skillTrained{
				activeLevel:  o.ActiveSkillLevel,
				description:  o.TypeDescription,
				groupName:    groupName,
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
			s := fyne.Size{Width: 14, Height: 14}
			f := canvas.ImageFillContain
			x1 := canvas.NewImageFromResource(theme.MediaStopIcon())
			x1.FillMode = f
			x1.SetMinSize(s)
			x2 := canvas.NewImageFromResource(theme.MediaStopIcon())
			x2.FillMode = f
			x2.SetMinSize(s)
			x3 := canvas.NewImageFromResource(theme.MediaStopIcon())
			x3.FillMode = f
			x3.SetMinSize(s)
			x4 := canvas.NewImageFromResource(theme.MediaStopIcon())
			x4.FillMode = f
			x4.SetMinSize(s)
			x5 := canvas.NewImageFromResource(theme.MediaStopIcon())
			x5.FillMode = f
			x5.SetMinSize(s)
			x := container.NewHBox(
				x1,
				x2,
				x3,
				x4,
				x5,
				widget.NewLabel("Capital Shipboard Compression Technology"))
			return x
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			label := row.Objects[5].(*widget.Label)
			skill, err := getItemUntypedList[skillTrained](a.skills, id)
			if err != nil {
				slog.Error("Failed to render skill item in skill catalogue UI", "err", err)
				label.SetText("ERROR")
				return
			}
			label.SetText(skill.name)
			for i := range 5 {
				y := row.Objects[i].(*canvas.Image)
				if skill.activeLevel > i {
					y.Resource = a.levelTrained
				} else if skill.trainedLevel > i {
					y.Resource = a.levelBlocked
				} else {
					y.Resource = a.levelUnTrained
				}
				y.Refresh()
			}
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		o, err := getItemUntypedList[skillTrained](a.skills, id)
		if err != nil {
			t := "Failed to access skill item"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
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
		dlg := dialog.NewCustom("Skill Details", "OK", s, a.ui.window)
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.ui.window.Canvas().Size().Width,
			Height: 0.8 * a.ui.window.Canvas().Size().Height,
		})
		g.UnselectAll()
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
		_, ok, err := a.ui.service.DictionaryTime(eveCategoriesKeyLastUpdated)
		if err != nil {
			return "", 0, err
		}
		if !ok {
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
	c := a.ui.currentChar()
	total := humanizedNullInt64(c.TotalSP, "?")
	unallocated := humanizedNullInt64(c.UnallocatedSP, "?")
	t := fmt.Sprintf("%s Total Skill Points (%s Unallocated)", total, unallocated)
	return t, widget.MediumImportance, nil
}

func (a *skillCatalogueArea) updateGroups() error {
	if !a.ui.hasCharacter() {
		return nil
	}
	gg, err := a.ui.service.ListCharacterSkillGroupsProgress(a.ui.currentCharID())
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
