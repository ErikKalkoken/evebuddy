// Package evenotification contains the business logic for dealing with EVE Online notifications.
//
// It defines the notification types and related categories
// and provides a service for rendering notifications titles and bodies.
package evenotification

import (
	"context"
	"fmt"
	"strings"
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
	// billing
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
	case app.CorpAppAcceptMsg:
		r = new(corpAppAcceptMsg)
	case app.CorpAppInvitedMsg:
		r = new(corpAppInvitedMsg)
	case app.CorpAppNewMsg:
		r = new(corpAppNewMsg)
	case app.CorpAppRejectCustomMsg:
		r = new(corpAppRejectCustomMsg)
	case app.CorpAppRejectMsg:
		r = new(corpAppRejectMsg)
	case app.CorpDividendMsg:
		r = new(corpDividendMsg)
	case app.CorpFriendlyFireDisableTimerCompleted:
		r = new(corpFriendlyFireDisableTimerCompleted)
	case app.CorpFriendlyFireDisableTimerStarted:
		r = new(corpFriendlyFireDisableTimerStarted)
	case app.CorpFriendlyFireEnableTimerCompleted:
		r = new(corpFriendlyFireEnableTimerCompleted)
	case app.CorpFriendlyFireEnableTimerStarted:
		r = new(corpFriendlyFireEnableTimerStarted)
	case app.CorpKicked:
		r = new(corpKicked)
	case app.CorpLiquidationMsg:
		r = new(corpLiquidationMsg)
	case app.CorpNewCEOMsg:
		r = new(corpNewCEOMsg)
	case app.CorpNewsMsg:
		r = new(corpNewsMsg)
	case app.CorpOfficeExpirationMsg:
		r = new(corpOfficeExpirationMsg)
	case app.CorpStructLostMsg:
		r = new(corpStructLostMsg)
	case app.CorpTaxChangeMsg:
		r = new(corpTaxChangeMsg)
	case app.CorpVoteCEORevokedMsg:
		r = new(corpVoteCEORevokedMsg)
	case app.CorpVoteMsg:
		r = new(corpVoteMsg)
	case app.CorporationGoalClosed:
		r = new(corporationGoalClosed)
	case app.CorporationGoalCompleted:
		r = new(corporationGoalCompleted)
	case app.CorporationGoalCreated:
		r = new(corporationGoalCreated)
	case app.CorporationGoalNameChange:
		r = new(corporationGoalNameChange)
	case app.CorporationLeft:
		r = new(corporationLeft)
	case app.OfficeLeaseCanceledInsufficientStandings:
		r = new(officeLeaseCanceledInsufficientStandings)
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
	case app.StructureCourierContractChanged:
		r = new(structureCourierContractChanged)
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
	case app.StructureLowReagentsAlert:
		r = new(structureLowReagentsAlert)
	case app.StructureNoReagentsAlert:
		r = new(structureNoReagentsAlert)
	case app.StructureOnline:
		r = new(structureOnline)
	case app.StructurePaintPurchased:
		r = new(structurePaintPurchased)
	case app.StructureServicesOffline:
		r = new(structureServicesOffline)
	case app.StructuresReinforcementChanged:
		r = new(structuresReinforcementChanged)
	case app.StructuresJobsCancelled:
		r = new(structuresJobsCancelled)
	case app.StructuresJobsPaused:
		r = new(structuresJobsPaused)
	case app.StructureUnanchoring:
		r = new(structureUnanchoring)
	case app.StructureUnderAttack:
		r = new(structureUnderAttack)
	case app.StructureWentHighPower:
		r = new(structureWentHighPower)
	case app.StructureWentLowPower:
		r = new(structureWentLowPower)
	// sov
	case app.EntosisCaptureStarted:
		r = new(entosisCaptureStarted)
	case app.SovAllClaimAcquiredMsg:
		r = new(sovAllClaimAcquiredMsg)
	case app.SovAllClaimLostMsg:
		r = new(sovAllClaimLostMsg)
	case app.SovCommandNodeEventStarted:
		r = new(sovCommandNodeEventStarted)
	case app.SovCorpBillLateMsg:
		r = new(sovCorpBillLateMsg)
	case app.SovCorpClaimFailMsg:
		r = new(sovCorpClaimFailMsg)
	case app.SovDisruptorMsg:
		r = new(sovDisruptorMsg)
	case app.SovStationEnteredFreeport:
		r = new(sovStationEnteredFreeport)
	case app.SovStructureDestroyed:
		r = new(sovStructureDestroyed)
	case app.SovStructureReinforced:
		r = new(sovStructureReinforced)
	case app.SovStructureSelfDestructCancel:
		r = new(sovStructureSelfDestructCancel)
	case app.SovStructureSelfDestructFinished:
		r = new(sovStructureSelfDestructFinished)
	case app.SovStructureSelfDestructRequested:
		r = new(sovStructureSelfDestructRequested)
	case app.SovereigntyIHDamageMsg:
		r = new(sovereigntyIHDamageMsg)
	case app.SovereigntySBUDamageMsg:
		r = new(sovereigntySBUDamageMsg)
	case app.SovereigntyTCUDamageMsg:
		r = new(sovereigntyTCUDamageMsg)
	// tower
	case app.TowerAlertMsg:
		r = new(towerAlertMsg)
	case app.TowerResourceAlertMsg:
		r = new(towerResourceAlertMsg)
	// faction warfare
	case app.FWAllianceKickMsg:
		r = new(fwAllianceKickMsg)
	case app.FWAllianceWarningMsg:
		r = new(fwAllianceWarningMsg)
	case app.FWCharKickMsg:
		r = new(fwCharKickMsg)
	case app.FWCharRankGainMsg:
		r = new(fwCharRankGainMsg)
	case app.FWCharRankLossMsg:
		r = new(fwCharRankLossMsg)
	case app.FWCharWarningMsg:
		r = new(fwCharWarningMsg)
	case app.FWCorpJoinMsg:
		r = new(fwCorpJoinMsg)
	case app.FWCorpKickMsg:
		r = new(fwCorpKickMsg)
	case app.FWCorpLeaveMsg:
		r = new(fwCorpLeaveMsg)
	case app.FWCorpWarningMsg:
		r = new(fwCorpWarningMsg)
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
	// kill reports and kill rights
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
	// skyhook and station
	case app.SkyhookDeployed:
		r = new(skyhookDeployed)
	case app.SkyhookDestroyed:
		r = new(skyhookDestroyed)
	case app.SkyhookLostShields:
		r = new(skyhookLostShields)
	case app.SkyhookOnline:
		r = new(skyhookOnline)
	case app.SkyhookUnderAttack:
		r = new(skyhookUnderAttack)
	case app.StationAggressionMsg1:
		r = new(stationAggressionMsg)
	case app.StationAggressionMsg2:
		r = new(stationAggressionMsg)
	case app.StationConquerMsg:
		r = new(stationConquerMsg)
	case app.StationServiceDisabled:
		r = new(stationServiceDisabled)
	case app.StationServiceEnabled:
		r = new(stationServiceEnabled)
	case app.StationStateChangeMsg:
		r = new(stationStateChangeMsg)
	// clone and character
	case app.AgentRetiredTrigravian:
		r = new(agentRetiredTrigravian)
	case app.AllMaintenanceBillMsg:
		r = new(allMaintenanceBillMsg)
	case app.CharMedalMsg:
		r = new(charMedalMsg)
	case app.CharTerminationMsg:
		r = new(charTerminationMsg)
	case app.CloneActivationMsg:
		r = new(cloneActivationMsg)
	case app.CloneActivationMsg2:
		r = new(cloneActivationMsg2)
	case app.CloneMovedMsg:
		r = new(cloneMovedMsg)
	case app.CloneRevokedMsg1:
		r = new(cloneRevokedMsg1)
	case app.CloneRevokedMsg2:
		r = new(cloneRevokedMsg2)
	case app.JumpCloneDeletedMsg1:
		r = new(jumpCloneDeletedMsg1)
	case app.JumpCloneDeletedMsg2:
		r = new(jumpCloneDeletedMsg2)
	// miscellaneous
	case app.AllAnchoringMsg:
		r = new(allAnchoringMsg)
	case app.AllStructureInvulnerableMsg:
		r = new(allStructureInvulnerableMsg)
	case app.AllStructVulnerableMsg:
		r = new(allStructVulnerableMsg)
	case app.AllianceCapitalChanged:
		r = new(allianceCapitalChanged)
	case app.BattlePunishFriendlyFire:
		r = new(battlPunishFriendlyFire)
	case app.BuddyConnectContactAdd:
		r = new(buddyConnectContactAdd)
	case app.CombatOperationFinished:
		r = new(combatOperationFinished)
	case app.ContactAdd:
		r = new(contactAdd)
	case app.ContactEdit:
		r = new(contactEdit)
	case app.ContainerPasswordMsg:
		r = new(containerPasswordMsg)
	case app.ContractRegionChangedToPochven:
		r = new(contractRegionChangedToPochven)
	case app.CustomsMsg:
		r = new(customsMsg)
	case app.DistrictAttacked:
		r = new(districtAttacked)
	case app.DustAppAcceptedMsg:
		r = new(dustAppAcceptedMsg)
	case app.ESSMainBankLink:
		r = new(essMainBankLink)
	case app.ExpertSystemExpired:
		r = new(expertSystemExpired)
	case app.ExpertSystemExpiryImminent:
		r = new(expertSystemExpiryImminent)
	case app.GameTimeAdded:
		r = new(gameTimeAdded)
	case app.GameTimeReceived:
		r = new(gameTimeReceived)
	case app.GameTimeSent:
		r = new(gameTimeSent)
	case app.GiftReceived:
		r = new(giftReceived)
	case app.IncursionCompletedMsg:
		r = new(incursionCompletedMsg)
	case app.IndustryOperationFinished:
		r = new(industryOperationFinished)
	case app.IndustryTeamAuctionLost:
		r = new(industryTeamAuctionLost)
	case app.IndustryTeamAuctionWon:
		r = new(industryTeamAuctionWon)
	case app.InvasionCompletedMsg:
		r = new(invasionCompletedMsg)
	case app.InvasionSystemLogin:
		r = new(invasionSystemLogin)
	case app.InvasionSystemStart:
		r = new(invasionSystemStart)
	case app.LocateCharMsg:
		r = new(locateCharMsg)
	case app.LPAutoRedeemed:
		r = new(lpAutoRedeemed)
	case app.MissionCanceledTriglavian:
		r = new(missionCanceledTriglavian)
	case app.MissionOfferExpirationMsg:
		r = new(missionOfferExpirationMsg)
	case app.MissionTimeoutMsg:
		r = new(missionTimeoutMsg)
	case app.NPCStandingsGained:
		r = new(npcStandingsGained)
	case app.NPCStandingsLost:
		r = new(npcStandingsLost)
	case app.OldLscMessages:
		r = new(oldLscMessages)
	case app.OperationFinished:
		r = new(operationFinished)
	case app.RaffleCreated:
		r = new(raffleCreated)
	case app.RaffleExpired:
		r = new(raffleExpired)
	case app.RaffleFinished:
		r = new(raffleFinished)
	case app.ReimbursementMsg:
		r = new(reimbursementMsg)
	case app.ResearchMissionAvailableMsg:
		r = new(researchMissionAvailableMsg)
	case app.SPAutoRedeemed:
		r = new(spAutoRedeemed)
	case app.SeasonalChallengeCompleted:
		r = new(seasonalChallengeCompleted)
	case app.SkinSequencingCompleted:
		r = new(skinSequencingCompleted)
	case app.StoryLineMissionAvailableMsg:
		r = new(storyLineMissionAvailableMsg)
	case app.TransactionReversalMsg:
		r = new(transactionReversalMsg)
	case app.TutorialMsg:
		r = new(tutorialMsg)
	// war
	case app.AllWarSurrenderMsg:
		r = new(allWarSurrenderMsg)
	case app.AcceptedAlly:
		r = new(acceptedAlly)
	case app.AcceptedSurrender:
		r = new(acceptedSurrender)
	case app.AllWarCorpJoinedAllianceMsg:
		r = new(allWarCorpJoinedAllianceMsg)
	case app.AllWarDeclaredMsg:
		r = new(allWarDeclaredMsg)
	case app.AllWarInvalidatedMsg:
		r = new(allWarInvalidatedMsg)
	case app.AllWarRetractedMsg:
		r = new(allWarRetractedMsg)
	case app.AllianceWarDeclaredV2:
		r = new(allianceWarDeclaredV2)
	case app.AllyContractCancelled:
		r = new(allyContractCancelled)
	case app.AllyJoinedWarAggressorMsg:
		r = new(allyJoinedWarAggressorMsg)
	case app.AllyJoinedWarAllyMsg:
		r = new(allyJoinedWarAllyMsg)
	case app.AllyJoinedWarDefenderMsg:
		r = new(allyJoinedWarDefenderMsg)
	case app.CorpBecameWarEligible:
		r = new(corpBecameWarEligible)
	case app.CorpNoLongerWarEligible:
		r = new(corpNoLongerWarEligible)
	case app.CorpWarDeclaredMsg:
		r = new(corpWarDeclaredMsg)
	case app.CorpWarDeclaredV2:
		r = new(corpWarDeclaredV2)
	case app.CorpWarFightingLegalMsg:
		r = new(corpWarFightingLegalMsg)
	case app.CorpWarInvalidatedMsg:
		r = new(corpWarInvalidatedMsg)
	case app.CorpWarRetractedMsg:
		r = new(corpWarRetractedMsg)
	case app.CorpWarSurrenderMsg:
		r = new(corpWarSurrenderMsg)
	case app.DeclareWar:
		r = new(declareWar)
	case app.MadeWarMutual:
		r = new(madeWarMutual)
	case app.MercOfferedNegotiationMsg:
		r = new(mercOfferedNegotiationMsg)
	case app.MercOfferRetractedMsg:
		r = new(mercOfferRetractedMsg)
	case app.MutualWarExpired:
		r = new(mutualWarExpired)
	case app.MutualWarInviteAccepted:
		r = new(mutualWarInviteAccepted)
	case app.MutualWarInviteRejected:
		r = new(mutualWarInviteRejected)
	case app.MutualWarInviteSent:
		r = new(mutualWarInviteSent)
	case app.OfferedSurrender:
		r = new(offeredSurrender)
	case app.OfferedToAlly:
		r = new(offeredToAlly)
	case app.OfferToAllyRetracted:
		r = new(offerToAllyRetracted)
	case app.RetractsWar:
		r = new(retractsWar)
	case app.WarAdopted:
		r = new(warAdopted)
	case app.WarAllyInherited:
		r = new(warAllyInherited)
	case app.WarAllyOfferDeclinedMsg:
		r = new(warAllyOfferDeclinedMsg)
	case app.WarConcordInvalidates:
		r = new(warConcordInvalidates)
	case app.WarDeclared:
		r = new(warDeclared)
	case app.WarEndedHqSecurityDrop:
		r = new(warEndedHqSecurityDrop)
	case app.WarHQRemovedFromSpace:
		r = new(warHQRemovedFromSpace)
	case app.WarInherited:
		r = new(warInherited)
	case app.WarInvalid:
		r = new(warInvalid)
	case app.WarRetracted:
		r = new(warRetracted)
	case app.WarRetractedByConcord:
		r = new(warRetractedByConcord)
	case app.WarSurrenderDeclinedMsg:
		r = new(warSurrenderDeclinedMsg)
	case app.WarSurrenderOfferMsg:
		r = new(warSurrenderOfferMsg)
	default:
		return nil, false
	}
	r.setEveUniverse(s.eus)
	return r, true
}

