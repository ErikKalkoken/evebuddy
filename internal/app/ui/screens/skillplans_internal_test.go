package screens

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

func TestSkillPlanCharacterPortraits(t *testing.T) {
	for _, mobile := range []bool{false, true} {
		t.Run(fmt.Sprintf("mobile=%v", mobile), func(t *testing.T) {
			db, st, _ := testutil.NewDBInMemory()
			defer db.Close()
			u := testdouble.NewUIFake(testdouble.UIParams{
				App: test.NewTempApp(t), Storage: st, IsMobile: mobile,
			})
			a := NewSkillPlans(u)
			a.rows = []skillPlanRow{
				{group: "Account A", character: characterservice.SkillPlanCharacter{
					Character: &app.EntityShort{ID: 90000001, Name: "First Pilot"},
				}},
				{group: "Account B", character: characterservice.SkillPlanCharacter{
					Character: &app.EntityShort{ID: 90000002, Name: "Second Pilot"},
				}},
			}
			row := a.results.CreateItem().(*fyne.Container)
			character := row.Objects[0].(*ui.EveEntityListItem)
			assert.True(t, character.IsAvatar)
			assert.Equal(t, fyne.TextTruncateEllipsis, character.Truncation)
			a.results.UpdateItem(0, row)
			assert.Contains(t, renderedTexts(character), "First Pilot")
			assert.Equal(t, "Account tag: Account A", row.Objects[1].(*widget.Label).Text)
			a.results.UpdateItem(1, row)
			assert.Contains(t, renderedTexts(character), "Second Pilot")
			assert.NotContains(t, renderedTexts(character), "First Pilot")
			assert.Equal(t, "Account tag: Account B", row.Objects[1].(*widget.Label).Text)
			row.Resize(fyne.NewSize(320, row.MinSize().Height))
			assert.LessOrEqual(t, row.MinSize().Width, float32(320))
		})
	}
}

func TestSkillPlanSkillRowLayout(t *testing.T) {
	for _, mobile := range []bool{false, true} {
		t.Run(fmt.Sprintf("mobile=%v", mobile), func(t *testing.T) {
			db, st, _ := testutil.NewDBInMemory()
			defer db.Close()
			u := testdouble.NewUIFake(testdouble.UIParams{
				App: test.NewTempApp(t), Storage: st, IsMobile: mobile,
			})
			a := NewSkillPlans(u)
			a.detailSkills = []app.SkillPlanSkillStatus{{
				Requirement: app.SkillPlanRequirement{TypeID: 1, Name: "Capacitor Systems Operation", Level: 3},
			}}
			width := float32(800)
			if mobile {
				width = 320
			}
			for _, tc := range []struct {
				name         string
				hasCharacter bool
				known        bool
				active       int64
				status       string
			}{
				{name: "plan requirements"},
				{name: "qualified", hasCharacter: true, known: true, active: 3, status: "Active: 3 | Met"},
				{name: "missing", hasCharacter: true, known: true, active: 1, status: "Active: 1 | Missing 2 level(s)"},
				{name: "unknown", hasCharacter: true, status: "Active: unknown"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					a.hasCharacter = tc.hasCharacter
					a.detailKnown = tc.known
					a.detailSkills[0].ActiveLevel = tc.active
					row := a.details.CreateItem().(*fyne.Container)
					a.details.UpdateItem(0, row)
					row.Resize(fyne.NewSize(width, row.MinSize().Height))
					header := row.Objects[0].(*fyne.Container)
					name := header.Objects[0].(*widget.Label)
					level := header.Objects[1].(*widget.Label)
					status := row.Objects[1].(*widget.Label)
					assert.Equal(t, "Capacitor Systems Operation", name.Text)
					assert.Equal(t, "Required: 3", level.Text)
					assert.Equal(t, fyne.TextAlignTrailing, level.Alignment)
					assert.Equal(t, name.Position().Y, level.Position().Y)
					assert.Less(t, name.Position().X, level.Position().X)
					assert.Greater(t, name.Size().Width, float32(0))
					assert.InDelta(t, width, level.Position().X+level.Size().Width, 0.1)
					assert.Equal(t, tc.status, status.Text)
					assert.Equal(t, tc.hasCharacter, status.Visible())
					if !tc.hasCharacter {
						assert.Equal(t, header.MinSize().Height, row.MinSize().Height, "no blank status line should be reserved")
					}
				})
			}
		})
	}
}

