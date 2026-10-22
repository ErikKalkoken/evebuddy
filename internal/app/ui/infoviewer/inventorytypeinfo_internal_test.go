package infoviewer

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// rowTexts returns rows as "label: value" and titles as "# label".
func rowTexts(rows []typeAttributeRow) []string {
	return xslices.Map(rows, func(r typeAttributeRow) string {
		if r.isTitle {
			return "# " + r.label
		}
		return r.label + ": " + r.value
	})
}

func TestInventoryTypeInfo_calcAttributesData(t *testing.T) {
	t.Run("should show attributes of a single group without title", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeArmorHitpoints: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeArmorHitpoints,
					DisplayName: optional.New("Armor Hitpoints"),
					Unit:        app.EveUnitHitpoints,
				},
				Value: 1500,
			},
		}
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
		// then
		assert.Equal(t, []string{"Armor Hitpoints: 1500 HP"}, rowTexts(got))
	})
	t.Run("should show groups in order with titles when there are several", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeTechLevel: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeTechLevel,
					DisplayName: optional.New("Tech Level"),
					Unit:        app.EveUnitNone,
				},
				Value: 2,
			},
			app.EveDogmaAttributeArmorHitpoints: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeArmorHitpoints,
					DisplayName: optional.New("Armor Hitpoints"),
					Unit:        app.EveUnitHitpoints,
				},
				Value: 1500,
			},
		}
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
		// then
		assert.Equal(t, []string{
			"# Armor",
			"Armor Hitpoints: 1500 HP",
			"# Miscellaneous",
			"Tech Level: 2",
		}, rowTexts(got))
	})
	t.Run("should not show zero capacity and mass", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeCapacity: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeCapacity,
					DisplayName: optional.New("Capacity"),
					Unit:        app.EveUnitVolume,
				},
				Value: 0,
			},
			app.EveDogmaAttributeMass: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeMass,
					DisplayName: optional.New("Mass"),
					Unit:        app.EveUnitMass,
				},
				Value: 0,
			},
			app.EveDogmaAttributeStructureHitpoints: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeStructureHitpoints,
					DisplayName: optional.New("Structure Hitpoints"),
					Unit:        app.EveUnitHitpoints,
				},
				Value: 300,
			},
		}
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
		// then
		assert.Equal(t, []string{"Structure Hitpoints: 300 HP"}, rowTexts(got))
	})
	t.Run("should show drone attributes when type has drone capacity", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeDroneCapacity: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeDroneCapacity,
					DisplayName: optional.New("Drone Capacity"),
					Unit:        app.EveUnitVolume,
				},
				Value: 25,
			},
			app.EveDogmaAttributeDroneBandwidth: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeDroneBandwidth,
					DisplayName: optional.New("Drone Bandwidth"),
					Unit:        app.EveUnitDroneBandwidth,
				},
				Value: 10,
			},
		}
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
		// then
		assert.Equal(t, []string{
			"Drone Capacity: 25 m3",
			"Drone Bandwidth: 10 Mbit/s",
		}, rowTexts(got))
	})
	t.Run("should not show drone attributes when type has no drone capacity", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeDroneCapacity: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeDroneCapacity,
					DisplayName: optional.New("Drone Capacity"),
					Unit:        app.EveUnitVolume,
				},
				Value: 0,
			},
			app.EveDogmaAttributeDroneBandwidth: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeDroneBandwidth,
					DisplayName: optional.New("Drone Bandwidth"),
					Unit:        app.EveUnitDroneBandwidth,
				},
				Value: 10,
			},
		}
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
		// then
		assert.Empty(t, got)
	})
	t.Run("should show jump drive attributes only with onboard jump drive", func(t *testing.T) {
		cases := []struct {
			name      string
			jumpDrive float64
			want      []string
		}{
			{"with jump drive", 1, []string{"Maximum Jump Range: 5.0 LY"}},
			{"without jump drive", 0, []string{}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				db, st, _ := testutil.NewDBOnDisk(t)
				t.Cleanup(func() { db.Close() })
				u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
				a := newInventoryTypeInfo(New(u), 42, 0)
				attributes := map[int64]*app.EveTypeDogmaAttribute{
					app.EveDogmaAttributeOnboardJumpDrive: {
						DogmaAttribute: &app.EveDogmaAttribute{
							ID:   app.EveDogmaAttributeOnboardJumpDrive,
							Unit: app.EveUnitNone,
						},
						Value: tc.jumpDrive,
					},
					app.EveDogmaAttributeMaximumJumpRange: {
						DogmaAttribute: &app.EveDogmaAttribute{
							ID:          app.EveDogmaAttributeMaximumJumpRange,
							DisplayName: optional.New("Maximum Jump Range"),
							Unit:        app.EveUnitLightYear,
						},
						Value: 5,
					},
				}
				// when
				got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
				// then
				assert.Equal(t, tc.want, rowTexts(got))
			})
		}
	})
	t.Run("should show warp speed with multiplier applied", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeShipWarpSpeed: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeShipWarpSpeed,
					DisplayName: optional.New("Warp Speed"),
					Unit:        app.EveUnitWarpSpeed,
				},
				Value: 1.5,
			},
			app.EveDogmaAttributeWarpSpeedMultiplier: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:   app.EveDogmaAttributeWarpSpeedMultiplier,
					Unit: app.EveUnitMultiplier,
				},
				Value: 3,
			},
		}
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, attributes)
		// then
		assert.Equal(t, []string{"Warp Speed: 4.5 AU/s"}, rowTexts(got))
	})
	t.Run("should show volume first in structure group", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeStructureHitpoints: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeStructureHitpoints,
					DisplayName: optional.New("Structure Hitpoints"),
					Unit:        app.EveUnitHitpoints,
				},
				Value: 300,
			},
		}
		et := &app.EveType{
			ID:             42,
			Volume:         optional.New(28_600.0),
			PackagedVolume: optional.New(2_500.0),
		}
		// when
		got := a.calcAttributesData(t.Context(), et, attributes)
		// then
		assert.Equal(t, []string{
			"Volume: 28,600 m3 (2,500 m3 Packaged)",
			"Structure Hitpoints: 300 HP",
		}, rowTexts(got))
	})
	t.Run("should show volume in miscellaneous group when there is no structure group", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeArmorHitpoints: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeArmorHitpoints,
					DisplayName: optional.New("Armor Hitpoints"),
					Unit:        app.EveUnitHitpoints,
				},
				Value: 1500,
			},
		}
		et := &app.EveType{
			ID:             42,
			Volume:         optional.New(5.0),
			PackagedVolume: optional.New(5.0),
		}
		// when
		got := a.calcAttributesData(t.Context(), et, attributes)
		// then
		assert.Equal(t, []string{
			"# Armor",
			"Armor Hitpoints: 1500 HP",
			"# Miscellaneous",
			"Volume: 5 m3",
		}, rowTexts(got))
	})
	t.Run("should show EVE ID in developer mode", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsDeveloperMode: true, Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		// when
		got := a.calcAttributesData(t.Context(), &app.EveType{ID: 42}, nil)
		// then
		assert.Equal(t, []string{"# Developer Mode", "EVE ID: 42"}, rowTexts(got))
	})
}

