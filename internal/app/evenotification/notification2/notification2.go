// Package notification2 defines types for parsing data of Eve notification types.
package notification2

type CorpBecameWarEligible struct {
}

type CorpNoLongerWarEligible struct {
}

type MutualWarInviteSent struct {
	AgainstID       int64 `yaml:"againstID"`
	DeclaredByID    int64 `yaml:"declaredByID"`
	ExpireTimeStamp int64 `yaml:"expireTimeStamp"`
}

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

type AllAnchoringCorp struct {
	AllianceID int64 `yaml:"allianceID"`
	CorpID     int64 `yaml:"corpID"`
}

type AllAnchoringTower struct {
	MoonID int64 `yaml:"moonID"`
	TypeID int64 `yaml:"typeID"`
}

type AllAnchoringMsg struct {
	AllianceID    int64               `yaml:"allianceID"`
	CorpID        int64               `yaml:"corpID"`
	CorpsPresent  []AllAnchoringCorp  `yaml:"corpsPresent"`
	MoonID        int64               `yaml:"moonID"`
	SolarSystemID int64               `yaml:"solarSystemID"`
	Towers        []AllAnchoringTower `yaml:"towers"`
	TypeID        int64               `yaml:"typeID"`
}

type MercOfferRetractedMsg struct {
	AggressorID int64 `yaml:"aggressorID"`
	DefenderID  int64 `yaml:"defenderID"`
	MercID      int64 `yaml:"mercID"`
}

type SkyhookDestroyed struct {
	ItemID              int64 `yaml:"itemID"`
	PlanetID            int64 `yaml:"planetID"`
	PlanetShowInfoData  []any `yaml:"planetShowInfoData"`
	SkyhookShowInfoData []any `yaml:"skyhookShowInfoData"`
	SolarsystemID       int64 `yaml:"solarsystemID"`
	TypeID              int64 `yaml:"typeID"`
}

type SkyhookDeployed struct {
	ItemID              int64  `yaml:"itemID"`
	OwnerCorpLinkData   []any  `yaml:"ownerCorpLinkData"`
	OwnerCorpName       string `yaml:"ownerCorpName"`
	PlanetID            int64  `yaml:"planetID"`
	PlanetShowInfoData  []any  `yaml:"planetShowInfoData"`
	SkyhookShowInfoData []any  `yaml:"skyhookShowInfoData"`
	SolarsystemID       int64  `yaml:"solarsystemID"`
	TimeLeft            int64  `yaml:"timeLeft"`
	TypeID              int64  `yaml:"typeID"`
}

type SkyhookOnline struct {
	ItemID              int64 `yaml:"itemID"`
	PlanetID            int64 `yaml:"planetID"`
	PlanetShowInfoData  []any `yaml:"planetShowInfoData"`
	SkyhookShowInfoData []any `yaml:"skyhookShowInfoData"`
	SolarsystemID       int64 `yaml:"solarsystemID"`
	TypeID              int64 `yaml:"typeID"`
}

type StructureLowReagentsAlert struct {
	SolarsystemID         int64 `yaml:"solarsystemID"`
	StructureID           int64 `yaml:"structureID"`
	StructureShowInfoData []any `yaml:"structureShowInfoData"`
	StructureTypeID       int64 `yaml:"structureTypeID"`
}

type StructureNoReagentsAlert struct {
	SolarsystemID         int64 `yaml:"solarsystemID"`
	StructureID           int64 `yaml:"structureID"`
	StructureShowInfoData []any `yaml:"structureShowInfoData"`
	StructureTypeID       int64 `yaml:"structureTypeID"`
}

type CorporationGoalCreated struct {
	CorporationID int64  `yaml:"corporation_id"`
	CreatorID     int64  `yaml:"creator_id"`
	GoalID        string `yaml:"goal_id"`
	GoalName      string `yaml:"goal_name"`
}

type CorporationGoalCompleted struct {
	CorporationID int64  `yaml:"corporation_id"`
	CreatorID     int64  `yaml:"creator_id"`
	GoalID        string `yaml:"goal_id"`
	GoalName      string `yaml:"goal_name"`
}

type CorporationGoalClosed struct {
	CloserID      int64  `yaml:"closer_id"`
	CorporationID int64  `yaml:"corporation_id"`
	CreatorID     int64  `yaml:"creator_id"`
	GoalID        string `yaml:"goal_id"`
	GoalName      string `yaml:"goal_name"`
}
