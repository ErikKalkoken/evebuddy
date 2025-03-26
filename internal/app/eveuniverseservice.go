package app

import (
	"context"
)

// EveUniverseService ...
type EveUniverseService interface {
	GetAllianceESI(ctx context.Context, allianceID int32) (*EveAlliance, error)
	GetAllianceCorporationsESI(ctx context.Context, allianceID int32) ([]*EveEntity, error)
	GetOrCreateCharacterESI(ctx context.Context, id int32) (*EveCharacter, error)
	GetCharacterESI(ctx context.Context, characterID int32) (*EveCharacter, error)
	// UpdateAllCharactersESI updates all known Eve characters from ESI.
	UpdateAllCharactersESI(ctx context.Context) error
	GetCorporationESI(ctx context.Context, corporationID int32) (*EveCorporation, error)
	GetDogmaAttribute(ctx context.Context, id int32) (*EveDogmaAttribute, error)
	GetOrCreateDogmaAttributeESI(ctx context.Context, id int32) (*EveDogmaAttribute, error)
	// FormatDogmaValue returns a formatted value.
	FormatDogmaValue(ctx context.Context, value float32, unitID EveUnitID) (string, int32)
	GetEntity(ctx context.Context, id int32) (*EveEntity, error)
	GetOrCreateEntityESI(ctx context.Context, id int32) (*EveEntity, error)
	// ToEntities returns the resolved EveEntities for a list of valid entity IDs.
	// It garantees a result for every ID and will map unknown IDs (including 0 & 1) to empty EveEntity objects.
	ToEntities(ctx context.Context, ids []int32) (map[int32]*EveEntity, error)
	// AddMissingEntities adds EveEntities from ESI for IDs missing in the database
	// and returns which IDs where indeed missing.
	//
	// Invalid IDs (e.g. 0, 1) will be ignored.
	AddMissingEntities(ctx context.Context, ids []int32) ([]int32, error)
	ListEntitiesByPartialName(ctx context.Context, partial string) ([]*EveEntity, error)
	GetType(ctx context.Context, id int32) (*EveType, error)
	GetOrCreateCategoryESI(ctx context.Context, id int32) (*EveCategory, error)
	GetOrCreateGroupESI(ctx context.Context, id int32) (*EveGroup, error)
	GetOrCreateTypeESI(ctx context.Context, id int32) (*EveType, error)
	AddMissingTypes(ctx context.Context, ids []int32) error
	UpdateCategoryWithChildrenESI(ctx context.Context, categoryID int32) error
	UpdateShipSkills(ctx context.Context) error
	ListTypeDogmaAttributesForType(ctx context.Context, typeID int32) ([]*EveTypeDogmaAttribute, error)
	GetLocation(ctx context.Context, id int64) (*EveLocation, error)
	ListLocations(ctx context.Context) ([]*EveLocation, error)
	// GetOrCreateLocationESI return a structure when it already exists
	// or else tries to fetch and create a new structure from ESI.
	//
	// Important: A token with the structure scope must be set in the context
	GetOrCreateLocationESI(ctx context.Context, id int64) (*EveLocation, error)
	GetOrCreateRegionESI(ctx context.Context, id int32) (*EveRegion, error)
	GetOrCreateConstellationESI(ctx context.Context, id int32) (*EveConstellation, error)
	GetOrCreateSolarSystemESI(ctx context.Context, id int32) (*EveSolarSystem, error)
	GetOrCreatePlanetESI(ctx context.Context, id int32) (*EvePlanet, error)
	GetOrCreateMoonESI(ctx context.Context, id int32) (*EveMoon, error)
	GetRouteESI(ctx context.Context, destination, origin *EveSolarSystem, flag RoutePreference) ([]*EveSolarSystem, error)
	GetMarketPrice(ctx context.Context, typeID int32) (*EveMarketPrice, error)
	GetOrCreateRaceESI(ctx context.Context, id int32) (*EveRace, error)
	GetOrCreateSchematicESI(ctx context.Context, id int32) (*EveSchematic, error)
	GetSolarSystemsESI(ctx context.Context, stargateIDs []int32) ([]*EveSolarSystem, error)
	GetSolarSystemPlanets(ctx context.Context, planets []EveSolarSystemPlanet) ([]*EvePlanet, error)
	GetStarTypeID(ctx context.Context, id int32) (int32, error)
	GetSolarSystemInfoESI(ctx context.Context, solarSystemID int32) (int32, []EveSolarSystemPlanet, []int32, []*EveEntity, []*EveLocation, error)
	GetRegionConstellationsESI(ctx context.Context, id int32) ([]*EveEntity, error)
	GetConstellationSolarSytemsESI(ctx context.Context, id int32) ([]*EveSolarSystem, error)
	// GetCharacterCorporationHistory returns a list of all the corporations a character has been a member of in descending order.
	GetCharacterCorporationHistory(ctx context.Context, characterID int32) ([]MembershipHistoryItem, error)
	// CharacterCorporationHistory returns a list of all the alliances a corporation has been a member of in descending order.
	GetCorporationAllianceHistory(ctx context.Context, corporationID int32) ([]MembershipHistoryItem, error)
	UpdateSection(ctx context.Context, section GeneralSection, forceUpdate bool) (bool, error)
	GetStationServicesESI(ctx context.Context, id int32) ([]string, error)
	ListEntitiesForIDs(ctx context.Context, ids []int32) ([]*EveEntity, error)
}
