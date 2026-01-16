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
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestExportTags(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := characterservice.NewFake(st)
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
	assert.ElementsMatch(t, []int32{c1.ID, c2.ID}, got.Tags[tag1.Name])
	assert.ElementsMatch(t, []int32{}, got.Tags[tag2.Name])
}

func TestImportTags(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()

	t.Run("can create tags for matching characters and version", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		x := characterservice.TagsExported{
			Tags:    map[string][]int32{"Alpha": {c1.ID, c2.ID}},
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
		got2 := xslices.Map(cc, func(x *app.EntityShort[int32]) int32 {
			return x.ID
		})
		assert.ElementsMatch(t, []int32{c1.ID, c2.ID}, got2)
	})

	t.Run("should return error when minor versions do not match", func(t *testing.T) {
		// given
		x := characterservice.TagsExported{
			Tags:    map[string][]int32{},
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
			Tags:    map[string][]int32{},
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
			Tags:    map[string][]int32{},
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
