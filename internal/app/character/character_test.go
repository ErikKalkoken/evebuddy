package character_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/dictionary"
	"github.com/stretchr/testify/assert"
)

func TestGetCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetCharacter(ctx, 42)
		// then
		assert.ErrorIs(t, err, character.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacter()
		// when
		x2, err := cs.GetCharacter(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestGetAnyCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := character.New(st, nil, nil)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetAnyCharacter(ctx)
		// then
		assert.ErrorIs(t, err, character.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacter()
		// when
		x2, err := cs.GetAnyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func newCharacterService(st *sqlite.Storage) *character.CharacterService {
	sc := statuscache.New(cache.New())
	eu := eveuniverse.New(st, nil)
	eu.StatusCacheService = sc
	s := character.New(st, nil, nil)
	s.DictionaryService = dictionary.New(st)
	s.EveUniverseService = eu
	s.StatusCacheService = sc
	return s
}
