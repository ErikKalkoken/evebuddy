package character

// FIXME

// func TestCharacterBiography_CanRenderWithData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(SkipUIReason)
// 	}
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
// 		Description: optional.New("This is a description"),
// 	})
// 	character := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
// 	test.ApplyTheme(t, test.Theme())
// 	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
// 	ui.SetCharacter(character)
// 	w := test.NewWindow(ui.characterBiography)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(600, 300))

// 	ui.characterBiography.update(t.Context())

// 	test.AssertImageMatches(t, "characterbiography/master.png", w.Canvas().Capture())
// }
