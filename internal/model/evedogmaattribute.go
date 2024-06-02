package model

type EveDogmaAttribute struct {
	ID           int32
	DefaultValue float32
	Description  string
	DisplayName  string
	IconID       int32
	Name         string
	IsHighGood   bool
	IsPublished  bool
	IsStackable  bool
	UnitID       int32
}
