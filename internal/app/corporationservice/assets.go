package corporationservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetsync"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CorporationService) ListAssets(ctx context.Context, corporationID int64) ([]*app.CorporationAsset, error) {
	assets, err := s.st.ListCorporationAssets(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	// Filter out unwanted assets
	assets = slices.DeleteFunc(assets, func(x *app.CorporationAsset) bool {
		return x.Type != nil && x.Type.ID == app.EveTypeAlliance
	})
	return assets, nil
}

// ListAllAssets returns the assets from all corporations.
func (s *CorporationService) ListAllAssets(ctx context.Context) ([]*app.CorporationAsset, error) {
	return s.st.ListAllCorporationAssets(ctx)
}

// CalculateAssetTotalValue returns the total value of all assets for a corporation.
func (s *CorporationService) CalculateAssetTotalValue(ctx context.Context, corporationID int64) (float64, error) {
	return s.st.CalculateCorporationAssetTotalValue(ctx, corporationID)
}

var nodeCategory2Division = map[asset.NodeCategory]app.Division{
	asset.NodeOffice1: app.Division1,
	asset.NodeOffice2: app.Division2,
	asset.NodeOffice3: app.Division3,
	asset.NodeOffice4: app.Division4,
	asset.NodeOffice5: app.Division5,
	asset.NodeOffice6: app.Division6,
	asset.NodeOffice7: app.Division7,
}

