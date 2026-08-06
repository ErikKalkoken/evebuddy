package skills

import (
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

type searchRow struct {
	activeLevel   int64
	characterID   int64
	characterName string
	groupID       int64
	groupName     string
	searchTarget  string
	skillName     string
	trainedLevel  int64
	typeID        int64
	typeName      string
}

type Search struct {
	widget.BaseWidget

	body            fyne.CanvasObject
	columnSorter    *xwidget.ColumnSorter[searchRow]
	footer          *widget.Label
	rows            []searchRow
	rowsFiltered    []searchRow
	search          *widget.Entry
	selectGroup     *kxwidget.FilterChipSelect
	selectType      *kxwidget.FilterChipSelect
	selectCharacter *kxwidget.FilterChipSelect
	sortButton      *xwidget.SortButton
	top             *widget.Label
	u               baseUI
}

const (
	searchColName = iota + 1
	searchColGroup
	searchColCharacter
)

func NewSearch(u baseUI) *Search {
	cols := []xwidget.DataColumn[searchRow]{{
		ID:    searchColName,
		Label: "Skill",
		Width: 300,
		Sort: func(a, b searchRow) int {
			return strings.Compare(a.searchTarget, b.searchTarget)
		},
		Update: func(r searchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.skillName)
		},
	}, {
		ID:    searchColGroup,
		Label: "Group",
		Width: 200,
		Sort: func(a, b searchRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
		Update: func(r searchRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.groupName)
		},
	}, ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[searchRow]{
		ColumnID: searchColCharacter,
		EIS:      u.EVEImage(),
		GetEntity: func(r searchRow) *app.EveEntity {
			return &app.EveEntity{
				ID:       r.characterID,
				Name:     r.characterName,
				Category: app.EveEntityCharacter,
			}
		},
		IsAvatar: true,
		Label:    "Character",
	})}

	columns := xwidget.NewDataColumns(cols)
	a := &Search{
		columnSorter: xwidget.NewColumnSorter(columns, searchColName, xwidget.SortAsc),
		footer:       ui.NewLabelWithTruncation(""),
		search:       widget.NewEntry(),
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
			a.columnSorter, a.filterRowsAsync, func(_ int, r searchRow) {
				u.InfoViewer().ShowType(r.typeID, r.characterID)
			})
	}

	// filters
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync(-1)
	})
	a.search.OnChanged = func(_ string) {
		a.filterRowsAsync(-1)
	}
	a.search.PlaceHolder = "Search items"
	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Group", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectCharacter = kxwidget.NewFilterChipSelect("Character", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectType = kxwidget.NewFilterChipSelectWithSearch("Type", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

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

func (a *Search) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(
		a.selectGroup,
		a.selectType,
		a.selectCharacter,
	)
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		filters.Add(a.sortButton)
		topBox.Add(a.search)
		topBox.Add(container.NewHScroll(filters))
	} else {
		topBox.Add(container.NewBorder(nil, nil, filters, nil, a.search))
	}
	c := container.NewBorder(topBox, a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *Search) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			character := widget.NewLabel("Template")
			group := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				title,
				group,
				character,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			box := co.(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(r.skillName)
			box[1].(*widget.Label).SetText(r.groupName)
			box[2].(*widget.Label).SetText(r.characterName)
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

func (a *Search) Focus() {
	a.u.MainWindow().Canvas().Focus(a.search)
}

func (a *Search) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	group := a.selectGroup.Selected
	character := a.selectCharacter.Selected
	type_ := a.selectType.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		if character != "" {
			rows = slices.DeleteFunc(rows, func(r searchRow) bool {
				return r.characterName != character
			})
		}
		if group != "" {
			rows = slices.DeleteFunc(rows, func(r searchRow) bool {
				return r.groupName != group
			})
		}
		if type_ != "" {
			rows = slices.DeleteFunc(rows, func(r searchRow) bool {
				return r.typeName != type_
			})
		}

		// search filter
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r searchRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// set data & refresh
		characterOptions := xslices.Map(rows, func(r searchRow) string {
			return r.characterName
		})
		groupOptions := xslices.Map(rows, func(r searchRow) string {
			return r.groupName
		})
		typeOptions := xslices.Map(rows, func(r searchRow) string {
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

func (a *Search) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
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
	var rows []searchRow
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

func (a *Search) fetchRows(ctx context.Context) ([]searchRow, error) {
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := a.u.Character().ListAllSkills(ctx)
	if err != nil {
		return nil, err
	}
	var rows []searchRow
	for _, s := range skills {
		if s.ActiveSkillLevel == 0 {
			continue
		}
		r := searchRow{
			activeLevel:   s.ActiveSkillLevel,
			characterID:   s.CharacterID,
			characterName: characters[s.CharacterID],
			groupID:       s.Type.Group.ID,
			groupName:     s.Type.Group.Name,
			searchTarget:  strings.ToLower(fmt.Sprintf("%s %d", s.Type.Name, s.ActiveSkillLevel)),
			skillName:     app.SkillDisplayName(s.Type.Name, s.ActiveSkillLevel),
			trainedLevel:  s.TrainedSkillLevel,
			typeID:        s.Type.ID,
			typeName:      s.Type.Name,
		}
		rows = append(rows, r)
	}
	return rows, nil
}
