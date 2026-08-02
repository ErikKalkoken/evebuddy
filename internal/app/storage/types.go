package storage

import (
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

// helper types
// used to ensure the correct parameter is passed to a function
// when multiple parameters have the same real type

type blueprintLocationName string
type blueprintLocationSecurity sql.NullFloat64
type blueprintTypeName string
type completedCharacterName sql.NullString
type description string
type endLocationName sql.NullString
type endSolarSystemID sql.NullInt64
type endSolarSystemName sql.NullString
type endSolarSystemSecurity sql.NullFloat64
type facilityName string
type facilitySecurity sql.NullFloat64
type groupName string
type issuer queries.EveEntity
type issuerCorporation queries.EveEntity
type locationName string
type nullAcceptor nullEveEntity
type nullAlliance nullEveEntity
type nullAssignee nullEveEntity
type nullCEO nullEveEntity
type nullCorporation nullEveEntity
type nullCreator nullEveEntity
type nullFaction nullEveEntity
type nullMilitiaCorporation nullEveEntity
type nullStation nullEveEntity
type outputLocationName string
type outputLocationSecurity sql.NullFloat64
type productTypeName sql.NullString
type regionName string
type skillName string
type startLocationName sql.NullString
type startSolarSystemID sql.NullInt64
type startSolarSystemName sql.NullString
type startSolarSystemSecurity sql.NullFloat64
type stationName string
type stationSecurity sql.NullFloat64
type typeName string
