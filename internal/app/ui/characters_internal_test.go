package ui

// FIXME

// func TestOverviewUpdateCharacters(t *testing.T) {
// 	db, st, factory := testutil.New()
// 	defer db.Close()
// 	u := newUI(st)
// 	ctx := context.Background()
// 	t.Run("can update a character", func(t *testing.T) {
// 		// given
// 		testutil.TruncateTables(db)
// 		a := CharacterOverview{
// 			u: u,
// 		}
// 		factory.CreateCharacter()
// 		// when
// 		_, err := a.updateCharacters()
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Len(t, a.rows, 1)
// 		}
// 	})
// 	t.Run("can handle empty location", func(t *testing.T) {
// 		// given
// 		testutil.TruncateTables(db)
// 		a := CharacterOverview{
// 			u: u,
// 		}
// 		if err := st.UpdateOrCreateEveLocation(ctx, storage.UpdateOrCreateLocationParams{
// 			ID:   99,
// 			Name: "Dummy",
// 		}); err != nil {
// 			t.Fatal(err)
// 		}
// 		factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
// 			LocationID: optional.New(int64(99)),
// 		})
// 		// when
// 		_, err := a.updateCharacters()
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Len(t, a.rows, 1)
// 		}
// 	})
// }

// func newUI(st *storage.Storage) *BaseUI {
// 	u := &BaseUI{cs: newCharacterService(st)}
// 	return u
// }

// func newCharacterService(st *storage.Storage) app.CharacterService {
// 	sc := statuscache.New(memcache.New())
// 	eu := eveuniverse.New(st, nil)
// 	eu.StatusCacheService = sc
// 	s := character.New(st, nil, nil)
// 	s.EveUniverseService = eu
// 	s.StatusCacheService = sc
// 	return s
// }