func TestInventoryTypeInfo_calcFittingData(t *testing.T) {
	t.Run("should show fitting attributes in order", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributeHighSlots: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeHighSlots,
					DisplayName: optional.New("High Slots"),
					Unit:        app.EveUnitFittingSlots,
				},
				Value: 4,
			},
			app.EveDogmaAttributeCPUOutput: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeCPUOutput,
					DisplayName: optional.New("CPU Output"),
					Unit:        app.EveUnitTeraflops,
				},
				Value: 150,
			},
			app.EveDogmaAttributeArmorHitpoints: {
				DogmaAttribute: &app.EveDogmaAttribute{
					ID:          app.EveDogmaAttributeArmorHitpoints,
					DisplayName: optional.New("Armor Hitpoints"),
					Unit:        app.EveUnitHitpoints,
				},
				Value: 1500,
			},
		}
		// when
		got := a.calcFittingData(t.Context(), attributes)
		// then
		assert.Equal(t, []string{"CPU Output: 150 tf", "High Slots: 4"}, rowTexts(got))
	})
}

func TestInventoryTypeInfo_calcRequiredSkills(t *testing.T) {
	t.Run("should return required skills with the character's levels", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateCharacter()
		skill1 := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Gallente Frigate"})
		skill2 := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Drones"})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c.ID,
			TypeID:            skill1.ID,
			ActiveSkillLevel:  2,
			TrainedSkillLevel: 3,
		})
		a := newInventoryTypeInfo(New(u), 42, c.ID)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributePrimarySkillID:      {Value: float64(skill1.ID)},
			app.EveDogmaAttributePrimarySkillLevel:   {Value: 3},
			app.EveDogmaAttributeSecondarySkillID:    {Value: float64(skill2.ID)},
			app.EveDogmaAttributeSecondarySkillLevel: {Value: 1},
		}
		// when
		got, err := a.calcRequiredSkills(t.Context(), c.ID, attributes)
		// then
		require.NoError(t, err)
		assert.Equal(t, []requiredSkill{
			{rank: 1, name: "Gallente Frigate", typeID: skill1.ID, activeLevel: 2, requiredLevel: 3, trainedLevel: 3},
			{rank: 2, name: "Drones", typeID: skill2.ID, requiredLevel: 1},
		}, got)
	})
	t.Run("should ignore skills without a required level", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateCharacter()
		skill := factory.CreateEveType()
		a := newInventoryTypeInfo(New(u), 42, c.ID)
		attributes := map[int64]*app.EveTypeDogmaAttribute{
			app.EveDogmaAttributePrimarySkillID: {Value: float64(skill.ID)},
		}
		// when
		got, err := a.calcRequiredSkills(t.Context(), c.ID, attributes)
		// then
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestInventoryTypeInfo_fetchMarketItems(t *testing.T) {
	t.Run("should show average price and Janice error", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       42,
			AveragePrice: optional.New(1234.5),
		})
		a := newInventoryTypeInfo(New(u), 42, 0)
		// when
		got := a.fetchMarketItems(t.Context(), 42)
		// then
		assert.Equal(t, []attributeItem{
			newAttributeItem("Average price", "1,234.50 ISK"),
			newAttributeItem("Janice prices", "ERROR: fetch prices from janice: missing API key"),
		}, got)
	})
	t.Run("should show unknown average price", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		// when
		got := a.fetchMarketItems(t.Context(), 42)
		// then
		require.NotEmpty(t, got)
		assert.Equal(t, newAttributeItem("Average price", "?"), got[0])
	})
	t.Run("should show Janice prices", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Activate()
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder("GET", "https://janice.e-351.com/api/rest/v2/pricer/42",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"buyVolume":  1000,
				"sellVolume": 2000,
				"immediatePrices": map[string]any{
					"buyPrice":  90.5,
					"sellPrice": 100.25,
				},
			}))
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), JaniceAPIKey: "DUMMY", Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		// when
		got := a.fetchMarketItems(t.Context(), 42)
		// then
		assert.Equal(t, []attributeItem{
			newAttributeItem("Average price", "?"),
			newAttributeItem("Jita sell price", "100.25 ISK"),
			newAttributeItem("Jita buy price", "90.50 ISK"),
			newAttributeItem("Jita sell volume", "2,000"),
			newAttributeItem("Jita buy volume", "1,000"),
		}, got)
	})
	t.Run("should return nothing when canceled", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		// when
		got := a.fetchMarketItems(ctx, 42)
		// then
		assert.Nil(t, got)
	})
}