func TestSkillPlanSkillRowHeightChanges(t *testing.T) {
	for _, mobile := range []bool{false, true} {
		t.Run(fmt.Sprintf("mobile=%v", mobile), func(t *testing.T) {
			db, st, _ := testutil.NewDBInMemory()
			defer db.Close()
			u := testdouble.NewUIFake(testdouble.UIParams{
				App: test.NewTempApp(t), Storage: st, IsMobile: mobile,
			})
			a := NewSkillPlans(u)
			a.selected = app.SkillPlan{Name: "Travel", Requirements: []app.SkillPlanRequirement{
				{TypeID: 1, Name: "Navigation", Level: 4},
				{TypeID: 2, Name: "Afterburner", Level: 3},
			}}
			rendered := make(map[int]fyne.CanvasObject)
			update := a.details.UpdateItem
			a.details.UpdateItem = func(id widget.ListItemID, co fyne.CanvasObject) {
				update(id, co)
				rendered[id] = co
			}
			w := u.MainWindow()
			defer w.Close()
			w.SetContent(a)
			w.Resize(fyne.NewSize(800, 740))
			a.showCharacter(nil)
			a.details.Resize(fyne.NewSize(320, 300))
			a.details.Refresh()
			require.Contains(t, rendered, 0)
			compactHeight := rendered[0].Size().Height
			assert.Equal(t, rendered[0].MinSize().Height, compactHeight, "rendered rows must not retain the old two-line template height")
			assert.Equal(t, rendered[0].(*fyne.Container).Objects[0].MinSize().Height, compactHeight)

			a.showCharacter(&characterservice.SkillPlanCharacter{
				Character:   &app.EntityShort{ID: 1, Name: "Pilot"},
				Skills:      a.selected.Evaluate(map[int64]int64{1: 4, 2: 1}),
				SkillsKnown: true,
				UpdatedAt:   time.Now(),
			})
			assert.Greater(t, rendered[0].Size().Height, compactHeight, "character status must still fit when shown")
			assert.Equal(t, rendered[0].MinSize().Height, rendered[0].Size().Height)
			a.showCharacter(nil)
			assert.Equal(t, compactHeight, rendered[0].Size().Height, "clearing the character should collapse the status space again")
		})
	}
}

func TestSkillPlansOverview(t *testing.T) {
	for _, mobile := range []bool{false, true} {
		t.Run(fmt.Sprintf("mobile=%v", mobile), func(t *testing.T) {
			db, st, f := testutil.NewDBInMemory()
			defer db.Close()
			ctx := t.Context()
			u := testdouble.NewUIFake(testdouble.UIParams{
				App: test.NewTempApp(t), Storage: st, IsMobile: mobile,
				Go: func(job func(context.Context)) { job(ctx) },
			})
			group := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
			skill := f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation", IsPublished: true})
			qualified := f.CreateCharacter()
			missing := f.CreateCharacter()
			unknown := f.CreateCharacter()
			for _, c := range []*app.Character{qualified, missing} {
				f.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{CharacterID: c.ID, Section: app.SectionCharacterSkills})
			}
			f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
				CharacterID: qualified.ID, TypeID: skill.ID, ActiveSkillLevel: 4, TrainedSkillLevel: 5,
			})
			for _, name := range []string{"Account A", "Fleet"} {
				tag, err := u.Character().CreateTag(ctx, name)
				require.NoError(t, err)
				require.NoError(t, u.Character().AddTagToCharacter(ctx, qualified.ID, tag.ID))
			}
			id, err := u.Character().SaveSkillPlan(ctx, 0, "Travel", "Navigation IV")
			require.NoError(t, err)
			_, err = u.Character().SaveSkillPlan(ctx, 0, "Expert", "Navigation V")
			require.NoError(t, err)
			a := NewSkillPlans(u)
			w := u.MainWindow()
			w.SetContent(a)
			defer w.Close()
			size := fyne.NewSize(1000, 700)
			if mobile {
				size = fyne.NewSize(360, 740)
			}
			w.Resize(size)
			assert.LessOrEqual(t, a.MinSize().Width, size.Width, "overview must fit the screen width")
			a.update(ctx)
			assert.Equal(t, "Expert", a.planSelect.Selected)
			a.planSelect.SetSelected("Travel")
			assert.Equal(t, id, a.selected.ID)
			require.Len(t, a.characters, 3)
			require.Len(t, a.rows, 4)
			require.Len(t, a.detailSkills, 1)
			assert.False(t, a.hasCharacter)
			if mobile {
				assert.Nil(t, a.edit)
				assert.Nil(t, a.remove)
			} else {
				assert.False(t, a.edit.Disabled())
				assert.False(t, a.remove.Disabled())
			}

			a.qualifiedOnly.SetChecked(true)
			require.Len(t, a.rows, 2)
			a.tagSelect.SetSelected("Account A")
			require.Len(t, a.rows, 1)
			a.results.Select(0)
			assert.True(t, a.hasCharacter)
			assert.True(t, a.detailKnown)
			assert.Equal(t, int64(4), a.detailSkills[0].ActiveLevel)
			assert.True(t, a.detailSkills[0].Met())
			a.search.SetText("no such character")
			assert.Empty(t, a.rows)
			assert.False(t, a.hasCharacter, "hidden characters must not leave misleading detail levels")
			a.search.SetText("")
			a.tagSelect.ClearSelected()
			a.qualifiedOnly.SetChecked(false)
			for i, r := range a.rows {
				if r.character.Character.ID == unknown.ID {
					a.results.Select(i)
				}

			}
			assert.True(t, a.hasCharacter)
			assert.False(t, a.detailKnown)
			assert.Contains(t, a.detailTitle.Text, "Unknown")

			// A newly completed skills sync refreshes the overview without restarting.
			f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
				CharacterID: missing.ID, TypeID: skill.ID, ActiveSkillLevel: 5, TrainedSkillLevel: 5,
			})
			u.Signals().CharacterSectionUpdated.Emit(ctx, app.CharacterSectionUpdated{CharacterID: missing.ID, Section: app.SectionCharacterSkills})
			a.qualifiedOnly.SetChecked(true)
			assert.Len(t, a.rows, 3)

			// Tag changes also refresh and clear a filter whose tag disappeared.
			a.tagSelect.SetSelected("Account A")
			require.NoError(t, u.Character().DeleteAllTags(ctx))
			u.Signals().TagsChanged.Emit(ctx, struct{}{})
			assert.Empty(t, a.tagSelect.Selected)
			require.Len(t, a.rows, 2)
			assert.Equal(t, "Untagged", a.rows[0].group)

			require.NoError(t, u.Character().DeleteSkillPlan(ctx, id))
			a.update(ctx)
			assert.Equal(t, "Expert", a.planSelect.Selected)
			assert.Len(t, a.rows, 1)
			require.NoError(t, u.Character().DeleteSkillPlan(ctx, a.selected.ID))
			a.update(ctx)
			assert.Empty(t, a.rows)
			assert.Empty(t, a.detailSkills)
			if !mobile {
				assert.True(t, a.edit.Disabled())
				assert.True(t, a.remove.Disabled())
			}
			assert.Contains(t, a.summary.Text, "No skill plans")
		})
	}
}

