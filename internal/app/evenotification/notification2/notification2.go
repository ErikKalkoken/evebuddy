// Package notification2 defines types for parsing data of Eve notification types.
package notification2

type MercenaryDenAttacked struct {
	AggressorAllianceName    string  `yaml:"aggressorAllianceName"`
	AggressorCharacterID     int64   `yaml:"aggressorCharacterID"`
	AggressorCorporationName string  `yaml:"aggressorCorporationName"`
	ArmorPercentage          float64 `yaml:"armorPercentage"`
	HullPercentage           float64 `yaml:"hullPercentage"`
	ItemID                   int64   `yaml:"itemID"`
	MercenaryDenShowInfoData []any   `yaml:"mercenaryDenShowInfoData"`
	PlanetID                 int64   `yaml:"planetID"`
	PlanetShowInfoData       []any   `yaml:"planetShowInfoData"`
	ShieldPercentage         float64 `yaml:"shieldPercentage"`
	SolarSystemID            int64   `yaml:"solarsystemID"`
	TypeID                   int64   `yaml:"typeID"`
}

type MercenaryDenReinforced struct {
	AggressorAllianceName    string `yaml:"aggressorAllianceName"`
	AggressorCharacterID     int64  `yaml:"aggressorCharacterID"`
	AggressorCorporationName string `yaml:"aggressorCorporationName"`
	ItemID                   int64  `yaml:"itemID"`
	MercenaryDenShowInfoData []any  `yaml:"mercenaryDenShowInfoData"`
	PlanetID                 int64  `yaml:"planetID"`
	PlanetShowInfoData       []any  `yaml:"planetShowInfoData"`
	SolarSystemID            int64  `yaml:"solarsystemID"`
	TimestampEntered         int64  `yaml:"timestampEntered"`
	TimestampExited          int64  `yaml:"timestampExited"`
	TypeID                   int64  `yaml:"typeID"`
}
