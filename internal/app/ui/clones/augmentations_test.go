package clones

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestAugmentations_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("should include characters with and without implants", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c1.ID})
		c2 := factory.CreateCharacter()
		a := NewAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))

		// when
		a.Update(t.Context())

		// then
		got := set.Collect(xiter.MapSlice(a.treeData.Children(nil), func(x *augmentationNode) int64 {
			return x.characterID
		}))
		want := set.Of(c1.ID, c2.ID)
		assert.Equal(t, want, got)
	})
}
