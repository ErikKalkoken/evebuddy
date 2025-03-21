package app

import "context"

type EveUniverseService interface {
	AddMissingEntities(context.Context, []int32) ([]int32, error)
	AddMissingEveTypes(context.Context, []int32) error
	FormatDogmaValue(context.Context, float32, EveUnitID) (string, int32)
	GetAllianceCorporationsESI(context.Context, int32) ([]*EveEntity, error)
	GetAllianceESI(context.Context, int32) (*EveAlliance, error)
	GetCharacterCorporationHistory(context.Context, int32) ([]MembershipHistoryItem, error)
	GetCharacterESI(context.Context, int32) (*EveCharacter, error)
	GetConstellationSolarSytemsESI(context.Context, int32) ([]*EveSolarSystem, error)
	GetCorporationAllianceHistory(context.Context, int32) ([]MembershipHistoryItem, error)
	GetCorporationESI(context.Context, int32) (*EveCorporation, error)
	GetLocation(context.Context, int64) (*EveLocation, error)
	GetMarketPrice(context.Context, int32) (*EveMarketPrice, error)
	GetOrCreateCharacterESI(context.Context, int32) (*EveCharacter, error)
	GetOrCreateConstellationESI(context.Context, int32) (*EveConstellation, error)
	GetOrCreateEntityESI(context.Context, int32) (*EveEntity, error)
	GetOrCreateLocationESI(context.Context, int64) (*EveLocation, error)
	GetOrCreatePlanetESI(context.Context, int32) (*EvePlanet, error)
	GetOrCreateRegionESI(context.Context, int32) (*EveRegion, error)
	GetOrCreateSchematicESI(context.Context, int32) (*EveSchematic, error)
	GetOrCreateSolarSystemESI(context.Context, int32) (*EveSolarSystem, error)
	GetOrCreateTypeESI(context.Context, int32) (*EveType, error)
	GetPlanets(context.Context, []EveSolarSystemPlanet) ([]*EvePlanet, error)
	GetRegionConstellationsESI(context.Context, int32) ([]*EveEntity, error)
	GetSolarSystemInfoESI(ctx context.Context, solarSystemID int32) (int32, []EveSolarSystemPlanet, []int32, []*EveEntity, []*EveLocation, error)
	GetSolarSystemsESI(context.Context, []int32) ([]*EveSolarSystem, error)
	GetStarTypeID(context.Context, int32) (int32, error)
	GetType(context.Context, int32) (*EveType, error)
	ListEntitiesByPartialName(context.Context, string) ([]*EveEntity, error)
	ListLocations(context.Context) ([]*EveLocation, error)
	ListTypeDogmaAttributesForType(context.Context, int32) ([]*EveTypeDogmaAttribute, error)
	ToEveEntities(context.Context, []int32) (map[int32]*EveEntity, error)
	UpdateSection(context.Context, GeneralSection, bool) (bool, error)
}
