package app

type CharacterContractItem struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int
	RawQuantity int
	RecordID    int64
	Type        *EveType
}
