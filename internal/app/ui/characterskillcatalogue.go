package ui

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type skillGroupRow struct {
	id      int64
	name    string
	trained int
	total   int
}

func (g skillGroupRow) completionP() float64 {
	return float64(g.trained) / float64(g.total*5)
}

type skillRow struct {
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

	character         atomic.Pointer[app.Character]
	groupRows         []skillGroupRow
	groupsGrid        fyne.CanvasObject
	levelBlocked      *theme.ErrorThemedResource
	levelTrained      *theme.PrimaryThemedResource
	levelUnTrained    *theme.DisabledResource
	selectedGroupID   int64
	skillRows         []skillRow
	skillRowsFiltered []skillRow
	skillsGrid        fyne.CanvasObject
	top               *widget.Label
	footer            *widget.Label
	u                 *baseUI
}

func newCharacterSkillCatalogue(u *baseUI) *characterSkillCatalogue {
	a := &characterSkillCatalogue{
		footer:            newLabelWithTruncation(),
		groupRows:         make([]skillGroupRow, 0),
		levelBlocked:      theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:      theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained:    theme.NewDisabledResource(theme.MediaStopIcon()),
		skillRows:         make([]skillRow, 0),
		skillRowsFiltered: make([]skillRow, 0),
		top:               newLabelWithWrapping(),
		u:                 u,
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
	c := container.NewBorder(a.top, a.footer, nil, nil, s)
	return widget.NewSimpleRenderer(c)
}

func (a *characterSkillCatalogue) makeGroupsGrid() fyne.CanvasObject {
	length := func() int {
		return len(a.groupRows)
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
		if id >= len(a.groupRows) {
			return
		}
		group := a.groupRows[id]
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
			if id >= len(a.groupRows) {
				unselectAll()
				return
			}
			group := a.groupRows[id]
			a.selectedGroupID = group.id
			a.filterRowsAsync()
		}
	}
	return makeGridOrList(a.u.isMobile, length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *characterSkillCatalogue) makeSkillsGrid() fyne.CanvasObject {
	length := func() int {
		return len(a.skillRowsFiltered)
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
		if id >= len(a.skillRowsFiltered) {
			return
		}
		skill := a.skillRowsFiltered[id]
		row := co.(*fyne.Container).Objects
		label := row[0].(*widget.Label)
		label.SetText(skill.name)
		level := row[1].(*skillLevel)
		level.Set(skill.activeLevel, skill.trainedLevel, 0)
	}
	makeOnSelected := func(unselectAll func()) func(int) {
		unselectAll()
		return func(id int) {
			if id >= len(a.skillRowsFiltered) {
				return
			}
			skill := a.skillRowsFiltered[id]
			a.u.ShowTypeInfoWindowWithCharacter(skill.id, characterIDOrZero(a.character.Load()))
		}
	}
	return makeGridOrList(a.u.isMobile, length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *characterSkillCatalogue) filterRowsAsync() {
	skills := slices.Clone(a.skillRows)
	groupID := a.selectedGroupID

	go func() {
		if groupID > 0 {
			skills = slices.DeleteFunc(skills, func(x skillRow) bool {
				return x.groupID != groupID
			})
		}
		total := len(skills)

		slices.SortFunc(skills, func(a, b skillRow) int {
			return strings.Compare(a.name, b.name)
		})

		footer := fmt.Sprintf("Showing %d / %d skills", len(skills), total)

		fyne.Do(func() {
			a.footer.SetText(footer)
			a.skillRowsFiltered = skills
			a.skillsGrid.Refresh()
		})
	}()
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
			clear(a.skillRowsFiltered)
			clear(a.groupRows)
			a.groupsGrid.Refresh()
		})
		unselectTasks()
	}

	if !a.u.scs.HasGeneralSection(app.SectionEveTypes) {
		clear()
		setTop("No data yet", widget.WarningImportance)
		return
	}
	sg, err := a.u.eus.ListSkillGroups(ctx)
	if err != nil {
		slog.Error("Failed to refresh skill catalogue UI", "err", err)
		clear()
		setTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}
	var totalSkills int
	groups := make(map[int64]skillGroupRow)
	for _, g := range sg {
		groups[g.ID] = skillGroupRow{
			id:    g.ID,
			name:  g.Name,
			total: g.SkillCount,
		}
		totalSkills += g.SkillCount
	}

	c := a.character.Load()
	if c == nil {
		clear()
		setTop("No character", widget.LowImportance)
		return
	}
	characterID := characterIDOrZero(c)
	if !a.u.scs.HasCharacterSection(characterID, app.SectionCharacterSkills) {
		clear()
		setTop("No data yet", widget.WarningImportance)
		return
	}
	oo, err := a.u.cs.ListSkills(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh skill catalogue UI", "err", err)
		clear()
		setTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}

	var trainedTotal int
	var skillRows []skillRow
	for _, o := range oo {
		skillRows = append(skillRows, skillRow{
			activeLevel:  o.ActiveSkillLevel,
			description:  o.Type.Description,
			groupID:      o.Type.Group.ID,
			groupName:    o.Type.Group.Name,
			name:         o.Type.Name,
			trainedLevel: o.TrainedSkillLevel,
		})
		g := groups[o.Type.Group.ID]
		g.trained += int(o.ActiveSkillLevel)
		groups[o.Type.Group.ID] = g
		trainedTotal += int(o.ActiveSkillLevel)
	}

	top := fmt.Sprintf("%s Total SP (%s Unallocated)",
		c.TotalSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}), c.UnallocatedSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}),
	)
	setTop(top, widget.MediumImportance)
	unselectTasks()

	groupRows := slices.SortedFunc(maps.Values(groups), func(a, b skillGroupRow) int {
		return strings.Compare(a.name, b.name)
	})

	groupRows = slices.Insert(groupRows, 0, skillGroupRow{
		id:      0,
		name:    "All",
		trained: trainedTotal,
		total:   totalSkills,
	})

	fyne.Do(func() {
		a.groupRows = groupRows
		a.groupsGrid.Refresh()
		switch x := a.groupsGrid.(type) {
		case *widget.GridWrap:
			x.Select(widget.GridWrapItemID(a.selectedGroupID))
		case *widget.List:
			x.Select(widget.ListItemID(a.selectedGroupID))
		}
		a.skillRows = skillRows
		a.filterRowsAsync()
	})
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