func TestInventoryTypeInfo_update(t *testing.T) {
	t.Run("should show type without attributes", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		g := factory.CreateEveGroup(storage.CreateEveGroupParams{Name: "Mineral"})
		// created directly, because the factory always adds a volume
		err := st.CreateEveType(t.Context(), storage.CreateEveTypeParams{
			ID:          34,
			GroupID:     g.ID,
			Name:        "Tritanium",
			Description: "The main building block",
		})
		require.NoError(t, err)
		a := newInventoryTypeInfo(New(u), 34, 0)
		var title string
		a.setTitle = func(s string) {
			title = s
		}
		// when
		err = a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "Tritanium", a.name.Text)
		assert.Equal(t, "Mineral", title)
		assert.Equal(t, "The main building block", a.description.Text)
		assert.Equal(t, []string{"Description"}, tabTitles(a))
		assert.False(t, a.eveMarketBrowser.Visible())
		assert.False(t, a.janice.Visible())
		assert.False(t, a.characterName.Visible())
	})
	t.Run("should show name when description is empty", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		g := factory.CreateEveGroup()
		// created directly, because the factory always adds a description
		err := st.CreateEveType(t.Context(), storage.CreateEveTypeParams{ID: 34, GroupID: g.ID, Name: "Tritanium"})
		require.NoError(t, err)
		a := newInventoryTypeInfo(New(u), 34, 0)
		a.setTitle = func(_ string) {}
		// when
		err = a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "Tritanium", a.description.Text)
	})
	t.Run("should show market for tradeable type", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		var jobs []func(context.Context)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(job func(context.Context)) { jobs = append(jobs, job) },
			Storage: st,
		})
		et := factory.CreateEveType(storage.CreateEveTypeParams{MarketGroupID: optional.New[int64](4)})
		a := newInventoryTypeInfo(New(u), et.ID, 0)
		a.setTitle = func(_ string) {}
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"Description", "Attributes", "Market"}, tabTitles(a))
		assert.Equal(t, "Description", a.tabs.Selected().Text)
		assert.True(t, a.eveMarketBrowser.Visible())
		assert.True(t, a.janice.Visible())
		assert.Len(t, jobs, 1)
	})
	t.Run("should select market tab when preferred", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		u.Settings().SetPreferMarketTab(true)
		et := factory.CreateEveType(storage.CreateEveTypeParams{MarketGroupID: optional.New[int64](4)})
		a := newInventoryTypeInfo(New(u), et.ID, 0)
		a.setTitle = func(_ string) {}
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, "Market", a.tabs.Selected().Text)
	})
	t.Run("should select attributes tab for ships", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategoryShip})
		g := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategoryShip})
		et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: g.ID})
		da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID:     app.EveDogmaAttributeArmorHitpoints,
			UnitID: app.EveUnitHitpoints,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        et.ID,
			DogmaAttributeID: da.ID,
			Value:            1500,
		})
		da2 := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID:     app.EveDogmaAttributeHighSlots,
			UnitID: app.EveUnitFittingSlots,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        et.ID,
			DogmaAttributeID: da2.ID,
			Value:            4,
		})
		a := newInventoryTypeInfo(New(u), et.ID, 0)
		a.setTitle = func(_ string) {}
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"Description", "Attributes", "Fittings"}, tabTitles(a))
		assert.Equal(t, "Attributes", a.tabs.Selected().Text)
	})
	t.Run("should show character and requirements for skills", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		c := factory.CreateCharacter()
		factory.GetOrCreateEveEntity(app.EveEntity{ID: c.ID, Name: c.EveCharacter.Name, Category: app.EveEntityCharacter})
		factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		g := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: app.EveCategorySkill})
		requiredSkill := factory.CreateEveType()
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:       c.ID,
			TypeID:            requiredSkill.ID,
			ActiveSkillLevel:  5,
			TrainedSkillLevel: 5,
		})
		et := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: g.ID})
		for id, v := range map[int64]float64{
			app.EveDogmaAttributePrimarySkillID:    float64(requiredSkill.ID),
			app.EveDogmaAttributePrimarySkillLevel: 3,
		} {
			da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: id})
			factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
				EveTypeID:        et.ID,
				DogmaAttributeID: da.ID,
				Value:            v,
			})
		}
		a := newInventoryTypeInfo(New(u), et.ID, c.ID)
		a.setTitle = func(_ string) {}
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, c.EveCharacter.Name, a.characterName.Text)
		assert.True(t, a.characterName.Visible())
		assert.True(t, a.checkIcon.Visible())
		assert.Equal(t, boolIconResource(true).Name(), a.checkIcon.Resource.Name())
		assert.Equal(t, "Requirements", a.tabs.Selected().Text)
	})
	t.Run("should show missing skills", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Go:      func(_ func(context.Context)) {},
			Storage: st,
		})
		c := factory.CreateCharacter()
		factory.GetOrCreateEveEntity(app.EveEntity{ID: c.ID, Name: c.EveCharacter.Name, Category: app.EveEntityCharacter})
		requiredSkill := factory.CreateEveType()
		et := factory.CreateEveType()
		for id, v := range map[int64]float64{
			app.EveDogmaAttributePrimarySkillID:    float64(requiredSkill.ID),
			app.EveDogmaAttributePrimarySkillLevel: 3,
		} {
			da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: id})
			factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
				EveTypeID:        et.ID,
				DogmaAttributeID: da.ID,
				Value:            v,
			})
		}
		a := newInventoryTypeInfo(New(u), et.ID, c.ID)
		a.setTitle = func(_ string) {}
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.True(t, a.checkIcon.Visible())
		assert.Equal(t, boolIconResource(false).Name(), a.checkIcon.Resource.Name())
	})
}

