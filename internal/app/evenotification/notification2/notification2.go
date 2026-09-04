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

// MercOfferRetractedMsg represents data for a merc offer retracted notification.
type MercOfferRetractedMsg struct {
	AggressorID int64 `yaml:"aggressorID"`
	DefenderID  int64 `yaml:"defenderID"`
	MercID      int64 `yaml:"mercID"`
}

// MutualWarInviteSent represents mutual war invite sent data.
type MutualWarInviteSent struct {
	AgainstID       int64 `yaml:"againstID"`
	DeclaredByID    int64 `yaml:"declaredByID"`
	ExpireTimeStamp int64 `yaml:"expireTimeStamp"`
}

// MutualWarInviteAccepted represents mutual war invite accepted data.
type MutualWarInviteAccepted struct {
	AgainstID    int64 `yaml:"againstID"`
	DeclaredByID int64 `yaml:"declaredByID"`
}

// MutualWarInviteRejected represents mutual war invite rejected data.
type MutualWarInviteRejected struct {
	AgainstID    int64 `yaml:"againstID"`
	DeclaredByID int64 `yaml:"declaredByID"`
}

// MutualWarExpired represents mutual war expired data.
type MutualWarExpired struct {
	AgainstID    int64 `yaml:"againstID"`
	DeclaredByID int64 `yaml:"declaredByID"`
}

// CorporationGoalMsg contains data for corporation goal notifications.
type CorporationGoalMsg struct {
	CorporationID int64  `yaml:"corporation_id"`
	CreatorID     int64  `yaml:"creator_id"`
	GoalID        string `yaml:"goal_id"`
	GoalName      string `yaml:"goal_name"`
}

// CorporationGoalNameChange contains data for corp goal name change notification.
type CorporationGoalNameChange struct {
	CorporationID int64  `yaml:"corporation_id"`
	GoalID        string `yaml:"goal_id"`
	NewName       string `yaml:"new_name"`
	OldName       string `yaml:"old_name"`
}

// AllAnchoringMsg contains data for alliance anchoring notification.
type AllAnchoringMsg struct {
	AllianceID    int64 `yaml:"allianceID"`
	CorpID        int64 `yaml:"corpID"`
	MoonID        int64 `yaml:"moonID"`
	SolarSystemID int64 `yaml:"solarSystemID"`
	TypeID        int64 `yaml:"typeID"`
}

// FWAllianceKickMsg contains data for FW alliance kick notification.
type FWAllianceKickMsg struct {
	AllianceID int64 `yaml:"allianceID"`
	FactionID  int64 `yaml:"factionID"`
}

// FWCharKickMsg contains data for FW character kick notification.
type FWCharKickMsg struct {
	FactionID int64 `yaml:"factionID"`
}

// FWCharWarningMsg contains data for FW character warning notification.
type FWCharWarningMsg struct {
	CurrentStanding  float64 `yaml:"currentStanding"`
	FactionID        int64   `yaml:"factionID"`
	RequiredStanding float64 `yaml:"requiredStanding"`
}

// SkyhookDeployed contains data for skyhook deployed notification.
type SkyhookDeployed struct {
	ItemID        int64 `yaml:"itemID"`
	PlanetID      int64 `yaml:"planetID"`
	SolarSystemID int64 `yaml:"solarsystemID"`
	TypeID        int64 `yaml:"typeID"`
}

// SkyhookDestroyed contains data for skyhook destroyed notification.
type SkyhookDestroyed struct {
	AggressorAllianceID  int64 `yaml:"aggressorAllianceID"`
	AggressorCharacterID int64 `yaml:"aggressorCharacterID"`
	AggressorCorpID      int64 `yaml:"aggressorCorpID"`
	ItemID               int64 `yaml:"itemID"`
	PlanetID             int64 `yaml:"planetID"`
	SolarSystemID        int64 `yaml:"solarsystemID"`
	TypeID               int64 `yaml:"typeID"`
}

// SkyhookOnline contains data for skyhook online notification.
type SkyhookOnline struct {
	ItemID        int64 `yaml:"itemID"`
	PlanetID      int64 `yaml:"planetID"`
	SolarSystemID int64 `yaml:"solarsystemID"`
	TypeID        int64 `yaml:"typeID"`
}

