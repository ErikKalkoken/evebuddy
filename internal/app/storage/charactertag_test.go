package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestTag(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		testutil.MustTruncateTables(db)
		t1 := factory.CreateCharacterTag()
		// when
		t2, err := st.GetTag(ctx, t1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, t2, t1)
		}
	})
	t.Run("can list ordered by name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		t1 := factory.CreateCharacterTag("Charlie")
		t2 := factory.CreateCharacterTag("Alpha")
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
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		// when
		_, err := st.CreateTag(ctx, tag.Name)
		// then
		assert.ErrorIs(t, err, app.ErrAlreadyExists)
	})
	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		t1 := factory.CreateCharacterTag()
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
		testutil.MustTruncateTables(db)
		t1 := factory.CreateCharacterTag()
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
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		character1 := factory.CreateCharacter()
		factory.CreateCharacter()
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
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		character1 := factory.CreateCharacter()
		err := st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character1.ID,
			TagID:       tag.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		character2 := factory.CreateCharacter()
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
				xassert.EqualSet(t, want, got)
			}
		}
	})
	t.Run("can list tags for character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		factory.CreateCharacterTag()
		character := factory.CreateCharacter()
		factory.CreateCharacter()
		err := st.CreateCharactersCharacterTag(ctx, storage.CreateCharacterTagParams{
			CharacterID: character.ID,
			TagID:       tag.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		got, err := st.ListCharacterTagsForCharacter(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, []*app.CharacterTag{tag}, got)
		}
	})
}
