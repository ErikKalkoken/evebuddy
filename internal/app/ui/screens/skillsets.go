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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Status of a character for a skill set.
const (
	skillSetsStatusQualified = "Qualified"
	skillSetsStatusMissing   = "Missing skills"
	skillSetsStatusUnknown   = "Unknown"
)

// skillSetsStatusOptions returns the status filter options. Unknown is not offered.
func skillSetsStatusOptions() []string {
	return []string{skillSetsStatusQualified, skillSetsStatusMissing}
}

// Names of the skill set filters, used as labels on desktop and as option names on mobile.
const (
	skillSetsFilterStatus = "Status"
	skillSetsFilterTag    = "Tag"
)

// skillSetsFilter is the selected value of each skill set filter. Empty means not filtered.
type skillSetsFilter struct {
	status string
	tag    string
}

// match reports whether row r passes all filters.
func (f skillSetsFilter) match(r skillSetRow) bool {
	switch {
	case f.status != "" && r.status() != f.status,
		f.tag != "" && !r.tags.Contains(f.tag):
		return false
	}
	return true
}

// skillSetSkill is a character's active level for one required skill.
type skillSetSkill struct {
	activeLevel   int64
	name          string
	requiredLevel int
	typeID        int64
}

func (s skillSetSkill) met() bool {
	return s.activeLevel >= int64(s.requiredLevel)
}

// skillSetCharacter is the data of a character needed to evaluate skill sets.
type skillSetCharacter struct {
	id     int64
	levels map[int64]int64 // active level by skill type ID; nil when the character has no skills yet
	name   string
	tags   set.Set[string]
}

// skillSetRow is a character evaluated against a skill set.
type skillSetRow struct {
	characterID   int64
	characterName string
	skills        []skillSetSkill
	skillsKnown   bool
	tags          set.Set[string]
}

func newSkillSetRow(c skillSetCharacter, s app.SkillSet) skillSetRow {
	r := skillSetRow{
		characterID:   c.id,
		characterName: c.name,
		skillsKnown:   c.levels != nil,
		tags:          c.tags,
	}
	for _, x := range s.Requirements {
		r.skills = append(r.skills, skillSetSkill{
			activeLevel:   c.levels[x.TypeID],
			name:          x.Name,
			requiredLevel: x.Level,
			typeID:        x.TypeID,
		})
	}
	return r
}

func (r skillSetRow) metCount() int {
	var n int
	for _, s := range r.skills {
		if s.met() {
			n++
		}
	}
	return n
}

func (r skillSetRow) status() string {
	switch {
	case !r.skillsKnown:
		return skillSetsStatusUnknown
	case r.metCount() == len(r.skills):
		return skillSetsStatusQualified
	default:
		return skillSetsStatusMissing
	}
}

func (r skillSetRow) statusImportance() widget.Importance {
	switch r.status() {
	case skillSetsStatusQualified:
		return widget.SuccessImportance
	case skillSetsStatusMissing:
		return widget.WarningImportance
	}
	return widget.MediumImportance
}

func (r skillSetRow) statusColor() fyne.ThemeColorName {
	switch r.status() {
	case skillSetsStatusQualified:
		return theme.ColorNameSuccess
	case skillSetsStatusMissing:
		return theme.ColorNameWarning
	}
	return theme.ColorNameForeground
}

func (r skillSetRow) skillsDisplay() string {
	if !r.skillsKnown {
		return "?"
	}
	return fmt.Sprintf("%d / %d", r.metCount(), len(r.skills))
}

// skillSetCharacterItem is a character row in the skill sets list on mobile.
type skillSetCharacterItem struct {
	widget.BaseWidget

	name   *widget.Label
	skills *widget.Label
	status *widget.Label
}

func newSkillSetCharacterItem() *skillSetCharacterItem {
	w := &skillSetCharacterItem{
		name:   widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		skills: widget.NewLabel(""),
		status: widget.NewLabel(""),
	}
	w.name.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)
	return w
}

