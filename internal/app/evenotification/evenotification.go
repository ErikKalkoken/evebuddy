// Package evenotification contains the business logic for dealing with Eve Online notifications.
// It defines the notification types and related categories
// and provides a service for rendering notifications titles and bodies.
package evenotification

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type setInt32 = set.Set[int32]

// Type represents a notification type.
type Type string

const (
	AcceptedAlly                              Type = "AcceptedAlly"
	AcceptedSurrender                         Type = "AcceptedSurrender"
	AgentRetiredTrigravian                    Type = "AgentRetiredTrigravian"
	AllAnchoringMsg                           Type = "AllAnchoringMsg"
	AllMaintenanceBillMsg                     Type = "AllMaintenanceBillMsg"
	AllStructureInvulnerableMsg               Type = "AllStrucInvulnerableMsg"
	AllStructVulnerableMsg                    Type = "AllStructVulnerableMsg"
	AllWarCorpJoinedAllianceMsg               Type = "AllWarCorpJoinedAllianceMsg"
	AllWarDeclaredMsg                         Type = "AllWarDeclaredMsg"
	AllWarInvalidatedMsg                      Type = "AllWarInvalidatedMsg"
	AllWarRetractedMsg                        Type = "AllWarRetractedMsg"
	AllWarSurrenderMsg                        Type = "AllWarSurrenderMsg"
	AllianceCapitalChanged                    Type = "AllianceCapitalChanged"
	AllianceWarDeclaredV2                     Type = "AllianceWarDeclaredV2"
	AllyContractCancelled                     Type = "AllyContractCancelled"
	AllyJoinedWarAggressorMsg                 Type = "AllyJoinedWarAggressorMsg"
	AllyJoinedWarAllyMsg                      Type = "AllyJoinedWarAllyMsg"
	AllyJoinedWarDefenderMsg                  Type = "AllyJoinedWarDefenderMsg"
	BattlePunishFriendlyFire                  Type = "BattlePunishFriendlyFire"
	BillOutOfMoneyMsg                         Type = "BillOutOfMoneyMsg"
	BillPaidCorpAllMsg                        Type = "BillPaidCorpAllMsg"
	BountyClaimMsg                            Type = "BountyClaimMsg"
	BountyESSShared                           Type = "BountyESSShared"
	BountyESSTaken                            Type = "BountyESSTaken"
	BountyPlacedAlliance                      Type = "BountyPlacedAlliance"
	BountyPlacedChar                          Type = "BountyPlacedChar"
	BountyPlacedCorp                          Type = "BountyPlacedCorp"
	BountyYourBountyClaimed                   Type = "BountyYourBountyClaimed"
	BuddyConnectContactAdd                    Type = "BuddyConnectContactAdd"
	CharAppAcceptMsg                          Type = "CharAppAcceptMsg"
	CharAppRejectMsg                          Type = "CharAppRejectMsg"
	CharAppWithdrawMsg                        Type = "CharAppWithdrawMsg"
	CharLeftCorpMsg                           Type = "CharLeftCorpMsg"
	CharMedalMsg                              Type = "CharMedalMsg"
	CharTerminationMsg                        Type = "CharTerminationMsg"
	CloneActivationMsg                        Type = "CloneActivationMsg"
	CloneActivationMsg2                       Type = "CloneActivationMsg2"
	CloneMovedMsg                             Type = "CloneMovedMsg"
	CloneRevokedMsg1                          Type = "CloneRevokedMsg1"
	CloneRevokedMsg2                          Type = "CloneRevokedMsg2"
	CombatOperationFinished                   Type = "CombatOperationFinished"
	ContactAdd                                Type = "ContactAdd"
	ContactEdit                               Type = "ContactEdit"
	ContainerPasswordMsg                      Type = "ContainerPasswordMsg"
	ContractRegionChangedToPochven            Type = "ContractRegionChangedToPochven"
	CorpAllBillMsg                            Type = "CorpAllBillMsg"
	CorpAppAcceptMsg                          Type = "CorpAppAcceptMsg"
	CorpAppInvitedMsg                         Type = "CorpAppInvitedMsg"
	CorpAppNewMsg                             Type = "CorpAppNewMsg"
	CorpAppRejectCustomMsg                    Type = "CorpAppRejectCustomMsg"
	CorpAppRejectMsg                          Type = "CorpAppRejectMsg"
	CorpBecameWarEligible                     Type = "CorpBecameWarEligible"
	CorpDividendMsg                           Type = "CorpDividendMsg"
	CorpFriendlyFireDisableTimerCompleted     Type = "CorpFriendlyFireDisableTimerCompleted"
	CorpFriendlyFireDisableTimerStarted       Type = "CorpFriendlyFireDisableTimerStarted"
	CorpFriendlyFireEnableTimerCompleted      Type = "CorpFriendlyFireEnableTimerCompleted"
	CorpFriendlyFireEnableTimerStarted        Type = "CorpFriendlyFireEnableTimerStarted"
	CorpKicked                                Type = "CorpKicked"
	CorpLiquidationMsg                        Type = "CorpLiquidationMsg"
	CorpNewCEOMsg                             Type = "CorpNewCEOMsg"
	CorpNewsMsg                               Type = "CorpNewsMsg"
	CorpNoLongerWarEligible                   Type = "CorpNoLongerWarEligible"
	CorpOfficeExpirationMsg                   Type = "CorpOfficeExpirationMsg"
	CorpStructLostMsg                         Type = "CorpStructLostMsg"
	CorpTaxChangeMsg                          Type = "CorpTaxChangeMsg"
	CorpVoteCEORevokedMsg                     Type = "CorpVoteCEORevokedMsg"
	CorpVoteMsg                               Type = "CorpVoteMsg"
	CorpWarDeclaredMsg                        Type = "CorpWarDeclaredMsg"
	CorpWarDeclaredV2                         Type = "CorpWarDeclaredV2"
	CorpWarFightingLegalMsg                   Type = "CorpWarFightingLegalMsg"
	CorpWarInvalidatedMsg                     Type = "CorpWarInvalidatedMsg"
	CorpWarRetractedMsg                       Type = "CorpWarRetractedMsg"
	CorpWarSurrenderMsg                       Type = "CorpWarSurrenderMsg"
	CorporationGoalClosed                     Type = "CorporationGoalClosed"
	CorporationGoalCompleted                  Type = "CorporationGoalCompleted"
	CorporationGoalCreated                    Type = "CorporationGoalCreated"
	CorporationGoalNameChange                 Type = "CorporationGoalNameChange"
	CorporationLeft                           Type = "CorporationLeft"
	CustomsMsg                                Type = "CustomsMsg"
	DeclareWar                                Type = "DeclareWar"
	DistrictAttacked                          Type = "DistrictAttacked"
	DustAppAcceptedMsg                        Type = "DustAppAcceptedMsg"
	ESSMainBankLink                           Type = "ESSMainBankLink"
	EntosisCaptureStarted                     Type = "EntosisCaptureStarted"
	ExpertSystemExpired                       Type = "ExpertSystemExpired"
	ExpertSystemExpiryImminent                Type = "ExpertSystemExpiryImminent"
	FWAllianceKickMsg                         Type = "FWAllianceKickMsg"
	FWAllianceWarningMsg                      Type = "FWAllianceWarningMsg"
	FWCharKickMsg                             Type = "FWCharKickMsg"
	FWCharRankGainMsg                         Type = "FWCharRankGainMsg"
	FWCharRankLossMsg                         Type = "FWCharRankLossMsg"
	FWCharWarningMsg                          Type = "FWCharWarningMsg"
	FWCorpJoinMsg                             Type = "FWCorpJoinMsg"
	FWCorpKickMsg                             Type = "FWCorpKickMsg"
	FWCorpLeaveMsg                            Type = "FWCorpLeaveMsg"
	FWCorpWarningMsg                          Type = "FWCorpWarningMsg"
	FacWarCorpJoinRequestMsg                  Type = "FacWarCorpJoinRequestMsg"
	FacWarCorpJoinWithdrawMsg                 Type = "FacWarCorpJoinWithdrawMsg"
	FacWarCorpLeaveRequestMsg                 Type = "FacWarCorpLeaveRequestMsg"
	FacWarCorpLeaveWithdrawMsg                Type = "FacWarCorpLeaveWithdrawMsg"
	FacWarLPDisqualifiedEvent                 Type = "FacWarLPDisqualifiedEvent"
	FacWarLPDisqualifiedKill                  Type = "FacWarLPDisqualifiedKill"
	FacWarLPPayoutEvent                       Type = "FacWarLPPayoutEvent"
	FacWarLPPayoutKill                        Type = "FacWarLPPayoutKill"
	GameTimeAdded                             Type = "GameTimeAdded"
	GameTimeReceived                          Type = "GameTimeReceived"
	GameTimeSent                              Type = "GameTimeSent"
	GiftReceived                              Type = "GiftReceived"
	IHubDestroyedByBillFailure                Type = "IHubDestroyedByBillFailure"
	IncursionCompletedMsg                     Type = "IncursionCompletedMsg"
	IndustryOperationFinished                 Type = "IndustryOperationFinished"
	IndustryTeamAuctionLost                   Type = "IndustryTeamAuctionLost"
	IndustryTeamAuctionWon                    Type = "IndustryTeamAuctionWon"
	InfrastructureHubBillAboutToExpire        Type = "InfrastructureHubBillAboutToExpire"
	InsuranceExpirationMsg                    Type = "InsuranceExpirationMsg"
	InsuranceFirstShipMsg                     Type = "InsuranceFirstShipMsg"
	InsuranceInvalidatedMsg                   Type = "InsuranceInvalidatedMsg"
	InsuranceIssuedMsg                        Type = "InsuranceIssuedMsg"
	InsurancePayoutMsg                        Type = "InsurancePayoutMsg"
	InvasionCompletedMsg                      Type = "InvasionCompletedMsg"
	InvasionSystemLogin                       Type = "InvasionSystemLogin"
	InvasionSystemStart                       Type = "InvasionSystemStart"
	JumpCloneDeletedMsg1                      Type = "JumpCloneDeletedMsg1"
	JumpCloneDeletedMsg2                      Type = "JumpCloneDeletedMsg2"
	KillReportFinalBlow                       Type = "KillReportFinalBlow"
	KillReportVictim                          Type = "KillReportVictim"
	KillRightAvailable                        Type = "KillRightAvailable"
	KillRightAvailableOpen                    Type = "KillRightAvailableOpen"
	KillRightEarned                           Type = "KillRightEarned"
	KillRightUnavailable                      Type = "KillRightUnavailable"
	KillRightUnavailableOpen                  Type = "KillRightUnavailableOpen"
	KillRightUsed                             Type = "KillRightUsed"
	LPAutoRedeemed                            Type = "LPAutoRedeemed"
	LocateCharMsg                             Type = "LocateCharMsg"
	MadeWarMutual                             Type = "MadeWarMutual"
	MercOfferRetractedMsg                     Type = "MercOfferRetractedMsg"
	MercOfferedNegotiationMsg                 Type = "MercOfferedNegotiationMsg"
	MissionCanceledTriglavian                 Type = "MissionCanceledTriglavian"
	MissionOfferExpirationMsg                 Type = "MissionOfferExpirationMsg"
	MissionTimeoutMsg                         Type = "MissionTimeoutMsg"
	MoonminingAutomaticFracture               Type = "MoonminingAutomaticFracture"
	MoonminingExtractionCancelled             Type = "MoonminingExtractionCancelled"
	MoonminingExtractionFinished              Type = "MoonminingExtractionFinished"
	MoonminingExtractionStarted               Type = "MoonminingExtractionStarted"
	MoonminingLaserFired                      Type = "MoonminingLaserFired"
	MutualWarExpired                          Type = "MutualWarExpired"
	MutualWarInviteAccepted                   Type = "MutualWarInviteAccepted"
	MutualWarInviteRejected                   Type = "MutualWarInviteRejected"
	MutualWarInviteSent                       Type = "MutualWarInviteSent"
	NPCStandingsGained                        Type = "NPCStandingsGained"
	NPCStandingsLost                          Type = "NPCStandingsLost"
	OfferToAllyRetracted                      Type = "OfferToAllyRetracted"
	OfferedSurrender                          Type = "OfferedSurrender"
	OfferedToAlly                             Type = "OfferedToAlly"
	OfficeLeaseCanceledInsufficientStandings  Type = "OfficeLeaseCanceledInsufficientStandings"
	OldLscMessages                            Type = "OldLscMessages"
	OperationFinished                         Type = "OperationFinished"
	OrbitalAttacked                           Type = "OrbitalAttacked"
	OrbitalReinforced                         Type = "OrbitalReinforced"
	OwnershipTransferred                      Type = "OwnershipTransferred"
	RaffleCreated                             Type = "RaffleCreated"
	RaffleExpired                             Type = "RaffleExpired"
	RaffleFinished                            Type = "RaffleFinished"
	ReimbursementMsg                          Type = "ReimbursementMsg"
	ResearchMissionAvailableMsg               Type = "ResearchMissionAvailableMsg"
	RetractsWar                               Type = "RetractsWar"
	SPAutoRedeemed                            Type = "SPAutoRedeemed"
	SeasonalChallengeCompleted                Type = "SeasonalChallengeCompleted"
	SkinSequencingCompleted                   Type = "SkinSequencingCompleted"
	SkyhookDeployed                           Type = "SkyhookDeployed"
	SkyhookDestroyed                          Type = "SkyhookDestroyed"
	SkyhookLostShields                        Type = "SkyhookLostShields"
	SkyhookOnline                             Type = "SkyhookOnline"
	SkyhookUnderAttack                        Type = "SkyhookUnderAttack"
	SovAllClaimAcquiredMsg                    Type = "SovAllClaimAquiredMsg"
	SovAllClaimLostMsg                        Type = "SovAllClaimLostMsg"
	SovCommandNodeEventStarted                Type = "SovCommandNodeEventStarted"
	SovCorpBillLateMsg                        Type = "SovCorpBillLateMsg"
	SovCorpClaimFailMsg                       Type = "SovCorpClaimFailMsg"
	SovDisruptorMsg                           Type = "SovDisruptorMsg"
	SovStationEnteredFreeport                 Type = "SovStationEnteredFreeport"
	SovStructureDestroyed                     Type = "SovStructureDestroyed"
	SovStructureReinforced                    Type = "SovStructureReinforced"
	SovStructureSelfDestructCancel            Type = "SovStructureSelfDestructCancel"
	SovStructureSelfDestructFinished          Type = "SovStructureSelfDestructFinished"
	SovStructureSelfDestructRequested         Type = "SovStructureSelfDestructRequested"
	SovereigntyIHDamageMsg                    Type = "SovereigntyIHDamageMsg"
	SovereigntySBUDamageMsg                   Type = "SovereigntySBUDamageMsg"
	SovereigntyTCUDamageMsg                   Type = "SovereigntyTCUDamageMsg"
	StationAggressionMsg1                     Type = "StationAggressionMsg1"
	StationAggressionMsg2                     Type = "StationAggressionMsg2"
	StationConquerMsg                         Type = "StationConquerMsg"
	StationServiceDisabled                    Type = "StationServiceDisabled"
	StationServiceEnabled                     Type = "StationServiceEnabled"
	StationStateChangeMsg                     Type = "StationStateChangeMsg"
	StoryLineMissionAvailableMsg              Type = "StoryLineMissionAvailableMsg"
	StructureAnchoring                        Type = "StructureAnchoring"
	StructureCourierContractChanged           Type = "StructureCourierContractChanged"
	StructureDestroyed                        Type = "StructureDestroyed"
	StructureFuelAlert                        Type = "StructureFuelAlert"
	StructureImpendingAbandonmentAssetsAtRisk Type = "StructureImpendingAbandonmentAssetsAtRisk"
	StructureItemsDelivered                   Type = "StructureItemsDelivered"
	StructureItemsMovedToSafety               Type = "StructureItemsMovedToSafety"
	StructureLostArmor                        Type = "StructureLostArmor"
	StructureLostShields                      Type = "StructureLostShields"
	StructureLowReagentsAlert                 Type = "StructureLowReagentsAlert"
	StructureNoReagentsAlert                  Type = "StructureNoReagentsAlert"
	StructureOnline                           Type = "StructureOnline"
	StructurePaintPurchased                   Type = "StructurePaintPurchased"
	StructureServicesOffline                  Type = "StructureServicesOffline"
	StructureUnanchoring                      Type = "StructureUnanchoring"
	StructureUnderAttack                      Type = "StructureUnderAttack"
	StructureWentHighPower                    Type = "StructureWentHighPower"
	StructureWentLowPower                     Type = "StructureWentLowPower"
	StructuresJobsCancelled                   Type = "StructuresJobsCancelled"
	StructuresJobsPaused                      Type = "StructuresJobsPaused"
	StructuresReinforcementChanged            Type = "StructuresReinforcementChanged"
	TowerAlertMsg                             Type = "TowerAlertMsg"
	TowerResourceAlertMsg                     Type = "TowerResourceAlertMsg"
	TransactionReversalMsg                    Type = "TransactionReversalMsg"
	TutorialMsg                               Type = "TutorialMsg"
	WarAdopted                                Type = "WarAdopted "
	WarAllyInherited                          Type = "WarAllyInherited"
	WarAllyOfferDeclinedMsg                   Type = "WarAllyOfferDeclinedMsg"
	WarConcordInvalidates                     Type = "WarConcordInvalidates"
	WarDeclared                               Type = "WarDeclared"
	WarEndedHqSecurityDrop                    Type = "WarEndedHqSecurityDrop"
	WarHQRemovedFromSpace                     Type = "WarHQRemovedFromSpace"
	WarInherited                              Type = "WarInherited"
	WarInvalid                                Type = "WarInvalid"
	WarRetracted                              Type = "WarRetracted"
	WarRetractedByConcord                     Type = "WarRetractedByConcord"
	WarSurrenderDeclinedMsg                   Type = "WarSurrenderDeclinedMsg"
	WarSurrenderOfferMsg                      Type = "WarSurrenderOfferMsg"
)

