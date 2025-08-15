package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestTraining_CanRenderWithActiveTraining(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{
		ID:            ec.ID,
		TotalSP:       optional.New(10_000_000),
		UnallocatedSP: optional.New(1_000_000),
	})
	now := time.Now().UTC()
	et := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Dummy Skill"})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
		CharacterID:   character.ID,
		StartDate:     now.Add(-1 * time.Hour),
		FinishDate:    now.Add(3 * time.Hour),
		EveTypeID:     et.ID,
		FinishedLevel: 3,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterSkillqueue,
		CompletedAt: now,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.training)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.training.update()

	test.AssertImageMatches(t, "training/active.png", w.Canvas().Capture())
}

func TestTraining_CanRenderWithInActiveTraining(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{
		ID:            ec.ID,
		TotalSP:       optional.New(10_000_000),
		UnallocatedSP: optional.New(1_000_000),
	})
	now := time.Now().UTC()
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterSkillqueue,
		CompletedAt: now,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.training)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.training.update()

	test.AssertImageMatches(t, "training/inactive.png", w.Canvas().Capture())
}

func TestTraining_CanRenderWithoutData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.training)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.training.update()

	test.AssertImageMatches(t, "training/minimal.png", w.Canvas().Capture())
}

func TestTraining_Filter(t *testing.T) {
	now := time.Now().UTC()
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Alpha",
	})
	character1 := factory.CreateCharacter(storage.CreateCharacterParams{
		ID:            ec1.ID,
		TotalSP:       optional.New(10_000_000),
		UnallocatedSP: optional.New(1_000_000),
	})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
		CharacterID: character1.ID,
		StartDate:   now.Add(-1 * time.Hour),
		FinishDate:  now.Add(3 * time.Hour),
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character1.ID,
		Section:     app.SectionCharacterSkillqueue,
		CompletedAt: now,
	})

	ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bravo",
	})
	character2 := factory.CreateCharacter(storage.CreateCharacterParams{
		ID:            ec2.ID,
		TotalSP:       optional.New(10_000_000),
		UnallocatedSP: optional.New(1_000_000),
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character2.ID,
		Section:     app.SectionCharacterSkillqueue,
		CompletedAt: now,
	})
	tag := factory.CreateCharacterTag()
	factory.AddCharacterToTag(tag, character2)
	factory.CreateCharacterTag()
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ui.training.update()

	t.Run("no filter", func(t *testing.T) {
		ui.training.selectStatus.SetSelected("")
		ui.training.selectTag.SetSelected("")

		got := xslices.Map(ui.training.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Alpha", "Bravo"}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("filter active", func(t *testing.T) {
		ui.training.selectStatus.SetSelected(trainingStatusActive)
		ui.training.selectTag.SetSelected("")

		got := xslices.Map(ui.training.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Alpha"}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("filter inactive", func(t *testing.T) {
		ui.training.selectStatus.SetSelected(trainingStatusInActive)
		ui.training.selectTag.SetSelected("")

		got := xslices.Map(ui.training.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Bravo"}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("filter tag", func(t *testing.T) {
		ui.training.selectStatus.SetSelected("")
		ui.training.selectTag.SetSelected(tag.Name)

		got := xslices.Map(ui.training.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Bravo"}
		assert.ElementsMatch(t, want, got)
	})
}
