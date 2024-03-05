package storage

import "fmt"

// An entity in Eve Online
type EveEntity struct {
	Category string
	ID       int32 `gorm:"primaryKey"`
	Name     string
}

// Save updates or creates an eve entity.
func (e *EveEntity) Save() error {
	err := db.Save(e).Error
	return err
}

// FetchEntityIDs returns all existing entity IDs.
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

// GetEveEntity return an EveEntity object if it exists or nil.
func GetEveEntity(id int32) (*EveEntity, error) {
	var e EveEntity
	err := db.Limit(1).Find(&e, id).Error
	if err != nil {
		return nil, err
	}
	if e.ID == 0 {
		return nil, fmt.Errorf("EveEntity object not found for ID %d", id)
	}
	return &e, nil
}
