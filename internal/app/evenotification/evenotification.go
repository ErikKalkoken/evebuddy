// Package evenotification contains the EveNotification service.
package evenotification

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
)

const (
	AcceptedAlly                              = "AcceptedAlly"
	AcceptedSurrender                         = "AcceptedSurrender"
	AgentRetiredTrigravian                    = "AgentRetiredTrigravian"
	AllAnchoringMsg                           = "AllAnchoringMsg"
	AllMaintenanceBillMsg                     = "AllMaintenanceBillMsg"
	AllStrucInvulnerableMsg                   = "AllStrucInvulnerableMsg"
	AllStructVulnerableMsg                    = "AllStructVulnerableMsg"
	AllWarCorpJoinedAllianceMsg               = "AllWarCorpJoinedAllianceMsg"
	AllWarDeclaredMsg                         = "AllWarDeclaredMsg"
	AllWarInvalidatedMsg                      = "AllWarInvalidatedMsg"
	AllWarRetractedMsg                        = "AllWarRetractedMsg"
	AllWarSurrenderMsg                        = "AllWarSurrenderMsg"
	AllianceCapitalChanged                    = "AllianceCapitalChanged"
	AllianceWarDeclaredV2                     = "AllianceWarDeclaredV2"
	AllyContractCancelled                     = "AllyContractCancelled"
	AllyJoinedWarAggressorMsg                 = "AllyJoinedWarAggressorMsg"
	AllyJoinedWarAllyMsg                      = "AllyJoinedWarAllyMsg"
	AllyJoinedWarDefenderMsg                  = "AllyJoinedWarDefenderMsg"
	BattlePunishFriendlyFire                  = "BattlePunishFriendlyFire"
	BillOutOfMoneyMsg                         = "BillOutOfMoneyMsg"
	BillPaidCorpAllMsg                        = "BillPaidCorpAllMsg"
	BountyClaimMsg                            = "BountyClaimMsg"
	BountyESSShared                           = "BountyESSShared"
	BountyESSTaken                            = "BountyESSTaken"
	BountyPlacedAlliance                      = "BountyPlacedAlliance"
	BountyPlacedChar                          = "BountyPlacedChar"
	BountyPlacedCorp                          = "BountyPlacedCorp"
	BountyYourBountyClaimed                   = "BountyYourBountyClaimed"
	BuddyConnectContactAdd                    = "BuddyConnectContactAdd"
	CharAppAcceptMsg                          = "CharAppAcceptMsg"
	CharAppRejectMsg                          = "CharAppRejectMsg"
	CharAppWithdrawMsg                        = "CharAppWithdrawMsg"
	CharLeftCorpMsg                           = "CharLeftCorpMsg"
	CharMedalMsg                              = "CharMedalMsg"
	CharTerminationMsg                        = "CharTerminationMsg"
	CloneActivationMsg                        = "CloneActivationMsg"
	CloneActivationMsg2                       = "CloneActivationMsg2"
	CloneMovedMsg                             = "CloneMovedMsg"
	CloneRevokedMsg1                          = "CloneRevokedMsg1"
	CloneRevokedMsg2                          = "CloneRevokedMsg2"
	CombatOperationFinished                   = "CombatOperationFinished"
	ContactAdd                                = "ContactAdd"
	ContactEdit                               = "ContactEdit"
	ContainerPasswordMsg                      = "ContainerPasswordMsg"
	ContractRegionChangedToPochven            = "ContractRegionChangedToPochven"
	CorpAllBillMsg                            = "CorpAllBillMsg"
	CorpAppAcceptMsg                          = "CorpAppAcceptMsg"
	CorpAppInvitedMsg                         = "CorpAppInvitedMsg"
	CorpAppNewMsg                             = "CorpAppNewMsg"
	CorpAppRejectCustomMsg                    = "CorpAppRejectCustomMsg"
	CorpAppRejectMsg                          = "CorpAppRejectMsg"
	CorpBecameWarEligible                     = "CorpBecameWarEligible"
	CorpDividendMsg                           = "CorpDividendMsg"
	CorpFriendlyFireDisableTimerCompleted     = "CorpFriendlyFireDisableTimerCompleted"
	CorpFriendlyFireDisableTimerStarted       = "CorpFriendlyFireDisableTimerStarted"
	CorpFriendlyFireEnableTimerCompleted      = "CorpFriendlyFireEnableTimerCompleted"
	CorpFriendlyFireEnableTimerStarted        = "CorpFriendlyFireEnableTimerStarted"
	CorpKicked                                = "CorpKicked"
	CorpLiquidationMsg                        = "CorpLiquidationMsg"
	CorpNewCEOMsg                             = "CorpNewCEOMsg"
	CorpNewsMsg                               = "CorpNewsMsg"
	CorpNoLongerWarEligible                   = "CorpNoLongerWarEligible"
	CorpOfficeExpirationMsg                   = "CorpOfficeExpirationMsg"
	CorpStructLostMsg                         = "CorpStructLostMsg"
	CorpTaxChangeMsg                          = "CorpTaxChangeMsg"
	CorpVoteCEORevokedMsg                     = "CorpVoteCEORevokedMsg"
	CorpVoteMsg                               = "CorpVoteMsg"
	CorpWarDeclaredMsg                        = "CorpWarDeclaredMsg"
	CorpWarDeclaredV2                         = "CorpWarDeclaredV2"
	CorpWarFightingLegalMsg                   = "CorpWarFightingLegalMsg"
	CorpWarInvalidatedMsg                     = "CorpWarInvalidatedMsg"
	CorpWarRetractedMsg                       = "CorpWarRetractedMsg"
	CorpWarSurrenderMsg                       = "CorpWarSurrenderMsg"
	CorporationGoalClosed                     = "CorporationGoalClosed"
	CorporationGoalCompleted                  = "CorporationGoalCompleted"
	CorporationGoalCreated                    = "CorporationGoalCreated"
	CorporationGoalNameChange                 = "CorporationGoalNameChange"
	CorporationLeft                           = "CorporationLeft"
	CustomsMsg                                = "CustomsMsg"
	DeclareWar                                = "DeclareWar"
	DistrictAttacked                          = "DistrictAttacked"
	DustAppAcceptedMsg                        = "DustAppAcceptedMsg"
	ESSMainBankLink                           = "ESSMainBankLink"
	EntosisCaptureStarted                     = "EntosisCaptureStarted"
	ExpertSystemExpired                       = "ExpertSystemExpired"
	ExpertSystemExpiryImminent                = "ExpertSystemExpiryImminent"
	FWAllianceKickMsg                         = "FWAllianceKickMsg"
	FWAllianceWarningMsg                      = "FWAllianceWarningMsg"
	FWCharKickMsg                             = "FWCharKickMsg"
	FWCharRankGainMsg                         = "FWCharRankGainMsg"
	FWCharRankLossMsg                         = "FWCharRankLossMsg"
	FWCharWarningMsg                          = "FWCharWarningMsg"
	FWCorpJoinMsg                             = "FWCorpJoinMsg"
	FWCorpKickMsg                             = "FWCorpKickMsg"
	FWCorpLeaveMsg                            = "FWCorpLeaveMsg"
	FWCorpWarningMsg                          = "FWCorpWarningMsg"
	FacWarCorpJoinRequestMsg                  = "FacWarCorpJoinRequestMsg"
	FacWarCorpJoinWithdrawMsg                 = "FacWarCorpJoinWithdrawMsg"
	FacWarCorpLeaveRequestMsg                 = "FacWarCorpLeaveRequestMsg"
	FacWarCorpLeaveWithdrawMsg                = "FacWarCorpLeaveWithdrawMsg"
	FacWarLPDisqualifiedEvent                 = "FacWarLPDisqualifiedEvent"
	FacWarLPDisqualifiedKill                  = "FacWarLPDisqualifiedKill"
	FacWarLPPayoutEvent                       = "FacWarLPPayoutEvent"
	FacWarLPPayoutKill                        = "FacWarLPPayoutKill"
	GameTimeAdded                             = "GameTimeAdded"
	GameTimeReceived                          = "GameTimeReceived"
	GameTimeSent                              = "GameTimeSent"
	GiftReceived                              = "GiftReceived"
	IHubDestroyedByBillFailure                = "IHubDestroyedByBillFailure"
	IncursionCompletedMsg                     = "IncursionCompletedMsg"
	IndustryOperationFinished                 = "IndustryOperationFinished"
	IndustryTeamAuctionLost                   = "IndustryTeamAuctionLost"
	IndustryTeamAuctionWon                    = "IndustryTeamAuctionWon"
	InfrastructureHubBillAboutToExpire        = "InfrastructureHubBillAboutToExpire"
	InsuranceExpirationMsg                    = "InsuranceExpirationMsg"
	InsuranceFirstShipMsg                     = "InsuranceFirstShipMsg"
	InsuranceInvalidatedMsg                   = "InsuranceInvalidatedMsg"
	InsuranceIssuedMsg                        = "InsuranceIssuedMsg"
	InsurancePayoutMsg                        = "InsurancePayoutMsg"
	InvasionCompletedMsg                      = "InvasionCompletedMsg"
	InvasionSystemLogin                       = "InvasionSystemLogin"
	InvasionSystemStart                       = "InvasionSystemStart"
	JumpCloneDeletedMsg1                      = "JumpCloneDeletedMsg1"
	JumpCloneDeletedMsg2                      = "JumpCloneDeletedMsg2"
	KillReportFinalBlow                       = "KillReportFinalBlow"
	KillReportVictim                          = "KillReportVictim"
	KillRightAvailable                        = "KillRightAvailable"
	KillRightAvailableOpen                    = "KillRightAvailableOpen"
	KillRightEarned                           = "KillRightEarned"
	KillRightUnavailable                      = "KillRightUnavailable"
	KillRightUnavailableOpen                  = "KillRightUnavailableOpen"
	KillRightUsed                             = "KillRightUsed"
	LPAutoRedeemed                            = "LPAutoRedeemed"
	LocateCharMsg                             = "LocateCharMsg"
	MadeWarMutual                             = "MadeWarMutual"
	MercOfferRetractedMsg                     = "MercOfferRetractedMsg"
	MercOfferedNegotiationMsg                 = "MercOfferedNegotiationMsg"
	MissionCanceledTriglavian                 = "MissionCanceledTriglavian"
	MissionOfferExpirationMsg                 = "MissionOfferExpirationMsg"
	MissionTimeoutMsg                         = "MissionTimeoutMsg"
	MoonminingAutomaticFracture               = "MoonminingAutomaticFracture"
	MoonminingExtractionCancelled             = "MoonminingExtractionCancelled"
	MoonminingExtractionFinished              = "MoonminingExtractionFinished"
	MoonminingExtractionStarted               = "MoonminingExtractionStarted"
	MoonminingLaserFired                      = "MoonminingLaserFired"
	MutualWarExpired                          = "MutualWarExpired"
	MutualWarInviteAccepted                   = "MutualWarInviteAccepted"
	MutualWarInviteRejected                   = "MutualWarInviteRejected"
	MutualWarInviteSent                       = "MutualWarInviteSent"
	NPCStandingsGained                        = "NPCStandingsGained"
	NPCStandingsLost                          = "NPCStandingsLost"
	OfferToAllyRetracted                      = "OfferToAllyRetracted"
	OfferedSurrender                          = "OfferedSurrender"
	OfferedToAlly                             = "OfferedToAlly"
	OfficeLeaseCanceledInsufficientStandings  = "OfficeLeaseCanceledInsufficientStandings"
	OldLscMessages                            = "OldLscMessages"
	OperationFinished                         = "OperationFinished"
	OrbitalAttacked                           = "OrbitalAttacked"
	OrbitalReinforced                         = "OrbitalReinforced"
	OwnershipTransferred                      = "OwnershipTransferred"
	RaffleCreated                             = "RaffleCreated"
	RaffleExpired                             = "RaffleExpired"
	RaffleFinished                            = "RaffleFinished"
	ReimbursementMsg                          = "ReimbursementMsg"
	ResearchMissionAvailableMsg               = "ResearchMissionAvailableMsg"
	RetractsWar                               = "RetractsWar"
	SPAutoRedeemed                            = "SPAutoRedeemed"
	SeasonalChallengeCompleted                = "SeasonalChallengeCompleted"
	SkinSequencingCompleted                   = "SkinSequencingCompleted"
	SkyhookDeployed                           = "SkyhookDeployed"
	SkyhookDestroyed                          = "SkyhookDestroyed"
	SkyhookLostShields                        = "SkyhookLostShields"
	SkyhookOnline                             = "SkyhookOnline"
	SkyhookUnderAttack                        = "SkyhookUnderAttack"
	SovAllClaimAquiredMsg                     = "SovAllClaimAquiredMsg"
	SovAllClaimLostMsg                        = "SovAllClaimLostMsg"
	SovCommandNodeEventStarted                = "SovCommandNodeEventStarted"
	SovCorpBillLateMsg                        = "SovCorpBillLateMsg"
	SovCorpClaimFailMsg                       = "SovCorpClaimFailMsg"
	SovDisruptorMsg                           = "SovDisruptorMsg"
	SovStationEnteredFreeport                 = "SovStationEnteredFreeport"
	SovStructureDestroyed                     = "SovStructureDestroyed"
	SovStructureReinforced                    = "SovStructureReinforced"
	SovStructureSelfDestructCancel            = "SovStructureSelfDestructCancel"
	SovStructureSelfDestructFinished          = "SovStructureSelfDestructFinished"
	SovStructureSelfDestructRequested         = "SovStructureSelfDestructRequested"
	SovereigntyIHDamageMsg                    = "SovereigntyIHDamageMsg"
	SovereigntySBUDamageMsg                   = "SovereigntySBUDamageMsg"
	SovereigntyTCUDamageMsg                   = "SovereigntyTCUDamageMsg"
	StationAggressionMsg1                     = "StationAggressionMsg1"
	StationAggressionMsg2                     = "StationAggressionMsg2"
	StationConquerMsg                         = "StationConquerMsg"
	StationServiceDisabled                    = "StationServiceDisabled"
	StationServiceEnabled                     = "StationServiceEnabled"
	StationStateChangeMsg                     = "StationStateChangeMsg"
	StoryLineMissionAvailableMsg              = "StoryLineMissionAvailableMsg"
	StructureAnchoring                        = "StructureAnchoring"
	StructureCourierContractChanged           = "StructureCourierContractChanged"
	StructureDestroyed                        = "StructureDestroyed"
	StructureFuelAlert                        = "StructureFuelAlert"
	StructureImpendingAbandonmentAssetsAtRisk = "StructureImpendingAbandonmentAssetsAtRisk"
	StructureItemsDelivered                   = "StructureItemsDelivered"
	StructureItemsMovedToSafety               = "StructureItemsMovedToSafety"
	StructureLostArmor                        = "StructureLostArmor"
	StructureLostShields                      = "StructureLostShields"
	StructureLowReagentsAlert                 = "StructureLowReagentsAlert"
	StructureNoReagentsAlert                  = "StructureNoReagentsAlert"
	StructureOnline                           = "StructureOnline"
	StructurePaintPurchased                   = "StructurePaintPurchased"
	StructureServicesOffline                  = "StructureServicesOffline"
	StructureUnanchoring                      = "StructureUnanchoring"
	StructureUnderAttack                      = "StructureUnderAttack"
	StructureWentHighPower                    = "StructureWentHighPower"
	StructureWentLowPower                     = "StructureWentLowPower"
	StructuresJobsCancelled                   = "StructuresJobsCancelled"
	StructuresJobsPaused                      = "StructuresJobsPaused"
	StructuresReinforcementChanged            = "StructuresReinforcementChanged"
	TowerAlertMsg                             = "TowerAlertMsg"
	TowerResourceAlertMsg                     = "TowerResourceAlertMsg"
	TransactionReversalMsg                    = "TransactionReversalMsg"
	TutorialMsg                               = "TutorialMsg"
	WarAdopted                                = "WarAdopted "
	WarAllyInherited                          = "WarAllyInherited"
	WarAllyOfferDeclinedMsg                   = "WarAllyOfferDeclinedMsg"
	WarConcordInvalidates                     = "WarConcordInvalidates"
	WarDeclared                               = "WarDeclared"
	WarEndedHqSecurityDrop                    = "WarEndedHqSecurityDrop"
	WarHQRemovedFromSpace                     = "WarHQRemovedFromSpace"
	WarInherited                              = "WarInherited"
	WarInvalid                                = "WarInvalid"
	WarRetracted                              = "WarRetracted"
	WarRetractedByConcord                     = "WarRetractedByConcord"
	WarSurrenderDeclinedMsg                   = "WarSurrenderDeclinedMsg"
	WarSurrenderOfferMsg                      = "WarSurrenderOfferMsg"
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

// EveNotificationService is a service for rendering notifications.
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