func (nt Type) String() string {
	return string(nt)
}

// Display returns a string representation for display.
func (nt Type) Display() string {
	var b strings.Builder
	var last rune
	s := nt.String()
	for i, r := range s {
		var next rune
		if i < len(s)-1 {
			next = rune(s[i+1])
		}
		if last != 0 {
			if unicode.IsUpper(r) && (unicode.IsLower(last) || unicode.IsLower(next)) {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
		last = r
	}
	return b.String()
}

// SupportedTypes returns all supported notification types.
func SupportedTypes() set.Set[Type] {
	var s = set.Of(
		AllWarSurrenderMsg,
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
		CorpWarSurrenderMsg,
		DeclareWar,
		IHubDestroyedByBillFailure,
		InfrastructureHubBillAboutToExpire,
		MoonminingAutomaticFracture,
		MoonminingExtractionCancelled,
		MoonminingExtractionFinished,
		MoonminingExtractionStarted,
		MoonminingLaserFired,
		OrbitalAttacked,
		OrbitalReinforced,
		OwnershipTransferred,
		StructureAnchoring,
		StructureDestroyed,
		StructureFuelAlert,
		StructureImpendingAbandonmentAssetsAtRisk,
		StructureItemsDelivered,
		StructureItemsMovedToSafety,
		StructureLostArmor,
		StructureLostShields,
		StructureOnline,
		StructureServicesOffline,
		StructuresReinforcementChanged,
		StructureUnanchoring,
		StructureUnderAttack,
		StructureWentHighPower,
		StructureWentLowPower,
		TowerAlertMsg,
		TowerResourceAlertMsg,
		WarAdopted,
		WarDeclared,
		WarHQRemovedFromSpace,
		WarInherited,
		WarInvalid,
		WarRetractedByConcord,
		SovAllClaimAcquiredMsg,
		SovCommandNodeEventStarted,
		SovAllClaimLostMsg,
		EntosisCaptureStarted,
		SovStructureReinforced,
		SovStructureDestroyed,
	)
	return s
}

// notificationRenderer represents the interface every notification renderer needs to confirm with.
type notificationRenderer interface {
	// entityIDs returns the Entity IDs used by a notification (if any).
	entityIDs(text string) (setInt32, error)
	// render returns the rendered title and body for a notification.
	render(ctx context.Context, text string, timestamp time.Time) (string, string, error)
	// setEveUniverse initialized access to the EveUniverseService service and must be called before render().
	setEveUniverse(*eveuniverseservice.EveUniverseService)
}

// baseRenderer represents the base renderer for all notification types.
//
// Each notification type has a renderer which can produce the title and string for a notification.
// In addition the renderer can return the Entity IDs of a notification,
// which allows refetching Entities for multile notifications in bulk before rendering.
//
// All rendered should embed baseRenderer and implement the render method.
// Renderers that want to return Entity IDs must also overwrite entityIDs.
type baseRenderer struct {
	eus *eveuniverseservice.EveUniverseService
}

func (br *baseRenderer) setEveUniverse(eus *eveuniverseservice.EveUniverseService) {
	br.eus = eus
}

// entityIDs returns the Entity IDs used by a notification (if any).
//
// Must be overwritten by a notification rendered that want to return IDs.
func (br baseRenderer) entityIDs(_ string) (setInt32, error) {
	return setInt32{}, nil
}

// EveNotificationService is a service for rendering notifications.
type EveNotificationService struct {
	eus *eveuniverseservice.EveUniverseService
}

func New(eus *eveuniverseservice.EveUniverseService) *EveNotificationService {
	s := &EveNotificationService{eus: eus}
	return s
}

// EntityIDs returns the Entity IDs used in a notification.
// This is useful to resolve Entity IDs in bulk for all notifications,
// before rendering them one by one.
// Returns an empty set when notification does not use Entity IDs.
// Returns [app.ErrNotFound] for unsuported notification types.
func (s *EveNotificationService) EntityIDs(type_, text string) (setInt32, error) {
	if type_ == "" {
		return setInt32{}, app.ErrNotFound
	}
	r, found := s.makeRenderer(Type(type_))
	if !found {
		return setInt32{}, app.ErrNotFound
	}
	return r.entityIDs(text)
}

// RenderESI renders title and body for all supported notification types and returns them.
// Returns [app.ErrNotFound] for unsuported notification types.
func (s *EveNotificationService) RenderESI(ctx context.Context, type_, text string, timestamp time.Time) (title string, body string, err error) {
	if type_ == "" {
		return "", "", app.ErrNotFound
	}
	r, found := s.makeRenderer(Type(type_))
	if !found {
		return "", "", app.ErrNotFound
	}
	title, body, err = r.render(ctx, text, timestamp)
	if err != nil {
		return "", "", err
	}
	return title, body, nil
}

func (s *EveNotificationService) makeRenderer(type_ Type) (notificationRenderer, bool) {
	m := map[Type]notificationRenderer{
		// billing
		BillOutOfMoneyMsg:                  &billOutOfMoneyMsg{},
		BillPaidCorpAllMsg:                 &billPaidCorpAllMsg{},
		CorpAllBillMsg:                     &corpAllBillMsg{},
		InfrastructureHubBillAboutToExpire: &infrastructureHubBillAboutToExpire{},
		IHubDestroyedByBillFailure:         &iHubDestroyedByBillFailure{},
		// corporate
		CharAppAcceptMsg:       &charAppAcceptMsg{},
		CharAppRejectMsg:       &charAppRejectMsg{},
		CharAppWithdrawMsg:     &charAppWithdrawMsg{},
		CharLeftCorpMsg:        &charLeftCorpMsg{},
		CorpAppInvitedMsg:      &corpAppInvitedMsg{},
		CorpAppNewMsg:          &corpAppNewMsg{},
		CorpAppRejectCustomMsg: &corpAppRejectCustomMsg{},
		// moonmining
		MoonminingAutomaticFracture:   &moonminingAutomaticFracture{},
		MoonminingExtractionCancelled: &moonminingExtractionCancelled{},
		MoonminingExtractionFinished:  &moonminingExtractionFinished{},
		MoonminingExtractionStarted:   &moonminingExtractionStarted{},
		MoonminingLaserFired:          &moonminingLaserFired{},
		// orbital
		OrbitalAttacked:   &orbitalAttacked{},
		OrbitalReinforced: &orbitalReinforced{},
		// structures
		OwnershipTransferred:                      &ownershipTransferred{},
		StructureAnchoring:                        &structureAnchoring{},
		StructureDestroyed:                        &structureDestroyed{},
		StructureFuelAlert:                        &structureFuelAlert{},
		StructureImpendingAbandonmentAssetsAtRisk: &structureImpendingAbandonmentAssetsAtRisk{},
		StructureItemsDelivered:                   &structureItemsDelivered{},
		StructureItemsMovedToSafety:               &structureItemsMovedToSafety{},
		StructureLostArmor:                        &structureLostArmor{},
		StructureLostShields:                      &structureLostShields{},
		StructureOnline:                           &structureOnline{},
		StructureServicesOffline:                  &structureServicesOffline{},
		StructuresReinforcementChanged:            &structuresReinforcementChanged{},
		StructureUnanchoring:                      &structureUnanchoring{},
		StructureUnderAttack:                      &structureUnderAttack{},
		StructureWentHighPower:                    &structureWentHighPower{},
		StructureWentLowPower:                     &structureWentLowPower{},
		// sov
		EntosisCaptureStarted:      &entosisCaptureStarted{},
		SovAllClaimAcquiredMsg:     &sovAllClaimAcquiredMsg{},
		SovAllClaimLostMsg:         &sovAllClaimLostMsg{},
		SovCommandNodeEventStarted: &sovCommandNodeEventStarted{},
		SovStructureDestroyed:      &sovStructureDestroyed{},
		SovStructureReinforced:     &sovStructureReinforced{},
		// tower
		TowerAlertMsg:         &towerAlertMsg{},
		TowerResourceAlertMsg: &towerResourceAlertMsg{},
		// war
		AllWarSurrenderMsg:    &allWarSurrenderMsg{},
		CorpWarSurrenderMsg:   &corpWarSurrenderMsg{},
		DeclareWar:            &declareWar{},
		WarAdopted:            &warAdopted{},
		WarDeclared:           &warDeclared{},
		WarHQRemovedFromSpace: &warHQRemovedFromSpace{},
		WarInherited:          &warInherited{},
		WarInvalid:            &warInvalid{},
		WarRetractedByConcord: &warRetractedByConcord{},
	}
	r, found := m[type_]
	if !found {
		return nil, found
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

func makeEveWhoCharacterURL(id int32) string {
	return fmt.Sprintf("https://evewho.com/character/%d", id)
}

func makeEveEntityProfileLink(e *app.EveEntity) string {
	if e == nil {
		return ""
	}
	var url string
	switch e.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(e.Name, dotlanAlliance)
	case app.EveEntityCharacter:
		url = makeEveWhoCharacterURL(e.ID)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(e.Name, dotlanCorporation)
	default:
		return e.Name
	}
	return makeMarkDownLink(e.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}
