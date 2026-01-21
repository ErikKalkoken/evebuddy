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

var locationFlagFromDBValue = map[string]app.LocationFlag{
	"":                                    app.FlagUndefined,
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
	"Unknown":                             app.FlagUnknown,
}

var locationFlagToDBValue = map[app.LocationFlag]string{}

var locationTypeFromDBValue = map[string]app.LocationType{
	"":             app.TypeUndefined,
	"station":      app.TypeStation,
	"solar_system": app.TypeSolarSystem,
	"item":         app.TypeItem,
	"other":        app.TypeOther,
}

var locationTypeToDBValue = map[app.LocationType]string{}

func init() {
	for k, v := range locationFlagFromDBValue {
		locationFlagToDBValue[v] = k
	}
	for k, v := range locationTypeFromDBValue {
		locationTypeToDBValue[v] = k
	}
}

type CreateCharacterAssetParams struct {
	CharacterID     int32
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

func (st *Storage) CreateCharacterAsset(ctx context.Context, arg CreateCharacterAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterAsset: %+v: %w", arg, err)

	}
	if arg.CharacterID == 0 || arg.EveTypeID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.CreateCharacterAsset(ctx, queries.CreateCharacterAssetParams{
		CharacterID:     int64(arg.CharacterID),
		EveTypeID:       int64(arg.EveTypeID),
		IsBlueprintCopy: arg.IsBlueprintCopy,
		IsSingleton:     arg.IsSingleton,
		ItemID:          arg.ItemID,
		LocationFlag:    locationFlagToDBValue[arg.LocationFlag],
		LocationID:      arg.LocationID,
		LocationType:    locationTypeToDBValue[arg.LocationType],
		Name:            arg.Name,
		Quantity:        int64(arg.Quantity),
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCharacterAssets(ctx context.Context, characterID int32, itemIDs set.Set[int64]) error {
	return st.qRW.DeleteCharacterAssets(ctx, queries.DeleteCharacterAssetsParams{
		CharacterID: int64(characterID),
		ItemIds:     slices.Collect(itemIDs.All()),
	})
}

func (st *Storage) GetCharacterAsset(ctx context.Context, characterID int32, itemID int64) (*app.CharacterAsset, error) {
	r, err := st.qRO.GetCharacterAsset(ctx, queries.GetCharacterAssetParams{
		CharacterID: int64(characterID),
		ItemID:      itemID,
	})
	if err != nil {
		return nil, fmt.Errorf("get character asset for character %d: %w", characterID, convertGetError(err))
	}
	o := characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	return o, nil
}

func (st *Storage) CalculateCharacterAssetTotalValue(ctx context.Context, characterID int32) (float64, error) {
	v, err := st.qRO.CalculateCharacterAssetTotalValue(ctx, int64(characterID))
	if err != nil {
		return 0, fmt.Errorf("calculate character asset for character %d: %w", characterID, err)
	}
	return v.Float64, nil
}

func (st *Storage) ListCharacterAssetIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterAssetIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list character asset IDs: %w", err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListCharacterAssetsInShipHangar(ctx, queries.ListCharacterAssetsInShipHangarParams{
		CharacterID:   int64(characterID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: app.EveCategoryShip,
	})
	if err != nil {
		return nil, fmt.Errorf("list assets in ship hangar for character ID %d: %w", characterID, err)
	}
	ii2 := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		ii2[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListCharacterAssetsInItemHangar(ctx, queries.ListCharacterAssetsInItemHangarParams{
		CharacterID:   int64(characterID),
		LocationID:    locationID,
		LocationFlag:  "Hangar",
		EveCategoryID: app.EveCategoryShip,
	})
	if err != nil {
		return nil, fmt.Errorf("list assets in item hangar for character ID %d: %w", characterID, err)
	}
	ii2 := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		ii2[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListCharacterAssetsInLocation(ctx, queries.ListCharacterAssetsInLocationParams{
		CharacterID: int64(characterID),
		LocationID:  locationID,
	})
	if err != nil {
		return nil, fmt.Errorf("list assets in location for character ID %d: %w", characterID, err)
	}
	ii2 := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		ii2[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return ii2, nil
}

func (st *Storage) ListAllCharacterAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListAllCharacterAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all character assets: %w", err)
	}
	oo := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func (st *Storage) ListCharacterAssets(ctx context.Context, characterID int32) ([]*app.CharacterAsset, error) {
	rows, err := st.qRO.ListCharacterAssets(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list assets for character ID %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterAsset, len(rows))
	for i, r := range rows {
		oo[i] = characterAssetFromDBModel(r.CharacterAsset, r.EveType, r.EveGroup, r.EveCategory, r.Price)
	}
	return oo, nil
}

func characterAssetFromDBModel(ca queries.CharacterAsset, t queries.EveType, g queries.EveGroup, c queries.EveCategory, p sql.NullFloat64) *app.CharacterAsset {
	if ca.CharacterID == 0 {
		panic("missing character ID")
	}
	locationFlag, found := locationFlagFromDBValue[ca.LocationFlag]
	if !found {
		locationFlag = app.FlagUnknown
	}
	locationType, found := locationTypeFromDBValue[ca.LocationType]
	if !found {
		locationType = app.TypeUnknown
	}
	o := &app.CharacterAsset{
		CharacterID: int32(ca.CharacterID),
		Asset: app.Asset{
			IsBlueprintCopy: ca.IsBlueprintCopy,
			IsSingleton:     ca.IsSingleton,
			ItemID:          ca.ItemID,
			LocationFlag:    locationFlag,
			LocationID:      ca.LocationID,
			LocationType:    locationType,
			Name:            ca.Name,
			Price:           optional.FromNullFloat64(p),
			Quantity:        int(ca.Quantity),
			Type:            eveTypeFromDBModel(t, g, c),
		},
	}
	return o
}

type UpdateCharacterAssetParams struct {
	CharacterID  int32
	ItemID       int64
	LocationFlag app.LocationFlag
	LocationID   int64
	LocationType app.LocationType
	Quantity     int32
}

func (st *Storage) UpdateCharacterAsset(ctx context.Context, arg UpdateCharacterAssetParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterAsset: %+v: %w", arg, err)

	}
	if arg.CharacterID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.UpdateCharacterAsset(ctx, queries.UpdateCharacterAssetParams{
		CharacterID:  int64(arg.CharacterID),
		ItemID:       arg.ItemID,
		LocationFlag: locationFlagToDBValue[arg.LocationFlag],
		LocationID:   arg.LocationID,
		LocationType: locationTypeToDBValue[arg.LocationType],
		Quantity:     int64(arg.Quantity),
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}

type UpdateCharacterAssetNameParams struct {
	CharacterID int32
	ItemID      int64
	Name        string
}

func (st *Storage) UpdateCharacterAssetName(ctx context.Context, arg UpdateCharacterAssetNameParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterAssetName: %+v: %w", arg, err)

	}
	if arg.CharacterID == 0 || arg.ItemID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.UpdateCharacterAssetName(ctx, queries.UpdateCharacterAssetNameParams{
		CharacterID: int64(arg.CharacterID),
		ItemID:      arg.ItemID,
		Name:        arg.Name,
	}); err != nil {
		return wrapErr(err)
	}
	return nil
}
