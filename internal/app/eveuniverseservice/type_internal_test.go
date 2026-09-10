package eveuniverseservice

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateTypes(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	esiClient := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	s := &EVEUniverseService{st: st, esiClient: esiClient, concurrencyLimit: -1}
	ctx := context.Background()

	categoryIDRx := regexp.MustCompile(`/categories/(\d+)`)
	groups := map[int64][]int{
		25: {587},
		26: {588},
	}
	categoryGroups := map[int64][]int{
		app.EveCategoryShip:  {25},
		app.EveCategorySkill: {26},
	}
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi.evetech.net/universe/categories/\d+`,
		func(req *http.Request) (*http.Response, error) {
			m := categoryIDRx.FindStringSubmatch(req.URL.Path)
			id, _ := strconv.ParseInt(m[1], 10, 64)
			return httpmock.NewJsonResponse(200, map[string]any{
				"category_id": id,
				"groups":      categoryGroups[id],
				"name":        "Category",
				"published":   true,
			})
		},
	)
	groupIDRx := regexp.MustCompile(`/groups/(\d+)`)
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi.evetech.net/universe/groups/\d+`,
		func(req *http.Request) (*http.Response, error) {
			m := groupIDRx.FindStringSubmatch(req.URL.Path)
			id, _ := strconv.ParseInt(m[1], 10, 64)
			return httpmock.NewJsonResponse(200, map[string]any{
				"category_id": 6,
				"group_id":    id,
				"name":        "Group",
				"published":   true,
				"types":       groups[id],
			})
		},
	)
	typeIDRx := regexp.MustCompile(`/types/(\d+)`)
	httpmock.RegisterResponder(
		"GET",
		`=~^https://esi.evetech.net/universe/types/\d+`,
		func(req *http.Request) (*http.Response, error) {
			m := typeIDRx.FindStringSubmatch(req.URL.Path)
			id, _ := strconv.ParseInt(m[1], 10, 64)
			groupID := int64(25)
			if id == 588 {
				groupID = 26
			}
			return httpmock.NewJsonResponse(200, map[string]any{
				"description": "A type",
				"group_id":    groupID,
				"name":        "Type",
				"published":   true,
				"type_id":     id,
			})
		},
	)

	t.Run("should update ship and skill types and report added ones", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		added, err := s.updateTypes(ctx)
		// then
		if err != nil {
			t.Fatal(err)
		}
		xassert.Equal(t, true, added.Contains(587))
		xassert.Equal(t, true, added.Contains(588))
	})
}

func TestFormatDogmaValue(t *testing.T) {
	cases := []struct {
		name string
		args formatDogmaValueParams
		s    string
		v    int64
	}{
		{"AbsolutePercent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitAbsolutePercent}, "4%", 0},
		{"Acceleration", formatDogmaValueParams{value: 3, unitID: app.EveUnitAcceleration}, "3 m/s²", 0},
		{"AttributeID", formatDogmaValueParams{
			value:  3,
			unitID: app.EveUnitAttributeID,
			getDogmaAttribute: func(context.Context, int64) (*app.EveDogmaAttribute, error) {
				return &app.EveDogmaAttribute{
					DisplayName: optional.New("attribute"),
					IconID:      optional.New[int64](42),
				}, nil
			}},
			"attribute", 42},
		{"AttributePoints", formatDogmaValueParams{value: 42, unitID: app.EveUnitAttributePoints}, "42 points", 0},
		{"CapacitorUnits", formatDogmaValueParams{value: 123.45, unitID: app.EveUnitCapacitorUnits}, "123.5 GJ", 0},
		{"DroneBandwidth", formatDogmaValueParams{value: 10, unitID: app.EveUnitDroneBandwidth}, "10 Mbit/s", 0},
		{"Hitpoints", formatDogmaValueParams{value: 123, unitID: app.EveUnitHitpoints}, "123 HP", 0},
		{"InverseAbsolutePercent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitInverseAbsolutePercent}, "96%", 0},
		{"LengthSmall", formatDogmaValueParams{value: 123, unitID: app.EveUnitLength}, "123 m", 0},
		{"LengthBig", formatDogmaValueParams{value: 1234, unitID: app.EveUnitLength}, "1.234 km", 0},
		{"Level", formatDogmaValueParams{value: 5, unitID: app.EveUnitLevel}, "Level 5", 0},
		{"LightYear", formatDogmaValueParams{value: 1.23, unitID: app.EveUnitLightYear}, "1.2 LY", 0},
		{"Mass", formatDogmaValueParams{value: 42, unitID: app.EveUnitMass}, "42 kg", 0},
		{"MegaWatts", formatDogmaValueParams{value: 42, unitID: app.EveUnitMegaWatts}, "42 MW", 0},
		{"Millimeters", formatDogmaValueParams{value: 42, unitID: app.EveUnitMillimeters}, "42 mm", 0},
		{"Milliseconds", formatDogmaValueParams{value: 60_000, unitID: app.EveUnitMilliseconds}, "1 minute", 0},
		{"Multiplier", formatDogmaValueParams{value: 1.2345, unitID: app.EveUnitMultiplier}, "1.2345 x", 0},
		{"Percent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitPercentage}, "4%", 0},
		{"Teraflops", formatDogmaValueParams{value: 42, unitID: app.EveUnitTeraflops}, "42 tf", 0},
		{"Volume", formatDogmaValueParams{value: 10_001, unitID: app.EveUnitVolume}, "10,001 m3", 0},
		{"WarpSpeed", formatDogmaValueParams{value: 1.2345, unitID: app.EveUnitWarpSpeed}, "1.2345 AU/s", 0},
		{"TypeID", formatDogmaValueParams{value: 42, unitID: app.EveUnitTypeID, getType: func(ctx context.Context, i int64) (*app.EveType, error) {
			return &app.EveType{Name: "type", IconID: optional.New[int64](88)}, nil
		}}, "type", 88},
		{"Units", formatDogmaValueParams{value: 42, unitID: app.EveUnitUnits}, "42 units", 0},
		{"None", formatDogmaValueParams{value: 42, unitID: app.EveUnitNone}, "42", 0},
		{"Hardpoints", formatDogmaValueParams{value: 42, unitID: app.EveUnitHardpoints}, "42", 0},
		{"FittingSlots", formatDogmaValueParams{value: 42, unitID: app.EveUnitFittingSlots}, "42", 0},
		{"Slot", formatDogmaValueParams{value: 42, unitID: app.EveUnitSlot}, "42", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, v := formatDogmaValue(context.Background(), tc.args)
			xassert.Equal(t, tc.s, s)
			xassert.Equal(t, tc.v, v)
		})
	}
}
