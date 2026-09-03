package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Option in search skill widget
const (
	searchSkillActive     = "Active skills"
	searchSkillRestricted = "Restricted skills"
	searchSkillAll        = "All skills"
)

type skillSearchRow struct {
	activeLevel        int64
	activeLevelRoman   string
	characterID        int64
	characterName      string
	groupID            int64
	groupName          string
	searchTarget       string
	skillPoints        int64
	skillPointsDisplay string
	trainedLevel       int64
	trainedLevelRoman  string
	typeID             int64
	typeName           string
}

func (r skillSearchRow) skillNameCondensed() string {
	skill := r.typeName
	if r.activeLevel == 0 && r.trainedLevel > 0 {
		skill += fmt.Sprintf(" [%s]", r.trainedLevelRoman)
	} else {
		skill += fmt.Sprintf(" %s", r.activeLevelRoman)
		if r.activeLevel != r.trainedLevel {
			skill += fmt.Sprintf(" [%s]", r.trainedLevelRoman)
		}
	}
	return skill
}

type SkillSearch struct {
	widget.BaseWidget

	body            fyne.CanvasObject
	columnSorter    *xwidget.ColumnSorter[skillSearchRow]
	footer          *widget.Label
	rows            []skillSearchRow
	rowsFiltered    []skillSearchRow
	searchEntry     *xwidget.SearchEntry
	selectCharacter *kxwidget.FilterChipSelect
	selectGroup     *kxwidget.FilterChipSelect
	selectSkill     *kxwidget.FilterChipSelect
	selectType      *kxwidget.FilterChipSelect
	sortChip        *kxwidget.SortChip
	top             *widget.Label
	u               baseUI
}

func NewSkillSearch(u baseUI) *SkillSearch {
	cols := []xwidget.DataColumn[skillSearchRow]{{
		Label: "Skill",
		Width: 275,
		Sort: func(a, b skillSearchRow) int {
			return strings.Compare(a.searchTarget, b.searchTarget)
		},
		Update: func(r skillSearchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.typeName)
		},
	}, {
		Label: "Active",
		Width: 70,
		Sort: func(a, b skillSearchRow) int {
			return cmp.Compare(a.activeLevel, b.activeLevel)
		},
		Update: func(r skillSearchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(
				fmt.Sprint(r.activeLevelRoman),
				widget.RichTextStyle{Alignment: fyne.TextAlignCenter},
			)
		},
	}, {
		Label: "Trained",
		Width: 70,
		Sort: func(a, b skillSearchRow) int {
			return cmp.Compare(a.trainedLevel, b.trainedLevel)
		},
		Update: func(r skillSearchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(
				fmt.Sprint(r.trainedLevelRoman),
				widget.RichTextStyle{Alignment: fyne.TextAlignCenter},
			)
		},
	}, {
		Label: "Skill Points",
		Width: 90,
		Sort: func(a, b skillSearchRow) int {
			return cmp.Compare(a.skillPoints, b.skillPoints)
		},
		Update: func(r skillSearchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(
				r.skillPointsDisplay,
				widget.RichTextStyle{Alignment: fyne.TextAlignTrailing},
			)
		},
	}, {
		Label: "Group",
		Width: 180,
		Sort: func(a, b skillSearchRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
		Update: func(r skillSearchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.groupName)
		},
	}, ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[skillSearchRow]{
		EIS: u.EVEImage(),
		GetEntity: func(r skillSearchRow) *app.EveEntity {
			return &app.EveEntity{
				ID:       r.characterID,
				Name:     r.characterName,
				Category: app.EveEntityCharacter,
			}
		},
		IsAvatar: true,
		Label:    "Character",
		Width:    250,
	})}

	columns := xwidget.NewDataColumns(cols)
	a := &SkillSearch{
		columnSorter: xwidget.NewColumnSorter(columns, "Skill", xwidget.SortAsc),
		footer:       ui.NewLabelWithTruncation(""),
		top:          ui.NewLabelWithWrapping(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if a.u.IsMobile() {
		a.body = a.makeDataList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter, a.filterRowsAsync, func(_ int, r skillSearchRow) {
				u.InfoViewer().ShowType(r.typeID, r.characterID)
			})
	}

	// filters
	a.searchEntry = xwidget.NewSearchEntry("Search skills", func(_ string) {
		a.filterRowsAsync(-1)
	})

	a.selectSkill = kxwidget.NewFilterChipSelect("", []string{
		searchSkillActive,
		searchSkillRestricted,
		searchSkillAll,
	}, func(_ string) {
		a.filterRowsAsync(-1)
	})
	a.selectSkill.Selected = searchSkillActive
	a.selectSkill.SortDisabled = true

	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Group", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectCharacter = kxwidget.NewFilterChipSelect("Character", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectType = kxwidget.NewFilterChipSelectWithSearch("Type", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync(-1)
	})

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterSkills {
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})

	return a
}