// fromLDAPTime converts an ldap time to golang time
func fromLDAPTime(ldapTime int64) time.Time {
	return time.Unix((ldapTime/10000000)-11644473600, 0).UTC()
}

// fromLDAPDuration converts an ldap duration to golang duration
func fromLDAPDuration(ldapDuration int64) time.Duration {
	return time.Duration(ldapDuration/10) * time.Microsecond
}

type dotlanType = uint

const (
	dotlanAlliance dotlanType = iota
	dotlanCorporation
	dotlanSolarSystem
	dotlanRegion
)

func makeDotLanProfileURL(name string, typ dotlanType) string {
	const baseURL = "https://evemaps.dotlan.net"
	var path string
	m := map[dotlanType]string{
		dotlanAlliance:    "alliance",
		dotlanCorporation: "corp",
		dotlanSolarSystem: "system",
		dotlanRegion:      "region",
	}
	path, ok := m[typ]
	if !ok {
		return name
	}
	name2 := strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s/%s/%s", baseURL, path, name2)
}

func makeSolarSystemLink(ess *app.EveSolarSystem) string {
	x := fmt.Sprintf(
		"%s (%s)",
		makeMarkDownLink(ess.Name, makeDotLanProfileURL(ess.Name, dotlanSolarSystem)),
		ess.Constellation.Region.Name,
	)
	return x
}

func makeCorporationLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanCorporation))
}

func makeAllianceLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanAlliance))
}

func makeEveWhoCharacterURL(id int64) string {
	return fmt.Sprintf("https://evewho.com/character/%d", id)
}

func makeEveEntityProfileLink(o *app.EveEntity) string {
	if o == nil {
		return "?"
	}
	var url string
	switch o.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(o.Name, dotlanAlliance)
	case app.EveEntityCharacter:
		url = makeEveWhoCharacterURL(o.ID)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(o.Name, dotlanCorporation)
	default:
		return o.Name
	}
	return makeMarkDownLink(o.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}
