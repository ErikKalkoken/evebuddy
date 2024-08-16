package evenotification

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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
	MoonminingExtractionStarted        = "MoonminingExtractionStarted"
	MoonminingExtractionFinished       = "MoonminingExtractionFinished"
	MoonminingAutomaticFracture        = "MoonminingAutomaticFracture"
	MoonminingExtractionCancelled      = "MoonminingExtractionCancelled"
	MoonminingLaserFired               = "MoonminingLaserFired"
	WarDeclared                        = "WarDeclared"
	WarInherited                       = "WarInherited"
	AllWarSurrenderMsg                 = "AllWarSurrenderMsg"
	CorpWarSurrenderMsg                = "CorpWarSurrenderMsg"
	WarHQRemovedFromSpace              = "WarHQRemovedFromSpace"
	WarAdopted                         = "WarAdopted "
	WarInvalid                         = "WarInvalid"
	WarRetractedByConcord              = "WarRetractedByConcord"
	StructureServicesOffline           = "StructureServicesOffline"
	StructuresReinforcementChanged     = "StructuresReinforcementChanged"
)

var notificationTypes = []string{
	StructuresReinforcementChanged,
	WarRetractedByConcord,
	WarDeclared,
	WarInvalid,
	WarAdopted,
	WarHQRemovedFromSpace,
	StructureServicesOffline,
	AllWarSurrenderMsg,
	WarInherited,
	CorpWarSurrenderMsg,
	BillOutOfMoneyMsg,
	BillPaidCorpAllMsg,
	CharAppAcceptMsg,
	MoonminingExtractionFinished,
	MoonminingAutomaticFracture,
	MoonminingExtractionCancelled,
	MoonminingLaserFired,
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
	MoonminingExtractionStarted,
	StructureWentHighPower,
	StructureWentLowPower,
	TowerAlertMsg,
	TowerResourceAlertMsg,
}

// NotificationTypesSupported returns a list of all supported notification types.
func NotificationTypesSupported() []string {
	return notificationTypes
}

// EveNotificationService provides services to handle notifications
type EveNotificationService struct {
	EveUniverseService *eveuniverse.EveUniverseService
}

func New() *EveNotificationService {
	s := &EveNotificationService{}
	return s
}

// RenderESI renders title and body for all supported notification types and returns them.
// Returns empty title and body for unsupported notification types.
func (s *EveNotificationService) RenderESI(ctx context.Context, type_, text string, timestamp time.Time) (optional.Optional[string], optional.Optional[string], error) {
	switch type_ {
	case BillOutOfMoneyMsg,
		BillPaidCorpAllMsg,
		CorpAllBillMsg,
		InfrastructureHubBillAboutToExpire,
		IHubDestroyedByBillFailure:
		return s.renderBilling(ctx, type_, text)

	case CharAppAcceptMsg,
		CharAppRejectMsg,
		CharAppWithdrawMsg,
		CharLeftCorpMsg,
		CorpAppInvitedMsg,
		CorpAppNewMsg,
		CorpAppRejectCustomMsg:
		return s.renderCorporate(ctx, type_, text)

	case OrbitalAttacked,
		OrbitalReinforced:
		return s.renderOrbital(ctx, type_, text)

	case MoonminingExtractionStarted,
		MoonminingExtractionFinished,
		MoonminingAutomaticFracture,
		MoonminingExtractionCancelled,
		MoonminingLaserFired:
		return s.renderMoonMining(ctx, type_, text)

	case OwnershipTransferred,
		StructureAnchoring,
		StructureDestroyed,
		StructureFuelAlert,
		StructureLostArmor,
		StructureLostShields,
		StructureOnline,
		StructuresReinforcementChanged,
		StructureServicesOffline,
		StructureUnanchoring,
		StructureUnderAttack,
		StructureWentHighPower,
		StructureWentLowPower:
		return s.renderStructure(ctx, type_, text, timestamp)

	case TowerAlertMsg,
		TowerResourceAlertMsg:
		return s.renderTower(ctx, type_, text)
	case AllWarSurrenderMsg,
		CorpWarSurrenderMsg,
		WarAdopted,
		WarDeclared,
		WarHQRemovedFromSpace,
		WarInherited,
		WarInvalid,
		WarRetractedByConcord:
		return s.renderWar(ctx, type_, text)
	}
	return optional.Optional[string]{}, optional.Optional[string]{}, nil
}
