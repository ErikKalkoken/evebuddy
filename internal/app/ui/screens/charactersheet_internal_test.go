package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// infoLinkText returns the text an info link shows, or "" when it is cleared.
func infoLinkText(t *testing.T, w *ui.InfoLink) string {
	t.Helper()
	stack := test.WidgetRenderer(w).Objects()[0].(*fyne.Container)
	link := stack.Objects[0].(*widget.Hyperlink)
	if !link.Visible() {
		return ""
	}
	return link.Text
}

func TestCharacterSheet_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newSheet := func(t *testing.T) *CharacterSheet {
		return NewCharacterSheet(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
	}
	t.Run("shows the character", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		ship, err := st.GetEveType(t.Context(), c.ShipTypeID.MustValue())
		require.NoError(t, err)
		location, err := st.GetLocation(t.Context(), c.LocationID.MustValue())
		require.NoError(t, err)
		a := newSheet(t)
		a.character.Store(c)

		a.update(t.Context())

		assert.Equal(t, c.EveCharacter.Name, infoLinkText(t, a.nameLink))
		assert.Equal(t, ship.Name, infoLinkText(t, a.shipLink))
		assert.Equal(t, location.Name, infoLinkText(t, a.locationLink))
		assert.Equal(t, c.EveCharacter.Birthday.Format(app.DateTimeFormat), a.bornLabel.Text)
	})
	t.Run("shows ship and location of the freshly loaded character", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		ship, err := st.GetEveType(t.Context(), c.ShipTypeID.MustValue())
		require.NoError(t, err)
		location, err := st.GetLocation(t.Context(), c.LocationID.MustValue())
		require.NoError(t, err)
		stale := *c
		stale.ShipTypeID = optional.New(factory.CreateEveType().ID)
		stale.LocationID = optional.New(factory.CreateEveLocationStructure().ID)
		a := newSheet(t)
		a.character.Store(&stale)

		a.update(t.Context())

		assert.Equal(t, ship.Name, infoLinkText(t, a.shipLink))
		assert.Equal(t, location.Name, infoLinkText(t, a.locationLink))
	})
	t.Run("clears all fields without character", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		a := newSheet(t)
		a.character.Store(c)
		a.update(t.Context())
		a.character.Store(nil)

		a.update(t.Context())

		assert.Equal(t, "", infoLinkText(t, a.nameLink))
		assert.Equal(t, "", infoLinkText(t, a.shipLink))
		assert.Equal(t, "", a.bornLabel.Text)
	})
}
