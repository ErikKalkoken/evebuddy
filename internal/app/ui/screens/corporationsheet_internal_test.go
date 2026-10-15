package screens

import (
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

func TestCorporationSheet_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newSheet := func(t *testing.T, isCorpMode bool) *CorporationSheet {
		return NewCorporationSheet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}), isCorpMode)
	}
	t.Run("shows the corporation in corporation mode", func(t *testing.T) {
		c := factory.CreateCorporation()
		a := newSheet(t, true)
		a.corporation.Store(c)

		a.update(t.Context())

		assert.Equal(t, c.EveCorporation.Name, infoLinkText(t, a.nameLink))
		assert.Equal(t, c.EveCorporation.Ticker, a.tickerLabel.Text)
		assert.Equal(t, humanize.Comma(c.EveCorporation.MemberCount), a.membersLabel.Text)
	})
	t.Run("shows current corporation data in corporation mode", func(t *testing.T) {
		c := factory.CreateCorporation()
		stale := *c.EveCorporation
		stale.MemberCount = c.EveCorporation.MemberCount + 1
		a := newSheet(t, true)
		a.corporation.Store(&app.Corporation{ID: c.ID, EveCorporation: &stale})

		a.update(t.Context())

		assert.Equal(t, humanize.Comma(c.EveCorporation.MemberCount), a.membersLabel.Text)
	})
	t.Run("shows the corporation and roles of the character", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		err := st.UpdateCharacterRoles(t.Context(), c.ID, set.Of(app.RoleAccountant, app.RoleAuditor))
		require.NoError(t, err)
		ec := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{ID: c.EveCharacter.Corporation.ID})
		a := newSheet(t, false)
		a.character.Store(c)

		a.update(t.Context())

		assert.Equal(t, ec.Name, infoLinkText(t, a.nameLink))
		want := slices.Sorted(slices.Values([]string{app.RoleAccountant.Display(), app.RoleAuditor.Display()}))
		assert.Equal(t, strings.Join(want, "\n"), a.rolesLabel.Text)
	})
	t.Run("clears all fields without character", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		a := newSheet(t, false)
		a.character.Store(c)
		a.update(t.Context())
		a.character.Store(nil)

		a.update(t.Context())

		assert.Equal(t, "", infoLinkText(t, a.nameLink))
		assert.Equal(t, "", a.tickerLabel.Text)
		assert.Equal(t, "", a.rolesLabel.Text)
	})
}

func TestCorporationSheet_CharacterChangedCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	c := factory.CreateCharacterFull()
	ec := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{ID: c.EveCharacter.Corporation.ID})
	stale := *c // character as stored before joining its current corporation
	staleEveCharacter := *c.EveCharacter
	staleEveCharacter.Corporation = factory.CreateEveEntityCorporation()
	stale.EveCharacter = &staleEveCharacter
	a := NewCorporationSheet(u, false)
	a.character.Store(&stale)

	u.Signals().EveUniverseSectionChanged.Emit(t.Context(), app.EveUniverseSectionUpdated{
		Section: app.SectionEveCharacters,
		Changed: set.Of(c.ID),
	})

	assert.Equal(t, ec.Name, infoLinkText(t, a.nameLink))
	assert.Equal(t, ec.ID, a.character.Load().EveCharacter.Corporation.ID)
}
