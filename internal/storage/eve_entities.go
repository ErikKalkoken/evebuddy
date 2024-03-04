package storage

// An entity in Eve Online
type EveEntity struct {
	Category string
	ID       int32 `gorm:"primaryKey"`
	Name     string
}