func (w *skillSetCharacterItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.New(
		layout.NewCustomPaddedVBoxLayout(-theme.Padding()),
		container.NewBorder(nil, nil, nil, w.status, w.name),
		w.skills,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *skillSetCharacterItem) set(r skillSetRow) {
	w.name.SetText(r.characterName)
	w.status.Importance = r.statusImportance()
	w.status.SetText(r.status())
	skills := r.skillsDisplay()
	if r.skillsKnown {
		skills += " skills" // no column header on mobile
	}
	w.skills.SetText(skills)
}

// SkillSets is a view that shows which characters meet the requirements of a skill set.
type SkillSets struct {
	widget.BaseWidget

	body                fyne.CanvasObject
	characters          []skillSetCharacter
	columnSorter        *xwidget.ColumnSorter[skillSetRow]
	createButton        *widget.Button
	editButton          *widget.Button
	errorLabel          *widget.Label
	filterChip          *xwidget.FilterChipCompact // only on mobile
	filterRun           xsync.LatestRun
	footerLabel         *widget.Label
	isUpdatingSetSelect bool
	rows                []skillSetRow
	rowsFiltered        []skillSetRow
	selectStatus        *kxwidget.FilterChipSelect // select chips only on desktop
	selectTag           *kxwidget.FilterChipSelect
	selectedSet         app.SkillSet
	setSelect           *widget.Select
	sets                []app.SkillSet
	sortChip            *kxwidget.SortChip
	u                   baseUI
	updateRun           xsync.LatestRun
}

func NewSkillSets(u baseUI) *SkillSets {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[skillSetRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[skillSetRow]{
			EIS: u.EVEImage(),
			GetEntity: func(r skillSetRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			Label: "Status",
			Width: 120,
			Sort: func(a, b skillSetRow) int {
				return strings.Compare(a.status(), b.status())
			},
			Update: func(r skillSetRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.status(), widget.RichTextStyle{
					ColorName: r.statusColor(),
				})
			},
		}, {
			Label: "Skills",
			Width: 75,
			Sort: func(a, b skillSetRow) int {
				return cmp.Compare(a.metCount(), b.metCount())
			},
			Update: func(r skillSetRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.skillsDisplay(), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
				})
			},
		}})
	a := &SkillSets{
		columnSorter: xwidget.NewColumnSorter(columns, "Character", xwidget.SortAsc),
		errorLabel:   ui.NewLabelWithWrapping(""),
		footerLabel:  ui.NewLabelWithWrapping(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()

	showDetail := func(r skillSetRow) {
		showSkillSetDetailWindow(a.u, a.selectedSet, r)
	}
	if !a.u.IsMobile() {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r skillSetRow) {
				showDetail(r)
			},
		)
	} else {
		l := xwidget.NewStripedList(
			func() int {
				return len(a.rowsFiltered)
			},
			func() fyne.CanvasObject {
				return newSkillSetCharacterItem()
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				if id < 0 || id >= len(a.rowsFiltered) {
					return
				}
				co.(*skillSetCharacterItem).set(a.rowsFiltered[id])
			},
		)
		l.OnSelected = func(id widget.ListItemID) {
			defer l.UnselectAll()
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			showDetail(a.rowsFiltered[id])
		}
		a.body = l
	}

	a.setSelect = widget.NewSelect(nil, func(name string) {
		if a.isUpdatingSetSelect {
			return
		}
		a.selectSet(name)
	})
	a.setSelect.PlaceHolder = "No skill set"
	a.createButton = widget.NewButtonWithIcon("Create", theme.ContentAddIcon(), func() {
		a.showEditor(app.SkillSet{})
	})
	a.editButton = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		a.showEditor(a.selectedSet)
	})
	a.editButton.Disable()

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		a.selectStatus = kxwidget.NewFilterChipSelect(skillSetsFilterStatus, skillSetsStatusOptions(), func(string) {
			a.filterRowsAsync("")
		})
		a.selectStatus.SortDisabled = true
		a.selectTag = kxwidget.NewFilterChipSelect(skillSetsFilterTag, nil, func(string) {
			a.filterRowsAsync("")
		})
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// signals
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
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *SkillSets) CreateRenderer() fyne.WidgetRenderer {
	sets := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(a.createButton, a.editButton),
		a.setSelect,
	)
	var filters fyne.CanvasObject
	if a.u.IsMobile() {
		filters = container.NewHBox(a.filterChip, a.sortChip)
	} else {
		filters = container.NewHScroll(container.NewHBox(a.selectStatus, a.selectTag))
	}
	c := container.NewBorder(
		container.NewVBox(sets, filters),
		container.NewVBox(a.errorLabel, a.footerLabel),
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

// showEditor shows a dialog for creating a skill set or editing an existing one.
// A skill set with ID 0 creates a new one.
func (a *SkillSets) showEditor(s app.SkillSet) {
	sets := slices.Clone(a.sets)
	a.u.Go(func(ctx context.Context) {
		skills, err := a.u.EVEUniverse().ListSkillTypes(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			ui.NotifyErrorAndLog("Failed to load skills", err, a.u.DisplaySnackbar)
			return
		}
		fyne.Do(func() {
			var d dialog.Dialog
			editor := newSkillSetEditor(a.u.MainWindow(), s, sets, skills,
				func(s2 app.SkillSet) {
					d.Hide()
					a.u.Go(func(ctx context.Context) {
						a.saveSkillSet(ctx, s2)
					})
				},
				func() {
					d.Hide()
				},
				func() {
					a.showDeleteConfirm(s, d)
				},
			)
			title := "Create skill set"
			if s.ID != 0 {
				title = "Edit skill set"
			}
			d = dialog.NewCustomWithoutButtons(title, editor, a.u.MainWindow())
			xdesktop.DisableShortcutsForDialog(d, a.u.MainWindow())
			_, size := a.u.MainWindow().Canvas().InteractiveArea()
			if a.u.IsMobile() {
				d.Resize(fyne.NewSize(size.Width, min(d.MinSize().Height, size.Height)))
			} else {
				d.Resize(fyne.NewSize(min(600, size.Width*0.8), min(d.MinSize().Height, size.Height*0.8)))
			}
			d.Show()
		})
	})
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *SkillSets) currentFilter() skillSetsFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return skillSetsFilter{
			status: s[skillSetsFilterStatus],
			tag:    s[skillSetsFilterTag],
		}
	}
	return skillSetsFilter{
		status: a.selectStatus.Selected,
		tag:    a.selectTag.Selected,
	}
}

