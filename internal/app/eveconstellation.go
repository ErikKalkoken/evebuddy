package app

// EveConstellation is a constellation in Eve Online.
type EveConstellation struct {
	ID     int32
	Name   string
	Region *EveRegion
}
