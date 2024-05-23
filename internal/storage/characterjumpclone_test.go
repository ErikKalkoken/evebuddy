package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestCharacterJumpClone(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new empty clone", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		location := factory.CreateLocationStructure()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			LocationID:  location.ID,
			Name:        "dummy",
		}
		// when
		err := r.CreateCharacterJumpClone(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterJumpClone(ctx, c.ID, 5)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(5), x.JumpCloneID)
				assert.Equal(t, "dummy", x.Name)
				assert.Equal(t, location.ID, x.Location.ID)
				assert.Equal(t, location.Name, x.Location.Name)
				assert.Equal(t, location.SolarSystem.Constellation.Region.ID, x.Region.ID)
				assert.Equal(t, location.SolarSystem.Constellation.Region.Name, x.Region.Name)
			}
		}
	})
	t.Run("can create new clone with implants", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		location := factory.CreateLocationStructure()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			Implants:    []int32{eveType.ID},
			LocationID:  location.ID,
			Name:        "dummy",
		}
		// when
		err := r.CreateCharacterJumpClone(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterJumpClone(ctx, c.ID, 5)
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
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		location := factory.CreateLocationStructure()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			Implants:    []int32{eveType.ID},
			LocationID:  location.ID,
			Name:        "dummy",
		}
		// when
		err := r.ReplaceCharacterJumpClones(ctx, c.ID, []storage.CreateCharacterJumpCloneParams{arg})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterJumpClone(ctx, c.ID, 5)
			if assert.NoError(t, err) {
				assert.Equal(t, location.ID, x.Location.ID)
				if assert.NotEmpty(t, x.Implants) {
					y := x.Implants[0]
					assert.Equal(t, eveType, y.EveType)
				}
			}
		}
	})
	t.Run("can list clones", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		// when
		oo, err := r.ListCharacterJumpClones(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 1)
			x2 := oo[0]
			assert.Equal(t, x1, x2)
		}
	})
}
