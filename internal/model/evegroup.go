package model

const (
	EveGroupAuditLogFreightContainer     = 649
	EveGroupAuditLogSecureCargoContainer = 448
	EveGroupCargoContainer               = 12
	EveGroupSecureCargoContainer         = 340
)

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}
