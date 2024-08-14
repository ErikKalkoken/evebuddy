package evenotification

const (
	BillOutOfMoneyMsg                  = "BillOutOfMoneyMsg"
	BillPaidCorpAllMsg                 = "BillPaidCorpAllMsg"
	CharAppAcceptMsg                   = "CharAppAcceptMsg"
	CharAppRejectMsg                   = "CharAppRejectMsg"
	CharAppWithdrawMsg                 = "CharAppWithdrawMsg"
	CharLeftCorpMsg                    = "CharLeftCorpMsg"
	CorpAllBillMsg                     = "CorpAllBillMsg"
	CorpAppInvitedMsg                  = "CorpAppInvitedMsg"
	CorpAppNewMsg                      = "CorpAppNewMsg"
	CorpAppRejectCustomMsg             = "CorpAppRejectCustomMsg"
	IHubDestroyedByBillFailure         = "IHubDestroyedByBillFailure"
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
	TowerAlertMsg                      = "TowerAlertMsg"
	TowerResourceAlertMsg              = "TowerResourceAlertMsg"
)

var notificationTypes = []string{
	BillOutOfMoneyMsg,
	BillPaidCorpAllMsg,
	CharAppAcceptMsg,
	CharAppRejectMsg,
	CharAppWithdrawMsg,
	CharLeftCorpMsg,
	CorpAllBillMsg,
	CorpAppInvitedMsg,
	CorpAppNewMsg,
	CorpAppRejectCustomMsg,
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
	TowerAlertMsg,
	TowerResourceAlertMsg,
}

// NotificationTypesSupported returns a list of all supported notification types.
func NotificationTypesSupported() []string {
	return notificationTypes
}