func TestSkillPlansSyncWarning(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	ctx := t.Context()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App: test.NewTempApp(t), Storage: st,
		Go: func(job func(context.Context)) { job(ctx) },
	})
	a := NewSkillPlans(u)
	c := f.CreateCharacter()
	f.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: c.ID, Section: app.SectionCharacterSkills,
		CompletedAt: time.Now().Add(-time.Hour), ErrorMessage: "ESI unavailable",
	})
	group := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
	skill := f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation", IsPublished: true})
	f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID, TypeID: skill.ID, ActiveSkillLevel: 5, TrainedSkillLevel: 5})
	_, err := u.Character().SaveSkillPlan(ctx, 0, "Travel", "Navigation I")
	require.NoError(t, err)
	a.update(ctx)
	a.results.Select(0)
	assert.True(t, a.detailKnown, "retain the last successful snapshot on sync error")
	assert.Contains(t, a.detailTitle.Text, "Latest sync failed: ESI unavailable")
}

func TestSkillPlansMobileReadOnly(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	ctx := t.Context()
	var messages []string
	var jobs int
	u := testdouble.NewUIFake(testdouble.UIParams{
		App: test.NewTempApp(t), Storage: st, IsMobile: true,
		Go: func(job func(context.Context)) {
			jobs++
			job(ctx)
		},
		DisplaySnackbarFunc: func(text string) { messages = append(messages, text) },
	})
	group := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
	f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation", IsPublished: true})
	_, err := u.Character().SaveSkillPlan(ctx, 0, "Travel", "Navigation IV")
	require.NoError(t, err)
	before, err := st.ListSkillPlans(ctx)
	require.NoError(t, err)
	a := NewSkillPlans(u)
	w := u.MainWindow()
	defer w.Close()
	w.SetContent(a)
	w.Resize(fyne.NewSize(360, 740))
	a.update(ctx)
	assert.Equal(t, "Travel", a.planSelect.Selected)
	require.Len(t, a.detailSkills, 1)
	assert.Nil(t, a.edit)
	assert.Nil(t, a.remove)
	texts := renderedTexts(a)
	for _, action := range []string{"New / Import", "Edit", "Delete", "Save plan", "Paste skill list"} {
		assert.NotContains(t, texts, action)
	}
	jobs = 0
	a.showEditor(app.SkillPlan{})
	a.showEditor(a.selected)
	a.deleteSelected()
	assert.Zero(t, jobs, "mobile mutation actions must not start background work")
	assert.Nil(t, w.Canvas().Overlays().Top(), "mobile must not open editors or delete confirmations")
	assert.Equal(t, []string{
		"Skill plans are view-only on mobile.",
		"Skill plans are view-only on mobile.",
		"Skill plans are view-only on mobile.",
	}, messages)
	after, err := st.ListSkillPlans(ctx)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	a.selectPlan(app.SkillPlan{})
	assert.Contains(t, a.summary.Text, "view-only on mobile")
	assert.NotContains(t, a.summary.Text, "New / Import")
}