// StructureLowReagentsAlert contains data for low reagents notification.
type StructureLowReagentsAlert struct {
	ListOfTypesAndQty     [][]int64     `yaml:"listOfTypesAndQty"`
	SolarsystemID         int64         `yaml:"solarsystemID"`
	StructureID           int64         `yaml:"structureID"`
	StructureShowInfoData []interface{} `yaml:"structureShowInfoData"`
	StructureTypeID       int64         `yaml:"structureTypeID"`
}

// StructureNoReagentsAlert contains data for no reagents notification.
type StructureNoReagentsAlert = StructureLowReagentsAlert

// StructurePaintPurchased contains data for structure paint purchased notification.
type StructurePaintPurchased struct {
	CharID          int64 `yaml:"charID"`
	SolarsystemID   int64 `yaml:"solarsystemID"`
	StructureID     int64 `yaml:"structureID"`
	StructureTypeID int64 `yaml:"structureTypeID"`
}

// StructuresJobsCancelled contains data for structure jobs cancelled notification.
type StructuresJobsCancelled struct {
	NumJobs               int32         `yaml:"numJobs"`
	SolarsystemID         int64         `yaml:"solarsystemID"`
	StructureID           int64         `yaml:"structureID"`
	StructureShowInfoData []interface{} `yaml:"structureShowInfoData"`
	StructureTypeID       int64         `yaml:"structureTypeID"`
}

// StructuresJobsPaused contains data for structure jobs paused notification.
type StructuresJobsPaused = StructuresJobsCancelled

// StructureCourierContractChanged contains data for structure courier contract changed notification.
type StructureCourierContractChanged struct {
	SolarsystemID   int64 `yaml:"solarsystemID"`
	StructureID     int64 `yaml:"structureID"`
	StructureTypeID int64 `yaml:"structureTypeID"`
}

// CloneRevokedMsg1 contains data for clone revoked notification (variant 1).
type CloneRevokedMsg1 struct {
	NewStationID int64 `yaml:"newStationID"`
	StationID    int64 `yaml:"stationID"`
}

// AgentRetiredTrigravian contains data for Triglavian agent retired notification.
type AgentRetiredTrigravian struct {
	AgentID int64 `yaml:"agentID"`
	TypeID  int64 `yaml:"typeID"`
}

// AllianceWarDeclaredV2 contains data for V2 alliance war declaration.
type AllianceWarDeclaredV2 struct {
	AgainstID    int64   `yaml:"againstID"`
	Cost         float64 `yaml:"cost"`
	DeclaredByID int64   `yaml:"declaredByID"`
	DelayHours   int32   `yaml:"delayHours"`
}

// CorpOfficeExpirationMsg contains data for corp office expiration notification.
type CorpOfficeExpirationMsg struct {
	CorpID       int64  `yaml:"corpID"`
	DaysToExpiry int32  `yaml:"daysToExpiry"`
	StationName  string `yaml:"stationName"`
}

// CorpStructLostMsg contains data for corp structure lost notification.
type CorpStructLostMsg struct {
	SolarsystemID   int64 `yaml:"solarsystemID"`
	StructureID     int64 `yaml:"structureID"`
	StructureTypeID int64 `yaml:"structureTypeID"`
}

// CorpVoteCEORevokedMsg contains data for CEO vote revoked notification.
type CorpVoteCEORevokedMsg struct {
	CharID int64 `yaml:"charID"`
	CorpID int64 `yaml:"corpID"`
}

// CorpWarDeclaredV2 contains data for V2 corp war declaration.
type CorpWarDeclaredV2 struct {
	AgainstID    int64   `yaml:"againstID"`
	Cost         float64 `yaml:"cost"`
	DeclaredByID int64   `yaml:"declaredByID"`
}

// CorporationLeft contains data for corporation left alliance notification.
type CorporationLeft struct {
	AllianceID int64 `yaml:"allianceID"`
	CorpID     int64 `yaml:"corpID"`
}

// DistrictAttacked contains data for district attacked notification.
type DistrictAttacked struct {
	AggressorID   int64 `yaml:"aggressorID"`
	DistrictID    int64 `yaml:"districtID"`
	SolarSystemID int64 `yaml:"solarSystemID"`
	TypeID        int64 `yaml:"typeID"`
}