func tabTitles(a *inventoryTypeInfo) []string {
	return xslices.Map(a.tabs.Items, func(x *container.TabItem) string {
		return x.Text
	})
}

func TestInventoryTypeInfo_makeRequirementsTab(t *testing.T) {
	t.Run("should not create tab without required skills", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		// when
		got := a.makeRequirementsTab(nil)
		// then
		assert.Nil(t, got)
	})
	t.Run("should show status of required skills", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := newInventoryTypeInfo(New(u), 42, 0)
		tab := a.makeRequirementsTab([]requiredSkill{
			{rank: 1, name: "Drones", typeID: 3436, requiredLevel: 1},
		})
		require.NotNil(t, tab)
		w := test.NewWindow(tab.Content)
		defer w.Close()
		// when
		w.Resize(fyne.NewSize(400, 300))
		// then
		assert.Contains(t, labelTexts(tab.Content), "Skill not injected")
	})
	t.Run("should show skill when selected", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			Go:       func(_ func(context.Context)) {},
			IsOnline: true,
			Storage:  st,
		})
		iw := New(u)
		a := newInventoryTypeInfo(iw, 42, 0)
		tab := a.makeRequirementsTab([]requiredSkill{
			{rank: 1, name: "Drones", typeID: 3436, requiredLevel: 1},
		})
		require.NotNil(t, tab)
		var list *widget.List
		for _, o := range test.LaidOutObjects(tab.Content) {
			if x, ok := o.(*widget.List); ok {
				list = x
			}
		}
		require.NotNil(t, list)
		// when
		list.Select(0)
		// then
		assert.Equal(t, &showParams{variant: Type, entityID: 3436}, iw.current)
	})
}
