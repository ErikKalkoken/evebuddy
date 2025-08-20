package app

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

//go:generate go tool stringer -type=EveNotificationType

// EveNotificationType represents a notification type in Eve Online.
type EveNotificationType uint

const (
	UnknownNotification EveNotificationType = iota
	AcceptedAlly
	AcceptedSurrender
	AgentRetiredTrigravian
	AllAnchoringMsg
	AllMaintenanceBillMsg
	AllStructureInvulnerableMsg
	AllStructVulnerableMsg
	AllWarCorpJoinedAllianceMsg
	AllWarDeclaredMsg
	AllWarInvalidatedMsg
	AllWarRetractedMsg
	AllWarSurrenderMsg
	AllianceCapitalChanged
	AllianceWarDeclaredV2
	AllyContractCancelled
	AllyJoinedWarAggressorMsg
	AllyJoinedWarAllyMsg
	AllyJoinedWarDefenderMsg
	BattlePunishFriendlyFire
	BillOutOfMoneyMsg
	BillPaidCorpAllMsg
	BountyClaimMsg
	BountyESSShared
	BountyESSTaken
	BountyPlacedAlliance
	BountyPlacedChar
	BountyPlacedCorp
	BountyYourBountyClaimed
	BuddyConnectContactAdd
	CharAppAcceptMsg
	CharAppRejectMsg
	CharAppWithdrawMsg
	CharLeftCorpMsg
	CharMedalMsg
	CharTerminationMsg
	CloneActivationMsg
	CloneActivationMsg2
	CloneMovedMsg
	CloneRevokedMsg1
	CloneRevokedMsg2
	CombatOperationFinished
	ContactAdd
	ContactEdit
	ContainerPasswordMsg
	ContractRegionChangedToPochven
	CorpAllBillMsg
	CorpAppAcceptMsg
	CorpAppInvitedMsg
	CorpAppNewMsg
	CorpAppRejectCustomMsg
	CorpAppRejectMsg
	CorpBecameWarEligible
	CorpDividendMsg
	CorpFriendlyFireDisableTimerCompleted
	CorpFriendlyFireDisableTimerStarted
	CorpFriendlyFireEnableTimerCompleted
	CorpFriendlyFireEnableTimerStarted
	CorpKicked
	CorpLiquidationMsg
	CorpNewCEOMsg
	CorpNewsMsg
	CorpNoLongerWarEligible
	CorpOfficeExpirationMsg
	CorpStructLostMsg
	CorpTaxChangeMsg
	CorpVoteCEORevokedMsg
	CorpVoteMsg
	CorpWarDeclaredMsg
	CorpWarDeclaredV2
	CorpWarFightingLegalMsg
	CorpWarInvalidatedMsg
	CorpWarRetractedMsg
	CorpWarSurrenderMsg
	CorporationGoalClosed
	CorporationGoalCompleted
	CorporationGoalCreated
	CorporationGoalNameChange
	CorporationLeft
	CustomsMsg
	DeclareWar
	DistrictAttacked
	DustAppAcceptedMsg
	ESSMainBankLink
	EntosisCaptureStarted
	ExpertSystemExpired
	ExpertSystemExpiryImminent
	FWAllianceKickMsg
	FWAllianceWarningMsg
	FWCharKickMsg
	FWCharRankGainMsg
	FWCharRankLossMsg
	FWCharWarningMsg
	FWCorpJoinMsg
	FWCorpKickMsg
	FWCorpLeaveMsg
	FWCorpWarningMsg
	FacWarCorpJoinRequestMsg
	FacWarCorpJoinWithdrawMsg
	FacWarCorpLeaveRequestMsg
	FacWarCorpLeaveWithdrawMsg
	FacWarLPDisqualifiedEvent
	FacWarLPDisqualifiedKill
	FacWarLPPayoutEvent
	FacWarLPPayoutKill
	GameTimeAdded
	GameTimeReceived
	GameTimeSent
	GiftReceived
	IHubDestroyedByBillFailure
	IncursionCompletedMsg
	IndustryOperationFinished
	IndustryTeamAuctionLost
	IndustryTeamAuctionWon
	InfrastructureHubBillAboutToExpire
	InsuranceExpirationMsg
	InsuranceFirstShipMsg
	InsuranceInvalidatedMsg
	InsuranceIssuedMsg
	InsurancePayoutMsg
	InvasionCompletedMsg
	InvasionSystemLogin
	InvasionSystemStart
	JumpCloneDeletedMsg1
	JumpCloneDeletedMsg2
	KillReportFinalBlow
	KillReportVictim
	KillRightAvailable
	KillRightAvailableOpen
	KillRightEarned
	KillRightUnavailable
	KillRightUnavailableOpen
	KillRightUsed
	LPAutoRedeemed
	LocateCharMsg
	MadeWarMutual
	MercOfferRetractedMsg
	MercOfferedNegotiationMsg
	MissionCanceledTriglavian
	MissionOfferExpirationMsg
	MissionTimeoutMsg
	MoonminingAutomaticFracture
	MoonminingExtractionCancelled
	MoonminingExtractionFinished
	MoonminingExtractionStarted
	MoonminingLaserFired
	MutualWarExpired
	MutualWarInviteAccepted
	MutualWarInviteRejected
	MutualWarInviteSent
	NPCStandingsGained
	NPCStandingsLost
	OfferToAllyRetracted
	OfferedSurrender
	OfferedToAlly
	OfficeLeaseCanceledInsufficientStandings
	OldLscMessages
	OperationFinished
	OrbitalAttacked
	OrbitalReinforced
	OwnershipTransferred
	RaffleCreated
	RaffleExpired
	RaffleFinished
	ReimbursementMsg
	ResearchMissionAvailableMsg
	RetractsWar
	SPAutoRedeemed
	SeasonalChallengeCompleted
	SkinSequencingCompleted
	SkyhookDeployed
	SkyhookDestroyed
	SkyhookLostShields
	SkyhookOnline
	SkyhookUnderAttack
	SovAllClaimAcquiredMsg
	SovAllClaimLostMsg
	SovCommandNodeEventStarted
	SovCorpBillLateMsg
	SovCorpClaimFailMsg
	SovDisruptorMsg
	SovStationEnteredFreeport
	SovStructureDestroyed
	SovStructureReinforced
	SovStructureSelfDestructCancel
	SovStructureSelfDestructFinished
	SovStructureSelfDestructRequested
	SovereigntyIHDamageMsg
	SovereigntySBUDamageMsg
	SovereigntyTCUDamageMsg
	StationAggressionMsg1
	StationAggressionMsg2
	StationConquerMsg
	StationServiceDisabled
	StationServiceEnabled
	StationStateChangeMsg
	StoryLineMissionAvailableMsg
	StructureAnchoring
	StructureCourierContractChanged
	StructureDestroyed
	StructureFuelAlert
	StructureImpendingAbandonmentAssetsAtRisk
	StructureItemsDelivered
	StructureItemsMovedToSafety
	StructureLostArmor
	StructureLostShields
	StructureLowReagentsAlert
	StructureNoReagentsAlert
	StructureOnline
	StructurePaintPurchased
	StructureServicesOffline
	StructureUnanchoring
	StructureUnderAttack
	StructureWentHighPower
	StructureWentLowPower
	StructuresJobsCancelled
	StructuresJobsPaused
	StructuresReinforcementChanged
	TowerAlertMsg
	TowerResourceAlertMsg
	TransactionReversalMsg
	TutorialMsg
	WarAdopted
	WarAllyInherited
	WarAllyOfferDeclinedMsg
	WarConcordInvalidates
	WarDeclared
	WarEndedHqSecurityDrop
	WarHQRemovedFromSpace
	WarInherited
	WarInvalid
	WarRetracted
	WarRetractedByConcord
	WarSurrenderDeclinedMsg
	WarSurrenderOfferMsg
)

