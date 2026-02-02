package storage_test

import (
	"context"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestListCharacterShipsAbilities(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	// given
	s1 := factory.CreateEveShipSkill()
	s2 := factory.CreateEveShipSkill()
	c := factory.CreateCharacter()
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel: 1,
		CharacterID:      c.ID,
		EveTypeID:        s1.SkillTypeID,
	})
	// when
	x, err := st.ListCharacterShipsAbilities(ctx, c.ID)
	// then
	require.NoError(t, err)
	require.Len(t, x, 2)
	got := maps.Collect(xiter.MapSlice2(x, func(v *app.CharacterShipAbility) (int32, bool) {
		return v.Type.ID, v.CanFly
	}))
	want := map[int32]bool{
		s1.ShipTypeID: true,
		s2.ShipTypeID: false,
	}
	assert.Equal(t, want, got)
}

func TestListCharacterShipsSkills(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	// given
	ss := factory.CreateEveShipSkill(storage.CreateShipSkillParams{
		Rank: 2,
	})
	shipType, err := st.GetEveType(ctx, ss.ShipTypeID)
	if err != nil {
		t.Fatal(err)
	}
	c := factory.CreateCharacter()
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel: 1,
		CharacterID:      c.ID,
		EveTypeID:        ss.SkillTypeID,
	})
	// when
	x, err := st.ListCharacterShipSkills(ctx, c.ID, shipType.ID)
	// then
	require.NoError(t, err)
	require.Len(t, x, 1)
	got := x[0]
	assert.EqualValues(t, 1, got.SkillLevel)
	assert.EqualValues(t, ss.SkillName, got.SkillName)
	assert.EqualValues(t, 2, got.Rank)
	assert.EqualValues(t, ss.SkillTypeID, got.SkillTypeID)

}
