package characterservice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestExportTags(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()

	// given
	tag1 := factory.CreateCharacterTag()
	c1 := factory.CreateCharacter()
	factory.AddCharacterToTag(tag1, c1)
	c2 := factory.CreateCharacter()
	factory.AddCharacterToTag(tag1, c2)
	tag2 := factory.CreateCharacterTag()

	// when
	buf := new(bytes.Buffer)
	err := s.WriteTags(ctx, buf, "0.1.0")

	// then
	require.NoError(t, err)

	var got characterservice.TagsExported
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)

	assert.Contains(t, got.Tags, tag1.Name)
	assert.Contains(t, got.Tags, tag2.Name)
	assert.ElementsMatch(t, []int64{c1.ID, c2.ID}, got.Tags[tag1.Name])
	assert.ElementsMatch(t, []int64{}, got.Tags[tag2.Name])
}

func TestImportTags(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()

	t.Run("can create tags for matching characters and version", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		x := characterservice.TagsExported{
			Tags:    map[string][]int64{"Alpha": {c1.ID, c2.ID}},
			Version: "0.1.0",
		}
		b, err := json.Marshal(x)
		require.NoError(t, err)
		file := bytes.NewReader(b)

		// when
		err = s.ReadAndReplaceTags(ctx, file, "0.1.0")

		// then
		require.NoError(t, err)

		tags, err := st.ListTagsByName(ctx)
		require.NoError(t, err)
		got := xslices.Map(tags, func(x *app.CharacterTag) string {
			return x.Name
		})
		assert.ElementsMatch(t, []string{"Alpha"}, got)

		tag := tags[0]
		cc, err := st.ListCharactersForCharacterTag(ctx, tag.ID)
		require.NoError(t, err)
		got2 := xslices.Map(cc, func(x *app.EntityShort) int64 {
			return x.ID
		})
		assert.ElementsMatch(t, []int64{c1.ID, c2.ID}, got2)
	})

	t.Run("should return error when minor versions do not match", func(t *testing.T) {
		// given
		x := characterservice.TagsExported{
			Tags:    map[string][]int64{},
			Version: "0.1.0",
		}
		b, err := json.Marshal(x)
		require.NoError(t, err)
		file := bytes.NewReader(b)
		// when
		err = s.ReadAndReplaceTags(ctx, file, "0.2.0")
		// then
		assert.Error(t, err)
	})

	t.Run("should return error when major versions do not match", func(t *testing.T) {
		// given
		x := characterservice.TagsExported{
			Tags:    map[string][]int64{},
			Version: "0.1.0",
		}
		b, err := json.Marshal(x)
		require.NoError(t, err)
		file := bytes.NewReader(b)
		// when
		err = s.ReadAndReplaceTags(ctx, file, "1.1.0")
		// then
		assert.Error(t, err)
	})

	t.Run("should not return error when minor versions are the same", func(t *testing.T) {
		x := characterservice.TagsExported{
			Tags:    map[string][]int64{},
			Version: "0.1.0",
		}
		b, err := json.Marshal(x)
		require.NoError(t, err)
		file := bytes.NewReader(b)
		// when
		err = s.ReadAndReplaceTags(ctx, file, "0.1.1")
		assert.NoError(t, err)
	})
}

func TestCreateTag(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can create a new tag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		got, err := s.CreateTag(ctx, "Alpha")
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Alpha", got.Name)
			tags, err := st.ListTagsByName(ctx)
			if assert.NoError(t, err) && assert.Len(t, tags, 1) {
				assert.Equal(t, "Alpha", tags[0].Name)
			}
		}
	})
}

func TestDeleteTag(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can delete a tag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		// when
		err := s.DeleteTag(ctx, tag.ID)
		// then
		if assert.NoError(t, err) {
			tags, err := st.ListTagsByName(ctx)
			if assert.NoError(t, err) {
				assert.Empty(t, tags)
			}
		}
	})
}

func TestDeleteAllTags(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can delete all tags", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterTag()
		factory.CreateCharacterTag()
		// when
		err := s.DeleteAllTags(ctx)
		// then
		if assert.NoError(t, err) {
			tags, err := st.ListTagsByName(ctx)
			if assert.NoError(t, err) {
				assert.Empty(t, tags)
			}
		}
	})
}

func TestListTagsByName(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can list tags sorted by name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterTag("Zulu")
		factory.CreateCharacterTag("Alpha")
		// when
		got, err := s.ListTagsByName(ctx)
		// then
		if assert.NoError(t, err) && assert.Len(t, got, 2) {
			assert.Equal(t, "Alpha", got[0].Name)
			assert.Equal(t, "Zulu", got[1].Name)
		}
	})
}

func TestRenameTag(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can rename a tag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag("Old")
		// when
		err := s.RenameTag(ctx, tag.ID, "New")
		// then
		if assert.NoError(t, err) {
			tags, err := st.ListTagsByName(ctx)
			if assert.NoError(t, err) && assert.Len(t, tags, 1) {
				assert.Equal(t, "New", tags[0].Name)
			}
		}
	})
}

func TestAddAndRemoveTagFromCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can add and remove a tag from a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		c := factory.CreateCharacter()
		// when
		err := s.AddTagToCharacter(ctx, c.ID, tag.ID)
		// then
		if assert.NoError(t, err) {
			cc, err := st.ListCharacterTagsForCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, cc, 1)
			}
		}
		// when
		err = s.RemoveTagFromCharacter(ctx, c.ID, tag.ID)
		// then
		if assert.NoError(t, err) {
			cc, err := st.ListCharacterTagsForCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Empty(t, cc)
			}
		}
	})
}

func TestListCharactersForTag(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can split characters into tagged and others", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag()
		c1 := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c1)
		c2 := factory.CreateCharacter()
		// when
		tagged, others, err := s.ListCharactersForTag(ctx, tag.ID)
		// then
		if assert.NoError(t, err) {
			taggedIDs := xslices.Map(tagged, func(x *app.EntityShort) int64 { return x.ID })
			otherIDs := xslices.Map(others, func(x *app.EntityShort) int64 { return x.ID })
			assert.ElementsMatch(t, []int64{c1.ID}, taggedIDs)
			assert.ElementsMatch(t, []int64{c2.ID}, otherIDs)
		}
	})
}

func TestListTagsForCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can list tag names for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		// when
		got, err := s.ListTagsForCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, got.Contains("Alpha"))
		}
	})
	t.Run("returns empty set when character has no tags", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		got, err := s.ListTagsForCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, got.Size())
		}
	})
}
