package storage

// An Eve Online character
type Character struct {
	ID   int32 `gorm:"primaryKey"`
	Name string
}

func (c *Character) Save() error {
	if err := db.Save(c).Error; err != nil {
		return err
	}
	return nil
}
