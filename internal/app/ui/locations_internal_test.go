package ui

// func TestLocations_CanRenderWithData(t *testing.T) {
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
// 		Name: "Bruce Wayne",
// 	})
// 	homeSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
// 		SecurityStatus: 0.3,
// 	})
// 	home := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
// 		Name:          "Batcave",
// 		SolarSystemID: optional.From(homeSystem.ID),
// 	})
// 	character := factory.CreateCharacterFull(storage.CreateCharacterParams{
// 		AssetValue:    optional.From(12_000_000_000.0),
// 		HomeID:        optional.From(home.ID),
// 		ID:            ec.ID,
// 		WalletBalance: optional.From(23_000_000.0),
// 		LastLoginAt:   optional.From(time.Now().Add(-24 * 7 * 2 * time.Hour)),
// 	})
// 	factory.CreateCharacterMail(storage.CreateCharacterMailParams{
// 		CharacterID: character.ID,
// 		IsRead:      false,
// 	})
// 	test.ApplyTheme(t, test.Theme())
// 	ui := NewFakeBaseUI(st, test.NewTempApp(t), true)
// 	x := ui.characters
// 	w := test.NewWindow(x)
// 	defer w.Close()
// 	w.Resize(fyne.NewSize(1700, 300))

// 	x.update()

// 	test.AssertImageMatches(t, "locations/master.png", w.Canvas().Capture())
// }