// selectSet makes the skill set with name the selected set and shows its rows.
// An empty or unknown name clears the selection.
func (a *SkillSets) selectSet(name string) {
	a.selectedSet = app.SkillSet{}
	for _, s := range a.sets {
		if s.Name == name {
			a.selectedSet = s
			break
		}
	}
	a.rows = make([]skillSetRow, 0, len(a.characters))
	if a.selectedSet.ID != 0 {
		a.editButton.Enable()
		for _, c := range a.characters {
			a.rows = append(a.rows, newSkillSetRow(c, a.selectedSet))
		}
	} else {
		a.editButton.Disable()
	}
	a.filterRowsAsync("")
}

func (a *SkillSets) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.Start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)
	hasSets := len(a.sets) > 0

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r skillSetRow) bool {
			return !filter.match(r)
		})
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		var footer string
		if hasSets {
			footer = fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)
		} else {
			footer = "No skill sets"
		}
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r skillSetRow) set.Set[string] {
			return r.tags
		})...).All())

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footerLabel.SetText(footer)
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoiceOrdered(skillSetsFilterStatus, skillSetsStatusOptions()),
					xwidget.NewFilterOptionMultiChoice(skillSetsFilterTag, tagOptions),
				)
			} else {
				a.selectTag.SetOptions(tagOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	})
}

// update reloads all skill sets and characters and keeps the selected skill set when possible.
func (a *SkillSets) update(ctx context.Context) {
	a.updateAndSelect(ctx, 0)
}

