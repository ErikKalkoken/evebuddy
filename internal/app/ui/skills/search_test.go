package skills

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSearch(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	a := NewSearch(u)

	t.Run("should fetch skills", func(t *testing.T) {
		// given
		cs := f.CreateCharacterSkill()

		// when
		a.update(t.Context())

		// then
		assert.Len(t, a.rows, 1)
		r := a.rows[0]
		xassert.Equal(t, r.typeName, cs.Type.Name)
		xassert.Equal(t, r.activeLevel, cs.ActiveSkillLevel)
		xassert.Equal(t, r.trainedLevel, cs.TrainedSkillLevel)
		xassert.Equal(t, r.skillPoints, cs.SkillPointsInSkill)
		xassert.Equal(t, r.characterID, cs.CharacterID)
	})
}
