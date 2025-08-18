package characterservice

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

func TestUpdateCharacterRolesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should update roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/roles/`,
			httpmock.NewJsonResponderOrPanic(200, map[string][]string{
				"roles": {
					"Director",
					"Station_Manager",
				},
			}),
		)
		// when
		changed, err := s.updateRolesESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterRoles,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			got, err := st.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				want := set.Of(app.RoleDirector, app.RoleStationManager)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}
