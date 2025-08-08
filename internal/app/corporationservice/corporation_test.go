package corporationservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	t.Run("can delete corporations with no member character", func(t *testing.T) {
		testutil.TruncateTables(db)
		corp := factory.CreateCorporation()
		factory.CreateCorporation()
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				CorporationIDs: set.Of(corp.ID),
			}})
		changed, err := s.RemoveStaleCorporations(ctx)
		if assert.NoError(t, err) {
			assert.True(t, changed)
			want := set.Of(corp.ID)
			got, err := s.ListCorporationIDs(ctx)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("report false when nothing deleted", func(t *testing.T) {
		testutil.TruncateTables(db)
		corp := factory.CreateCorporation()
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				CorporationIDs: set.Of(corp.ID),
			}})
		changed, err := s.RemoveStaleCorporations(ctx)
		if assert.NoError(t, err) {
			assert.False(t, changed)
			want := set.Of(corp.ID)
			got, err := s.ListCorporationIDs(ctx)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("report false when no corporations", func(t *testing.T) {
		testutil.TruncateTables(db)
		s := corporationservice.NewFake(st, corporationservice.Params{
			CharacterService: &corporationservice.CharacterServiceFake{
				CorporationIDs: set.Of[int32](),
			}})
		changed, err := s.RemoveStaleCorporations(ctx)
		if assert.NoError(t, err) {
			assert.False(t, changed)
			want := set.Of[int32]()
			got, err := s.ListCorporationIDs(ctx)
			if assert.NoError(t, err) {
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}
