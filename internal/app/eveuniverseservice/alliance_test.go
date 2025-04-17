package eveuniverseservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestGetEveAllianceCorporationsESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, ""),
	})
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
			httpmock.NewJsonResponderOrPanic(200, []int32{102, 103}),
		)
		// when
		oo, err := s.GetAllianceCorporationsESI(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			got := xslices.Map(oo, func(a *app.EveEntity) int32 {
				return a.ID
			})
			want := []int32{103, 102}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty list when there are no corporations", func(t *testing.T) {
		// given
		const allianceID = 42
		testutil.TruncateTables(db)
		factory.CreateEveEntityAlliance(app.EveEntity{ID: allianceID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/alliances/%d/corporations/", allianceID),
			httpmock.NewJsonResponderOrPanic(200, []int32{}),
		)
		// when
		oo, err := s.GetAllianceCorporationsESI(ctx, allianceID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 0)
		}
	})
}
