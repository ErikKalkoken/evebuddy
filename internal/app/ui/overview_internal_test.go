package ui

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestOverviewUpdateCharacters(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	u := newUI(st)
	ctx := context.Background()
	t.Run("can update a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := OverviewArea{
			u: u,
		}
		factory.CreateCharacter()
		// when
		_, err := a.updateCharacters()
		// then
		if assert.NoError(t, err) {
			assert.Len(t, a.rows, 1)
		}
	})
	t.Run("can handle empty location", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := OverviewArea{
			u: u,
		}
		if err := st.UpdateOrCreateEveLocation(ctx, storage.UpdateOrCreateLocationParams{
			ID:   99,
			Name: "Dummy",
		}); err != nil {
			t.Fatal(err)
		}
		factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LocationID: optional.New(int64(99)),
		})
		// when
		_, err := a.updateCharacters()
		// then
		if assert.NoError(t, err) {
			assert.Len(t, a.rows, 1)
		}
	})
}

func newUI(st *storage.Storage) *BaseUI {
	u := &BaseUI{}
	u.CharacterService = newCharacterService(st)
	return u
}

func newCharacterService(st *storage.Storage) *character.CharacterService {
	sc := statuscache.New(memcache.New())
	eu := eveuniverse.New(st, nil)
	eu.StatusCacheService = sc
	s := character.New(st, nil, nil)
	s.EveUniverseService = eu
	s.StatusCacheService = sc
	return s
}
