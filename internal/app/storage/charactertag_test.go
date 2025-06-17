package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestTag(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		got, err := st.CreateTag(ctx, "name")
		// then
		if assert.NoError(t, err) {
			assert.NotEqual(t, got.ID, 0)
			assert.Equal(t, got.Name, "name")
		}
	})
	t.Run("can get by ID", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		t1 := factory.CreateTag()
		// when
		t2, err := st.GetTag(ctx, t1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, t2, t1)
		}
	})
	t.Run("can list ordered by name", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		t1 := factory.CreateTag("Charlie")
		t2 := factory.CreateTag("Alpha")
		// when
		oo, err := st.ListTagsByName(ctx)
		// then
		if assert.NoError(t, err) {
			want := []int64{t2.ID, t1.ID}
			got := xslices.Map(oo, func(x *app.CharacterTag) int64 {
				return x.ID
			})
			assert.Equal(t, want, got)
		}
	})
	t.Run("raise specfic error when tyring to create new with existing name", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		tag := factory.CreateTag()
		// when
		_, err := st.CreateTag(ctx, tag.Name)
		// then
		assert.ErrorIs(t, err, app.ErrAlreadyExists)
	})
	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		t1 := factory.CreateTag()
		// when
		err := st.UpdateTagName(ctx, t1.ID, "alpha")
		// then
		if assert.NoError(t, err) {
			t2, err := st.GetTag(ctx, t1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "alpha", t2.Name)
			}
		}
	})
	t.Run("can delete tag", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		t1 := factory.CreateTag()
		// when
		err := st.DeleteTag(ctx, t1.ID)
		// then
		if assert.NoError(t, err) {
			_, err := st.GetTag(ctx, t1.ID)
			assert.Error(t, err, app.ErrNotFound)
		}
	})
}

func TestCharacterTag(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can add tag to character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		tag := factory.CreateTag()
		character1 := factory.CreateCharacterMinimal()
		factory.CreateCharacterMinimal()
		// when
		err := st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character1.ID,
			TagID:       tag.ID,
		})
		// then
		if assert.NoError(t, err) {
			got, err := st.ListCharactersForCharacterTag(ctx, tag.ID)
			if assert.NoError(t, err) {
				assert.Equal(
					t,
					[]*app.EntityShort[int32]{
						{
							ID:   character1.ID,
							Name: character1.EveCharacter.Name,
						},
					},
					got,
				)
			}
		}
	})
	t.Run("can remove tag from character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		tag := factory.CreateTag()
		character1 := factory.CreateCharacterMinimal()
		err := st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character1.ID,
			TagID:       tag.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		character2 := factory.CreateCharacterMinimal()
		err = st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character2.ID,
			TagID:       tag.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = st.DeleteCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character1.ID,
			TagID:       tag.ID,
		})
		// then
		if assert.NoError(t, err) {
			cc, err := st.ListCharactersForCharacterTag(ctx, tag.ID)
			if assert.NoError(t, err) {
				want := set.Of(character2.ID)
				got := set.Of(xslices.Map(cc, func(x *app.EntityShort[int32]) int32 {
					return x.ID
				})...)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
	t.Run("can list tags for character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		tag := factory.CreateTag()
		factory.CreateTag()
		character := factory.CreateCharacterMinimal()
		factory.CreateCharacterMinimal()
		// when
		err := st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character.ID,
			TagID:       tag.ID,
		})
		// then
		if assert.NoError(t, err) {
			got, err := st.ListCharacterTagsForCharacter(ctx, character.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, []*app.CharacterTag{tag}, got)
			}
		}
	})
}
