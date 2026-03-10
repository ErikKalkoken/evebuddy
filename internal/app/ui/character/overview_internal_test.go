package character

// func TestCharacters_CanRenderWithData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(SkipUIReason)
// 	}
// 	test.ApplyTheme(t, test.Theme())
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()

// 	alliance := factory.CreateEveEntityAlliance(app.EveEntity{
// 		Name: "Wayne Inc.",
// 	})
// 	corporation := factory.CreateEveEntityCorporation(app.EveEntity{
// 		Name: "Wayne Technolgy",
// 	})
// 	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
// 		AllianceID:     optional.New(alliance.ID),
// 		Birthday:       time.Now().Add(-24 * 365 * 3 * time.Hour),
// 		CorporationID:  corporation.ID,
// 		Name:           "Bruce Wayne",
// 		SecurityStatus: optional.New(-10.0),
// 	})
// 	homeSystem := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
// 		SecurityStatus: 0.3,
// 		Name:           "Abune",
// 	})
// 	location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
// 		Name:          "Batcave",
// 		SolarSystemID: optional.New(homeSystem.ID),
// 	})
// 	ship := factory.CreateEveType(storage.CreateEveTypeParams{
// 		Name: "Merlin",
// 	})
// 	character := factory.CreateCharacterFull(storage.CreateCharacterParams{
// 		AssetValue:    optional.New(12_000_000_000.0),
// 		LocationID:    optional.New(location.ID),
// 		ID:            ec.ID,
// 		WalletBalance: optional.New(23_000_000.0),
// 		LastLoginAt:   optional.New(time.Now().Add(-24 * 7 * 2 * time.Hour)),
// 		TotalSP:       optional.New(100_000),
// 		ShipID:        optional.New(ship.ID),
// 	})
// 	factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
// 		CharacterID: character.ID,
// 		IsRead:      optional.New(false),
// 	})

// 	cases := []struct {
// 		name      string
// 		isDesktop bool
// 		filename  string
// 		size      fyne.Size
// 	}{
// 		{"desktop", true, "desktop_full", fyne.NewSize(1000, 600)},
// 		{"mobile", false, "mobile_full", fyne.NewSize(500, 800)},
// 	}
// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			ui := MakeFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
// 			w := test.NewWindow(ui.characterOverview)
// 			defer w.Close()
// 			w.Resize(tc.size)

// 			ui.characterOverview.Update(t.Context())

// 			test.AssertImageMatches(t, "characters/"+tc.filename+".png", w.Canvas().Capture())
// 		})
// 	}
// }

// func TestCharacters_CanRenderWitoutData(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(SkipUIReason)
// 	}
// 	test.ApplyTheme(t, test.Theme())
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()

// 	corporation := factory.CreateEveEntityCorporation(app.EveEntity{
// 		Name: "Wayne Technolgy",
// 	})
// 	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
// 		Birthday:       time.Now().Add(-24 * 365 * 3 * time.Hour),
// 		CorporationID:  corporation.ID,
// 		Name:           "Bruce Wayne",
// 		SecurityStatus: optional.New(-10.0),
// 	})
// 	factory.CreateCharacter(storage.CreateCharacterParams{
// 		ID: ec.ID,
// 	})

// 	cases := []struct {
// 		name      string
// 		isDesktop bool
// 		filename  string
// 		size      fyne.Size
// 	}{
// 		{"desktop", true, "desktop_minimal", fyne.NewSize(1000, 600)},
// 		{"mobile", false, "mobile_minimal", fyne.NewSize(500, 800)},
// 	}
// 	for _, tc := range cases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			ui := MakeFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
// 			w := test.NewWindow(ui.characterOverview)
// 			defer w.Close()
// 			w.Resize(tc.size)

// 			ui.characterOverview.Update(t.Context())

// 			test.AssertImageMatches(t, "characters/"+tc.filename+".png", w.Canvas().Capture())
// 		})
// 	}
// }
