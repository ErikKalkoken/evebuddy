package characterwindow

// func TestCharacterAdmin_CanRenderWithData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(SkipUIReason)
// 	}
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	t.Run("normal", func(t *testing.T) {
// 		testutil.MustTruncateTables(db)
// 		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
// 			Name: "Bruce Wayne",
// 		})
// 		character := factory.CreateCharacter(storage.CreateCharacterParams{
// 			ID: ec.ID,
// 		})
// 		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
// 			CharacterID: character.ID,
// 			Scopes:      app.Scopes(),
// 		})
// 		test.ApplyTheme(t, test.Theme())
// 		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
// 		ui.SetCharacter(character)
// 		a := newCharacterAdmin(&manageCharacters{
// 			u: ui,
// 		})
// 		w := test.NewWindow(a)
// 		defer w.Close()
// 		w.Resize(fyne.NewSize(600, 300))

// 		a.update(t.Context())

// 		test.AssertImageMatches(t, "managedcharacters/master.png", w.Canvas().Capture())
// 	})
// }
