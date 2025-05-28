package storage_test

// FIXME

// func TestListCharacterShipsAbilities(t *testing.T) {
// 	db, st, factory := testutil.New()
// 	defer db.Close()
// 	ctx := context.Background()
// 	// given
// 	c := factory.CreateCharacter()
// 	et := factory.CreateEveType(storage.CreateEveTypeParams{Name: "alpha"})
// 	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
// 		CharacterID:      c.ID,
// 		ActiveSkillLevel: 1,
// 	})
// 	// when
// 	got, err := st.ListCharacterShipsAbilities(ctx, c.ID, "alpha")
// 	if assert.NoError(t, err) && assert.Len(t, got, 1) {
// 		assert.Equal(t, et.ID, got[0].Type.ID)
// 	}
// }
