package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) AssetTotalValue(ctx context.Context, characterID int32) (optional.Optional[float64], error) {
	return s.st.GetCharacterAssetValue(ctx, characterID)
}

func (s *CharacterService) ListAssets(ctx context.Context, characterID int32) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssets(ctx, characterID)
}

func (s *CharacterService) ListAllAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	return s.st.ListAllCharacterAssets(ctx)
}

var (
	locationFlagFromESIValue = map[string]app.LocationFlag{
		"AssetSafety":                         app.FlagAssetSafety,
		"AutoFit":                             app.FlagAutoFit,
		"BoosterBay":                          app.FlagBoosterBay,
		"CapsuleerDeliveries":                 app.FlagCapsuleerDeliveries,
		"Cargo":                               app.FlagCargo,
		"CorporationGoalDeliveries":           app.FlagCorporationGoalDeliveries,
		"CorpseBay":                           app.FlagCorpseBay,
		"Deliveries":                          app.FlagDeliveries,
		"DroneBay":                            app.FlagDroneBay,
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
		"InfrastructureHangar":                app.FlagInfrastructureHangar,
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
		"QuafeBay":                            app.FlagQuafeBay,
		"RigSlot0":                            app.FlagRigSlot0,
		"RigSlot1":                            app.FlagRigSlot1,
		"RigSlot2":                            app.FlagRigSlot2,
		"RigSlot3":                            app.FlagRigSlot3,
		"RigSlot4":                            app.FlagRigSlot4,
		"RigSlot5":                            app.FlagRigSlot5,
		"RigSlot6":                            app.FlagRigSlot6,
		"RigSlot7":                            app.FlagRigSlot7,
		"ShipHangar":                          app.FlagShipHangar,
		"Skill":                               app.FlagSkill,
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
		"StructureDeedBay":                    app.FlagStructureDeedBay,
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
		"Wardrobe":                            app.FlagWardrobe,
	}
	locationTypeFromESIValue = map[string]app.LocationType{
		"":             app.TypeUndefined,
		"station":      app.TypeStation,
		"solar_system": app.TypeSolarSystem,
		"item":         app.TypeItem,
		"other":        app.TypeOther,
	}
)

func (s *CharacterService) updateAssetsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterAssets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdAssets")
			assets, err := xgoesi.FetchPages(
				func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdAssetsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received assets from ESI", "count", len(assets), "characterID", characterID)
			return assets, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			assets := data.([]esi.GetCharactersCharacterIdAssets200Ok)
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
			currentIDs, err := s.st.ListCharacterAssetIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var updated, created int
			for _, a := range assets {
				locationFlag, found := locationFlagFromESIValue[a.LocationFlag]
				if !found {
					locationFlag = app.FlagUnknown
					slog.Warn("Unknown location flag encountered", "characterID", characterID, "item", a)
				}
				locationType, found := locationTypeFromESIValue[a.LocationType]
				if !found {
					locationType = app.TypeUnknown
					slog.Warn("Unknown location type encountered", "characterID", characterID, "item", a)
				}
				if currentIDs.Contains(a.ItemId) {
					arg := storage.UpdateCharacterAssetParams{
						CharacterID:  characterID,
						ItemID:       a.ItemId,
						LocationFlag: locationFlag,
						LocationID:   a.LocationId,
						LocationType: locationType,
						Quantity:     a.Quantity,
					}
					if err := s.st.UpdateCharacterAsset(ctx, arg); err != nil {
						return err
					}
					updated++
				} else {
					arg := storage.CreateCharacterAssetParams{
						CharacterID:     characterID,
						EveTypeID:       a.TypeId,
						IsBlueprintCopy: a.IsBlueprintCopy,
						IsSingleton:     a.IsSingleton,
						ItemID:          a.ItemId,
						LocationFlag:    locationFlag,
						LocationID:      a.LocationId,
						LocationType:    locationType,
						Quantity:        a.Quantity,
					}
					if err := s.st.CreateCharacterAsset(ctx, arg); err != nil {
						return err
					}
					created++
				}
			}
			slog.Info("Stored character assets", "characterID", characterID, "created", created, "updated", updated)

			// Remove obsolete assets
			if ids := set.Difference(currentIDs, incomingIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterAssets(ctx, characterID, ids); err != nil {
					return err
				}
				slog.Info("Deleted obsolete character assets", "characterID", characterID, "count", ids.Size())
			}
			if _, err := s.UpdateAssetTotalValue(ctx, characterID); err != nil {
				return err
			}

			// update names
			assets2, err := s.st.ListCharacterAssets(ctx, characterID)
			if err != nil {
				return err
			}
			names := make(map[int64]string)
			for _, a := range assets2 {
				if a.CanHaveName() {
					names[a.ItemID] = a.Name
				}
			}
			names2, err := s.fetchAssetNamesESI(ctx, characterID, slices.Collect(maps.Keys(names)))
			if err != nil {
				return err
			}
			slog.Debug("Received character asset names from ESI", "count", len(names2), "characterID", characterID)
			var changed set.Set[int64]
			for id, name := range names {
				if names2[id] != name {
					changed.Add(id)
				}
			}
			for id := range changed.All() {
				err := s.st.UpdateCharacterAssetName(ctx, storage.UpdateCharacterAssetNameParams{
					CharacterID: characterID,
					ItemID:      id,
					Name:        names2[id],
				})
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func (s *CharacterService) fetchAssetNamesESI(ctx context.Context, characterID int32, ids []int64) (map[int64]string, error) {
	const assetNamesMaxIDs = 999
	results := make([][]esi.PostCharactersCharacterIdAssetsNames200Ok, 0)
	if len(ids) > 0 {
		ctx = xgoesi.NewContextWithOperationID(ctx, "PostCharactersCharacterIdAssetsNames")
		for chunk := range slices.Chunk(ids, assetNamesMaxIDs) {
			names, _, err := s.esiClient.ESI.AssetsApi.PostCharactersCharacterIdAssetsNames(ctx, characterID, chunk, nil)
			if err != nil {
				// We can live temporarily without asset names and will try again to fetch them next time
				// If some of the requests have succeeded we will use those names
				slog.Warn("Failed to fetch asset names", "characterID", characterID, "err", err)
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

func (s *CharacterService) UpdateAssetTotalValue(ctx context.Context, characterID int32) (float64, error) {
	v, err := s.st.CalculateCharacterAssetTotalValue(ctx, characterID)
	if err != nil {
		return 0, err
	}
	if err := s.st.UpdateCharacterAssetValue(ctx, characterID, optional.New(v)); err != nil {
		return 0, err
	}
	return v, nil
}
