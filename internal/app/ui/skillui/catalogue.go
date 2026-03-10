package skillui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/awidget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	skillsAllSkill          = "All skills"
	skillsMySkill           = "My skills"
	skillsHavePrerequisites = "Have prerequisites for"
	skillsQueued            = "Queued"
	skillsFullyTrained      = "Fully trained"
)

type skillRow struct {
	description      string
	groupID          int64
	groupName        string
	hasPrerequisites bool
	levelActive      int64
	levelQueued      int64
	levelTrained     int64
	name             string
	searchTarget     string
	spMax            optional.Optional[int]
	spTrained        int
	typeID           int64
}

type Catalogue struct {
	widget.BaseWidget

	character      atomic.Pointer[app.Character]
	footer         *widget.Label
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
	rows           []skillRow
	rowsFiltered   []skillRow
	search         *widget.Entry
	selectGroup    *kxwidget.FilterChipSelect
	selectMain     *kxwidget.FilterChipSelect
	skills         fyne.CanvasObject
	top            *widget.Label
	u              ui
	sortButton     *xwidget.SortButton
	columnSorter   *xwidget.ColumnSorter[skillRow]
}

func NewCatalogue(u ui) *Catalogue {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[skillRow]{{
		ID:    1,
		Label: "Name",
		Sort: func(a, b skillRow) int {
			return strings.Compare(a.name, b.name)
		},
	}, {
		ID:    2,
		Label: "Level trained",
		Sort: func(a, b skillRow) int {
			return cmp.Compare(a.levelTrained, b.levelTrained)
		},
	}, {
		ID:    3,
		Label: "Skillpoints trained",
		Sort: func(a, b skillRow) int {
			return cmp.Compare(a.spTrained, b.spTrained)
		},
	}}),
		1,
		xwidget.SortAsc,
	)
	a := &Catalogue{
		columnSorter:   columnSorter,
		footer:         awidget.NewLabelWithTruncation(""),
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: theme.NewDisabledResource(theme.MediaStopIcon()),
		search:         widget.NewEntry(),
		top:            awidget.NewLabelWithWrapping(""),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.skills = a.makeSkillsGrid()

	a.search.OnChanged = func(_ string) {
		a.filterRowsAsync()
	}
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync()
	})
	a.search.PlaceHolder = "Search skills"

	a.selectGroup = kxwidget.NewFilterChipSelect("Group", []string{}, func(string) {
		a.filterRowsAsync()
	})
	a.selectMain = kxwidget.NewFilterChipSelect("", []string{
		skillsAllSkill,
		skillsMySkill,
		skillsHavePrerequisites,
		skillsQueued,
		skillsFullyTrained,
	}, func(string) {
		a.filterRowsAsync()
	})
	a.selectMain.Selected = skillsAllSkill
	a.selectMain.SortDisabled = true
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync()
	}, a.u.MainWindow())

	// signals
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	})
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		characterID := a.character.Load().IDOrZero()
		if characterID == 0 {
			return
		}
		if arg.Section == app.SectionEveTypes {
			a.update(ctx)
		}
	})
	return a
}

func (a *Catalogue) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectGroup, a.selectMain, a.sortButton)
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		topBox.Add(a.search)
		topBox.Add(container.NewHScroll(filter))
	} else {
		topBox.Add(container.NewBorder(nil, nil, filter, nil, a.search))
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.skills,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Catalogue) makeSkillsGrid() fyne.CanvasObject {
	length := func() int {
		return len(a.rowsFiltered)
	}
	makeCreateItem := func(trunc fyne.TextTruncation) func() fyne.CanvasObject {
		return func() fyne.CanvasObject {
			title := ttwidget.NewLabel("Capital Shipboard Compression Technology")
			title.Truncation = trunc
			c := container.NewBorder(
				nil,
				nil,
				awidget.NewSkillLevel(),
				nil,
				title,
			)
			return c
		}
	}
	updateItem := func(id int, co fyne.CanvasObject) {
		if id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		row := co.(*fyne.Container).Objects
		label := row[0].(*ttwidget.Label)
		label.SetText(r.name)

		tt := r.description + "\n\n"
		var levelText string
		if r.levelTrained > 0 {
			levelText = fmt.Sprintf("Level %s", ihumanize.RomanLetter(r.levelTrained))
		} else {
			tt += "Not trained"
		}
		tt += fmt.Sprintf(
			"%s    %s / %s skillpoints",
			levelText,
			ihumanize.Comma(r.spTrained),
			r.spMax.StringFunc("?", func(v int) string {
				return ihumanize.Comma(v)
			}),
		)
		label.SetToolTip(tt)

		level := row[1].(*awidget.SkillLevel)
		level.Set(r.levelActive, r.levelTrained, r.levelQueued)
	}
	makeOnSelected := func(unselectAll func()) func(int) {
		return func(id int) {
			defer unselectAll()
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			a.u.InfoWindow().ShowTypeWithCharacter(r.typeID, a.character.Load().IDOrZero())
		}
	}
	return makeGridOrList(a.u.IsMobile(), length, makeCreateItem, updateItem, makeOnSelected)
}

