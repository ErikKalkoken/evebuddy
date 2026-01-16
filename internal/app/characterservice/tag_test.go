package characterservice_test

import (
	"context"
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
	got, err := s.ExportTags(ctx)

	// then
	require.NoError(t, err)

	assert.Contains(t, got, tag1.Name)
	assert.Contains(t, got, tag2.Name)
	assert.ElementsMatch(t, []int32{c1.ID, c2.ID}, got[tag1.Name])
	assert.ElementsMatch(t, []int32{}, got[tag2.Name])
}

func TestImportTags(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := characterservice.NewFake(st)
	ctx := context.Background()

	t.Run("can create tags for matching characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()

		// when
		err := s.ImportTags(ctx, map[string][]int32{
			"Alpha": {c1.ID, c2.ID},
			"Bravo": {},
		})

		// then
		require.NoError(t, err)

		tags, err := st.ListTagsByName(ctx)
		require.NoError(t, err)
		got := xslices.Map(tags, func(x *app.CharacterTag) string {
			return x.Name
		})
		assert.ElementsMatch(t, []string{"Alpha", "Bravo"}, got)

		for _, tag := range tags {
			cc, err := st.ListCharactersForCharacterTag(ctx, tag.ID)
			require.NoError(t, err)
			got := xslices.Map(cc, func(x *app.EntityShort[int32]) int32 {
				return x.ID
			})
			switch tag.Name {
			case "Alpha":
				assert.ElementsMatch(t, []int32{c1.ID, c2.ID}, got)
			case "Bravo":
				assert.ElementsMatch(t, []int32{}, got)
			}
		}
	})

	t.Run("should ignore missing characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()

		// when
		err := s.ImportTags(ctx, map[string][]int32{
			"Alpha": {c1.ID, 42},
		})

		// then
		require.NoError(t, err)

		tags, err := st.ListTagsByName(ctx)
		require.NoError(t, err)
		gotNames := xslices.Map(tags, func(x *app.CharacterTag) string {
			return x.Name
		})
		assert.ElementsMatch(t, []string{"Alpha"}, gotNames)

		tag := tags[0]
		cc, err := st.ListCharactersForCharacterTag(ctx, tag.ID)
		require.NoError(t, err)
		gotIDs := xslices.Map(cc, func(x *app.EntityShort[int32]) int32 {
			return x.ID
		})
		assert.ElementsMatch(t, []int32{c1.ID}, gotIDs)
	})

}
