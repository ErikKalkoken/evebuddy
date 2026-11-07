package characterservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetsync"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CharacterService) AssetTotalValue(ctx context.Context, characterID int64) (optional.Optional[float64], error) {
	return s.st.GetCharacterAssetValue(ctx, characterID)
}

func (s *CharacterService) ListAssets(ctx context.Context, characterID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssets(ctx, characterID)
}

func (s *CharacterService) ListAllAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	return s.st.ListAllCharacterAssets(ctx)
}

var locationFlagFromESIValue = map[string]app.LocationFlag{
	"AssetSafety":                         app.FlagAssetSafety,
	"AutoFit":                             app.FlagAutoFit,
	"BoosterBay":                          app.FlagBoosterBay,
	"CapsuleerDeliveries":                 app.FlagCapsuleerDeliveries,
	"Cargo":                               app.FlagCargo,
	"CorporationGoalDeliveries":           app.FlagCorporationGoalDeliveries,
	"CorpseBay":                           app.FlagCorpseBay,
	"Deliveries":                          app.FlagDeliveries,
	"DroneBay":                            app.FlagDroneBay,
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

var locationTypeFromESIValue = map[string]app.LocationType{
	"":             app.TypeUndefined,
	"station":      app.TypeStation,
	"solar_system": app.TypeSolarSystem,
	"item":         app.TypeItem,
	"other":        app.TypeOther,
}

func (s *CharacterService) updateAssetsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterAssets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	changed, err := s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdAssets")
			assets, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CharactersCharacterIdAssetsGetInner, *http.Response, error) {
					return s.esiClient.AssetsAPI.GetCharactersCharacterIdAssets(ctx, characterID).Page(page).Execute()
				})
			if err != nil {
				return false, err
			}
			slices.SortFunc(assets, func(a, b esi.CharactersCharacterIdAssetsGetInner) int {
				return cmp.Compare(a.ItemId, b.ItemId)
			})
			slog.Debug("Received assets from ESI", "count", len(assets), "characterID", characterID)
			return assets, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			assets := xslices.Map(data.([]esi.CharactersCharacterIdAssetsGetInner), func(x esi.CharactersCharacterIdAssetsGetInner) assetsync.ESIAsset {
				return assetsync.ESIAsset(x)
			})
			typeIDs, locationIDs := assetsync.ExtractIDs(assets)
			if err := s.eus.AddMissing(ctx, set.Set[int64]{}, locationIDs, typeIDs); err != nil {
				return false, err
			}
			currentIDs, err := s.st.ListCharacterAssetIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			logAttr := slog.Int64("characterID", characterID)
			err = assetsync.Sync(
				ctx,
				assets,
				currentIDs,
				locationFlagFromESIValue,
				locationTypeFromESIValue,
				func(ctx context.Context, item assetsync.Item) error {
					return s.st.CreateCharacterAsset(ctx, storage.CreateCharacterAssetParams{
						CharacterID:     characterID,
						TypeID:          item.TypeID,
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
					return s.st.UpdateCharacterAsset(ctx, storage.UpdateCharacterAssetParams{
						CharacterID:  characterID,
						IsSingleton:  item.IsSingleton,
						ItemID:       item.ItemID,
						LocationFlag: item.LocationFlag,
						LocationID:   item.LocationID,
						LocationType: item.LocationType,
						Quantity:     item.Quantity,
					})
				},
				func(ctx context.Context, ids set.Set[int64]) error {
					return s.st.DeleteCharacterAssets(ctx, characterID, ids)
				},
				logAttr,
			)
			if err != nil {
				return false, err
			}
			stored, err := s.st.ListCharacterAssets(ctx, characterID)
			if err != nil {
				return false, err
			}
			err = assetsync.SyncNames(
				ctx,
				xslices.Map(stored, func(x *app.CharacterAsset) *app.Asset {
					return &x.Asset
				}),
				func(ctx context.Context, ids []int64) ([]assetsync.ESIAssetName, error) {
					ctx = xgoesi.NewContextWithOperationID(ctx, "PostCharactersCharacterIdAssetsNames")
					names, _, err := s.esiClient.AssetsAPI.PostCharactersCharacterIdAssetsNames(ctx, characterID).RequestBody(ids).Execute()
					if err != nil {
						return nil, err
					}
					return xslices.Map(names, func(x esi.CharactersCharacterIdAssetsNamesPostInner) assetsync.ESIAssetName {
						return assetsync.ESIAssetName(x)
					}), nil
				},
				func(ctx context.Context, itemID int64, name string) error {
					return s.st.UpdateCharacterAssetName(ctx, storage.UpdateCharacterAssetNameParams{
						CharacterID: characterID,
						ItemID:      itemID,
						Name:        name,
					})
				},
				nil,
				logAttr,
			)
			if err != nil {
				return false, err
			}
			return true, nil
		},
	)
	if err != nil {
		return false, err
	}

	// update calculated values
	c, err := s.st.GetCharacter(ctx, arg.characterID)
	if err != nil {
		return false, err
	}
	if arg.forceUpdate || changed || c.AssetValue.IsEmpty() {
		if err := s.updateAssetValue(ctx, arg.characterID); err != nil {
			return false, err
		}
	}
	return changed, nil
}

func (s *CharacterService) updateAssetValue(ctx context.Context, characterID int64) error {
	v, err := s.st.CalculateCharacterAssetTotalValue(ctx, characterID)
	if err != nil {
		return err
	}
	if err := s.st.UpdateCharacterAssetValue(ctx, characterID, optional.New(v)); err != nil {
		return err
	}
	return nil
}
