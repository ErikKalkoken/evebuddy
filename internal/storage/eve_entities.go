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
