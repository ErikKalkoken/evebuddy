package screens

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// skillSetEditor is the content of the dialog for creating or editing a skill set.
type skillSetEditor struct {
	widget.BaseWidget

	onSave func(app.SkillSet)

	cancelButton *widget.Button
	deleteButton *widget.Button   // nil for a new set
	levelButtons []*widget.Button // of the last shown level dialog
	levelDialog  dialog.Dialog    // last shown level dialog
	nameEntry    *widget.Entry
	parent       fyne.Window
	skillsEntry  *widget.Entry
	saveButton   *widget.Button
	searchEntry  *xwidget.CompletionEntry

	// Untouched empty fields are not flagged, so cancelling a new form does not flash errors.
	nameEdited, skillsEdited bool
	nameValidator            fyne.StringValidator
	set                      app.SkillSet
	skillsByName             map[string]*app.EveType
}

func newSkillSetEditor(
	parent fyne.Window,
	s app.SkillSet,
	sets []app.SkillSet,
	skills []*app.EveType,
	onSave func(app.SkillSet),
	onCancel func(),
	onDelete func(),
) *skillSetEditor {
	skillsByName := make(map[string]*app.EveType, len(skills))
	for _, et := range skills {
		skillsByName[strings.ToLower(et.Name)] = et
	}
	skillNames := xslices.Map(skills, func(et *app.EveType) string {
		return et.Name
	})
	slices.Sort(skillNames)
	w := &skillSetEditor{
		nameEntry:     widget.NewEntry(),
		nameValidator: newSkillSetNameValidator(sets, s.ID),
		onSave:        onSave,
		parent:        parent,
		skillsEntry:   widget.NewMultiLineEntry(),
		set:           s,
		searchEntry:   xwidget.NewCompletionEntry(skillNames),
		skillsByName:  skillsByName,
	}
	w.ExtendBaseWidget(w)

	w.nameEntry.SetText(s.Name)
	w.nameEntry.Validator = func(text string) error {
		if !w.nameEdited && text == "" {
			return nil
		}
		return w.nameValidator(text)
	}
	w.nameEntry.OnChanged = func(string) {
		w.nameEdited = true
		w.updateSaveButton()
	}

	var lines []string
	for _, r := range s.Requirements {
		lines = append(lines, fmt.Sprintf("%s %s", r.Name, ihumanize.RomanLetter(r.Level)))
	}
	w.skillsEntry.SetMinRowsVisible(10) // fits all entry fields above the soft keyboard on mobile
	w.skillsEntry.PlaceHolder = "Navigation IV\nAfterburner 3"
	w.skillsEntry.SetText(strings.Join(lines, "\n"))
	w.skillsEntry.Validator = func(text string) error {
		if !w.skillsEdited && text == "" {
			return nil
		}
		_, err := parseSkillSetRequirements(text, w.skillsByName)
		return err
	}
	w.skillsEntry.OnChanged = func(string) {
		w.skillsEdited = true
		w.updateSaveButton()
	}

	w.searchEntry.PlaceHolder = "Search skills to add…"
	w.searchEntry.SuggestionsAbove = true // avoids conflict with soft keyboard and smaller windows
	w.searchEntry.VisibleSuggestions = 7  // stays below the name field with 10 skill rows
	w.searchEntry.OnPicked = func(name string) {
		if et, ok := w.skillsByName[strings.ToLower(name)]; ok {
			w.showLevelDialog(et)
		}
	}
	w.searchEntry.OnSubmitted = func(text string) {
		if et, ok := w.skillsByName[strings.ToLower(strings.TrimSpace(text))]; ok {
			w.showLevelDialog(et)
		}
	}

	confirm := "Create"
	if s.ID != 0 {
		confirm = "Save"
	}
	w.saveButton = widget.NewButtonWithIcon(confirm, theme.ConfirmIcon(), w.save)
	w.saveButton.Importance = widget.HighImportance
	w.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), onCancel)
	if s.ID != 0 {
		w.deleteButton = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), onDelete)
		w.deleteButton.Importance = widget.DangerImportance
	}
	w.updateSaveButton()
	return w
}

func (w *skillSetEditor) CreateRenderer() fyne.WidgetRenderer {
	buttons := container.NewHBox(layout.NewSpacer(), w.cancelButton, w.saveButton)
	if w.deleteButton != nil {
		buttons.Objects = slices.Insert(buttons.Objects, 0, fyne.CanvasObject(w.deleteButton))
	}
	nameItem := widget.NewFormItem("Name", w.nameEntry)
	skillsItem := widget.NewFormItem("Skills", w.skillsEntry)
	form := widget.NewForm(nameItem, skillsItem)
	form.Orientation = widget.Vertical
	p := theme.Padding()
	c := container.NewVBox(
		form,
		xwidget.NewSpacer(fyne.NewSquareSize(4*p)),
		w.searchEntry,
		xwidget.NewSpacer(fyne.NewSquareSize(4*p)),
		buttons,
	)
	return widget.NewSimpleRenderer(c)
}

