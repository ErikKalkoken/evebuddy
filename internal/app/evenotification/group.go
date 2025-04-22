package evenotification

import (
	"strings"
	"unicode"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type Type string

// A specific notification type
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

var supportedTypes = []Type{
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
}

// SupportedGroups returns a list of all supported notification types.
func SupportedGroups() []Type {
	return supportedTypes
}

var Type2group = map[Type]app.NotificationGroup{
	AcceptedAlly:                              app.GroupWar,
	AcceptedSurrender:                         app.GroupWar,
	AgentRetiredTrigravian:                    app.GroupUnknown,
	AllAnchoringMsg:                           app.GroupSovereignty,
	AllMaintenanceBillMsg:                     app.GroupBills,
	AllStructureInvulnerableMsg:               app.GroupStructure,
	AllStructVulnerableMsg:                    app.GroupStructure,
	AllWarCorpJoinedAllianceMsg:               app.GroupWar,
	AllWarDeclaredMsg:                         app.GroupWar,
	AllWarInvalidatedMsg:                      app.GroupWar,
	AllWarRetractedMsg:                        app.GroupWar,
	AllWarSurrenderMsg:                        app.GroupWar,
	AllianceCapitalChanged:                    app.GroupMiscellaneous,
	AllianceWarDeclaredV2:                     app.GroupWar,
	AllyContractCancelled:                     app.GroupWar,
	AllyJoinedWarAggressorMsg:                 app.GroupWar,
	AllyJoinedWarAllyMsg:                      app.GroupWar,
	AllyJoinedWarDefenderMsg:                  app.GroupWar,
	BattlePunishFriendlyFire:                  app.GroupUnknown,
	BillOutOfMoneyMsg:                         app.GroupBills,
	BillPaidCorpAllMsg:                        app.GroupBills,
	BountyClaimMsg:                            app.GroupUnknown,
	BountyESSShared:                           app.GroupUnknown,
	BountyESSTaken:                            app.GroupUnknown,
	BountyPlacedAlliance:                      app.GroupUnknown,
	BountyPlacedChar:                          app.GroupUnknown,
	BountyPlacedCorp:                          app.GroupUnknown,
	BountyYourBountyClaimed:                   app.GroupUnknown,
	BuddyConnectContactAdd:                    app.GroupUnknown,
	CharAppAcceptMsg:                          app.GroupCorporate,
	CharAppRejectMsg:                          app.GroupCorporate,
	CharAppWithdrawMsg:                        app.GroupCorporate,
	CharLeftCorpMsg:                           app.GroupCorporate,
	CharMedalMsg:                              app.GroupCorporate,
	CharTerminationMsg:                        app.GroupCorporate,
	CloneActivationMsg:                        app.GroupUnknown,
	CloneActivationMsg2:                       app.GroupUnknown,
	CloneMovedMsg:                             app.GroupUnknown,
	CloneRevokedMsg1:                          app.GroupUnknown,
	CloneRevokedMsg2:                          app.GroupUnknown,
	CombatOperationFinished:                   app.GroupUnknown,
	ContactAdd:                                app.GroupContacts,
	ContactEdit:                               app.GroupContacts,
	ContainerPasswordMsg:                      app.GroupUnknown,
	ContractRegionChangedToPochven:            app.GroupUnknown,
	CorpAllBillMsg:                            app.GroupBills,
	CorpAppAcceptMsg:                          app.GroupCorporate,
	CorpAppInvitedMsg:                         app.GroupCorporate,
	CorpAppNewMsg:                             app.GroupCorporate,
	CorpAppRejectCustomMsg:                    app.GroupCorporate,
	CorpAppRejectMsg:                          app.GroupCorporate,
	CorpBecameWarEligible:                     app.GroupCorporate,
	CorpDividendMsg:                           app.GroupCorporate,
	CorpFriendlyFireDisableTimerCompleted:     app.GroupCorporate,
	CorpFriendlyFireDisableTimerStarted:       app.GroupCorporate,
	CorpFriendlyFireEnableTimerCompleted:      app.GroupCorporate,
	CorpFriendlyFireEnableTimerStarted:        app.GroupCorporate,
	CorpKicked:                                app.GroupCorporate,
	CorpLiquidationMsg:                        app.GroupCorporate,
	CorpNewCEOMsg:                             app.GroupCorporate,
	CorpNewsMsg:                               app.GroupCorporate,
	CorpNoLongerWarEligible:                   app.GroupCorporate,
	CorpOfficeExpirationMsg:                   app.GroupCorporate,
	CorpStructLostMsg:                         app.GroupCorporate,
	CorpTaxChangeMsg:                          app.GroupCorporate,
	CorpVoteCEORevokedMsg:                     app.GroupCorporate,
	CorpVoteMsg:                               app.GroupCorporate,
	CorpWarDeclaredMsg:                        app.GroupWar,
	CorpWarDeclaredV2:                         app.GroupWar,
	CorpWarFightingLegalMsg:                   app.GroupWar,
	CorpWarInvalidatedMsg:                     app.GroupWar,
	CorpWarRetractedMsg:                       app.GroupWar,
	CorpWarSurrenderMsg:                       app.GroupWar,
	CorporationGoalClosed:                     app.GroupCorporate,
	CorporationGoalCompleted:                  app.GroupCorporate,
	CorporationGoalCreated:                    app.GroupCorporate,
	CorporationGoalNameChange:                 app.GroupCorporate,
	CorporationLeft:                           app.GroupCorporate,
	CustomsMsg:                                app.GroupMiscellaneous,
	DeclareWar:                                app.GroupWar,
	DistrictAttacked:                          app.GroupWar,
	DustAppAcceptedMsg:                        app.GroupMiscellaneous,
	ESSMainBankLink:                           app.GroupUnknown,
	EntosisCaptureStarted:                     app.GroupSovereignty,
	ExpertSystemExpired:                       app.GroupMiscellaneous,
	ExpertSystemExpiryImminent:                app.GroupMiscellaneous,
	FWAllianceKickMsg:                         app.GroupFactionWarfare,
	FWAllianceWarningMsg:                      app.GroupFactionWarfare,
	FWCharKickMsg:                             app.GroupFactionWarfare,
	FWCharRankGainMsg:                         app.GroupFactionWarfare,
	FWCharRankLossMsg:                         app.GroupFactionWarfare,
	FWCharWarningMsg:                          app.GroupFactionWarfare,
	FWCorpJoinMsg:                             app.GroupFactionWarfare,
	FWCorpKickMsg:                             app.GroupFactionWarfare,
	FWCorpLeaveMsg:                            app.GroupFactionWarfare,
	FWCorpWarningMsg:                          app.GroupFactionWarfare,
	FacWarCorpJoinRequestMsg:                  app.GroupFactionWarfare,
	FacWarCorpJoinWithdrawMsg:                 app.GroupFactionWarfare,
	FacWarCorpLeaveRequestMsg:                 app.GroupFactionWarfare,
	FacWarCorpLeaveWithdrawMsg:                app.GroupFactionWarfare,
	FacWarLPDisqualifiedEvent:                 app.GroupFactionWarfare,
	FacWarLPDisqualifiedKill:                  app.GroupFactionWarfare,
	FacWarLPPayoutEvent:                       app.GroupFactionWarfare,
	FacWarLPPayoutKill:                        app.GroupFactionWarfare,
	GameTimeAdded:                             app.GroupUnknown,
	GameTimeReceived:                          app.GroupUnknown,
	GameTimeSent:                              app.GroupUnknown,
	GiftReceived:                              app.GroupUnknown,
	IHubDestroyedByBillFailure:                app.GroupSovereignty,
	IncursionCompletedMsg:                     app.GroupUnknown,
	IndustryOperationFinished:                 app.GroupUnknown,
	IndustryTeamAuctionLost:                   app.GroupUnknown,
	IndustryTeamAuctionWon:                    app.GroupUnknown,
	InfrastructureHubBillAboutToExpire:        app.GroupSovereignty,
	InsuranceExpirationMsg:                    app.GroupUnknown,
	InsuranceFirstShipMsg:                     app.GroupUnknown,
	InsuranceInvalidatedMsg:                   app.GroupUnknown,
	InsuranceIssuedMsg:                        app.GroupUnknown,
	InsurancePayoutMsg:                        app.GroupUnknown,
	InvasionCompletedMsg:                      app.GroupUnknown,
	InvasionSystemLogin:                       app.GroupUnknown,
	InvasionSystemStart:                       app.GroupUnknown,
	JumpCloneDeletedMsg1:                      app.GroupUnknown,
	JumpCloneDeletedMsg2:                      app.GroupUnknown,
	KillReportFinalBlow:                       app.GroupUnknown,
	KillReportVictim:                          app.GroupUnknown,
	KillRightAvailable:                        app.GroupUnknown,
	KillRightAvailableOpen:                    app.GroupUnknown,
	KillRightEarned:                           app.GroupUnknown,
	KillRightUnavailable:                      app.GroupUnknown,
	KillRightUnavailableOpen:                  app.GroupUnknown,
	KillRightUsed:                             app.GroupUnknown,
	LPAutoRedeemed:                            app.GroupUnknown,
	LocateCharMsg:                             app.GroupUnknown,
	MadeWarMutual:                             app.GroupUnknown,
	MercOfferRetractedMsg:                     app.GroupWar,
	MercOfferedNegotiationMsg:                 app.GroupWar,
	MissionCanceledTriglavian:                 app.GroupUnknown,
	MissionOfferExpirationMsg:                 app.GroupUnknown,
	MissionTimeoutMsg:                         app.GroupUnknown,
	MoonminingAutomaticFracture:               app.GroupMoonMining,
	MoonminingExtractionCancelled:             app.GroupMoonMining,
	MoonminingExtractionFinished:              app.GroupMoonMining,
	MoonminingExtractionStarted:               app.GroupMoonMining,
	MoonminingLaserFired:                      app.GroupMoonMining,
	MutualWarExpired:                          app.GroupWar,
	MutualWarInviteAccepted:                   app.GroupWar,
	MutualWarInviteRejected:                   app.GroupWar,
	MutualWarInviteSent:                       app.GroupWar,
	NPCStandingsGained:                        app.GroupUnknown,
	NPCStandingsLost:                          app.GroupUnknown,
	OfferToAllyRetracted:                      app.GroupWar,
	OfferedSurrender:                          app.GroupWar,
	OfferedToAlly:                             app.GroupWar,
	OfficeLeaseCanceledInsufficientStandings:  app.GroupCorporate,
	OldLscMessages:                            app.GroupUnknown,
	OperationFinished:                         app.GroupUnknown,
	OrbitalAttacked:                           app.GroupStructure,
	OrbitalReinforced:                         app.GroupStructure,
	OwnershipTransferred:                      app.GroupStructure,
	RaffleCreated:                             app.GroupUnknown,
	RaffleExpired:                             app.GroupUnknown,
	RaffleFinished:                            app.GroupUnknown,
	ReimbursementMsg:                          app.GroupUnknown,
	ResearchMissionAvailableMsg:               app.GroupUnknown,
	RetractsWar:                               app.GroupWar,
	SPAutoRedeemed:                            app.GroupUnknown,
	SeasonalChallengeCompleted:                app.GroupUnknown,
	SkinSequencingCompleted:                   app.GroupUnknown,
	SkyhookDeployed:                           app.GroupStructure,
	SkyhookDestroyed:                          app.GroupStructure,
	SkyhookLostShields:                        app.GroupStructure,
	SkyhookOnline:                             app.GroupStructure,
	SkyhookUnderAttack:                        app.GroupStructure,
	SovAllClaimAcquiredMsg:                    app.GroupSovereignty,
	SovAllClaimLostMsg:                        app.GroupSovereignty,
	SovCommandNodeEventStarted:                app.GroupSovereignty,
	SovCorpBillLateMsg:                        app.GroupSovereignty,
	SovCorpClaimFailMsg:                       app.GroupSovereignty,
	SovDisruptorMsg:                           app.GroupSovereignty,
	SovStationEnteredFreeport:                 app.GroupSovereignty,
	SovStructureDestroyed:                     app.GroupSovereignty,
	SovStructureReinforced:                    app.GroupSovereignty,
	SovStructureSelfDestructCancel:            app.GroupSovereignty,
	SovStructureSelfDestructFinished:          app.GroupSovereignty,
	SovStructureSelfDestructRequested:         app.GroupSovereignty,
	SovereigntyIHDamageMsg:                    app.GroupSovereignty,
	SovereigntySBUDamageMsg:                   app.GroupSovereignty,
	SovereigntyTCUDamageMsg:                   app.GroupSovereignty,
	StationAggressionMsg1:                     app.GroupUnknown,
	StationAggressionMsg2:                     app.GroupUnknown,
	StationConquerMsg:                         app.GroupUnknown,
	StationServiceDisabled:                    app.GroupUnknown,
	StationServiceEnabled:                     app.GroupUnknown,
	StationStateChangeMsg:                     app.GroupUnknown,
	StoryLineMissionAvailableMsg:              app.GroupUnknown,
	StructureAnchoring:                        app.GroupStructure,
	StructureCourierContractChanged:           app.GroupStructure,
	StructureDestroyed:                        app.GroupStructure,
	StructureFuelAlert:                        app.GroupStructure,
	StructureImpendingAbandonmentAssetsAtRisk: app.GroupStructure,
	StructureItemsDelivered:                   app.GroupStructure,
	StructureItemsMovedToSafety:               app.GroupStructure,
	StructureLostArmor:                        app.GroupStructure,
	StructureLostShields:                      app.GroupStructure,
	StructureLowReagentsAlert:                 app.GroupStructure,
	StructureNoReagentsAlert:                  app.GroupStructure,
	StructureOnline:                           app.GroupStructure,
	StructurePaintPurchased:                   app.GroupStructure,
	StructureServicesOffline:                  app.GroupStructure,
	StructureUnanchoring:                      app.GroupStructure,
	StructureUnderAttack:                      app.GroupStructure,
	StructureWentHighPower:                    app.GroupStructure,
	StructureWentLowPower:                     app.GroupStructure,
	StructuresJobsCancelled:                   app.GroupStructure,
	StructuresJobsPaused:                      app.GroupStructure,
	StructuresReinforcementChanged:            app.GroupStructure,
	TowerAlertMsg:                             app.GroupStructure,
	TowerResourceAlertMsg:                     app.GroupStructure,
	TransactionReversalMsg:                    app.GroupUnknown,
	TutorialMsg:                               app.GroupUnknown,
	WarAdopted:                                app.GroupWar,
	WarAllyInherited:                          app.GroupWar,
	WarAllyOfferDeclinedMsg:                   app.GroupWar,
	WarConcordInvalidates:                     app.GroupWar,
	WarDeclared:                               app.GroupWar,
	WarEndedHqSecurityDrop:                    app.GroupWar,
	WarHQRemovedFromSpace:                     app.GroupWar,
	WarInherited:                              app.GroupWar,
	WarInvalid:                                app.GroupWar,
	WarRetracted:                              app.GroupWar,
	WarRetractedByConcord:                     app.GroupWar,
	WarSurrenderDeclinedMsg:                   app.GroupWar,
	WarSurrenderOfferMsg:                      app.GroupWar,
}

var GroupTypes map[app.NotificationGroup][]Type

func init() {
	GroupTypes = make(map[app.NotificationGroup][]Type)
	for t, g := range Type2group {
		if g == app.GroupUnknown {
			g = app.GroupMiscellaneous
		}
		GroupTypes[g] = append(GroupTypes[g], t)
	}
}