// DustAppAcceptedMsg contains data for Dust app accepted notification.
type DustAppAcceptedMsg struct {
	CharID int64 `yaml:"charID"`
}

// ESSMainBankLink contains data for ESS main bank link notification.
type ESSMainBankLink struct {
	CharID        int64 `yaml:"charID"`
	SolarSystemID int64 `yaml:"solarSystemID"`
	TypeID        int64 `yaml:"typeID"`
}

// ExpertSystemExpired contains data for expert system expired notification.
type ExpertSystemExpired struct {
	TypeID int64 `yaml:"typeID"`
}

// ExpertSystemExpiryImminent contains data for imminent expert system expiry.
type ExpertSystemExpiryImminent struct {
	DaysUntilExpiry int32 `yaml:"daysUntilExpiry"`
	TypeID          int64 `yaml:"typeID"`
}

// IndustryOperationFinished contains data for industry operation finished notification.
type IndustryOperationFinished struct {
	BlueprintTypeID int64 `yaml:"blueprintTypeID"`
	ProductTypeID   int64 `yaml:"productTypeID"`
	Runs            int32 `yaml:"runs"`
	SolarSystemID   int64 `yaml:"solarSystemID"`
	StationID       int64 `yaml:"stationID"`
}

// IndustryTeamAuctionWon contains data for industry team auction won notification.
type IndustryTeamAuctionWon struct {
	SolarSystemID int64   `yaml:"solarSystemID"`
	TotalIsk      float64 `yaml:"totalIsk"`
	YourAmount    float64 `yaml:"yourAmount"`
}

// InvasionCompletedMsg contains data for invasion completed notification.
type InvasionCompletedMsg struct {
	SolarSystemID int64 `yaml:"solarSystemID"`
}

// InvasionSystemLogin contains data for invasion system login notification.
type InvasionSystemLogin struct {
	SolarSystemID int64 `yaml:"solarSystemID"`
}

// InvasionSystemStart contains data for invasion system start notification.
type InvasionSystemStart struct {
	SolarSystemID int64 `yaml:"solarSystemID"`
}

// LPAutoRedeemed contains data for LP auto-redeemed notification.
type LPAutoRedeemed struct {
	FactionID int64 `yaml:"factionID"`
	LP        int32 `yaml:"lp"`
}

// MissionCanceledTriglavian contains data for Triglavian mission canceled notification.
type MissionCanceledTriglavian struct {
	AgentID int64 `yaml:"agentID"`
}

// MissionTimeoutMsg contains data for mission timeout notification.
type MissionTimeoutMsg struct {
	AgentID int64 `yaml:"agentID"`
}

// OfferToAllyRetracted contains data for ally offer retracted notification.
type OfferToAllyRetracted struct {
	AggressorID int64 `yaml:"aggressorID"`
	DefenderID  int64 `yaml:"defenderID"`
	EnemyID     int64 `yaml:"enemyID"`
}

// OfficeLeaseCanceledInsufficientStandings contains data for office lease canceled notification.
type OfficeLeaseCanceledInsufficientStandings struct {
	CorpID    int64 `yaml:"corpID"`
	StationID int64 `yaml:"stationID"`
}

// RaffleCreated contains data for raffle created notification.
type RaffleCreated struct {
	RaffleID int64 `yaml:"raffleID"`
	TypeID   int64 `yaml:"typeID"`
}

// RaffleExpired contains data for raffle expired notification.
type RaffleExpired struct {
	RaffleID int64 `yaml:"raffleID"`
	TypeID   int64 `yaml:"typeID"`
}

// RaffleFinished contains data for raffle finished notification.
type RaffleFinished struct {
	IsWinner bool  `yaml:"isWinner"`
	RaffleID int64 `yaml:"raffleID"`
	TypeID   int64 `yaml:"typeID"`
}

// SPAutoRedeemed contains data for SP auto-redeemed notification.
type SPAutoRedeemed struct {
	Amount int32 `yaml:"amount"`
}

// SkinSequencingCompleted contains data for SKIN sequencing completed notification.
type SkinSequencingCompleted struct {
	TypeID int64 `yaml:"typeID"`
}

