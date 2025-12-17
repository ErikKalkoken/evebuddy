package corporationservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/kx/set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCorporation_UpdateCorporations(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	s := corporationservice.NewFake(st)
	t.Run("can delete corporations with no member character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		factory.CreateCorporation()
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		want := set.Of(corp.ID)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.EqualSet(t, want, got)
	})
	t.Run("report false when nothing deleted", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		corp := factory.CreateCorporation(character.EveCharacter.Corporation.ID)
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}

		assert.False(t, changed)
		want := set.Of(corp.ID)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.EqualSet(t, want, got)
	})
	t.Run("report false when no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, changed)
		want := set.Of[int32]()
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.EqualSet(t, want, got)
	})
	t.Run("can add missing corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: character.EveCharacter.Corporation.ID,
		})
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, changed)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(character.EveCharacter.Corporation.ID)
		xassert.EqualSet(t, want, got)
	})
	t.Run("should not add missing NPC corp", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		cc := factory.CreateEveEntityCorporation(app.EveEntity{
			ID: 1000115,
		})
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: cc.ID,
		})
		character := factory.CreateCharacter(storage.CreateCharacterParams{
			ID: ec.ID,
		})
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: character.EveCharacter.Corporation.ID,
		})
		changed, err := s.UpdateCorporations(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, changed)
		got, err := s.ListCorporationIDs(ctx)
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.Equal(t, 0, got.Size())
	})
}
