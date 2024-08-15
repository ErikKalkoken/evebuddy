package evenotification

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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
