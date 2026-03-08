package clonesui

// func TestAugmentations_CanRenderWithData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("UI tests are flakey")
// 	}
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
// 		Name: "Bruce Wayne",
// 		ID:   42,
// 	})
// 	character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
// 	et := factory.CreateEveType(storage.CreateEveTypeParams{
// 		Name: "Dummy Implant",
// 	})
// 	da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: app.EveDogmaAttributeImplantSlot})
// 	factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
// 		DogmaAttributeID: da.ID,
// 		EveTypeID:        et.ID,
// 		Value:            3,
// 	})
// 	factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{
// 		CharacterID: character.ID,
// 		TypeID:      et.ID,
// 	})
// 	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
// 		CharacterID: character.ID,
// 		Section:     app.SectionCharacterImplants,
// 	})
// 	test.ApplyTheme(t, test.Theme())
// 	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)

// 	a := NewAugmentations()

// 	w := test.NewWindow(a)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(600, 300))

// 	a.Update(t.Context())
// 	time.Sleep(50 * time.Millisecond)

// 	a.tree.OpenAllBranches()
// 	test.AssertImageMatches(t, "augmentations/master.png", w.Canvas().Capture())
// }
