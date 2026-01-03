package ui

// FIXME: This test is flakey and therefore disabled for now.
// See also related Fyne issue: https://github.com/fyne-io/fyne/issues/5847

// func TestCharacterAttributes_CanRenderWithData(t *testing.T) {
// if !*uiTestFlag {
// 	t.Skip(TestUIFlagReason)
// }
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	character := factory.CreateCharacterFull()
// 	factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{
// 		CharacterID:  character.ID,
// 		Charisma:     21,
// 		Intelligence: 22,
// 		Memory:       23,
// 		Perception:   24,
// 		Willpower:    25,
// 	})
// 	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
// 		CharacterID: character.ID,
// 		Section:     app.SectionCharacterAttributes,
// 	})
// 	test.ApplyTheme(t, test.Theme())
// 	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
// 	ui.setCharacter(character)
// 	w := test.NewWindow(ui.characterAttributes)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(600, 300))

// 	ui.characterAttributes.update()

// 	test.AssertImageMatches(t, "characterattributes/master.png", w.Canvas().Capture())
// }