// showLevelDialog asks for the level of skill et and adds it.
func (w *skillSetEditor) showLevelDialog(et *app.EveType) {
	name := widget.NewLabelWithStyle(et.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	name.Wrapping = fyne.TextWrapWord
	var d dialog.Dialog
	w.levelButtons = nil
	levels := container.NewGridWithColumns(5)
	p := theme.Padding()
	for level := 1; level <= 5; level++ {
		b := widget.NewButton(ihumanize.RomanLetter(level), func() {
			d.Hide()
			w.addSkill(et, level)
		})
		w.levelButtons = append(w.levelButtons, b)
		levels.Add(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), b))
	}
	spacer := xwidget.NewSpacer(fyne.NewSquareSize(p * 4))
	content := container.NewVBox(name, levels, spacer)
	d = dialog.NewCustom("Add skill", "Cancel", content, w.parent)
	d.SetOnClosed(func() {
		w.searchEntry.SetText("") // the picked name would not show suggestions again
	})
	w.levelDialog = d
	d.Show()
}

// addSkill appends skill et with level.
func (w *skillSetEditor) addSkill(et *app.EveType, level int) {
	line := et.Name + " " + ihumanize.RomanLetter(level)
	if current := strings.TrimRight(w.skillsEntry.Text, "\n"); current != "" {
		line = current + "\n" + line
	}
	w.skillsEntry.SetText(line)
}

func (w *skillSetEditor) save() {
	requirements, err := parseSkillSetRequirements(w.skillsEntry.Text, w.skillsByName)
	if err != nil {
		return // should not happen, because save is only enabled for valid input
	}
	w.onSave(app.SkillSet{
		ID:           w.set.ID,
		Name:         strings.TrimSpace(w.nameEntry.Text),
		Requirements: requirements,
	})
}

// updateSaveButton enables the save button only for valid input.
func (w *skillSetEditor) updateSaveButton() {
	_, err := parseSkillSetRequirements(w.skillsEntry.Text, w.skillsByName)
	if err != nil || w.nameValidator(w.nameEntry.Text) != nil {
		w.saveButton.Disable()
	} else {
		w.saveButton.Enable()
	}
}

// newSkillSetNameValidator returns a validator that rejects empty and existing skill set names.
// The name of the skill set being edited is allowed, so its case can be changed.
func newSkillSetNameValidator(sets []app.SkillSet, editedID int64) fyne.StringValidator {
	var names set.Set[string]
	for _, s := range sets {
		if s.ID == editedID {
			continue
		}
		names.Add(strings.ToLower(s.Name))
	}
	return func(text string) error {
		name := strings.TrimSpace(text)
		if name == "" {
			return errors.New("can not be empty")
		}
		if names.Contains(strings.ToLower(name)) {
			return errors.New("skill set with same name already exists")
		}
		return nil
	}
}

// parseSkillSetRequirements parses one "Skill name level" per line, with level as I-V or 1-5.
// skills maps lower case skill names to their types.
// Repeated skills are merged at their highest requested level.
func parseSkillSetRequirements(text string, skills map[string]*app.EveType) ([]app.SkillSetRequirement, error) {
	byID := make(map[int64]app.SkillSetRequirement)
	for i, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 2 {
			return nil, fmt.Errorf("line %d: expected a skill name followed by a level", i+1)
		}
		levelText := strings.ToUpper(fields[len(fields)-1])
		level := map[string]int{"I": 1, "II": 2, "III": 3, "IV": 4, "V": 5}[levelText]
		if level == 0 {
			level, _ = strconv.Atoi(levelText)
		}
		if level < 1 || level > 5 {
			return nil, fmt.Errorf("line %d: level must be I-V or 1-5", i+1)
		}
		name := strings.Join(fields[:len(fields)-1], " ")
		et, ok := skills[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("line %d: unknown skill %q", i+1, name)
		}
		if byID[et.ID].Level < level {
			byID[et.ID] = app.SkillSetRequirement{TypeID: et.ID, Name: et.Name, Level: level}
		}
	}
	if len(byID) == 0 {
		return nil, errors.New("needs at least one skill")
	}
	requirements := slices.Collect(maps.Values(byID))
	slices.SortFunc(requirements, func(a, b app.SkillSetRequirement) int {
		return strings.Compare(a.Name, b.Name)
	})
	return requirements, nil
}
