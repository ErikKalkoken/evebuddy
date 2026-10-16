package storage

import (
	"database/sql"
)

// helper types
// used to ensure the correct parameter is passed to a function
// when multiple parameters have the same real type

type blueprintLocationName string
type blueprintLocationSecurity sql.NullFloat64
type blueprintTypeName string
type completedCharacterName sql.NullString
type description string
type facilityName string
type facilitySecurity sql.NullFloat64
type groupName string
type locationName string
type nullAlliance nullEveEntity
type nullCEO nullEveEntity
type nullCorporation nullEveEntity
type nullCreator nullEveEntity
type nullFaction nullEveEntity
type nullMilitiaCorporation nullEveEntity
type nullRaceFaction nullEveEntity
type nullSolarSystem nullEveEntity
type nullStation nullEveEntity
type outputLocationName string
type outputLocationSecurity sql.NullFloat64
type productTypeName sql.NullString
type regionName string
type skillName string
type stationName string
type stationSecurity sql.NullFloat64
type typeName string
