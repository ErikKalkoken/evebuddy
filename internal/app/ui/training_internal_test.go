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
)

func TestTraining_CanRenderWithActiveTraining(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID:            ec.ID,
		TotalSP:       optional.From(10_000_000),
		UnallocatedSP: optional.From(1_000_000),
	})
	now := time.Now().UTC()
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
		CharacterID: character.ID,
		StartDate:   now.Add(-1 * time.Hour),
		FinishDate:  now.Add(3 * time.Hour),
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionSkillqueue,
		CompletedAt: now,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
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
	character := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID:            ec.ID,
		TotalSP:       optional.From(10_000_000),
		UnallocatedSP: optional.From(1_000_000),
	})
	now := time.Now().UTC()
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionSkillqueue,
		CompletedAt: now,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
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
	factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	test.ApplyTheme(t, test.Theme())
	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
	w := test.NewWindow(ui.training)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	ui.training.update()

	test.AssertImageMatches(t, "training/minimal.png", w.Canvas().Capture())
}
