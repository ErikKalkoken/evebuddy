// Package notification2 defines types for parsing data of Eve notification types.
package notification2

type MercenaryDenAttacked struct {
	AggressorAllianceName    string  `json:"aggressorAllianceName"`
	AggressorCharacterID     int64   `json:"aggressorCharacterID"`
	AggressorCorporationName string  `json:"aggressorCorporationName"`
	ArmorPercentage          float64 `json:"armorPercentage"`
	HullPercentage           float64 `json:"hullPercentage"`
	ItemID                   int64   `json:"itemID"`
	MercenaryDenShowInfoData []any   `json:"mercenaryDenShowInfoData"`
	PlanetID                 int64   `json:"planetID"`
	PlanetShowInfoData       []any   `json:"planetShowInfoData"`
	ShieldPercentage         float64 `json:"shieldPercentage"`
	SolarsystemID            int64   `json:"solarsystemID"`
	TypeID                   int64   `json:"typeID"`
}

type MercenaryDenReinforced struct {
	AggressorAllianceName    string `json:"aggressorAllianceName"`
	AggressorCharacterID     int64  `json:"aggressorCharacterID"`
	AggressorCorporationName string `json:"aggressorCorporationName"`
	ItemID                   int64  `json:"itemID"`
	MercenaryDenShowInfoData []any  `json:"mercenaryDenShowInfoData"`
	PlanetID                 int64  `json:"planetID"`
	PlanetShowInfoData       []any  `json:"planetShowInfoData"`
	SolarsystemID            int64  `json:"solarsystemID"`
	TimestampEntered         int64  `json:"timestampEntered"`
	TimestampExited          int64  `json:"timestampExited"`
	TypeID                   int64  `json:"typeID"`
}
