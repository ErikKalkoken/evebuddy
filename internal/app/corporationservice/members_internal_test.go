package corporationservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCorporationMembersESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should add new members and remove stale members", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		m1 := factory.CreateEveEntityCharacter()
		m2 := factory.CreateEveEntityCharacter()
		factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/members", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{m1.ID, m2.ID}),
		)
		// when
		changed, err := s.updateMembersESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(m1.ID, m2.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("should do nothing when member list is unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		m1 := factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/members", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{m1.Character.ID}),
		)
		// when
		changed, err := s.updateMembersESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.ListCorporationMemberIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(m1.Character.ID)
		xassert.Equal(t, want, got)
	})
}
