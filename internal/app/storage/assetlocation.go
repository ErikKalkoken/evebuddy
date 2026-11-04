package storage

import (
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// locationFlagFromDBValue translates the stored location flags of all owners.
// Which flags are valid for an owner is defined in the app package.
var locationFlagFromDBValue = map[string]app.LocationFlag{
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
	"CorpseBay":                           app.FlagCorpseBay,
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
	"StructureDeedBay":                    app.FlagStructureDeedBay,
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

var locationFlagToDBValue = map[app.LocationFlag]string{}

var locationTypeFromDBValue = map[string]app.LocationType{
	"":             app.TypeUndefined,
	"station":      app.TypeStation,
	"solar_system": app.TypeSolarSystem,
	"item":         app.TypeItem,
	"other":        app.TypeOther,
	"unknown":      app.TypeUnknown,
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

// isValidLocationFlag reports whether a location flag can be stored for an owner.
// allowed holds the flags which are valid for that owner.
// [app.FlagUnknown] is always valid, because it stands for flags ESI has newly added.
func isValidLocationFlag(allowed set.Set[app.LocationFlag], flag app.LocationFlag) bool {
	return flag == app.FlagUnknown || allowed.Contains(flag)
}
