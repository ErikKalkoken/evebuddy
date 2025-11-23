package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterJumpClone(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new empty clone", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		location := factory.CreateEveLocationStructure()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			LocationID:  location.ID,
			Name:        "dummy",
		}
		// when
		err := st.CreateCharacterJumpClone(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterJumpClone(ctx, c.ID, 5)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(5), x.CloneID)
				assert.Equal(t, "dummy", x.Name)
				assert.Equal(t, location.ToShort(), x.Location)
				assert.Equal(t, location.SolarSystem.Constellation.Region.ID, x.Region.ID)
				assert.Equal(t, location.SolarSystem.Constellation.Region.Name, x.Region.Name)
			}
		}
	})
	t.Run("can create new clone with implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		location := factory.CreateEveLocationStructure()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			Implants:    []int32{eveType.ID},
			LocationID:  location.ID,
			Name:        "dummy",
		}
		// when
		err := st.CreateCharacterJumpClone(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterJumpClone(ctx, c.ID, 5)
			if assert.NoError(t, err) {
				assert.Equal(t, location.ID, x.Location.ID)
				if assert.NotEmpty(t, x.Implants) {
					y := x.Implants[0]
					assert.Equal(t, eveType, y.EveType)
				}
			}
		}
	})
	t.Run("can replace existing clone", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		location := factory.CreateEveLocationStructure()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			Implants:    []int32{eveType.ID},
			LocationID:  location.ID,
			Name:        "dummy",
		}
		// when
		err := st.ReplaceCharacterJumpClones(ctx, c.ID, []storage.CreateCharacterJumpCloneParams{arg})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterJumpClone(ctx, c.ID, 5)
			if assert.NoError(t, err) {
				assert.Equal(t, location.ID, x.Location.ID)
				assert.Equal(t, "dummy", x.Name)
				if assert.NotEmpty(t, x.Implants) {
					y := x.Implants[0]
					assert.Equal(t, eveType, y.EveType)
				}
			}
		}
	})
	t.Run("can list clones for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		x2 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		// when
		oo, err := st.ListCharacterJumpClones(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			ids := xslices.Map(oo, func(a *app.CharacterJumpClone) int32 {
				return a.CloneID
			})
			assert.ElementsMatch(t, []int32{x1.CloneID, x2.CloneID}, ids)
		}
	})
	t.Run("can list clones for all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCharacterJumpClone()
		eveType := factory.CreateEveType()
		x2 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			Implants: []int32{eveType.ID},
		})
		// when
		oo, err := st.ListAllCharacterJumpClones(ctx)
		// then
		if assert.NoError(t, err) {
			ids := xslices.Map(oo, func(a *app.CharacterJumpClone2) int32 {
				return a.CloneID
			})
			assert.ElementsMatch(t, []int32{x1.CloneID, x2.CloneID}, ids)
		}
	})
}
