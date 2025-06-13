package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestIndustrySlot_CanRenderWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character1 := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec1.ID,
	})
	industry := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeIndustry})
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		CharacterID:      character1.ID,
		EveTypeID:        industry.ID,
		ActiveSkillLevel: 5,
	})
	massProduction := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeMassProduction})
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		CharacterID:      character1.ID,
		EveTypeID:        massProduction.ID,
		ActiveSkillLevel: 5,
	})
	advancedMassProduction := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedMassProduction})
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		CharacterID:      character1.ID,
		EveTypeID:        advancedMassProduction.ID,
		ActiveSkillLevel: 3,
	})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		CharacterID: character1.ID,
		ActivityID:  int32(app.Manufacturing),
		Status:      app.JobActive,
	})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		CharacterID: character1.ID,
		ActivityID:  int32(app.Manufacturing),
		Status:      app.JobActive,
	})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		CharacterID: character1.ID,
		ActivityID:  int32(app.Manufacturing),
		Status:      app.JobReady,
	})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		CharacterID: character1.ID,
		ActivityID:  int32(app.Manufacturing),
		Status:      app.JobDelivered,
	})
	ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Clark Kent",
	})
	character2 := factory.CreateCharacterMinimal(storage.CreateCharacterParams{
		ID: ec2.ID,
	})
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		CharacterID:      character2.ID,
		EveTypeID:        industry.ID,
		ActiveSkillLevel: 3,
	})
	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_full", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_full", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			test.ApplyTheme(t, test.Theme())
			ui := NewFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.slotsManufacturing)
			defer w.Close()
			w.Resize(tc.size)

			ui.slotsManufacturing.update()

			test.AssertImageMatches(t, "industryslot/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}

func TestIndustrySlot_CanRenderEmpty(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_empty", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_empty", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			test.ApplyTheme(t, test.Theme())
			ui := NewFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.slotsManufacturing)
			defer w.Close()
			w.Resize(tc.size)

			ui.slotsManufacturing.update()

			test.AssertImageMatches(t, "industryslot/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}