// SovCorpBillLateMsg contains data for sov corp bill late notification.
type SovCorpBillLateMsg struct {
	CorpID        int64 `yaml:"corpID"`
	SolarSystemID int64 `yaml:"solarSystemID"`
}

// SovCorpClaimFailMsg contains data for sov corp claim failed notification.
type SovCorpClaimFailMsg struct {
	CorpID        int64 `yaml:"corpID"`
	SolarSystemID int64 `yaml:"solarSystemID"`
}

// SovDisruptorMsg contains data for sovereignty disruptor notification.
type SovDisruptorMsg struct {
	AggressorID   int64 `yaml:"aggressorID"`
	SolarSystemID int64 `yaml:"solarSystemID"`
}

// StationAggressionMsg contains data for station aggression notification.
type StationAggressionMsg struct {
	AggressorAllianceID int64   `yaml:"aggressorAllianceID"`
	AggressorCorpID     int64   `yaml:"aggressorCorpID"`
	AggressorID         int64   `yaml:"aggressorID"`
	ArmorValue          float64 `yaml:"armorValue"`
	HullValue           float64 `yaml:"hullValue"`
	ShieldValue         float64 `yaml:"shieldValue"`
	SolarSystemID       int64   `yaml:"solarSystemID"`
	StationID           int64   `yaml:"stationID"`
	TypeID              int64   `yaml:"typeID"`
}

// StationConquerMsg contains data for station conquered notification.
type StationConquerMsg struct {
	NewOwnerCorpID int64 `yaml:"newOwnerCorpID"`
	OldOwnerCorpID int64 `yaml:"oldOwnerCorpID"`
	SolarSystemID  int64 `yaml:"solarSystemID"`
	StationID      int64 `yaml:"stationID"`
}

// StationStateChangeMsg contains data for station state change notification.
type StationStateChangeMsg struct {
	SolarSystemID int64  `yaml:"solarSystemID"`
	StationID     int64  `yaml:"stationID"`
	State         string `yaml:"state"`
}

// StoryLineMissionAvailableMsg contains data for story line mission available notification.
type StoryLineMissionAvailableMsg struct {
	AgentID int64 `yaml:"agentID"`
}

// TransactionReversalMsg contains data for transaction reversal notification.
type TransactionReversalMsg struct {
	Amount float64 `yaml:"amount"`
	CharID int64   `yaml:"charID"`
}

// TutorialMsg contains data for tutorial message notification.
type TutorialMsg struct {
	TutorialID int64 `yaml:"tutorialID"`
}

// WarAllyInherited contains data for war ally inherited notification.
type WarAllyInherited struct {
	AgainstID int64 `yaml:"againstID"`
	AllyID    int64 `yaml:"allyID"`
	OpenDate  int64 `yaml:"openDate"`
}

// WarConcordInvalidates contains data for CONCORD war invalidation notification.
type WarConcordInvalidates struct {
	AgainstID    int64 `yaml:"againstID"`
	DeclaredByID int64 `yaml:"declaredByID"`
}

// WarEndedHqSecurityDrop contains data for war ended by HQ security drop notification.
type WarEndedHqSecurityDrop struct {
	AgainstID    int64 `yaml:"againstID"`
	DeclaredByID int64 `yaml:"declaredByID"`
}

// WarRetracted contains data for war retracted notification.
type WarRetracted struct {
	AgainstID    int64 `yaml:"againstID"`
	DeclaredByID int64 `yaml:"declaredByID"`
	EndDate      int64 `yaml:"endDate"`
}

// BattlePunishFriendlyFire contains data for friendly fire punishment notification.
type BattlePunishFriendlyFire struct {
	BillAmount   float64 `yaml:"billAmount"`
	KilledCharID int64   `yaml:"killedCharID"`
	KillerCharID int64   `yaml:"killerCharID"`
}

// CombatOperationFinished contains data for combat operation finished notification.
type CombatOperationFinished struct {
	OperationID int64 `yaml:"operationID"`
}

// ContractRegionChangedToPochven contains data for contract moved to Pochven notification.
type ContractRegionChangedToPochven struct {
	ContractID int64 `yaml:"contractID"`
}