// updateAndSelect reloads all skill sets and characters and selects the skill set with setID.
// A setID of 0 keeps the current selection. Falls back to the first skill set.
func (a *SkillSets) updateAndSelect(ctx context.Context, setID int64) {
	isLatest := a.updateRun.Start()
	sets, characters, err := a.fetchData(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh skill sets UI", "err", err)
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.sets = nil
			a.characters = nil
			a.setSetOptions("")
			a.errorLabel.SetText("ERROR: " + a.u.ErrorDisplay(err))
			a.errorLabel.Show()
		})
		return
	}
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.errorLabel.Hide()
		a.sets = sets
		a.characters = characters
		if setID == 0 {
			setID = a.selectedSet.ID
		}
		var name string
		for _, s := range sets {
			if s.ID == setID {
				name = s.Name
				break
			}
		}
		if name == "" && len(sets) > 0 {
			name = sets[0].Name
		}
		a.setSetOptions(name)
	})
}

// setSetOptions updates the options of the skill set select from a.sets and selects name.
func (a *SkillSets) setSetOptions(name string) {
	a.isUpdatingSetSelect = true
	a.setSelect.SetOptions(xslices.Map(a.sets, func(s app.SkillSet) string {
		return s.Name
	}))
	if name != "" {
		a.setSelect.SetSelected(name)
	} else {
		a.setSelect.ClearSelected()
	}
	a.isUpdatingSetSelect = false
	if len(a.sets) == 0 {
		a.setSelect.Disable()
	} else {
		a.setSelect.Enable()
	}
	a.selectSet(name)
}

func (a *SkillSets) fetchData(ctx context.Context) ([]app.SkillSet, []skillSetCharacter, error) {
	sets, err := a.u.Character().ListSkillSets(ctx)
	if err != nil {
		return nil, nil, err
	}
	cc, err := a.u.Character().ListCharactersShort(ctx)
	if err != nil {
		return nil, nil, err
	}
	skills, err := a.u.Character().ListAllSkills(ctx)
	if err != nil {
		return nil, nil, err
	}
	levels := make(map[int64]map[int64]int64)
	for _, s := range skills {
		if levels[s.CharacterID] == nil {
			levels[s.CharacterID] = make(map[int64]int64)
		}
		levels[s.CharacterID][s.Type.ID] = s.ActiveSkillLevel
	}
	characters := make([]skillSetCharacter, 0, len(cc))
	for _, c := range cc {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, nil, err
		}
		characters = append(characters, skillSetCharacter{
			id:     c.ID,
			levels: levels[c.ID],
			name:   c.Name,
			tags:   tags,
		})
	}
	return sets, characters, nil
}

// showDeleteConfirm asks to delete skill set s and deletes it in the background after closing the editor.
func (a *SkillSets) showDeleteConfirm(s app.SkillSet, editor dialog.Dialog) {
	ui.ShowConfirm(
		"Delete skill set?",
		fmt.Sprintf("You are about to delete the skill set \"%s\". This action cannot be undone.", s.Name),
		"Delete",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			editor.Hide()
			a.u.Go(func(ctx context.Context) {
				err := a.u.Character().DeleteSkillSet(ctx, s.ID)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					ui.NotifyErrorAndLog("Failed to delete skill set", err, a.u.DisplaySnackbar)
					return
				}
				a.update(ctx)
			})
		},
		a.u.MainWindow(),
	)
}

// saveSkillSet creates or updates a skill set and selects it.
func (a *SkillSets) saveSkillSet(ctx context.Context, s app.SkillSet) {
	id := s.ID
	var err error
	if id == 0 {
		id, err = a.u.Character().CreateSkillSet(ctx, s)
	} else {
		err = a.u.Character().UpdateSkillSet(ctx, s)
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		ui.NotifyErrorAndLog("Failed to save skill set", err, a.u.DisplaySnackbar)
		return
	}
	a.updateAndSelect(ctx, id)
}
