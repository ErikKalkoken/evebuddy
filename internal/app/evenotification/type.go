package evenotification

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
}

// SupportedTypes returns a list of all supported notification types.
func SupportedTypes() []Type {
	return supportedTypes
}