func (a *SkillSearch) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(
		a.selectGroup,
		a.selectType,
		a.selectSkill,
		a.selectCharacter,
	)
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		filters.Add(a.sortChip)
		topBox.Add(a.searchEntry)
		topBox.Add(container.NewHScroll(filters))
	} else {
		topBox.Add(container.NewBorder(nil, nil, filters, nil, a.searchEntry))
	}
	c := container.NewBorder(topBox, a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *SkillSearch) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newSearchRowWidget(p)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}

			co.(*searchRowWidget).Set(a.rowsFiltered[id])
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.u.InfoViewer().ShowType(r.typeID, r.characterID)
	}
	return l
}

func (a *SkillSearch) Focus() {
	a.u.MainWindow().Canvas().Focus(a.searchEntry)
}

func (a *SkillSearch) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	group := a.selectGroup.Selected
	character := a.selectCharacter.Selected
	type_ := a.selectType.Selected
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		rows := slices.DeleteFunc(rows, func(r skillSearchRow) bool {
			switch a.selectSkill.Selected {
			case searchSkillActive:
				return r.activeLevel == 0
			case searchSkillRestricted:
				return r.activeLevel >= r.trainedLevel
			case searchSkillAll:
				return false
			}
			return true
		})
		if character != "" {
			rows = slices.DeleteFunc(rows, func(r skillSearchRow) bool {
				return r.characterName != character
			})
		}
		if group != "" {
			rows = slices.DeleteFunc(rows, func(r skillSearchRow) bool {
				return r.groupName != group
			})
		}
		if type_ != "" {
			rows = slices.DeleteFunc(rows, func(r skillSearchRow) bool {
				return r.typeName != type_
			})
		}

		// search
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r skillSearchRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// set data & refresh
		characterOptions := xslices.Map(rows, func(r skillSearchRow) string {
			return r.characterName
		})
		groupOptions := xslices.Map(rows, func(r skillSearchRow) string {
			return r.groupName
		})
		typeOptions := xslices.Map(rows, func(r skillSearchRow) string {
			return r.typeName
		})

		footer := fmt.Sprintf("Showing %s / %s items", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectCharacter.SetOptions(characterOptions)
			a.selectGroup.SetOptions(groupOptions)
			a.selectType.SetOptions(typeOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
			switch x := a.body.(type) {
			case *widget.Table:
				x.ScrollToTop()
			}
		})
	}()
}

func (a *SkillSearch) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			xslices.Clear(&a.rows)
			a.filterRowsAsync(-1)
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
			a.top.Show()
		})
	}
	var rows []skillSearchRow
	var err error
	rows, err = a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh skills", "err", err)
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.top.Hide()
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *SkillSearch) fetchRows(ctx context.Context) ([]skillSearchRow, error) {
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := a.u.Character().ListAllSkills(ctx)
	if err != nil {
		return nil, err
	}
	var rows []skillSearchRow
	for _, s := range skills {
		r := skillSearchRow{
			activeLevel:        s.ActiveSkillLevel,
			activeLevelRoman:   ihumanize.RomanLetter(s.ActiveSkillLevel),
			characterID:        s.CharacterID,
			characterName:      characters[s.CharacterID],
			groupID:            s.Type.Group.ID,
			groupName:          s.Type.Group.Name,
			searchTarget:       strings.ToLower(fmt.Sprintf("%s %d", s.Type.Name, s.ActiveSkillLevel)),
			skillPoints:        s.SkillPointsInSkill,
			skillPointsDisplay: ihumanize.Comma(s.SkillPointsInSkill),
			trainedLevel:       s.TrainedSkillLevel,
			trainedLevelRoman:  ihumanize.RomanLetter(s.TrainedSkillLevel),
			typeID:             s.Type.ID,
			typeName:           s.Type.Name,
		}
		rows = append(rows, r)
	}
	return rows, nil
}

type searchRowWidget struct {
	widget.BaseWidget

	skill       *widget.Label
	group       *widget.Label
	skillPoints *widget.Label
	character   *widget.Label
	padding     float32
}

func newSearchRowWidget(padding float32) *searchRowWidget {
	w := &searchRowWidget{
		skill:       widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		group:       widget.NewLabel(""),
		skillPoints: widget.NewLabel(""),
		character:   widget.NewLabel(""),
		padding:     padding,
	}

	w.ExtendBaseWidget(w)
	return w
}

func (w *searchRowWidget) Set(r skillSearchRow) {
	w.skill.SetText(r.skillNameCondensed())
	w.group.SetText(r.groupName)
	w.skillPoints.SetText(fmt.Sprintf("%s SP", r.skillPointsDisplay))
	w.character.SetText(r.characterName)
}

func (w *searchRowWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(-w.padding),
		w.skill,
		container.NewHBox(w.group, layout.NewSpacer(), w.skillPoints),
		w.character,
	)

	return widget.NewSimpleRenderer(c)
}
