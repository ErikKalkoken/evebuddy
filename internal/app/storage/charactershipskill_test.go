package storage_test

import (
	"context"
	"maps"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/stretchr/testify/assert"
)

func TestListCharacterShipsAbilities(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	// given
	ss := factory.CreateEveShipSkill()
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
	x, err := st.ListCharacterShipsAbilities(ctx, c.ID, shipType.Name)
	// then
	if assert.NoError(t, err) && assert.Len(t, x, 1) {
		got := maps.Collect(xiter.MapSlice2(x, func(v *app.CharacterShipAbility) (int32, bool) {
			return v.Type.ID, v.CanFly
		}))
		want := map[int32]bool{
			ss.ShipTypeID: true,
		}
		assert.Equal(t, want, got)
	}
}

func TestListCharacterShipsSkills(t *testing.T) {
	db, st, factory := testutil.New()
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
	if assert.NoError(t, err) && assert.Len(t, x, 1) {
		got := x[0]
		assert.EqualValues(t, 1, got.SkillLevel)
		assert.EqualValues(t, ss.SkillName, got.SkillName)
		assert.EqualValues(t, 2, got.Rank)
		assert.EqualValues(t, ss.SkillTypeID, got.SkillTypeID)
	}
}
