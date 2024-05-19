package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
)

type skillGroupProgress struct {
	trained int
	id      int32
	name    string
	total   int
}

func (g skillGroupProgress) completionP() float64 {
	return float64(g.trained) / float64(g.total)
}

// skillCatalogueArea is the UI area that shows the skill catalogue
type skillCatalogueArea struct {
	content *fyne.Container
	details *fyne.Container
	grid    *widget.GridWrap
	groups  binding.UntypedList
	total   *widget.Label
	ui      *ui
}

func (u *ui) NewSkillCatalogueArea() *skillCatalogueArea {
	a := &skillCatalogueArea{
		groups: binding.NewUntypedList(),
		total:  widget.NewLabel(""),
		ui:     u,
	}
	a.total.TextStyle.Bold = true
	a.grid = widget.NewGridWrap(
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
		func(id widget.ListItemID, item fyne.CanvasObject) {
			group, err := getFromBoundUntypedList[skillGroupProgress](a.groups, id)
			if err != nil {
				panic(err)
			}
			row := item.(*fyne.Container)
			pb := row.Objects[0].(*fyne.Container).Objects[0].(*widget.ProgressBar)
			pb.SetValue(group.completionP())
			c := row.Objects[0].(*fyne.Container).Objects[1].(*fyne.Container)
			name := c.Objects[0].(*widget.Label)
			name.SetText(group.name)
			total := c.Objects[2].(*widget.Label)
			total.SetText(humanize.Comma(int64(group.total)))
		},
	)
	a.grid.OnSelected = func(id widget.ListItemID) {
		group, err := getFromBoundUntypedList[skillGroupProgress](a.groups, id)
		if err != nil {
			panic(err)
		}
		l := a.details.Objects[0].(*widget.Label)
		l.SetText(group.name)
	}
	a.details = container.NewStack(widget.NewLabel(""))
	s := container.NewVSplit(a.grid, a.details)
	a.content = container.NewBorder(a.total, nil, nil, nil, s)

	// DUMMY data
	// groups := []skillGroupProgress{
	// 	{id: 1, name: "Engineering", total: 20, trained: 2},
	// 	{id: 2, name: "Navigation", total: 10, trained: 5},
	// 	{id: 3, name: "Spaceship Command", total: 10, trained: 3},
	// }
	// a.groups.Set(copyToUntypedSlice(groups))
	return a
}

func (a *skillCatalogueArea) Refresh() {
	err := a.updateGroups()
	if err != nil {
		panic(err)
	}
	s := "?"
	c := a.ui.CurrentChar()
	if c != nil {
		s = humanizedNullInt64(c.SkillPoints, "?")
	}
	a.total.SetText(fmt.Sprintf("%s Total Skill Points", s))
}

func (a *skillCatalogueArea) updateGroups() error {
	c := a.ui.CurrentChar()
	if c == nil {
		return nil
	}
	gg, err := a.ui.service.ListCharacterSkillGroupsProgress(c.ID)
	if err != nil {
		return err
	}
	groups := make([]skillGroupProgress, len(gg))
	for i, g := range gg {
		groups[i] = skillGroupProgress{
			trained: g.Trained,
			id:      g.ID,
			name:    g.Name,
			total:   g.Total,
		}
	}
	a.groups.Set(copyToUntypedSlice(groups))
	return nil
}