func TestSkillPlansEditor(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	ctx := t.Context()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App: test.NewTempApp(t), Storage: st,
		Go: func(job func(context.Context)) { job(ctx) },
	})
	group := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
	f.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, Name: "Navigation", IsPublished: true})
	skills, err := u.EVEUniverse().ListSkills(ctx)
	require.NoError(t, err)
	a := NewSkillPlans(u)
	w := u.MainWindow()
	defer w.Close()
	w.SetContent(a)
	size := fyne.NewSize(1000, 700)
	w.Resize(size)
	d := a.makeEditor(app.SkillPlan{}, skills)
	d.Show()
	assert.LessOrEqual(t, d.MinSize().Width, size.Width)
	assert.LessOrEqual(t, d.MinSize().Height, size.Height)
	var name, text *widget.Entry
	var save, paste, add *widget.Button
	var skillPicker *widget.SelectEntry
	var levelPicker *widget.Select
	var errorLabel *widget.Label
	var walk func(fyne.CanvasObject)
	walk = func(co fyne.CanvasObject) {
		switch x := co.(type) {
		case *widget.Entry:
			if x.PlaceHolder == "Plan name" {
				name = x
			} else if x.MultiLine {
				text = x
			}
		case *widget.Button:
			if x.Text == "Save plan" {
				save = x
			} else if x.Text == "Paste skill list" {
				paste = x
			} else if x.Text == "Add skill" {
				add = x
			}
		case *widget.SelectEntry:
			skillPicker = x
		case *widget.Select:
			levelPicker = x
		case *widget.Label:
			if x.Importance == widget.DangerImportance {
				errorLabel = x
			}
		case *fyne.Container:
			for _, child := range x.Objects {
				walk(child)
			}
		case *container.Scroll:
			walk(x.Content)
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(x).Objects() {
				walk(child)
			}
		}
	}
	walk(w.Canvas().Overlays().Top())
	require.NotNil(t, name)
	require.NotNil(t, text)
	require.NotNil(t, errorLabel)
	require.NotNil(t, save)
	require.NotNil(t, paste)
	require.NotNil(t, add)
	require.NotNil(t, skillPicker)
	require.NotNil(t, levelPicker)
	assert.True(t, add.Disabled())
	name.SetText("Travel")
	text.SetText("Not a skill IV")
	test.Tap(save)
	assert.Contains(t, errorLabel.Text, "unknown skill")
	assert.NotNil(t, w.Canvas().Overlays().Top(), "invalid input must leave the editor open")
	plans, err := st.ListSkillPlans(ctx)
	require.NoError(t, err)
	assert.Empty(t, plans)

	fyne.CurrentApp().Clipboard().SetContent("Navigation IV\nNavigation II")
	test.Tap(paste)
	assert.Equal(t, "Navigation IV\nNavigation II", text.Text)
	test.Tap(save)
	plans, err = st.ListSkillPlans(ctx)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "Travel", plans[0].Name)
	require.Len(t, plans[0].Requirements, 1)
	assert.Equal(t, 4, plans[0].Requirements[0].Level)
	assert.Equal(t, plans[0].ID, a.selected.ID)
	assert.Nil(t, w.Canvas().Overlays().Top())

	d = a.makeEditor(plans[0], skills)
	d.Show()
	walk(w.Canvas().Overlays().Top())
	assert.Equal(t, "Travel", name.Text)
	assert.Equal(t, "Navigation 4", text.Text)
	skillPicker.SetText("Navigation")
	levelPicker.SetSelected("5")
	assert.False(t, add.Disabled())
	test.Tap(add)
	test.Tap(save)
	edited, err := st.ListSkillPlans(ctx)
	require.NoError(t, err)
	require.Len(t, edited, 1)
	assert.Equal(t, plans[0].ID, edited[0].ID)
	assert.Equal(t, 5, edited[0].Requirements[0].Level)
	assert.Nil(t, w.Canvas().Overlays().Top())
}
