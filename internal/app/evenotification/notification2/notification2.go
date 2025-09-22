// Package notification2 defines types for parsing data of Eve notification types.
package notification2

type MercenaryDenAttacked struct {
	AggressorAllianceName    string  `json:"aggressorAllianceName"`
	AggressorCharacterID     int32   `json:"aggressorCharacterID"`
	AggressorCorporationName string  `json:"aggressorCorporationName"`
	ArmorPercentage          float32 `json:"armorPercentage"`
	HullPercentage           float32 `json:"hullPercentage"`
	ItemID                   int64   `json:"itemID"`
	MercenaryDenShowInfoData []any   `json:"mercenaryDenShowInfoData"`
	PlanetID                 int32   `json:"planetID"`
	PlanetShowInfoData       []any   `json:"planetShowInfoData"`
	ShieldPercentage         float32 `json:"shieldPercentage"`
	SolarsystemID            int32   `json:"solarsystemID"`
	TypeID                   int32   `json:"typeID"`
}

type MercenaryDenReinforced struct {
	AggressorAllianceName    string `json:"aggressorAllianceName"`
	AggressorCharacterID     int32  `json:"aggressorCharacterID"`
	AggressorCorporationName string `json:"aggressorCorporationName"`
	ItemID                   int64  `json:"itemID"`
	MercenaryDenShowInfoData []any  `json:"mercenaryDenShowInfoData"`
	PlanetID                 int32  `json:"planetID"`
	PlanetShowInfoData       []any  `json:"planetShowInfoData"`
	SolarsystemID            int32  `json:"solarsystemID"`
	TimestampEntered         int64  `json:"timestampEntered"`
	TimestampExited          int64  `json:"timestampExited"`
	TypeID                   int32  `json:"typeID"`
}
