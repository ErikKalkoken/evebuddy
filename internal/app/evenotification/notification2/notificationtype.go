// Package notification2 contains type definitions to unmarshal Eve notifications from ESI.
// It extends the notification package from goesi.
package notification2

type CorpAllBillMsgV2 struct {
	Amount      float64 `yaml:"amount"`
	BillTypeID  int32   `yaml:"billTypeID"`
	CreditorID  int32   `yaml:"creditorID"`
	CurrentDate int64   `yaml:"currentDate"`
	DebtorID    int32   `yaml:"debtorID"`
	DueDate     int64   `yaml:"dueDate"`
	ExternalID  int64   `yaml:"externalID"`
	ExternalID2 int64   `yaml:"externalID2"`
}

type InfrastructureHubBillAboutToExpire struct {
	BillID        int32 `yaml:"billID"`
	CorpID        int32 `yaml:"corpID"`
	DueDate       int64 `yaml:"dueDate"`
	SolarSystemID int32 `yaml:"solarSystemID"`
}

type IHubDestroyedByBillFailure struct {
	SolarSystemID   int32 `yaml:"solarSystemID"`
	StructureTypeID int64 `yaml:"structureTypeID"`
}

type OwnershipTransferredV2 struct {
	CharID          int32  `yaml:"charID"`
	NewOwnerCorpID  int32  `yaml:"newOwnerCorpID"`
	OldOwnerCorpID  int32  `yaml:"oldOwnerCorpID"`
	SolarSystemID   int32  `yaml:"solarSystemID"`
	StructureID     int64  `yaml:"structureID"`
	StructureName   string `yaml:"structureName"`
	StructureTypeID int32  `yaml:"structureTypeID"`
}

type StructureImpendingAbandonmentAssetsAtRisk struct {
	DaysUntilAbandon      int32  `yaml:"daysUntilAbandon"`
	IsCorpOwned           bool   `yaml:"isCorpOwned"`
	SolarSystemID         int32  `yaml:"solarsystemID"`
	StructureID           int64  `yaml:"structureID"`
	StructureLink         string `yaml:"structureLink"`
	StructureShowInfoData []any  `yaml:"structureShowInfoData"`
	StructureTypeID       int32  `yaml:"structureTypeID"`
}

type StructureItemsMovedToSafety struct {
	AssetSafetyDurationFull     int64  `yaml:"assetSafetyDurationFull"`
	AssetSafetyDurationMinimum  int64  `yaml:"assetSafetyDurationMinimum"`
	AssetSafetyFullTimestamp    int64  `yaml:"assetSafetyFullTimestamp"`
	AssetSafetyMinimumTimestamp int64  `yaml:"assetSafetyMinimumTimestamp"`
	IsCorpOwned                 bool   `yaml:"isCorpOwned"`
	NewStationID                int32  `yaml:"newStationID"`
	SolarSystemID               int32  `yaml:"solarsystemID"`
	StructureID                 int64  `yaml:"structureID"`
	StructureLink               string `yaml:"structureLink"`
	StructureShowInfoData       []any  `yaml:"structureShowInfoData"`
	StructureTypeID             int32  `yaml:"structureTypeID"`
}

type WarAdopted struct {
	AgainstID    int32 `yaml:"againstID"`
	AllianceID   int32 `yaml:"allianceID"`
	DeclaredByID int32 `yaml:"declaredByID"`
}

type WarDeclared struct {
	AgainstID    int32   `yaml:"againstID"`
	Cost         float64 `yaml:"cost"`
	DeclaredByID int32   `yaml:"declaredByID"`
	DelayHours   int32   `yaml:"delayHours"`
	HostileState bool    `yaml:"hostileState"`
	TimeStarted  int64   `yaml:"timeStarted"`
	WarHQ        string  `yaml:"warHQ"`
	WarHQIdType  []any   `yaml:"warHQ_IdType"`
}

type WarHQRemovedFromSpace struct {
	AgainstID    int32  `yaml:"againstID"`
	DeclaredByID int32  `yaml:"declaredByID"`
	TimeDeclared int64  `yaml:"timeDeclared"`
	WarHQ        string `yaml:"warHQ"`
}

type WarInherited struct {
	AgainstID    int32 `yaml:"againstID"`
	AllianceID   int32 `yaml:"allianceID"`
	DeclaredByID int32 `yaml:"declaredByID"`
	OpponentID   int32 `yaml:"opponentID"`
	QuitterID    int32 `yaml:"quitterID"`
}

type WarInvalid struct {
	AgainstID    int32 `yaml:"againstID"`
	DeclaredByID int32 `yaml:"declaredByID"`
	EndDate      int64 `yaml:"endDate"`
}

type WarRetractedByConcord struct {
	AgainstID    int32 `yaml:"againstID"`
	DeclaredByID int32 `yaml:"declaredByID"`
	EndDate      int64 `yaml:"endDate"`
}
