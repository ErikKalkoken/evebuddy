package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterRolesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create roles from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/roles", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string][]string{
				"roles": {
					"Director",
					"Station_Manager",
				},
			}),
		)
		// when
		changed, err := s.updateRolesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterRoles,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterRoles(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(app.RoleDirector, app.RoleStationManager)
		xassert.Equal(t, want, got)
	})
	t.Run("should update existing roles", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.SetCharacterRoles(c.ID, set.Of(app.RoleDirector, app.RoleAuditor))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/roles", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string][]string{
				"roles": {
					"Director",
					"Station_Manager",
				},
			}),
		)
		// when
		changed, err := s.updateRolesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterRoles,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterRoles(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(app.RoleDirector, app.RoleStationManager)
		xassert.Equal(t, want, got)
	})
	t.Run("should do nothing when roles did not change", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.SetCharacterRoles(c.ID, set.Of(app.RoleDirector, app.RoleStationManager))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/roles", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string][]string{
				"roles": {
					"Director",
					"Station_Manager",
				},
			}),
		)
		// when
		changed, err := s.updateRolesESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterRoles,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.ListCharacterRoles(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(app.RoleDirector, app.RoleStationManager)
		xassert.Equal(t, want, got)
	})
}
