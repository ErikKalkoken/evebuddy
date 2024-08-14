package evenotification

const (
	CorpAppNewMsg                      = "CorpAppNewMsg"
	CharAppRejectMsg                   = "CharAppRejectMsg"
	CorpAppInvitedMsg                  = "CorpAppInvitedMsg"
	BillOutOfMoneyMsg                  = "BillOutOfMoneyMsg"
	CorpAppRejectCustomMsg             = "CorpAppRejectCustomMsg"
	BillPaidCorpAllMsg                 = "BillPaidCorpAllMsg"
	CorpAllBillMsg                     = "CorpAllBillMsg"
	InfrastructureHubBillAboutToExpire = "InfrastructureHubBillAboutToExpire"
	OrbitalAttacked                    = "OrbitalAttacked"
	OrbitalReinforced                  = "OrbitalReinforced"
	OwnershipTransferred               = "OwnershipTransferred"
	StructureAnchoring                 = "StructureAnchoring"
	StructureDestroyed                 = "StructureDestroyed"
	StructureFuelAlert                 = "StructureFuelAlert"
	StructureLostArmor                 = "StructureLostArmor"
	StructureLostShields               = "StructureLostShields"
	StructureOnline                    = "StructureOnline"
	StructureUnanchoring               = "StructureUnanchoring"
	StructureUnderAttack               = "StructureUnderAttack"
	StructureWentHighPower             = "StructureWentHighPower"
	StructureWentLowPower              = "StructureWentLowPower"
	IHubDestroyedByBillFailure         = "IHubDestroyedByBillFailure"
)

var notificationTypes = []string{
	CorpAppNewMsg,
	CharAppRejectMsg,
	CorpAppRejectCustomMsg,
	CorpAppInvitedMsg,
	BillOutOfMoneyMsg,
	BillPaidCorpAllMsg,
	CorpAllBillMsg,
	IHubDestroyedByBillFailure,
	InfrastructureHubBillAboutToExpire,
	OrbitalAttacked,
	OrbitalReinforced,
	OwnershipTransferred,
	StructureAnchoring,
	StructureDestroyed,
	StructureFuelAlert,
	StructureLostArmor,
	StructureLostShields,
	StructureOnline,
	StructureUnanchoring,
	StructureUnderAttack,
	StructureWentHighPower,
	StructureWentLowPower,
}

// NotificationTypesSupported returns a list of all supported notification types.
func NotificationTypesSupported() []string {
	return notificationTypes
}
