package storage

// An entity in Eve Online
type EveEntity struct {
	Category string
	ID       int32 `gorm:"primaryKey"`
	Name     string
}

func (e *EveEntity) Save() error {
	err := db.Save(e).Error
	return err
}

// Return all existing entity IDs
func FetchEntityIDs() ([]int32, error) {
	var objs []EveEntity
	err := db.Select("id").Find(&objs).Error
	if err != nil {
		return nil, err
	}
	var ids []int32
	for _, o := range objs {
		ids = append(ids, o.ID)
	}
	return ids, nil
}

func GetOrCreateEveEntity(id int32) (*EveEntity, bool, error) {
	var e EveEntity
	err := db.Limit(1).Find(&e, id).Error
	if err != nil {
		return nil, false, err
	}
	created := false
	if e.ID == 0 {
		e.ID = id
		e.Save()
		created = true
	}
	return &e, created, nil
}