func (a *Catalogue) filterRowsAsync() {
	total := len(a.rows)
	rows := slices.Clone(a.rows)
	group := a.selectGroup.Selected
	main := a.selectMain.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		switch main {
		case skillsMySkill:
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return r.levelTrained == 0
			})
		case skillsHavePrerequisites:
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return !r.hasPrerequisites || r.levelActive == 5
			})
		case skillsQueued:
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return r.levelQueued == 0
			})
		case skillsFullyTrained:
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return r.levelActive < 5
			})
		}
		if group != "" {
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return r.groupName != group
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r skillRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		slices.SortFunc(rows, func(a, b skillRow) int {
			return strings.Compare(a.name, b.name)
		})

		groupOptions := xslices.Map(rows, func(r skillRow) string {
			return r.groupName
		})
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		footer := fmt.Sprintf("Showing %d / %d skills", len(rows), total)

		fyne.Do(func() {
			a.footer.SetText(footer)
			a.selectGroup.SetOptions(groupOptions)
			a.rowsFiltered = rows
			a.skills.Refresh()
		})
	}()
}

func (a *Catalogue) update(ctx context.Context) {
	setTop := func(t string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = t
			a.top.Importance = i
			a.top.Refresh()
		})
	}

	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.rowsFiltered = xslices.Reset(a.rowsFiltered)
		})
	}

	if !a.u.StatusCache().HasEveUniverseSection(app.SectionEveTypes) {
		reset()
		setTop("No data yet", widget.WarningImportance)
		return
	}

	characterID := a.character.Load().IDOrZero()
	if characterID == 0 {
		reset()
		setTop("No character", widget.LowImportance)
		return
	}

	if !a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterSkills) {
		reset()
		setTop("No data yet", widget.WarningImportance)
		return
	}

	c, err := a.u.Character().GetCharacter(ctx, characterID)
	if err != nil {
		slog.Error("Updating skill catalogue UI", "err", err)
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	a.character.Store(c)

	skills, err := a.u.Character().ListSkills(ctx, characterID)
	if err != nil {
		slog.Error("Updating skill catalogue UI", "err", err)
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	sq := app.NewCharacterSkillqueue()
	err = sq.Update(ctx, a.u.Character(), c.ID)
	if err != nil {
		slog.Error("Failed to update skill queue", "err", err)
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}

	queued := make(map[int64]int64)
	for it := range sq.All() {
		queued[it.SkillID] = max(queued[it.SkillID], it.FinishedLevel)
	}

	var rows []skillRow
	for _, o := range skills {
		rows = append(rows, skillRow{
			description:      o.Skill.Type.DescriptionPlain(),
			groupID:          o.Skill.Type.Group.ID,
			groupName:        o.Skill.Type.Group.Name,
			hasPrerequisites: o.HasPrerequisites,
			levelActive:      o.ActiveSkillLevel,
			levelQueued:      queued[o.Skill.Type.ID],
			levelTrained:     o.TrainedSkillLevel,
			name:             o.Skill.Type.Name,
			searchTarget:     strings.ToLower(o.Skill.Type.Name),
			spMax:            o.Skill.Skillpoints,
			spTrained:        int(o.SkillPointsInSkill),
			typeID:           o.Skill.Type.ID,
		})
	}

	totalSP := optional.Sum(c.TrainedSP, c.UnallocatedSP)
	top := fmt.Sprintf("%s Total SP (%s Unallocated)",
		totalSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}), c.UnallocatedSP.StringFunc("?", func(v int64) string {
			return ihumanize.Comma(v)
		}),
	)
	setTop(top, widget.MediumImportance)

	fyne.Do(func() {
		a.rows = rows
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
