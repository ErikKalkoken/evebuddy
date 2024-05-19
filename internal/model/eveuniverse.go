package model

const (
	EveCategoryIDSkill       = 16
	EveTypeIDSolarSystem     = 5
	EveTypeIDAssetSafetyWrap = 60
)

// EveCategory is a category in Eve Online.
type EveCategory struct {
	ID          int32
	IsPublished bool
	Name        string
}

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}

// EveType is a type in Eve Online.
type EveType struct {
	ID          int32
	Description string
	Group       *EveGroup
	IsPublished bool
	Name        string
}

// EveRegion is a region in Eve Online.
type EveRegion struct {
	Description string
	ID          int32
	Name        string
}

// EveConstellation is a constellation in Eve Online.
type EveConstellation struct {
	ID     int32
	Name   string
	Region *EveRegion
}

// EveSolarSystem is a solar system in Eve Online.
type EveSolarSystem struct {
	Constellation  *EveConstellation
	ID             int32
	Name           string
	SecurityStatus float64
}

type Position struct {
	X float64
	Y float64
	Z float64
}
