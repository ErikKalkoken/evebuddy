package app

// EveRegion is a region in Eve Online.
type EveRegion struct {
	Description string
	ID          int32
	Name        string
}

func (er EveRegion) ToEveEntity() *EveEntity {
	return &EveEntity{ID: er.ID, Name: er.Name, Category: EveEntityRegion}
}
