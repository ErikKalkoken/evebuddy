package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/kx/set"
	"github.com/stretchr/testify/assert"

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
		if assert.NoError(t, err) {
			x, err := st.GetCorporationMember(ctx, storage.CorporationMemberParams{
				CorporationID: c.ID,
				CharacterID:   member.ID,
			})
			if assert.NoError(t, err) {
				assert.EqualValues(t, c.ID, x.CorporationID)
				assert.EqualValues(t, member, x.Character)
			}
		}
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
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CorporationMember) int32 {
				return x.Character.ID
			})...)
			want := set.Of(e1.Character.ID, e2.Character.ID)
			assert.Equal(t, want, got)
		}
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
		if assert.NoError(t, err) {
			want := set.Of(e1.Character.ID, e2.Character.ID)
			xassert.EqualSet(t, want, got)
		}
	})
	t.Run("can delete members", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		e2 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		// when
		oo, err := st.ListCorporationMembers(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CorporationMember) int32 {
				return x.Character.ID
			})...)
			want := set.Of(e1.Character.ID, e2.Character.ID)
			assert.Equal(t, want, got)
		}
	})
}
