package corporationservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"regexp"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CorporationService) ListAssets(ctx context.Context, corporationID int32) ([]*app.CorporationAsset, error) {
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

func (s *CorporationService) ListAllAssets(ctx context.Context) ([]*app.CorporationAsset, error) {
	return s.st.ListAllCorporationAssets(ctx)
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
	locationTypeFromESIValue = map[string]app.LocationType{
		"":             app.TypeUndefined,
		"station":      app.TypeStation,
		"solar_system": app.TypeSolarSystem,
		"item":         app.TypeItem,
		"other":        app.TypeOther,
	}
	reCustomsOffice = regexp.MustCompile(`Customs Office \((.+)\)`)
)

func (s *CorporationService) updateAssetsESI(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCorporationAssets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdAssets")
			assets, err := xgoesi.FetchPages(
				func(pageNum int) ([]esi.GetCorporationsCorporationIdAssets200Ok, *http.Response, error) {
					opts := &esi.GetCorporationsCorporationIdAssetsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.AssetsApi.GetCorporationsCorporationIdAssets(ctx, arg.CorporationID, opts)
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received corporation assets from ESI", "count", len(assets), "corporationID", arg.CorporationID)
			return assets, nil
		},
		func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
			assets := data.([]esi.GetCorporationsCorporationIdAssets200Ok)
			incomingIDs := set.Of[int64]()
			for _, ca := range assets {
				incomingIDs.Add(ca.ItemId)
			}
			typeIDs := set.Of[int32]()
			locationIDs := set.Of[int64]()
			for _, ca := range assets {
				typeIDs.Add(ca.TypeId)
				if !incomingIDs.Contains(ca.LocationId) {
					locationIDs.Add(ca.LocationId) // location IDs that are not referencing other itemIDs are locations
				}
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			currentIDs, err := s.st.ListCorporationAssetIDs(ctx, arg.CorporationID)
			if err != nil {
				return err
			}
			var updated, created int
			for _, a := range assets {
				locationFlag, found := locationFlagFromESIValue[a.LocationFlag]
				if !found {
					locationFlag = app.FlagUnknown
					slog.Warn("Unknown location flag encountered", "corporationID", arg.CorporationID, "item", a)
				}
				locationType, found := locationTypeFromESIValue[a.LocationType]
				if !found {
					locationType = app.TypeUnknown
					slog.Warn("Unknown location type encountered", "corporationID", arg.CorporationID, "item", a)
				}
				if currentIDs.Contains(a.ItemId) {
					arg := storage.UpdateCorporationAssetParams{
						CorporationID: arg.CorporationID,
						ItemID:        a.ItemId,
						LocationFlag:  locationFlag,
						LocationID:    a.LocationId,
						LocationType:  locationType,
						Quantity:      a.Quantity,
					}
					if err := s.st.UpdateCorporationAsset(ctx, arg); err != nil {
						return err
					}
					updated++
				} else {
					arg := storage.CreateCorporationAssetParams{
						CorporationID:   arg.CorporationID,
						EveTypeID:       a.TypeId,
						IsBlueprintCopy: a.IsBlueprintCopy,
						IsSingleton:     a.IsSingleton,
						ItemID:          a.ItemId,
						LocationFlag:    locationFlag,
						LocationID:      a.LocationId,
						LocationType:    locationType,
						Quantity:        a.Quantity,
					}
					if err := s.st.CreateCorporationAsset(ctx, arg); err != nil {
						return err
					}
					created++
				}
			}
			slog.Info("Stored corporation assets", "corporationID", arg.CorporationID, "created", created, "updated", updated)

			// remove obsolete assets
			if ids := set.Difference(currentIDs, incomingIDs); ids.Size() > 0 {
				if err := s.st.DeleteCorporationAssets(ctx, arg.CorporationID, ids); err != nil {
					return err
				}
				slog.Info("Deleted obsolete corporation assets", "corporationID", arg.CorporationID, "count", ids.Size())
			}

			// update names
			assets2, err := s.st.ListCorporationAssets(ctx, arg.CorporationID)
			if err != nil {
				return err
			}
			names := make(map[int64]string)
			for _, a := range assets2 {
				if a.CanHaveName() {
					names[a.ItemID] = a.Name
				}
			}
			names2, err := s.fetchAssetNamesESI(ctx, arg.CorporationID, slices.Collect(maps.Keys(names)))
			if err != nil {
				return err
			}
			slog.Debug("Received corporation asset names from ESI", "count", len(names2), "corporationID", arg.CorporationID)

			modifyAssetNames(assets2, names2)

			var changed set.Set[int64]
			for id, name := range names {
				if names2[id] != name {
					changed.Add(id)
				}
			}
			for id := range changed.All() {
				err := s.st.UpdateCorporationAssetName(ctx, storage.UpdateCorporationAssetNameParams{
					CorporationID: arg.CorporationID,
					ItemID:        id,
					Name:          names2[id],
				})
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}

// modifyAssetNames modifies the names of specific asset types.
func modifyAssetNames(assets2 []*app.CorporationAsset, names2 map[int64]string) {
	for _, a := range assets2 {
		name, ok := names2[a.ItemID]
		if !ok {
			continue
		}
		if a.Type == nil {
			continue
		}
		switch a.Type.ID {
		case app.EveTypeCustomsOffice:
			match := reCustomsOffice.FindStringSubmatch(name)
			if len(match) < 2 {
				continue
			}
			names2[a.ItemID] = match[1]
		}
	}
}

func (s *CorporationService) fetchAssetNamesESI(ctx context.Context, corporationID int32, ids []int64) (map[int64]string, error) {
	const assetNamesMaxIDs = 999
	results := make([][]esi.PostCorporationsCorporationIdAssetsNames200Ok, 0)
	if len(ids) > 0 {
		ctx = xgoesi.NewContextWithOperationID(ctx, "PostCorporationsCorporationIdAssetsNames")
		for chunk := range slices.Chunk(ids, assetNamesMaxIDs) {
			names, _, err := s.esiClient.ESI.AssetsApi.PostCorporationsCorporationIdAssetsNames(ctx, corporationID, chunk, nil)
			if err != nil {
				// We can live temporarily without asset names and will try again to fetch them next time
				// If some of the requests have succeeded we will use those names
				slog.Warn("Failed to fetch asset names", "corporationID", corporationID, "err", err)
			}
			results = append(results, names)
		}
	}
	m := make(map[int64]string)
	for _, names := range results {
		for _, n := range names {
			if n.Name != "None" {
				m[n.ItemId] = n.Name
			}
		}
	}
	return m, nil
}
