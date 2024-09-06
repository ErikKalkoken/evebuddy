package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestCharacterImplant(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterImplantParams{
			EveTypeID:   eveType.ID,
			CharacterID: c.ID,
		}
		// when
		err := r.CreateCharacterImplant(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterImplant(ctx, c.ID, arg.EveTypeID)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType, x.EveType)
			}
		}
	})
	t.Run("can replace implants", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterImplantParams{
			EveTypeID:   eveType.ID,
			CharacterID: c.ID,
		}
		// when
		err := r.ReplaceCharacterImplants(ctx, c.ID, []storage.CreateCharacterImplantParams{arg})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterImplant(ctx, c.ID, arg.EveTypeID)
			if assert.NoError(t, err) {
				assert.Equal(t, eveType, x.EveType)
			}
		}
	})
	t.Run("can list implants", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		x2 := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		// when
		oo, err := r.ListCharacterImplants(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.New[int32]()
			for _, o := range oo {
				got.Add(o.EveType.ID)
			}
			want := set.NewFromSlice([]int32{x1.EveType.ID, x2.EveType.ID})
			assert.Equal(t, want, got)
		}
	})
}
