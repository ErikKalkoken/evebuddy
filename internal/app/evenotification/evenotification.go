// Package evenotification contains the business logic for dealing with EVE Online notifications.
//
// It provides a service for rendering notifications titles and bodies.
package evenotification

import (
	"context"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type EVEUniverse interface {
	GetOrCreateEntityESI(ctx context.Context, id int64) (*app.EveEntity, error)
	GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error)
	GetOrCreateMoonESI(ctx context.Context, id int64) (*app.EveMoon, error)
	GetOrCreatePlanetESI(ctx context.Context, id int64) (*app.EvePlanet, error)
	GetOrCreateSolarSystemESI(ctx context.Context, id int64) (*app.EveSolarSystem, error)
	GetOrCreateTypeESI(ctx context.Context, id int64) (*app.EveType, error)
	ToEntities(ctx context.Context, ids set.Set[int64]) (map[int64]*app.EveEntity, error)
}

// notificationRenderer represents the interface every notification renderer needs to confirm with.
type notificationRenderer interface {
	// entityIDs returns the Entity IDs used by a notification (if any).
	entityIDs(text string) (set.Set[int64], error)
	// render returns the rendered title and body for a goesi.
	render(ctx context.Context, text string, _ time.Time) (string, string, error)
	// setEveUniverse initialized access to the EveUniverseService service and must be called before render().
	setEveUniverse(EVEUniverse)
}

// baseRenderer represents the base renderer for all notification types.
//
// Each notification type has a renderer which can produce the title and string for a goesi.
// In addition the renderer can return the Entity IDs of a notification,
// which allows refetching Entities for multiple notifications in bulk before rendering.
//
// All rendered should embed baseRenderer and implement the render method.
// Renderers that want to return Entity IDs must also overwrite entityIDs.
type baseRenderer struct {
	eus EVEUniverse
}

func (br *baseRenderer) setEveUniverse(eus EVEUniverse) {
	br.eus = eus
}

// entityIDs returns the Entity IDs used by a notification (if any).
//
// Must be overwritten by a notification rendered that want to return IDs.
func (br baseRenderer) entityIDs(_ string) (set.Set[int64], error) {
	return set.Set[int64]{}, nil
}

// EVENotificationService is a service for rendering notifications.
type EVENotificationService struct {
	eus EVEUniverse
}

func New(eus EVEUniverse) *EVENotificationService {
	s := &EVENotificationService{eus: eus}
	return s
}

// EntityIDs returns the Entity IDs used in a goesi.
// This is useful to resolve Entity IDs in bulk for all notifications,
// before rendering them one by one.
// Returns an empty set when notification does not use Entity IDs.
// Returns [app.ErrNotFound] for unsupported notification types.
func (s *EVENotificationService) EntityIDs(nt app.EveNotificationType, text optional.Optional[string]) (set.Set[int64], error) {
	v, ok := text.Value()
	if !ok {
		return set.Set[int64]{}, nil
	}
	r, found := s.makeRenderer(app.EveNotificationType(nt))
	if !found {
		return set.Set[int64]{}, app.ErrNotFound
	}
	return r.entityIDs(v)
}

// RenderESI renders title and body for all supported notification types and returns them.
// Returns [app.ErrNotFound] for unsupported notification types.
func (s *EVENotificationService) RenderESI(ctx context.Context, nt app.EveNotificationType, text optional.Optional[string], timestamp time.Time) (title string, body string, err error) {
	v, ok := text.Value()
	if !ok {
		return "", "", nil
	}
	r, found := s.makeRenderer(app.EveNotificationType(nt))
	if !found {
		return "", "", app.ErrNotFound
	}
	title, body, err = r.render(ctx, v, timestamp)
	if err != nil {
		return "", "", err
	}
	return title, body, nil
}

