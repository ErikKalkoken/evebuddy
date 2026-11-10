package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationMember(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create members", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		member := factory.CreateEveEntityCharacter()
		// when
		err := st.CreateCorporationMember(ctx, storage.CorporationMemberParams{
			CorporationID: c.ID,
			CharacterID:   member.ID,
		})
		// then
		require.NoError(t, err)
		x, err := st.GetCorporationMember(ctx, storage.CorporationMemberParams{
			CorporationID: c.ID,
			CharacterID:   member.ID,
		})
		require.NoError(t, err)
		xassert.Equal(t, c.ID, x.CorporationID)
		xassert.Equal(t, member, x.Character)
	})
	t.Run("can list members", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		e2 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationMember()
		// when
		oo, err := st.ListCorporationMembers(ctx, c.ID)
		// then
		require.NoError(t, err)
		got := set.Of(xslices.Map(oo, func(x *app.CorporationMember) int64 {
			return x.Character.ID
		})...)
		want := set.Of(e1.Character.ID, e2.Character.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list member IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		e2 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationMember()
		// when
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(e1.Character.ID, e2.Character.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can replace members", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		kept := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		other := factory.CreateCorporationMember()
		added := factory.CreateEveEntityCharacter()
		// when
		changed, err := st.ReplaceCorporationMembers(ctx, c.ID, set.Of(kept.Character.ID, added.ID))
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(kept.Character.ID, added.ID)
		xassert.Equal(t, want, got)
		got2, err := st.ListCorporationMemberIDs(ctx, other.CorporationID)
		require.NoError(t, err)
		want2 := set.Of(other.Character.ID)
		xassert.Equal(t, want2, got2)
	})
	t.Run("should report no change when members are the same", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		// when
		changed, err := st.ReplaceCorporationMembers(ctx, c.ID, set.Of(e1.Character.ID))
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(e1.Character.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should remove all members when replaced with empty set", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		// when
		changed, err := st.ReplaceCorporationMembers(ctx, c.ID, set.Set[int64]{})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("should remove members in chunks", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		c := factory.CreateCorporation()
		factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		e3 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		// when
		changed, err := st.ReplaceCorporationMembers(ctx, c.ID, set.Of(e3.Character.ID))
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(e3.Character.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should keep members unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		e2 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		const unknownID = 42 // has no eve entity
		// when
		_, err := st.ReplaceCorporationMembers(ctx, c.ID, set.Of(e1.Character.ID, unknownID))
		// then
		require.Error(t, err)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(e1.Character.ID, e2.Character.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should return error when replacing with invalid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e := factory.CreateEveEntityCharacter()
		cases := []struct {
			name          string
			corporationID int64
			characterIDs  set.Set[int64]
		}{
			{"zero corporation ID", 0, set.Of(e.ID)},
			{"zero character ID", c.ID, set.Of(e.ID, 0)},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				_, err := st.ReplaceCorporationMembers(ctx, tc.corporationID, tc.characterIDs)
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