// CalculateAssetValueByDivision returns the total asset value per hangar division for a corporation.
func (s *CorporationService) CalculateAssetValueByDivision(ctx context.Context, corporationID int64) (map[app.Division]float64, error) {
	assets, err := s.ListAssets(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	locations, err := s.eus.ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	tree := asset.NewFromCorporationAssets(assets, locations)
	m := make(map[app.Division]float64)
	for _, loc := range tree.Locations() {
		for n := range loc.All() {
			d, ok := nodeCategory2Division[n.Category()]
			if !ok {
				continue
			}
			m[d] += n.TotalValue()
		}
	}
	return m, nil
}

var (
	locationFlagFromESIValue = map[string]app.LocationFlag{
		"AssetSafety":                         app.FlagAssetSafety,
		"AutoFit":                             app.FlagAutoFit,
		"Bonus":                               app.FlagBonus,
		"Booster":                             app.FlagBooster,
		"BoosterBay":                          app.FlagBoosterBay,
		"Capsule":                             app.FlagCapsule,
		"CapsuleerDeliveries":                 app.FlagCapsuleerDeliveries,
		"Cargo":                               app.FlagCargo,
		"CorpDeliveries":                      app.FlagCorpDeliveries,
		"CorpSAG1":                            app.FlagCorpSAG1,
		"CorpSAG2":                            app.FlagCorpSAG2,
		"CorpSAG3":                            app.FlagCorpSAG3,
		"CorpSAG4":                            app.FlagCorpSAG4,
		"CorpSAG5":                            app.FlagCorpSAG5,
		"CorpSAG6":                            app.FlagCorpSAG6,
		"CorpSAG7":                            app.FlagCorpSAG7,
		"CorporationGoalDeliveries":           app.FlagCorporationGoalDeliveries,
		"CrateLoot":                           app.FlagCrateLoot,
		"Deliveries":                          app.FlagDeliveries,
		"DroneBay":                            app.FlagDroneBay,
		"DustBattle":                          app.FlagDustBattle,
		"DustDatabank":                        app.FlagDustDatabank,
		"ExpeditionHold":                      app.FlagExpeditionHold,
		"FighterBay":                          app.FlagFighterBay,
		"FighterTube0":                        app.FlagFighterTube0,
		"FighterTube1":                        app.FlagFighterTube1,
		"FighterTube2":                        app.FlagFighterTube2,
		"FighterTube3":                        app.FlagFighterTube3,
		"FighterTube4":                        app.FlagFighterTube4,
		"FleetHangar":                         app.FlagFleetHangar,
		"FrigateEscapeBay":                    app.FlagFrigateEscapeBay,
		"Hangar":                              app.FlagHangar,
		"HangarAll":                           app.FlagHangarAll,
		"HiSlot0":                             app.FlagHiSlot0,
		"HiSlot1":                             app.FlagHiSlot1,
		"HiSlot2":                             app.FlagHiSlot2,
		"HiSlot3":                             app.FlagHiSlot3,
		"HiSlot4":                             app.FlagHiSlot4,
		"HiSlot5":                             app.FlagHiSlot5,
		"HiSlot6":                             app.FlagHiSlot6,
		"HiSlot7":                             app.FlagHiSlot7,
		"HiddenModifiers":                     app.FlagHiddenModifiers,
		"Implant":                             app.FlagImplant,
		"Impounded":                           app.FlagImpounded,
		"InfrastructureHangar":                app.FlagInfrastructureHangar,
		"JunkyardReprocessed":                 app.FlagJunkyardReprocessed,
		"JunkyardTrashed":                     app.FlagJunkyardTrashed,
		"LoSlot0":                             app.FlagLoSlot0,
		"LoSlot1":                             app.FlagLoSlot1,
		"LoSlot2":                             app.FlagLoSlot2,
		"LoSlot3":                             app.FlagLoSlot3,
		"LoSlot4":                             app.FlagLoSlot4,
		"LoSlot5":                             app.FlagLoSlot5,
		"LoSlot6":                             app.FlagLoSlot6,
		"LoSlot7":                             app.FlagLoSlot7,
		"Locked":                              app.FlagLocked,
		"MedSlot0":                            app.FlagMedSlot0,
		"MedSlot1":                            app.FlagMedSlot1,
		"MedSlot2":                            app.FlagMedSlot2,
		"MedSlot3":                            app.FlagMedSlot3,
		"MedSlot4":                            app.FlagMedSlot4,
		"MedSlot5":                            app.FlagMedSlot5,
		"MedSlot6":                            app.FlagMedSlot6,
		"MedSlot7":                            app.FlagMedSlot7,
		"MobileDepotHold":                     app.FlagMobileDepotHold,
		"MoonMaterialBay":                     app.FlagMoonMaterialBay,
		"OfficeFolder":                        app.FlagOfficeFolder,
		"Pilot":                               app.FlagPilot,
		"PlanetSurface":                       app.FlagPlanetSurface,
		"QuafeBay":                            app.FlagQuafeBay,
		"QuantumCoreRoom":                     app.FlagQuantumCoreRoom,
		"Reward":                              app.FlagReward,
		"RigSlot0":                            app.FlagRigSlot0,
		"RigSlot1":                            app.FlagRigSlot1,
		"RigSlot2":                            app.FlagRigSlot2,
		"RigSlot3":                            app.FlagRigSlot3,
		"RigSlot4":                            app.FlagRigSlot4,
		"RigSlot5":                            app.FlagRigSlot5,
		"RigSlot6":                            app.FlagRigSlot6,
		"RigSlot7":                            app.FlagRigSlot7,
		"SecondaryStorage":                    app.FlagSecondaryStorage,
		"ServiceSlot0":                        app.FlagServiceSlot0,
		"ServiceSlot1":                        app.FlagServiceSlot1,
		"ServiceSlot2":                        app.FlagServiceSlot2,
		"ServiceSlot3":                        app.FlagServiceSlot3,
		"ServiceSlot4":                        app.FlagServiceSlot4,
		"ServiceSlot5":                        app.FlagServiceSlot5,
		"ServiceSlot6":                        app.FlagServiceSlot6,
		"ServiceSlot7":                        app.FlagServiceSlot7,
		"ShipHangar":                          app.FlagShipHangar,
		"ShipOffline":                         app.FlagShipOffline,
		"Skill":                               app.FlagSkill,
		"SkillInTraining":                     app.FlagSkillInTraining,
		"SpecializedAmmoHold":                 app.FlagSpecializedAmmoHold,
		"SpecializedAsteroidHold":             app.FlagSpecializedAsteroidHold,
		"SpecializedCommandCenterHold":        app.FlagSpecializedCommandCenterHold,
		"SpecializedFuelBay":                  app.FlagSpecializedFuelBay,
		"SpecializedGasHold":                  app.FlagSpecializedGasHold,
		"SpecializedIceHold":                  app.FlagSpecializedIceHold,
		"SpecializedIndustrialShipHold":       app.FlagSpecializedIndustrialShipHold,
		"SpecializedLargeShipHold":            app.FlagSpecializedLargeShipHold,
		"SpecializedMaterialBay":              app.FlagSpecializedMaterialBay,
		"SpecializedMediumShipHold":           app.FlagSpecializedMediumShipHold,
		"SpecializedMineralHold":              app.FlagSpecializedMineralHold,
		"SpecializedOreHold":                  app.FlagSpecializedOreHold,
		"SpecializedPlanetaryCommoditiesHold": app.FlagSpecializedPlanetaryCommoditiesHold,
		"SpecializedSalvageHold":              app.FlagSpecializedSalvageHold,
		"SpecializedShipHold":                 app.FlagSpecializedShipHold,
		"SpecializedSmallShipHold":            app.FlagSpecializedSmallShipHold,
		"StructureActive":                     app.FlagStructureActive,
		"StructureFuel":                       app.FlagStructureFuel,
		"StructureInactive":                   app.FlagStructureInactive,
		"StructureOffline":                    app.FlagStructureOffline,
		"SubSystemBay":                        app.FlagSubSystemBay,
		"SubSystemSlot0":                      app.FlagSubSystemSlot0,
		"SubSystemSlot1":                      app.FlagSubSystemSlot1,
		"SubSystemSlot2":                      app.FlagSubSystemSlot2,
		"SubSystemSlot3":                      app.FlagSubSystemSlot3,
		"SubSystemSlot4":                      app.FlagSubSystemSlot4,
		"SubSystemSlot5":                      app.FlagSubSystemSlot5,
		"SubSystemSlot6":                      app.FlagSubSystemSlot6,
		"SubSystemSlot7":                      app.FlagSubSystemSlot7,
		"Unlocked":                            app.FlagUnlocked,
		"Wallet":                              app.FlagWallet,
		"Wardrobe":                            app.FlagWardrobe,
	}
	reCustomsOffice = regexp.MustCompile(`Customs Office \((.+)\)`)
)

func (s *CorporationService) updateAssetsESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationAssets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdAssets")
			assets, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CorporationsCorporationIdAssetsGetInner, *http.Response, error) {
					return s.esiClient.AssetsAPI.GetCorporationsCorporationIdAssets(ctx, arg.corporationID).Page(page).Execute()
				})
			if err != nil {
				return false, err
			}
			slices.SortFunc(assets, func(a, b esi.CorporationsCorporationIdAssetsGetInner) int {
				return cmp.Compare(a.ItemId, b.ItemId)
			})
			slog.Debug("Received corporation assets from ESI", "count", len(assets), "corporationID", arg.corporationID)
			return assets, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			assets := xslices.Map(data.([]esi.CorporationsCorporationIdAssetsGetInner), func(x esi.CorporationsCorporationIdAssetsGetInner) assetsync.ESIAsset {
				return assetsync.ESIAsset(x)
			})
			typeIDs, locationIDs := assetsync.ExtractIDs(assets)
			g := new(errgroup.Group)
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return false, err
			}
			currentIDs, err := s.st.ListCorporationAssetIDs(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			logAttr := slog.Int64("corporationID", arg.corporationID)
			err = assetsync.Sync(
				ctx,
				assets,
				currentIDs,
				locationFlagFromESIValue,
				func(ctx context.Context, item assetsync.Item) error {
					return s.st.CreateCorporationAsset(ctx, storage.CreateCorporationAssetParams{
						CorporationID:   arg.corporationID,
						EveTypeID:       item.TypeID,
						IsBlueprintCopy: item.IsBlueprintCopy,
						IsSingleton:     item.IsSingleton,
						ItemID:          item.ItemID,
						LocationFlag:    item.LocationFlag,
						LocationID:      item.LocationID,
						LocationType:    item.LocationType,
						Quantity:        item.Quantity,
					})
				},
				func(ctx context.Context, item assetsync.Item) error {
					return s.st.UpdateCorporationAsset(ctx, storage.UpdateCorporationAssetParams{
						CorporationID: arg.corporationID,
						IsSingleton:   item.IsSingleton,
						ItemID:        item.ItemID,
						LocationFlag:  item.LocationFlag,
						LocationID:    item.LocationID,
						LocationType:  item.LocationType,
						Quantity:      item.Quantity,
					})
				},
				func(ctx context.Context, ids set.Set[int64]) error {
					return s.st.DeleteCorporationAssets(ctx, arg.corporationID, ids)
				},
				logAttr,
			)
			if err != nil {
				return false, err
			}
			stored, err := s.st.ListCorporationAssets(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			err = assetsync.SyncNames(
				ctx,
				xslices.Map(stored, func(x *app.CorporationAsset) *app.Asset {
					return &x.Asset
				}),
				func(ctx context.Context, ids []int64) ([]assetsync.ESIAssetName, error) {
					ctx = xgoesi.NewContextWithOperationID(ctx, "PostCorporationsCorporationIdAssetsNames")
					names, _, err := s.esiClient.AssetsAPI.PostCorporationsCorporationIdAssetsNames(ctx, arg.corporationID).RequestBody(ids).Execute()
					if err != nil {
						return nil, err
					}
					return xslices.Map(names, func(x esi.CharactersCharacterIdAssetsNamesPostInner) assetsync.ESIAssetName {
						return assetsync.ESIAssetName(x)
					}), nil
				},
				func(ctx context.Context, itemID int64, name string) error {
					return s.st.UpdateCorporationAssetName(ctx, storage.UpdateCorporationAssetNameParams{
						CorporationID: arg.corporationID,
						ItemID:        itemID,
						Name:          name,
					})
				},
				modifyAssetName,
				logAttr,
			)
			if err != nil {
				return false, err
			}
			return true, nil
		},
	)
}

// modifyAssetName returns the name to store for specific asset types.
func modifyAssetName(a *app.Asset, name string) string {
	if a.Type == nil {
		return name
	}
	switch a.Type.ID {
	case app.EveTypeCustomsOffice:
		match := reCustomsOffice.FindStringSubmatch(name)
		if len(match) < 2 {
			return name
		}
		return match[1]
	}
	return name
}
