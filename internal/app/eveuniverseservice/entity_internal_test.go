package eveuniverseservice

import (
	"errors"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateEntityNameIfExists(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := &EVEUniverseService{st: st}
	t.Run("should update name when entity exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		e := factory.CreateEveEntity()
		// when
		err := s.updateEntityNameIfExists(t.Context(), e.ID, "New Name")
		// then
		if err != nil {
			t.Fatal(err)
		}
		o, err := st.GetEveEntity(t.Context(), e.ID)
		if err != nil {
			t.Fatal(err)
		}
		xassert.Equal(t, "New Name", o.Name)
	})
	t.Run("should do nothing when entity does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := s.updateEntityNameIfExists(t.Context(), 666, "New Name")
		// then
		if err != nil {
			t.Fatal(err)
		}
		_, err = st.GetEveEntity(t.Context(), 666)
		if !errors.Is(err, app.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}
