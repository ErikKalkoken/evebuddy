package storage

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

var locationFlagFromDBValue2 = map[string]app.LocationFlag{
	"":                                    app.FlagUndefined,
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
	"Unknown":                             app.FlagUnknown,
}

var locationFlagToDBValue2 = map[app.LocationFlag]string{}

var locationTypeFromDBValue2 = map[string]app.LocationType{
	"":             app.TypeUndefined,
	"station":      app.TypeStation,
	"solar_system": app.TypeSolarSystem,
	"item":         app.TypeItem,
	"other":        app.TypeOther,
}

var locationTypeToDBValue2 = map[app.LocationType]string{}

func init() {
	for k, v := range locationFlagFromDBValue2 {
		locationFlagToDBValue2[v] = k
	}
	for k, v := range locationTypeFromDBValue2 {
		locationTypeToDBValue2[v] = k
	}
}

type CreateCorporationAssetParams struct {
	CorporationID   int32
	EveTypeID       int32
	IsBlueprintCopy bool
	IsSingleton     bool
	ItemID          int64
	LocationFlag    app.LocationFlag
	LocationID      int64
	LocationType    app.LocationType
	Name            string
	Quantity        int32
}

func (st *Storage) CreateCorporationAsset(ctx context.Context, arg CreateCorporationAssetParams) error {
	if arg.CorporationID == 0 || arg.EveTypeID == 0 || arg.ItemID == 0 {
		return fmt.Errorf("CreateCorporationAsset: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateCorporationAssetParams{
		CorporationID:   int64(arg.CorporationID),
		EveTypeID:       int64(arg.EveTypeID),
		IsBlueprintCopy: arg.IsBlueprintCopy,
		IsSingleton:     arg.IsSingleton,
		ItemID:          arg.ItemID,
		LocationFlag:    locationFlagToDBValue2[arg.LocationFlag],
		LocationID:      arg.LocationID,
		LocationType:    locationTypeToDBValue2[arg.LocationType],
		Name:            arg.Name,
		Quantity:        int64(arg.Quantity),
	}
	if err := st.qRW.CreateCorporationAsset(ctx, arg2); err != nil {
		return fmt.Errorf("create corporation asset %+v, %w", arg, err)
	}
	return nil
}

func (st *Storage) DeleteCorporationAssets(ctx context.Context, corporationID int32, itemIDs set.Set[int64]) error {
	arg := queries.DeleteCorporationAssetsParams{
		CorporationID: int64(corporationID),
		ItemIds:       slices.Collect(itemIDs.All()),
	}
	return st.qRW.DeleteCorporationAssets(ctx, arg)
}

func (st *Storage) GetCorporationAsset(ctx context.Context, corporationID int32, itemID int64) (*app.CorporationAsset, error) {
	arg := queries.GetCorporationAssetParams{
		CorporationID: int64(corporationID),
		ItemID:        itemID,
	}
	r, err := st.qRO.GetCorporationAsset(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get corporation asset for corporation %d: %w", corporationID, convertGetError(err))
	}
	o := corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	return o, nil
}

func (st *Storage) CalculateCorporationAssetTotalValue(ctx context.Context, corporationID int32) (float64, error) {
	v, err := st.qRO.CalculateCorporationAssetTotalValue(ctx, int64(corporationID))
	if err != nil {
		return 0, fmt.Errorf("calculate corporation asset for corporation %d: %w", corporationID, err)
	}
	return v.Float64, nil
}

func (st *Storage) ListCorporationAssetIDs(ctx context.Context, corporationID int32) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationAssetIDs(ctx, int64(corporationID))
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list corporation asset IDs: %w", err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCorporationAssetsInShipHangar(ctx context.Context, corporationID int32, locationID int64) ([]*app.CorporationAsset, error) {
	rows, err := st.qRO.ListCorporationAssetsInShipHangar(ctx, queries.ListCorporationAssetsInShipHangarParams{
		CorporationID: int64(corporationID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: app.EveCategoryShip,
	})
	if err != nil {
		return nil, fmt.Errorf("list assets in ship hangar for corporation ID %d: %w", corporationID, err)
	}
	ii2 := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		ii2[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListCorporationAssetsInItemHangar(ctx context.Context, corporationID int32, locationID int64) ([]*app.CorporationAsset, error) {
	arg := queries.ListCorporationAssetsInItemHangarParams{
		CorporationID: int64(corporationID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: app.EveCategoryShip,
	}
	rows, err := st.qRO.ListCorporationAssetsInItemHangar(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list assets in item hangar for corporation ID %d: %w", corporationID, err)
	}
	ii2 := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		ii2[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListCorporationAssetsInLocation(ctx context.Context, corporationID int32, locationID int64) ([]*app.CorporationAsset, error) {
	arg := queries.ListCorporationAssetsInLocationParams{
		CorporationID: int64(corporationID),
		LocationID:    locationID,
	}
	rows, err := st.qRO.ListCorporationAssetsInLocation(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list assets in location for corporation ID %d: %w", corporationID, err)
	}
	ii2 := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		ii2[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

type UpdateCorporationAssetParams struct {
	CorporationID int32
	ItemID        int64
	LocationFlag  app.LocationFlag
	LocationID    int64
	LocationType  app.LocationType
	Name          string
	Quantity      int32
}

func (st *Storage) UpdateCorporationAsset(ctx context.Context, arg UpdateCorporationAssetParams) error {
	if arg.CorporationID == 0 || arg.ItemID == 0 {
		return fmt.Errorf("IDs must not be zero %+v", arg)
	}
	arg2 := queries.UpdateCorporationAssetParams{
		CorporationID: int64(arg.CorporationID),
		ItemID:        arg.ItemID,
		LocationFlag:  locationFlagToDBValue2[arg.LocationFlag],
		LocationID:    arg.LocationID,
		LocationType:  locationTypeToDBValue2[arg.LocationType],
		Name:          arg.Name,
		Quantity:      int64(arg.Quantity),
	}
	if err := st.qRW.UpdateCorporationAsset(ctx, arg2); err != nil {
		return fmt.Errorf("update corporation asset %+v, %w", arg, err)
	}
	return nil
}

func (st *Storage) ListAllCorporationAssets(ctx context.Context) ([]*app.CorporationAsset, error) {
	rows, err := st.qRO.ListAllCorporationAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all corporation assets: %w", err)
	}
	oo := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		oo[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func (st *Storage) ListCorporationAssets(ctx context.Context, corporationID int32) ([]*app.CorporationAsset, error) {
	rows, err := st.qRO.ListCorporationAssets(ctx, int64(corporationID))
	if err != nil {
		return nil, fmt.Errorf("list assets for corporation ID %d: %w", corporationID, err)
	}
	oo := make([]*app.CorporationAsset, len(rows))
	for i, r := range rows {
		oo[i] = corporationAssetFromDBModel(r.CorporationAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func corporationAssetFromDBModel(ca queries.CorporationAsset, t queries.EveType, g queries.EveGroup, c queries.EveCategory, p sql.NullFloat64) *app.CorporationAsset {
	if ca.CorporationID == 0 {
		panic("missing corporation ID")
	}
	locationFlag, found := locationFlagFromDBValue2[ca.LocationFlag]
	if !found {
		locationFlag = app.FlagUnknown
	}
	locationType, found := locationTypeFromDBValue2[ca.LocationType]
	if !found {
		locationType = app.TypeUnknown
	}
	o := &app.CorporationAsset{
		CorporationID:   int32(ca.CorporationID),
		Type:            eveTypeFromDBModel(t, g, c),
		IsBlueprintCopy: ca.IsBlueprintCopy,
		IsSingleton:     ca.IsSingleton,
		ItemID:          ca.ItemID,
		LocationFlag:    locationFlag,
		LocationID:      ca.LocationID,
		LocationType:    locationType,
		Name:            ca.Name,
		Quantity:        int(ca.Quantity),
		Price:           optional.FromNullFloat64(p),
	}
	return o
}
