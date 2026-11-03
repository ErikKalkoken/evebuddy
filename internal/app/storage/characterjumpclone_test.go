package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterJumpClone(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new empty clone", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		location := factory.CreateEveLocationStructure()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			LocationID:  location.ID,
			Name:        optional.New("dummy"),
		}
		// when
		err := st.CreateCharacterJumpClone(t.Context(), arg)
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterJumpClone(t.Context(), c.ID, 5)
		require.NoError(t, err)
		xassert.Equal(t, 5, x.CloneID)
		xassert.EqualOptional(t, "dummy", x.Name)
		xassert.Equal(t, location.ToEveLocationShort(), x.Location)
	})

	t.Run("can create new clone with implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		location := factory.CreateEveLocationStructure()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			Implants:    []int64{eveType.ID},
			LocationID:  location.ID,
			Name:        optional.New("dummy"),
		}
		// when
		err := st.CreateCharacterJumpClone(t.Context(), arg)
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterJumpClone(t.Context(), c.ID, 5)
		require.NoError(t, err)
		xassert.Equal(t, location.ID, x.Location.ID)
		if assert.NotEmpty(t, x.Implants) {
			y := x.Implants[0]
			xassert.Equal(t, eveType, y.EveType)
		}
	})

	t.Run("can replace existing clone", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		location := factory.CreateEveLocationStructure()
		eveType := factory.CreateEveType()
		arg := storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 5,
			Implants:    []int64{eveType.ID},
			LocationID:  location.ID,
			Name:        optional.New("dummy"),
		}
		// when
		err := st.ReplaceCharacterJumpClones(t.Context(), c.ID, []storage.CreateCharacterJumpCloneParams{arg})
		// then
		require.NoError(t, err)
		x, err := st.GetCharacterJumpClone(t.Context(), c.ID, 5)
		require.NoError(t, err)
		xassert.Equal(t, location.ID, x.Location.ID)
		xassert.EqualOptional(t, "dummy", x.Name)
		if assert.NotEmpty(t, x.Implants) {
			y := x.Implants[0]
			xassert.Equal(t, eveType, y.EveType)
		}
	})

	t.Run("should keep existing clones unchanged when replacing fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		location := factory.CreateEveLocationStructure()
		// when
		err := st.ReplaceCharacterJumpClones(t.Context(), c.ID, []storage.CreateCharacterJumpCloneParams{
			{
				CharacterID: c.ID,
				JumpCloneID: 5,
				LocationID:  location.ID,
			},
			{
				CharacterID: c.ID,
				JumpCloneID: 6,
			}, // invalid
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		oo, err := st.ListCharacterJumpClones(t.Context(), c.ID)
		require.NoError(t, err)
		got := xslices.Map(oo, func(x *app.CharacterJumpClone) int64 {
			return x.CloneID
		})
		xassert.Equal(t, []int64{x1.CloneID}, got)
	})

	t.Run("should return invalid error when replacing with clone of other character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c1.ID,
		})
		location := factory.CreateEveLocationStructure()
		// when
		err := st.ReplaceCharacterJumpClones(t.Context(), c1.ID, []storage.CreateCharacterJumpCloneParams{
			{
				CharacterID: c2.ID,
				JumpCloneID: 5,
				LocationID:  location.ID,
			},
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		oo1, err := st.ListCharacterJumpClones(t.Context(), c1.ID)
		require.NoError(t, err)
		got := xslices.Map(oo1, func(x *app.CharacterJumpClone) int64 {
			return x.CloneID
		})
		xassert.Equal(t, []int64{x1.CloneID}, got)
		oo2, err := st.ListCharacterJumpClones(t.Context(), c2.ID)
		require.NoError(t, err)
		assert.Empty(t, oo2)
	})

	t.Run("can list clones for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x1 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		x2 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
		})
		// when
		oo, err := st.ListCharacterJumpClones(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		ids := xslices.Map(oo, func(a *app.CharacterJumpClone) int64 {
			return a.CloneID
		})
		assert.ElementsMatch(t, []int64{x1.CloneID, x2.CloneID}, ids)
	})

	t.Run("can list clones for all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateCharacterJumpClone()
		eveType := factory.CreateEveType()
		x2 := factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			Implants: []int64{eveType.ID},
		})
		// when
		oo, err := st.ListAllCharacterJumpClones(t.Context())
		// then
		require.NoError(t, err)
		ids := xslices.Map(oo, func(a *app.CharacterJumpClone2) int64 {
			return a.CloneID
		})
		assert.ElementsMatch(t, []int64{x1.CloneID, x2.CloneID}, ids)
	})
}
