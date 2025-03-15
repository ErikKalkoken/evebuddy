package eveuniverse_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetEveAllianceCorporationsESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	s := eveuniverse.New(r, client)
	ctx := context.Background()
	t.Run("should return corporations", func(t *testing.T) {
		// given
		const allianceID = 42
		testutil.TruncateTables(db)
		factory.CreateEveEntityAlliance(app.EveEntity{ID: allianceID})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 101})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 102, Name: "Bravo"})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 103, Name: "Alpha"})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/alliances/%d/corporations/", allianceID),
			httpmock.NewJsonResponderOrPanic(200, []int32{101, 102}),
		)
		// when
		oo, err := s.GetEveAllianceCorporationsESI(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			got := slices.Collect(xiter.MapSlice(oo, func(a *app.EveEntity) int32 {
				return a.ID
			}))
			want := []int32{102, 101}
			assert.Equal(t, want, got)
		}
	})
}
