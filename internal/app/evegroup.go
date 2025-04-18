package app

const (
	EveGroupAuditLogFreightContainer     = 649
	EveGroupAuditLogSecureCargoContainer = 448
	EveGroupBlackOps                     = 898
	EveGroupCapitalIndustrialShip        = 883
	EveGroupCargoContainer               = 12
	EveGroupCarrier                      = 547
	EveGroupDreadnought                  = 485
	EveGroupExtractorControlUnits        = 1063
	EveGroupForceAuxiliary               = 1538
	EveGroupJumpFreighter                = 902
	EveGroupPlanet                       = 7
	EveGroupProcessors                   = 1028
	EveGroupSecureCargoContainer         = 340
	EveGroupSuperCarrier                 = 659
	EveGroupTitan                        = 30
)

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}
