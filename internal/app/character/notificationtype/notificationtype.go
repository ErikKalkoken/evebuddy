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
