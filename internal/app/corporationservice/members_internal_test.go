package corporationservice

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestUpdateCorporationMembersESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should add new members and remove stale members", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		m1 := factory.CreateEveEntityCharacter()
		m2 := factory.CreateEveEntityCharacter()
		factory.CreateCorporationMember(storage.CorporationMemberParams{
			CorporationID: c.ID,
		})
		data := []int32{m1.ID, m2.ID}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/members/`,
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		changed, err := s.updateMembersESI(ctx, app.CorporationUpdateSectionParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			got, err := st.ListCorporationMemberIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				want := set.Of(data...)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}
