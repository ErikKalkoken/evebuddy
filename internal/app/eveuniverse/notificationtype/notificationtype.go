package notificationtype

type CorpAllBillMsg struct {
	Amount      float64 `yaml:"amount"`
	BillTypeID  int32   `yaml:"billTypeID"`
	CreditorID  int32   `yaml:"creditorID"`
	CurrentDate int64   `yaml:"currentDate"`
	DebtorID    int32   `yaml:"debtorID"`
	DueDate     int64   `yaml:"dueDate"`
	ExternalID  int64   `yaml:"externalID"`
	ExternalID2 int64   `yaml:"externalID2"`
}

type OwnershipTransferred struct {
	CharID          int32  `yaml:"charID"`
	NewOwnerCorpID  int32  `yaml:"newOwnerCorpID"`
	OldOwnerCorpID  int32  `yaml:"oldOwnerCorpID"`
	SolarSystemID   int32  `yaml:"solarSystemID"`
	StructureID     int64  `yaml:"structureID"`
	StructureName   string `yaml:"structureName"`
	StructureTypeID int32  `yaml:"structureTypeID"`
}