func (s *EVENotificationService) makeRenderer(nt app.EveNotificationType) (notificationRenderer, bool) {
	var r notificationRenderer
	switch nt {
	// bounty
	case app.BountyClaimMsg:
		r = new(bountyClaimMsg)
	case app.BountyESSShared:
		r = new(bountyESSShared)
	case app.BountyESSTaken:
		r = new(bountyESSTaken)
	case app.BountyPlacedAlliance:
		r = new(bountyPlacedAlliance)
	case app.BountyPlacedChar:
		r = new(bountyPlacedChar)
	case app.BountyPlacedCorp:
		r = new(bountyPlacedCorp)
	case app.BountyYourBountyClaimed:
		r = new(bountyYourBountyClaimed)
	// clone
	case app.CloneActivationMsg:
		r = new(cloneActivationMsg)
	case app.CloneActivationMsg2:
		r = new(cloneActivationMsg2)
	case app.CloneMovedMsg:
		r = new(cloneMovedMsg)
	case app.CloneRevokedMsg2:
		r = new(cloneRevokedMsg2)
	case app.JumpCloneDeletedMsg1:
		r = new(jumpCloneDeletedMsg1)
	case app.JumpCloneDeletedMsg2:
		r = new(jumpCloneDeletedMsg2)
	// contact
	case app.ContactAdd:
		r = new(contactAdd)
	case app.ContactEdit:
		r = new(contactEdit)
	// customs
	case app.ContainerPasswordMsg:
		r = new(containerPasswordMsg)
	case app.CustomsMsg:
		r = new(customsMsg)
	// game time
	case app.GameTimeAdded:
		r = new(gameTimeAdded)
	case app.GameTimeReceived:
		r = new(gameTimeReceived)
	case app.GameTimeSent:
		r = new(gameTimeSent)
	// insurance
	case app.InsuranceExpirationMsg:
		r = new(insuranceExpirationMsg)
	case app.InsuranceFirstShipMsg:
		r = new(insuranceFirstShipMsg)
	case app.InsuranceInvalidatedMsg:
		r = new(insuranceInvalidatedMsg)
	case app.InsuranceIssuedMsg:
		r = new(insuranceIssuedMsg)
	case app.InsurancePayoutMsg:
		r = new(insurancePayoutMsg)
	// kill right
	case app.KillReportFinalBlow:
		r = new(killReportFinalBlow)
	case app.KillReportVictim:
		r = new(killReportVictim)
	case app.KillRightAvailable:
		r = new(killRightAvailable)
	case app.KillRightAvailableOpen:
		r = new(killRightAvailableOpen)
	case app.KillRightEarned:
		r = new(killRightEarned)
	case app.KillRightUnavailable:
		r = new(killRightUnavailable)
	case app.KillRightUnavailableOpen:
		r = new(killRightUnavailableOpen)
	case app.KillRightUsed:
		r = new(killRightUsed)
	// faction warfare
	case app.FacWarCorpJoinRequestMsg:
		r = new(facWarCorpJoinRequestMsg)
	case app.FacWarCorpJoinWithdrawMsg:
		r = new(facWarCorpJoinWithdrawMsg)
	case app.FacWarCorpLeaveRequestMsg:
		r = new(facWarCorpLeaveRequestMsg)
	case app.FacWarCorpLeaveWithdrawMsg:
		r = new(facWarCorpLeaveWithdrawMsg)
	case app.FacWarLPDisqualifiedEvent:
		r = new(facWarLPDisqualifiedEvent)
	case app.FacWarLPDisqualifiedKill:
		r = new(facWarLPDisqualifiedKill)
	case app.FacWarLPPayoutEvent:
		r = new(facWarLPPayoutEvent)
	case app.FacWarLPPayoutKill:
		r = new(facWarLPPayoutKill)
	case app.FWAllianceWarningMsg:
		r = new(fwAllianceWarningMsg)
	case app.FWCharRankGainMsg:
		r = new(fwCharRankGainMsg)
	case app.FWCharRankLossMsg:
		r = new(fwCharRankLossMsg)
	case app.FWCorpJoinMsg:
		r = new(fwCorpJoinMsg)
	case app.FWCorpKickMsg:
		r = new(fwCorpKickMsg)
	case app.FWCorpLeaveMsg:
		r = new(fwCorpLeaveMsg)
	case app.FWCorpWarningMsg:
		r = new(fwCorpWarningMsg)
	// miscellaneous
	case app.IncursionCompletedMsg:
		r = new(incursionCompletedMsg)
	case app.IndustryTeamAuctionLost:
		r = new(industryTeamAuctionLost)
	case app.LocateCharMsg:
		r = new(locateCharMsg)
	case app.MissionOfferExpirationMsg:
		r = new(missionOfferExpirationMsg)
	case app.OldLscMessages:
		r = new(oldLscMessages)
	case app.OperationFinished:
		r = new(operationFinished)
	case app.ReimbursementMsg:
		r = new(reimbursementMsg)
	case app.ResearchMissionAvailableMsg:
		r = new(researchMissionAvailableMsg)
	case app.SeasonalChallengeCompleted:
		r = new(seasonalChallengeCompleted)
	// billing
	case app.AllMaintenanceBillMsg:
		r = new(allMaintenanceBillMsg)
	case app.BillOutOfMoneyMsg:
		r = new(billOutOfMoneyMsg)
	case app.BillPaidCorpAllMsg:
		r = new(billPaidCorpAllMsg)
	case app.CorpAllBillMsg:
		r = new(corpAllBillMsg)
	case app.InfrastructureHubBillAboutToExpire:
		r = new(infrastructureHubBillAboutToExpire)
	case app.IHubDestroyedByBillFailure:
		r = new(iHubDestroyedByBillFailure)
	// corporate
	case app.CharAppAcceptMsg:
		r = new(charAppAcceptMsg)
	case app.CharAppRejectMsg:
		r = new(charAppRejectMsg)
	case app.CharAppWithdrawMsg:
		r = new(charAppWithdrawMsg)
	case app.CharLeftCorpMsg:
		r = new(charLeftCorpMsg)
	case app.CorpAppInvitedMsg:
		r = new(corpAppInvitedMsg)
	case app.CorpAppNewMsg:
		r = new(corpAppNewMsg)
	case app.CorpAppRejectCustomMsg:
		r = new(corpAppRejectCustomMsg)
	case app.BuddyConnectContactAdd:
		r = new(buddyConnectContactAdd)
	case app.CharMedalMsg:
		r = new(charMedalMsg)
	case app.CharTerminationMsg:
		r = new(charTerminationMsg)
	case app.CorpAppAcceptMsg:
		r = new(corpAppAcceptMsg)
	case app.CorpAppRejectMsg:
		r = new(corpAppRejectMsg)
	case app.CorpKicked:
		r = new(corpKicked)
	case app.CorpNewCEOMsg:
		r = new(corpNewCEOMsg)
	case app.CorpVoteMsg:
		r = new(corpVoteMsg)
	case app.CorpVoteCEORevokedMsg:
		r = new(corpVoteCEORevokedMsg)
	case app.CorpDividendMsg:
		r = new(corpDividendMsg)
	case app.CorpLiquidationMsg:
		r = new(corpLiquidationMsg)
	case app.CorpNewsMsg:
		r = new(corpNewsMsg)
	case app.CorpTaxChangeMsg:
		r = new(corpTaxChangeMsg)
	case app.CorpFriendlyFireDisableTimerCompleted:
		r = new(corpFriendlyFireDisableTimerCompleted)
	case app.CorpFriendlyFireDisableTimerStarted:
		r = new(corpFriendlyFireDisableTimerStarted)
	case app.CorpFriendlyFireEnableTimerCompleted:
		r = new(corpFriendlyFireEnableTimerCompleted)
	case app.CorpFriendlyFireEnableTimerStarted:
		r = new(corpFriendlyFireEnableTimerStarted)
	case app.GiftReceived:
		r = new(giftReceived)
	case app.CorpBecameWarEligible:
		r = new(corpBecameWarEligible)
	case app.CorpNoLongerWarEligible:
		r = new(corpNoLongerWarEligible)
	case app.CorporationGoalCreated:
		r = new(corporationGoalCreated)
	case app.CorporationGoalCompleted:
		r = new(corporationGoalCompleted)
	case app.CorporationGoalClosed:
		r = new(corporationGoalClosed)
	// moonmining
	case app.MoonminingAutomaticFracture:
		r = new(moonminingAutomaticFracture)
	case app.MoonminingExtractionCancelled:
		r = new(moonminingExtractionCancelled)
	case app.MoonminingExtractionFinished:
		r = new(moonminingExtractionFinished)
	case app.MoonminingExtractionStarted:
		r = new(moonminingExtractionStarted)
	case app.MoonminingLaserFired:
		r = new(moonminingLaserFired)
	// orbital
	case app.OrbitalAttacked:
		r = new(orbitalAttacked)
	case app.OrbitalReinforced:
		r = new(orbitalReinforced)
	// structures
	case app.MercenaryDenAttacked:
		r = new(mercenaryDenAttacked)
	case app.MercenaryDenReinforced:
		r = new(mercenaryDenReinforced)
	case app.OwnershipTransferred:
		r = new(ownershipTransferred)
	case app.StructureAnchoring:
		r = new(structureAnchoring)
	case app.StructureDestroyed:
		r = new(structureDestroyed)
	case app.StructureFuelAlert:
		r = new(structureFuelAlert)
	case app.StructureImpendingAbandonmentAssetsAtRisk:
		r = new(structureImpendingAbandonmentAssetsAtRisk)
	case app.StructureItemsDelivered:
		r = new(structureItemsDelivered)
	case app.StructureItemsMovedToSafety:
		r = new(structureItemsMovedToSafety)
	case app.StructureLostArmor:
		r = new(structureLostArmor)
	case app.StructureLostShields:
		r = new(structureLostShields)
	case app.StructureOnline:
		r = new(structureOnline)
	case app.StructureServicesOffline:
		r = new(structureServicesOffline)
	case app.StructuresReinforcementChanged:
		r = new(structuresReinforcementChanged)
	case app.StructureUnanchoring:
		r = new(structureUnanchoring)
	case app.StructureUnderAttack:
		r = new(structureUnderAttack)
	case app.StructureWentHighPower:
		r = new(structureWentHighPower)
	case app.StructureWentLowPower:
		r = new(structureWentLowPower)
	case app.SkyhookLostShields:
		r = new(skyhookLostShields)
	case app.SkyhookUnderAttack:
		r = new(skyhookUnderAttack)
	case app.SkyhookDeployed:
		r = new(skyhookDeployed)
	case app.SkyhookDestroyed:
		r = new(skyhookDestroyed)
	case app.SkyhookOnline:
		r = new(skyhookOnline)
	case app.StructureLowReagentsAlert:
		r = new(structureLowReagentsAlert)
	case app.StructureNoReagentsAlert:
		r = new(structureNoReagentsAlert)
	case app.StructuresJobsPaused:
		r = new(structuresJobsPaused)
	case app.StationServiceDisabled:
		r = new(stationServiceDisabled)
	case app.StationServiceEnabled:
		r = new(stationServiceEnabled)
	// sov
	case app.EntosisCaptureStarted:
		r = new(entosisCaptureStarted)
	case app.AllAnchoringMsg:
		r = new(allAnchoringMsg)
	case app.SovAllClaimAcquiredMsg:
		r = new(sovAllClaimAcquiredMsg)
	case app.SovAllClaimLostMsg:
		r = new(sovAllClaimLostMsg)
	case app.SovCommandNodeEventStarted:
		r = new(sovCommandNodeEventStarted)
	case app.SovStructureDestroyed:
		r = new(sovStructureDestroyed)
	case app.SovStructureReinforced:
		r = new(sovStructureReinforced)
	case app.SovereigntyIHDamageMsg:
		r = new(sovereigntyIHDamageMsg)
	case app.SovereigntySBUDamageMsg:
		r = new(sovereigntySBUDamageMsg)
	case app.SovereigntyTCUDamageMsg:
		r = new(sovereigntyTCUDamageMsg)
	case app.SovStationEnteredFreeport:
		r = new(sovStationEnteredFreeport)
	case app.SovStructureSelfDestructCancel:
		r = new(sovStructureSelfDestructCancel)
	case app.SovStructureSelfDestructFinished:
		r = new(sovStructureSelfDestructFinished)
	case app.SovStructureSelfDestructRequested:
		r = new(sovStructureSelfDestructRequested)
	// tower
	case app.TowerAlertMsg:
		r = new(towerAlertMsg)
	case app.TowerResourceAlertMsg:
		r = new(towerResourceAlertMsg)
	// war
	case app.AllWarSurrenderMsg:
		r = new(allWarSurrenderMsg)
	case app.MutualWarInviteSent:
		r = new(mutualWarInviteSent)
	case app.CorpWarSurrenderMsg:
		r = new(corpWarSurrenderMsg)
	case app.DeclareWar:
		r = new(declareWar)
	case app.WarAdopted:
		r = new(warAdopted)
	case app.WarDeclared:
		r = new(warDeclared)
	case app.WarHQRemovedFromSpace:
		r = new(warHQRemovedFromSpace)
	case app.WarInherited:
		r = new(warInherited)
	case app.WarInvalid:
		r = new(warInvalid)
	case app.WarRetractedByConcord:
		r = new(warRetractedByConcord)
	case app.WarConcordInvalidates:
		r = new(warConcordInvalidates)
	case app.WarRetracted:
		r = new(warRetracted)
	case app.AllianceWarDeclaredV2:
		r = new(allianceWarDeclaredV2)
	case app.CorpWarDeclaredV2:
		r = new(corpWarDeclaredV2)
	case app.AcceptedAlly:
		r = new(acceptedAlly)
	case app.AcceptedSurrender:
		r = new(acceptedSurrender)
	case app.AllianceCapitalChanged:
		r = new(allianceCapitalChanged)
	case app.AllWarCorpJoinedAllianceMsg:
		r = new(allWarCorpJoinedAllianceMsg)
	case app.AllWarDeclaredMsg:
		r = new(allWarDeclaredMsg)
	case app.AllWarInvalidatedMsg:
		r = new(allWarInvalidatedMsg)
	case app.AllWarRetractedMsg:
		r = new(allWarRetractedMsg)
	case app.AllyContractCancelled:
		r = new(allyContractCancelled)
	case app.AllyJoinedWarAggressorMsg:
		r = new(allyJoinedWarAggressorMsg)
	case app.AllyJoinedWarAllyMsg:
		r = new(allyJoinedWarAllyMsg)
	case app.AllyJoinedWarDefenderMsg:
		r = new(allyJoinedWarDefenderMsg)
	case app.MadeWarMutual:
		r = new(madeWarMutual)
	case app.OfferedSurrender:
		r = new(offeredSurrender)
	case app.OfferedToAlly:
		r = new(offeredToAlly)
	case app.RetractsWar:
		r = new(retractsWar)
	case app.WarAllyOfferDeclinedMsg:
		r = new(warAllyOfferDeclinedMsg)
	case app.WarSurrenderDeclinedMsg:
		r = new(warSurrenderDeclinedMsg)
	case app.WarSurrenderOfferMsg:
		r = new(warSurrenderOfferMsg)
	case app.CorpWarDeclaredMsg:
		r = new(corpWarDeclaredMsg)
	case app.CorpWarFightingLegalMsg:
		r = new(corpWarFightingLegalMsg)
	case app.CorpWarInvalidatedMsg:
		r = new(corpWarInvalidatedMsg)
	case app.CorpWarRetractedMsg:
		r = new(corpWarRetractedMsg)
	case app.MercOfferedNegotiationMsg:
		r = new(mercOfferedNegotiationMsg)
	case app.MercOfferRetractedMsg:
		r = new(mercOfferRetractedMsg)
	default:
		return nil, false
	}
	r.setEveUniverse(s.eus)
	return r, true
}