// Category returns the entity category for supported notifications
// and reports whether the category is defined.
func (nt EveNotificationType) Category() (EveEntityCategory, bool) {
	c, ok := notificationCategories[nt]
	if !ok {
		return EveEntityUndefined, false
	}
	return c, ok
}

// Display returns a string representation for display.
func (nt EveNotificationType) Display() string {
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

// Group returns the group of a notification.
func (nt EveNotificationType) Group() EveNotificationGroup {
	return notificationGroups[nt]
}

// notificationCategories maps supported types to their entity category.
var notificationCategories = map[EveNotificationType]EveEntityCategory{
	AllWarSurrenderMsg:                        EveEntityAlliance,
	BillOutOfMoneyMsg:                         EveEntityCorporation,
	BillPaidCorpAllMsg:                        EveEntityCorporation,
	CharAppAcceptMsg:                          EveEntityCorporation,
	CharAppRejectMsg:                          EveEntityCorporation,
	CharAppWithdrawMsg:                        EveEntityCorporation,
	CharLeftCorpMsg:                           EveEntityCorporation,
	CorpAllBillMsg:                            EveEntityCorporation,
	CorpAppInvitedMsg:                         EveEntityCorporation,
	CorpAppNewMsg:                             EveEntityCorporation,
	CorpAppRejectCustomMsg:                    EveEntityCorporation,
	CorpWarSurrenderMsg:                       EveEntityCorporation,
	DeclareWar:                                EveEntityCorporation,
	EntosisCaptureStarted:                     EveEntityAlliance,
	IHubDestroyedByBillFailure:                EveEntityAlliance,
	InfrastructureHubBillAboutToExpire:        EveEntityAlliance,
	MoonminingAutomaticFracture:               EveEntityCorporation,
	MoonminingExtractionCancelled:             EveEntityCorporation,
	MoonminingExtractionFinished:              EveEntityCorporation,
	MoonminingExtractionStarted:               EveEntityCorporation,
	MoonminingLaserFired:                      EveEntityCorporation,
	OrbitalAttacked:                           EveEntityCorporation,
	OrbitalReinforced:                         EveEntityCorporation,
	OwnershipTransferred:                      EveEntityCorporation,
	SovAllClaimAcquiredMsg:                    EveEntityAlliance,
	SovAllClaimLostMsg:                        EveEntityAlliance,
	SovCommandNodeEventStarted:                EveEntityAlliance,
	SovStructureDestroyed:                     EveEntityAlliance,
	SovStructureReinforced:                    EveEntityAlliance,
	StructureAnchoring:                        EveEntityCorporation,
	StructureDestroyed:                        EveEntityCorporation,
	StructureFuelAlert:                        EveEntityCorporation,
	StructureImpendingAbandonmentAssetsAtRisk: EveEntityCharacter,
	StructureItemsDelivered:                   EveEntityCharacter,
	StructureItemsMovedToSafety:               EveEntityCharacter,
	StructureLostArmor:                        EveEntityCorporation,
	StructureLostShields:                      EveEntityCorporation,
	StructureOnline:                           EveEntityCorporation,
	StructureServicesOffline:                  EveEntityCorporation,
	StructuresReinforcementChanged:            EveEntityCorporation,
	StructureUnanchoring:                      EveEntityCorporation,
	StructureUnderAttack:                      EveEntityCorporation,
	StructureWentHighPower:                    EveEntityCorporation,
	StructureWentLowPower:                     EveEntityCorporation,
	TowerAlertMsg:                             EveEntityCorporation,
	TowerResourceAlertMsg:                     EveEntityCorporation,
	WarAdopted:                                EveEntityCorporation,
	WarDeclared:                               EveEntityCorporation,
	WarHQRemovedFromSpace:                     EveEntityCorporation,
	WarInherited:                              EveEntityCorporation,
	WarInvalid:                                EveEntityCorporation,
	WarRetractedByConcord:                     EveEntityCorporation,
}

var supportedTypes set.Set[EveNotificationType]

// NotificationTypesSupported returns all supported notification types.
func NotificationTypesSupported() set.Set[EveNotificationType] {
	if supportedTypes.Size() == 0 {
		for nt := range notificationCategories {
			supportedTypes.Add(nt)
		}
	}
	return supportedTypes
}

// notificationGroups maps all known types to their group.
var notificationGroups = map[EveNotificationType]EveNotificationGroup{
	AcceptedAlly:                              GroupWar,
	AcceptedSurrender:                         GroupWar,
	AgentRetiredTrigravian:                    GroupUnknown,
	AllAnchoringMsg:                           GroupSovereignty,
	AllMaintenanceBillMsg:                     GroupBills,
	AllStructureInvulnerableMsg:               GroupStructure,
	AllStructVulnerableMsg:                    GroupStructure,
	AllWarCorpJoinedAllianceMsg:               GroupWar,
	AllWarDeclaredMsg:                         GroupWar,
	AllWarInvalidatedMsg:                      GroupWar,
	AllWarRetractedMsg:                        GroupWar,
	AllWarSurrenderMsg:                        GroupWar,
	AllianceCapitalChanged:                    GroupMiscellaneous,
	AllianceWarDeclaredV2:                     GroupWar,
	AllyContractCancelled:                     GroupWar,
	AllyJoinedWarAggressorMsg:                 GroupWar,
	AllyJoinedWarAllyMsg:                      GroupWar,
	AllyJoinedWarDefenderMsg:                  GroupWar,
	BattlePunishFriendlyFire:                  GroupUnknown,
	BillOutOfMoneyMsg:                         GroupBills,
	BillPaidCorpAllMsg:                        GroupBills,
	BountyClaimMsg:                            GroupUnknown,
	BountyESSShared:                           GroupUnknown,
	BountyESSTaken:                            GroupUnknown,
	BountyPlacedAlliance:                      GroupUnknown,
	BountyPlacedChar:                          GroupUnknown,
	BountyPlacedCorp:                          GroupUnknown,
	BountyYourBountyClaimed:                   GroupUnknown,
	BuddyConnectContactAdd:                    GroupUnknown,
	CharAppAcceptMsg:                          GroupCorporate,
	CharAppRejectMsg:                          GroupCorporate,
	CharAppWithdrawMsg:                        GroupCorporate,
	CharLeftCorpMsg:                           GroupCorporate,
	CharMedalMsg:                              GroupCorporate,
	CharTerminationMsg:                        GroupCorporate,
	CloneActivationMsg:                        GroupUnknown,
	CloneActivationMsg2:                       GroupUnknown,
	CloneMovedMsg:                             GroupUnknown,
	CloneRevokedMsg1:                          GroupUnknown,
	CloneRevokedMsg2:                          GroupUnknown,
	CombatOperationFinished:                   GroupUnknown,
	ContactAdd:                                GroupContacts,
	ContactEdit:                               GroupContacts,
	ContainerPasswordMsg:                      GroupUnknown,
	ContractRegionChangedToPochven:            GroupUnknown,
	CorpAllBillMsg:                            GroupBills,
	CorpAppAcceptMsg:                          GroupCorporate,
	CorpAppInvitedMsg:                         GroupCorporate,
	CorpAppNewMsg:                             GroupCorporate,
	CorpAppRejectCustomMsg:                    GroupCorporate,
	CorpAppRejectMsg:                          GroupCorporate,
	CorpBecameWarEligible:                     GroupCorporate,
	CorpDividendMsg:                           GroupCorporate,
	CorpFriendlyFireDisableTimerCompleted:     GroupCorporate,
	CorpFriendlyFireDisableTimerStarted:       GroupCorporate,
	CorpFriendlyFireEnableTimerCompleted:      GroupCorporate,
	CorpFriendlyFireEnableTimerStarted:        GroupCorporate,
	CorpKicked:                                GroupCorporate,
	CorpLiquidationMsg:                        GroupCorporate,
	CorpNewCEOMsg:                             GroupCorporate,
	CorpNewsMsg:                               GroupCorporate,
	CorpNoLongerWarEligible:                   GroupCorporate,
	CorpOfficeExpirationMsg:                   GroupCorporate,
	CorpStructLostMsg:                         GroupCorporate,
	CorpTaxChangeMsg:                          GroupCorporate,
	CorpVoteCEORevokedMsg:                     GroupCorporate,
	CorpVoteMsg:                               GroupCorporate,
	CorpWarDeclaredMsg:                        GroupWar,
	CorpWarDeclaredV2:                         GroupWar,
	CorpWarFightingLegalMsg:                   GroupWar,
	CorpWarInvalidatedMsg:                     GroupWar,
	CorpWarRetractedMsg:                       GroupWar,
	CorpWarSurrenderMsg:                       GroupWar,
	CorporationGoalClosed:                     GroupCorporate,
	CorporationGoalCompleted:                  GroupCorporate,
	CorporationGoalCreated:                    GroupCorporate,
	CorporationGoalNameChange:                 GroupCorporate,
	CorporationLeft:                           GroupCorporate,
	CustomsMsg:                                GroupMiscellaneous,
	DeclareWar:                                GroupWar,
	DistrictAttacked:                          GroupWar,
	DustAppAcceptedMsg:                        GroupMiscellaneous,
	ESSMainBankLink:                           GroupUnknown,
	EntosisCaptureStarted:                     GroupSovereignty,
	ExpertSystemExpired:                       GroupMiscellaneous,
	ExpertSystemExpiryImminent:                GroupMiscellaneous,
	FWAllianceKickMsg:                         GroupFactionWarfare,
	FWAllianceWarningMsg:                      GroupFactionWarfare,
	FWCharKickMsg:                             GroupFactionWarfare,
	FWCharRankGainMsg:                         GroupFactionWarfare,
	FWCharRankLossMsg:                         GroupFactionWarfare,
	FWCharWarningMsg:                          GroupFactionWarfare,
	FWCorpJoinMsg:                             GroupFactionWarfare,
	FWCorpKickMsg:                             GroupFactionWarfare,
	FWCorpLeaveMsg:                            GroupFactionWarfare,
	FWCorpWarningMsg:                          GroupFactionWarfare,
	FacWarCorpJoinRequestMsg:                  GroupFactionWarfare,
	FacWarCorpJoinWithdrawMsg:                 GroupFactionWarfare,
	FacWarCorpLeaveRequestMsg:                 GroupFactionWarfare,
	FacWarCorpLeaveWithdrawMsg:                GroupFactionWarfare,
	FacWarLPDisqualifiedEvent:                 GroupFactionWarfare,
	FacWarLPDisqualifiedKill:                  GroupFactionWarfare,
	FacWarLPPayoutEvent:                       GroupFactionWarfare,
	FacWarLPPayoutKill:                        GroupFactionWarfare,
	GameTimeAdded:                             GroupUnknown,
	GameTimeReceived:                          GroupUnknown,
	GameTimeSent:                              GroupUnknown,
	GiftReceived:                              GroupUnknown,
	IHubDestroyedByBillFailure:                GroupSovereignty,
	IncursionCompletedMsg:                     GroupUnknown,
	IndustryOperationFinished:                 GroupUnknown,
	IndustryTeamAuctionLost:                   GroupUnknown,
	IndustryTeamAuctionWon:                    GroupUnknown,
	InfrastructureHubBillAboutToExpire:        GroupSovereignty,
	InsuranceExpirationMsg:                    GroupUnknown,
	InsuranceFirstShipMsg:                     GroupUnknown,
	InsuranceInvalidatedMsg:                   GroupUnknown,
	InsuranceIssuedMsg:                        GroupUnknown,
	InsurancePayoutMsg:                        GroupUnknown,
	InvasionCompletedMsg:                      GroupUnknown,
	InvasionSystemLogin:                       GroupUnknown,
	InvasionSystemStart:                       GroupUnknown,
	JumpCloneDeletedMsg1:                      GroupUnknown,
	JumpCloneDeletedMsg2:                      GroupUnknown,
	KillReportFinalBlow:                       GroupUnknown,
	KillReportVictim:                          GroupUnknown,
	KillRightAvailable:                        GroupUnknown,
	KillRightAvailableOpen:                    GroupUnknown,
	KillRightEarned:                           GroupUnknown,
	KillRightUnavailable:                      GroupUnknown,
	KillRightUnavailableOpen:                  GroupUnknown,
	KillRightUsed:                             GroupUnknown,
	LPAutoRedeemed:                            GroupUnknown,
	LocateCharMsg:                             GroupUnknown,
	MadeWarMutual:                             GroupUnknown,
	MercOfferRetractedMsg:                     GroupWar,
	MercOfferedNegotiationMsg:                 GroupWar,
	MissionCanceledTriglavian:                 GroupUnknown,
	MissionOfferExpirationMsg:                 GroupUnknown,
	MissionTimeoutMsg:                         GroupUnknown,
	MoonminingAutomaticFracture:               GroupMoonMining,
	MoonminingExtractionCancelled:             GroupMoonMining,
	MoonminingExtractionFinished:              GroupMoonMining,
	MoonminingExtractionStarted:               GroupMoonMining,
	MoonminingLaserFired:                      GroupMoonMining,
	MutualWarExpired:                          GroupWar,
	MutualWarInviteAccepted:                   GroupWar,
	MutualWarInviteRejected:                   GroupWar,
	MutualWarInviteSent:                       GroupWar,
	NPCStandingsGained:                        GroupUnknown,
	NPCStandingsLost:                          GroupUnknown,
	OfferToAllyRetracted:                      GroupWar,
	OfferedSurrender:                          GroupWar,
	OfferedToAlly:                             GroupWar,
	OfficeLeaseCanceledInsufficientStandings:  GroupCorporate,
	OldLscMessages:                            GroupUnknown,
	OperationFinished:                         GroupUnknown,
	OrbitalAttacked:                           GroupStructure,
	OrbitalReinforced:                         GroupStructure,
	OwnershipTransferred:                      GroupStructure,
	RaffleCreated:                             GroupUnknown,
	RaffleExpired:                             GroupUnknown,
	RaffleFinished:                            GroupUnknown,
	ReimbursementMsg:                          GroupUnknown,
	ResearchMissionAvailableMsg:               GroupUnknown,
	RetractsWar:                               GroupWar,
	SPAutoRedeemed:                            GroupUnknown,
	SeasonalChallengeCompleted:                GroupUnknown,
	SkinSequencingCompleted:                   GroupUnknown,
	SkyhookDeployed:                           GroupStructure,
	SkyhookDestroyed:                          GroupStructure,
	SkyhookLostShields:                        GroupStructure,
	SkyhookOnline:                             GroupStructure,
	SkyhookUnderAttack:                        GroupStructure,
	SovAllClaimAcquiredMsg:                    GroupSovereignty,
	SovAllClaimLostMsg:                        GroupSovereignty,
	SovCommandNodeEventStarted:                GroupSovereignty,
	SovCorpBillLateMsg:                        GroupSovereignty,
	SovCorpClaimFailMsg:                       GroupSovereignty,
	SovDisruptorMsg:                           GroupSovereignty,
	SovStationEnteredFreeport:                 GroupSovereignty,
	SovStructureDestroyed:                     GroupSovereignty,
	SovStructureReinforced:                    GroupSovereignty,
	SovStructureSelfDestructCancel:            GroupSovereignty,
	SovStructureSelfDestructFinished:          GroupSovereignty,
	SovStructureSelfDestructRequested:         GroupSovereignty,
	SovereigntyIHDamageMsg:                    GroupSovereignty,
	SovereigntySBUDamageMsg:                   GroupSovereignty,
	SovereigntyTCUDamageMsg:                   GroupSovereignty,
	StationAggressionMsg1:                     GroupUnknown,
	StationAggressionMsg2:                     GroupUnknown,
	StationConquerMsg:                         GroupUnknown,
	StationServiceDisabled:                    GroupUnknown,
	StationServiceEnabled:                     GroupUnknown,
	StationStateChangeMsg:                     GroupUnknown,
	StoryLineMissionAvailableMsg:              GroupUnknown,
	StructureAnchoring:                        GroupStructure,
	StructureCourierContractChanged:           GroupStructure,
	StructureDestroyed:                        GroupStructure,
	StructureFuelAlert:                        GroupStructure,
	StructureImpendingAbandonmentAssetsAtRisk: GroupStructure,
	StructureItemsDelivered:                   GroupStructure,
	StructureItemsMovedToSafety:               GroupStructure,
	StructureLostArmor:                        GroupStructure,
	StructureLostShields:                      GroupStructure,
	StructureLowReagentsAlert:                 GroupStructure,
	StructureNoReagentsAlert:                  GroupStructure,
	StructureOnline:                           GroupStructure,
	StructurePaintPurchased:                   GroupStructure,
	StructureServicesOffline:                  GroupStructure,
	StructureUnanchoring:                      GroupStructure,
	StructureUnderAttack:                      GroupStructure,
	StructureWentHighPower:                    GroupStructure,
	StructureWentLowPower:                     GroupStructure,
	StructuresJobsCancelled:                   GroupStructure,
	StructuresJobsPaused:                      GroupStructure,
	StructuresReinforcementChanged:            GroupStructure,
	TowerAlertMsg:                             GroupStructure,
	TowerResourceAlertMsg:                     GroupStructure,
	TransactionReversalMsg:                    GroupUnknown,
	TutorialMsg:                               GroupUnknown,
	WarAdopted:                                GroupWar,
	WarAllyInherited:                          GroupWar,
	WarAllyOfferDeclinedMsg:                   GroupWar,
	WarConcordInvalidates:                     GroupWar,
	WarDeclared:                               GroupWar,
	WarEndedHqSecurityDrop:                    GroupWar,
	WarHQRemovedFromSpace:                     GroupWar,
	WarInherited:                              GroupWar,
	WarInvalid:                                GroupWar,
	WarRetracted:                              GroupWar,
	WarRetractedByConcord:                     GroupWar,
	WarSurrenderDeclinedMsg:                   GroupWar,
	WarSurrenderOfferMsg:                      GroupWar,
}

var groupTypes map[EveNotificationGroup]set.Set[EveNotificationType]

// NotificationGroupTypes returns the types of a [EveNotificationGroup].
func NotificationGroupTypes(g EveNotificationGroup) set.Set[EveNotificationType] {
	if groupTypes == nil {
		groupTypes = make(map[EveNotificationGroup]set.Set[EveNotificationType])
		for t, g := range notificationGroups {
			if g == GroupUnknown {
				g = GroupMiscellaneous
			}
			x := groupTypes[g]
			x.Add(t)
			groupTypes[g] = x
		}
	}
	return groupTypes[g]
}

// EveNotificationGroup represents a group of notification types.
type EveNotificationGroup uint

const (
	GroupBills EveNotificationGroup = iota + 1
	GroupFactionWarfare
	GroupContacts
	GroupCorporate
	GroupInsurance
	GroupInsurgencies
	GroupMoonMining
	GroupMiscellaneous
	GroupOld
	GroupSovereignty
	GroupStructure
	GroupWar

	GroupUnknown
	GroupUnread
	GroupAll
)

var group2Name = map[EveNotificationGroup]string{
	GroupAll:            "All",
	GroupBills:          "Bills",
	GroupContacts:       "Contacts",
	GroupCorporate:      "Corporate",
	GroupFactionWarfare: "Faction Warfare",
	GroupInsurance:      "Insurance",
	GroupInsurgencies:   "Insurgencies",
	GroupMiscellaneous:  "Miscellaneous",
	GroupMoonMining:     "Moon Mining",
	GroupOld:            "Old",
	GroupSovereignty:    "Sovereignty",
	GroupStructure:      "Structure",
	GroupUnknown:        "Unknown",
	GroupUnread:         "Unread",
	GroupWar:            "War",
}

func (c EveNotificationGroup) String() string {
	return group2Name[c]
}

// NotificationGroups returns a slice of all regular groups in alphabetical order.
func NotificationGroups() []EveNotificationGroup {
	return []EveNotificationGroup{
		GroupBills,
		GroupFactionWarfare,
		GroupContacts,
		GroupCorporate,
		GroupInsurance,
		GroupInsurgencies,
		GroupMoonMining,
		GroupMiscellaneous,
		GroupOld,
		GroupSovereignty,
		GroupStructure,
		GroupWar,
	}
}

var eveNotificationTypeFromString map[string]EveNotificationType

// EveNotificationTypeFromString returns the [EveNotificationType]
// for a matching string and reports whether a match was found
func EveNotificationTypeFromString(s string) (EveNotificationType, bool) {
	if eveNotificationTypeFromString == nil {
		eveNotificationTypeFromString = make(map[string]EveNotificationType)
		for nt := range notificationGroups {
			eveNotificationTypeFromString[nt.String()] = nt
		}
	}
	nt, found := eveNotificationTypeFromString[s]
	if !found {
		return UnknownNotification, false
	}
	return nt, true
}

type CharacterNotification struct {
	ID             int64
	Body           optional.Optional[string] // generated body text in markdown
	CharacterID    int32
	IsProcessed    bool
	IsRead         bool
	NotificationID int64
	Sender         *EveEntity
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string] // generated title text in markdown
	Type           EveNotificationType
}

// TitleDisplay returns the rendered title when it exists or else the fake tile.
func (cn *CharacterNotification) TitleDisplay() string {
	if cn.Title.IsEmpty() {
		return cn.TitleFake()
	}
	return cn.Title.ValueOrZero()
}

// TitleFake returns a title for output made from the name of the type.
func (cn *CharacterNotification) TitleFake() string {
	var b strings.Builder
	var last rune
	for _, r := range cn.Type.String() {
		if unicode.IsUpper(r) && unicode.IsLower(last) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		last = r
	}
	return b.String()
}

// BodyPlain returns the body of a notification as plain text.
func (cn *CharacterNotification) BodyPlain() (optional.Optional[string], error) {
	var b optional.Optional[string]
	if cn.Body.IsEmpty() {
		return b, nil
	}
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(cn.Body.ValueOrZero()), &buf); err != nil {
		return b, fmt.Errorf("convert notification body: %w", err)
	}
	b.Set(evehtml.Strip(buf.String()))
	return b, nil
}
