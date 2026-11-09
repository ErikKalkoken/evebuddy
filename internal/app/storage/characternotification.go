package storage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

var notificationTypeFromString = map[string]app.EveNotificationType{
	"AcceptedAlly":                              app.AcceptedAlly,
	"AcceptedSurrender":                         app.AcceptedSurrender,
	"AgentRetiredTrigravian":                    app.AgentRetiredTrigravian,
	"AllAnchoringMsg":                           app.AllAnchoringMsg,
	"AllianceCapitalChanged":                    app.AllianceCapitalChanged,
	"AllianceWarDeclaredV2":                     app.AllianceWarDeclaredV2,
	"AllMaintenanceBillMsg":                     app.AllMaintenanceBillMsg,
	"AllStrucInvulnerableMsg":                   app.AllStructureInvulnerableMsg,
	"AllStructVulnerableMsg":                    app.AllStructVulnerableMsg,
	"AllWarCorpJoinedAllianceMsg":               app.AllWarCorpJoinedAllianceMsg,
	"AllWarDeclaredMsg":                         app.AllWarDeclaredMsg,
	"AllWarInvalidatedMsg":                      app.AllWarInvalidatedMsg,
	"AllWarRetractedMsg":                        app.AllWarRetractedMsg,
	"AllWarSurrenderMsg":                        app.AllWarSurrenderMsg,
	"AllyContractCancelled":                     app.AllyContractCancelled,
	"AllyJoinedWarAggressorMsg":                 app.AllyJoinedWarAggressorMsg,
	"AllyJoinedWarAllyMsg":                      app.AllyJoinedWarAllyMsg,
	"AllyJoinedWarDefenderMsg":                  app.AllyJoinedWarDefenderMsg,
	"BattlePunishFriendlyFire":                  app.BattlePunishFriendlyFire,
	"BillOutOfMoneyMsg":                         app.BillOutOfMoneyMsg,
	"BillPaidCorpAllMsg":                        app.BillPaidCorpAllMsg,
	"BountyClaimMsg":                            app.BountyClaimMsg,
	"BountyESSShared":                           app.BountyESSShared,
	"BountyESSTaken":                            app.BountyESSTaken,
	"BountyPlacedAlliance":                      app.BountyPlacedAlliance,
	"BountyPlacedChar":                          app.BountyPlacedChar,
	"BountyPlacedCorp":                          app.BountyPlacedCorp,
	"BountyYourBountyClaimed":                   app.BountyYourBountyClaimed,
	"BuddyConnectContactAdd":                    app.BuddyConnectContactAdd,
	"CharAppAcceptMsg":                          app.CharAppAcceptMsg,
	"CharAppRejectMsg":                          app.CharAppRejectMsg,
	"CharAppWithdrawMsg":                        app.CharAppWithdrawMsg,
	"CharLeftCorpMsg":                           app.CharLeftCorpMsg,
	"CharMedalMsg":                              app.CharMedalMsg,
	"CharTerminationMsg":                        app.CharTerminationMsg,
	"CloneActivationMsg":                        app.CloneActivationMsg,
	"CloneActivationMsg2":                       app.CloneActivationMsg2,
	"CloneMovedMsg":                             app.CloneMovedMsg,
	"CloneRevokedMsg1":                          app.CloneRevokedMsg1,
	"CloneRevokedMsg2":                          app.CloneRevokedMsg2,
	"CombatOperationFinished":                   app.CombatOperationFinished,
	"ContactAdd":                                app.ContactAdd,
	"ContactEdit":                               app.ContactEdit,
	"ContainerPasswordMsg":                      app.ContainerPasswordMsg,
	"ContractRegionChangedToPochven":            app.ContractRegionChangedToPochven,
	"CorpAllBillMsg":                            app.CorpAllBillMsg,
	"CorpAppAcceptMsg":                          app.CorpAppAcceptMsg,
	"CorpAppInvitedMsg":                         app.CorpAppInvitedMsg,
	"CorpAppNewMsg":                             app.CorpAppNewMsg,
	"CorpAppRejectCustomMsg":                    app.CorpAppRejectCustomMsg,
	"CorpAppRejectMsg":                          app.CorpAppRejectMsg,
	"CorpBecameWarEligible":                     app.CorpBecameWarEligible,
	"CorpDividendMsg":                           app.CorpDividendMsg,
	"CorpFriendlyFireDisableTimerCompleted":     app.CorpFriendlyFireDisableTimerCompleted,
	"CorpFriendlyFireDisableTimerStarted":       app.CorpFriendlyFireDisableTimerStarted,
	"CorpFriendlyFireEnableTimerCompleted":      app.CorpFriendlyFireEnableTimerCompleted,
	"CorpFriendlyFireEnableTimerStarted":        app.CorpFriendlyFireEnableTimerStarted,
	"CorpKicked":                                app.CorpKicked,
	"CorpLiquidationMsg":                        app.CorpLiquidationMsg,
	"CorpNewCEOMsg":                             app.CorpNewCEOMsg,
	"CorpNewsMsg":                               app.CorpNewsMsg,
	"CorpNoLongerWarEligible":                   app.CorpNoLongerWarEligible,
	"CorpOfficeExpirationMsg":                   app.CorpOfficeExpirationMsg,
	"CorporationGoalClosed":                     app.CorporationGoalClosed,
	"CorporationGoalCompleted":                  app.CorporationGoalCompleted,
	"CorporationGoalCreated":                    app.CorporationGoalCreated,
	"CorporationGoalNameChange":                 app.CorporationGoalNameChange,
	"CorporationLeft":                           app.CorporationLeft,
	"CorpStructLostMsg":                         app.CorpStructLostMsg,
	"CorpTaxChangeMsg":                          app.CorpTaxChangeMsg,
	"CorpVoteCEORevokedMsg":                     app.CorpVoteCEORevokedMsg,
	"CorpVoteMsg":                               app.CorpVoteMsg,
	"CorpWarDeclaredMsg":                        app.CorpWarDeclaredMsg,
	"CorpWarDeclaredV2":                         app.CorpWarDeclaredV2,
	"CorpWarFightingLegalMsg":                   app.CorpWarFightingLegalMsg,
	"CorpWarInvalidatedMsg":                     app.CorpWarInvalidatedMsg,
	"CorpWarRetractedMsg":                       app.CorpWarRetractedMsg,
	"CorpWarSurrenderMsg":                       app.CorpWarSurrenderMsg,
	"CustomsMsg":                                app.CustomsMsg,
	"DeclareWar":                                app.DeclareWar,
	"DistrictAttacked":                          app.DistrictAttacked,
	"DustAppAcceptedMsg":                        app.DustAppAcceptedMsg,
	"EntosisCaptureStarted":                     app.EntosisCaptureStarted,
	"ESSMainBankLink":                           app.ESSMainBankLink,
	"ExpertSystemExpired":                       app.ExpertSystemExpired,
	"ExpertSystemExpiryImminent":                app.ExpertSystemExpiryImminent,
	"FacWarCorpJoinRequestMsg":                  app.FacWarCorpJoinRequestMsg,
	"FacWarCorpJoinWithdrawMsg":                 app.FacWarCorpJoinWithdrawMsg,
	"FacWarCorpLeaveRequestMsg":                 app.FacWarCorpLeaveRequestMsg,
	"FacWarCorpLeaveWithdrawMsg":                app.FacWarCorpLeaveWithdrawMsg,
	"FacWarLPDisqualifiedEvent":                 app.FacWarLPDisqualifiedEvent,
	"FacWarLPDisqualifiedKill":                  app.FacWarLPDisqualifiedKill,
	"FacWarLPPayoutEvent":                       app.FacWarLPPayoutEvent,
	"FacWarLPPayoutKill":                        app.FacWarLPPayoutKill,
	"FWAllianceKickMsg":                         app.FWAllianceKickMsg,
	"FWAllianceWarningMsg":                      app.FWAllianceWarningMsg,
	"FWCharKickMsg":                             app.FWCharKickMsg,
	"FWCharRankGainMsg":                         app.FWCharRankGainMsg,
	"FWCharRankLossMsg":                         app.FWCharRankLossMsg,
	"FWCharWarningMsg":                          app.FWCharWarningMsg,
	"FWCorpJoinMsg":                             app.FWCorpJoinMsg,
	"FWCorpKickMsg":                             app.FWCorpKickMsg,
	"FWCorpLeaveMsg":                            app.FWCorpLeaveMsg,
	"FWCorpWarningMsg":                          app.FWCorpWarningMsg,
	"GameTimeAdded":                             app.GameTimeAdded,
	"GameTimeReceived":                          app.GameTimeReceived,
	"GameTimeSent":                              app.GameTimeSent,
	"GiftReceived":                              app.GiftReceived,
	"IHubDestroyedByBillFailure":                app.IHubDestroyedByBillFailure,
	"IncursionCompletedMsg":                     app.IncursionCompletedMsg,
	"IndustryOperationFinished":                 app.IndustryOperationFinished,
	"IndustryTeamAuctionLost":                   app.IndustryTeamAuctionLost,
	"IndustryTeamAuctionWon":                    app.IndustryTeamAuctionWon,
	"InfrastructureHubBillAboutToExpire":        app.InfrastructureHubBillAboutToExpire,
	"InsuranceExpirationMsg":                    app.InsuranceExpirationMsg,
	"InsuranceFirstShipMsg":                     app.InsuranceFirstShipMsg,
	"InsuranceInvalidatedMsg":                   app.InsuranceInvalidatedMsg,
	"InsuranceIssuedMsg":                        app.InsuranceIssuedMsg,
	"InsurancePayoutMsg":                        app.InsurancePayoutMsg,
	"InvasionCompletedMsg":                      app.InvasionCompletedMsg,
	"InvasionSystemLogin":                       app.InvasionSystemLogin,
	"InvasionSystemStart":                       app.InvasionSystemStart,
	"JumpCloneDeletedMsg1":                      app.JumpCloneDeletedMsg1,
	"JumpCloneDeletedMsg2":                      app.JumpCloneDeletedMsg2,
	"KillReportFinalBlow":                       app.KillReportFinalBlow,
	"KillReportVictim":                          app.KillReportVictim,
	"KillRightAvailable":                        app.KillRightAvailable,
	"KillRightAvailableOpen":                    app.KillRightAvailableOpen,
	"KillRightEarned":                           app.KillRightEarned,
	"KillRightUnavailable":                      app.KillRightUnavailable,
	"KillRightUnavailableOpen":                  app.KillRightUnavailableOpen,
	"KillRightUsed":                             app.KillRightUsed,
	"LocateCharMsg":                             app.LocateCharMsg,
	"LPAutoRedeemed":                            app.LPAutoRedeemed,
	"MadeWarMutual":                             app.MadeWarMutual,
	"MercenaryDenAttacked":                      app.MercenaryDenAttacked,
	"MercenaryDenReinforced":                    app.MercenaryDenReinforced,
	"MercOfferedNegotiationMsg":                 app.MercOfferedNegotiationMsg,
	"MercOfferRetractedMsg":                     app.MercOfferRetractedMsg,
	"MissionCanceledTriglavian":                 app.MissionCanceledTriglavian,
	"MissionOfferExpirationMsg":                 app.MissionOfferExpirationMsg,
	"MissionTimeoutMsg":                         app.MissionTimeoutMsg,
	"MoonminingAutomaticFracture":               app.MoonminingAutomaticFracture,
	"MoonminingExtractionCancelled":             app.MoonminingExtractionCancelled,
	"MoonminingExtractionFinished":              app.MoonminingExtractionFinished,
	"MoonminingExtractionStarted":               app.MoonminingExtractionStarted,
	"MoonminingLaserFired":                      app.MoonminingLaserFired,
	"MutualWarExpired":                          app.MutualWarExpired,
	"MutualWarInviteAccepted":                   app.MutualWarInviteAccepted,
	"MutualWarInviteRejected":                   app.MutualWarInviteRejected,
	"MutualWarInviteSent":                       app.MutualWarInviteSent,
	"NPCStandingsGained":                        app.NPCStandingsGained,
	"NPCStandingsLost":                          app.NPCStandingsLost,
	"OfferedSurrender":                          app.OfferedSurrender,
	"OfferedToAlly":                             app.OfferedToAlly,
	"OfferToAllyRetracted":                      app.OfferToAllyRetracted,
	"OfficeLeaseCanceledInsufficientStandings":  app.OfficeLeaseCanceledInsufficientStandings,
	"OldLscMessages":                            app.OldLscMessages,
	"OperationFinished":                         app.OperationFinished,
	"OrbitalAttacked":                           app.OrbitalAttacked,
	"OrbitalReinforced":                         app.OrbitalReinforced,
	"OwnershipTransferred":                      app.OwnershipTransferred,
	"RaffleCreated":                             app.RaffleCreated,
	"RaffleExpired":                             app.RaffleExpired,
	"RaffleFinished":                            app.RaffleFinished,
	"ReimbursementMsg":                          app.ReimbursementMsg,
	"ResearchMissionAvailableMsg":               app.ResearchMissionAvailableMsg,
	"RetractsWar":                               app.RetractsWar,
	"SeasonalChallengeCompleted":                app.SeasonalChallengeCompleted,
	"SkinSequencingCompleted":                   app.SkinSequencingCompleted,
	"SkyhookDeployed":                           app.SkyhookDeployed,
	"SkyhookDestroyed":                          app.SkyhookDestroyed,
	"SkyhookLostShields":                        app.SkyhookLostShields,
	"SkyhookOnline":                             app.SkyhookOnline,
	"SkyhookUnderAttack":                        app.SkyhookUnderAttack,
	"SovAllClaimAquiredMsg":                     app.SovAllClaimAcquiredMsg,
	"SovAllClaimLostMsg":                        app.SovAllClaimLostMsg,
	"SovCommandNodeEventStarted":                app.SovCommandNodeEventStarted,
	"SovCorpBillLateMsg":                        app.SovCorpBillLateMsg,
	"SovCorpClaimFailMsg":                       app.SovCorpClaimFailMsg,
	"SovDisruptorMsg":                           app.SovDisruptorMsg,
	"SovereigntyIHDamageMsg":                    app.SovereigntyIHDamageMsg,
	"SovereigntySBUDamageMsg":                   app.SovereigntySBUDamageMsg,
	"SovereigntyTCUDamageMsg":                   app.SovereigntyTCUDamageMsg,
	"SovStationEnteredFreeport":                 app.SovStationEnteredFreeport,
	"SovStructureDestroyed":                     app.SovStructureDestroyed,
	"SovStructureReinforced":                    app.SovStructureReinforced,
	"SovStructureSelfDestructCancel":            app.SovStructureSelfDestructCancel,
	"SovStructureSelfDestructFinished":          app.SovStructureSelfDestructFinished,
	"SovStructureSelfDestructRequested":         app.SovStructureSelfDestructRequested,
	"SPAutoRedeemed":                            app.SPAutoRedeemed,
	"StationAggressionMsg1":                     app.StationAggressionMsg1,
	"StationAggressionMsg2":                     app.StationAggressionMsg2,
	"StationConquerMsg":                         app.StationConquerMsg,
	"StationServiceDisabled":                    app.StationServiceDisabled,
	"StationServiceEnabled":                     app.StationServiceEnabled,
	"StationStateChangeMsg":                     app.StationStateChangeMsg,
	"StoryLineMissionAvailableMsg":              app.StoryLineMissionAvailableMsg,
	"StructureAnchoring":                        app.StructureAnchoring,
	"StructureCourierContractChanged":           app.StructureCourierContractChanged,
	"StructureDestroyed":                        app.StructureDestroyed,
	"StructureFuelAlert":                        app.StructureFuelAlert,
	"StructureImpendingAbandonmentAssetsAtRisk": app.StructureImpendingAbandonmentAssetsAtRisk,
	"StructureItemsDelivered":                   app.StructureItemsDelivered,
	"StructureItemsMovedToSafety":               app.StructureItemsMovedToSafety,
	"StructureLostArmor":                        app.StructureLostArmor,
	"StructureLostShields":                      app.StructureLostShields,
	"StructureLowReagentsAlert":                 app.StructureLowReagentsAlert,
	"StructureNoReagentsAlert":                  app.StructureNoReagentsAlert,
	"StructureOnline":                           app.StructureOnline,
	"StructurePaintPurchased":                   app.StructurePaintPurchased,
	"StructureServicesOffline":                  app.StructureServicesOffline,
	"StructuresJobsCancelled":                   app.StructuresJobsCancelled,
	"StructuresJobsPaused":                      app.StructuresJobsPaused,
	"StructuresReinforcementChanged":            app.StructuresReinforcementChanged,
	"StructureUnanchoring":                      app.StructureUnanchoring,
	"StructureUnderAttack":                      app.StructureUnderAttack,
	"StructureWentHighPower":                    app.StructureWentHighPower,
	"StructureWentLowPower":                     app.StructureWentLowPower,
	"TowerAlertMsg":                             app.TowerAlertMsg,
	"TowerResourceAlertMsg":                     app.TowerResourceAlertMsg,
	"TransactionReversalMsg":                    app.TransactionReversalMsg,
	"TutorialMsg":                               app.TutorialMsg,
	"WarAdopted ":                               app.WarAdopted,
	"WarAllyInherited":                          app.WarAllyInherited,
	"WarAllyOfferDeclinedMsg":                   app.WarAllyOfferDeclinedMsg,
	"WarConcordInvalidates":                     app.WarConcordInvalidates,
	"WarDeclared":                               app.WarDeclared,
	"WarEndedHqSecurityDrop":                    app.WarEndedHqSecurityDrop,
	"WarHQRemovedFromSpace":                     app.WarHQRemovedFromSpace,
	"WarInherited":                              app.WarInherited,
	"WarInvalid":                                app.WarInvalid,
	"WarRetracted":                              app.WarRetracted,
	"WarRetractedByConcord":                     app.WarRetractedByConcord,
	"WarSurrenderDeclinedMsg":                   app.WarSurrenderDeclinedMsg,
	"WarSurrenderOfferMsg":                      app.WarSurrenderOfferMsg,
}

// EveNotificationTypeFromESIString returns a notifications from a matching ESI string
// or [app.UnknownNotification] if not found.
func EveNotificationTypeFromESIString(name string) (app.EveNotificationType, bool) {
	nt, ok := notificationTypeFromString[name]
	if !ok {
		return app.UnknownNotification, false
	}
	return nt, true
}

var notificationTypeToString map[app.EveNotificationType]string

// EveNotificationTypeToESIString returns the ESI string for a notification
// and reports whether it was found.
func EveNotificationTypeToESIString(nt app.EveNotificationType) (string, bool) {
	if notificationTypeToString == nil {
		notificationTypeToString = make(map[app.EveNotificationType]string)
		for k, v := range notificationTypeFromString {
			notificationTypeToString[v] = k
		}
	}
	s, ok := notificationTypeToString[nt]
	if !ok {
		return "", false
	}
	return s, true
}

type CreateCharacterNotificationParams struct {
	Body           optional.Optional[string]
	CharacterID    int64
	IsProcessed    bool
	IsRead         bool
	NotificationID int64
	RecipientID    optional.Optional[int64]
	SenderID       int64
	Text           optional.Optional[string]
	Timestamp      time.Time
	Title          optional.Optional[string]
	Type           string
}

// CreateCharacterNotification stores a notification and creates its type if missing.
func (st *Storage) CreateCharacterNotification(ctx context.Context, arg CreateCharacterNotificationParams) error {
	return st.CreateCharacterNotifications(ctx, []CreateCharacterNotificationParams{arg})
}

// CreateCharacterNotifications stores all notifications or none of them.
// Missing notification types are created.
func (st *Storage) CreateCharacterNotifications(ctx context.Context, args []CreateCharacterNotificationParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterNotifications: %w", err)
	}
	wrapArgErr := func(arg CreateCharacterNotificationParams, err error) error {
		return fmt.Errorf("CreateCharacterNotifications: %+v: %w", struct {
			CharacterID    int64
			NotificationID int64
			SenderID       int64
			Type           string
		}{
			arg.CharacterID,
			arg.NotificationID,
			arg.SenderID,
			arg.Type,
		}, err)
	}
	if len(args) == 0 {
		return nil
	}
	// validate before the transaction to avoid blocking other writers for an invalid batch
	var names set.Set[string]
	for _, arg := range args {
		if arg.CharacterID == 0 || arg.NotificationID == 0 || arg.SenderID == 0 || arg.Type == "" {
			return wrapArgErr(arg, app.ErrInvalid)
		}
		names.Add(arg.Type)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	rows, err := qtx.ListNotificationTypeIDsByName(ctx, slices.Collect(names.All()))
	if err != nil {
		return wrapErr(err)
	}
	typeIDs := make(map[string]int64, len(rows))
	for _, r := range rows {
		typeIDs[r.Name] = r.ID
	}
	for name := range names.All() {
		if _, found := typeIDs[name]; found {
			continue
		}
		id, err := qtx.CreateNotificationType(ctx, name)
		if err != nil {
			return wrapErr(err)
		}
		typeIDs[name] = id
	}
	for _, arg := range args {
		err := qtx.CreateCharacterNotification(ctx, queries.CreateCharacterNotificationParams{
			Body:           optional.ToNullString(arg.Body),
			CharacterID:    arg.CharacterID,
			IsRead:         arg.IsRead,
			IsProcessed:    arg.IsProcessed,
			NotificationID: arg.NotificationID,
			RecipientID:    optional.ToNullInt64(arg.RecipientID),
			SenderID:       arg.SenderID,
			Text:           arg.Text.ValueOrZero(),
			Timestamp:      arg.Timestamp.UTC(), // stored times must be UTC, because SQLite compares them as text
			Title:          optional.ToNullString(arg.Title),
			TypeID:         typeIDs[arg.Type],
		})
		if err != nil {
			return wrapArgErr(arg, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCharacterNotifications(ctx context.Context, characterID int64, notificationIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterNotifications: %d %s: %w", characterID, notificationIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if notificationIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterNotifications(ctx, queries.DeleteCharacterNotificationsParams{
		CharacterID:     characterID,
		NotificationIds: slices.Collect(notificationIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCharacterNotification(ctx context.Context, characterID int64, notificationID int64) (*app.CharacterNotification, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterNotification: %d %d: %w", characterID, notificationID, err)
	}
	if characterID == 0 || notificationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterNotification(ctx, queries.GetCharacterNotificationParams{
		CharacterID:    characterID,
		NotificationID: notificationID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
	if !found {
		nt = app.UnknownNotification
	}
	o := characterNotificationFromDBModel(
		r.CharacterNotification,
		r.EveEntity,
		nt,
		nullEveEntity{
			category: r.RecipientCategory,
			id:       r.CharacterNotification.RecipientID,
			name:     r.RecipientName,
		},
	)
	return o, err
}

func (st *Storage) ListCharacterNotificationIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterNotificationIDs: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCharacterNotificationIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListAllCharacterNotifications(ctx context.Context) ([]*app.CharacterNotification, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListAllCharacterNotifications: %w", err)
	}
	rows, err := st.qRO.ListAllCharacterNotifications(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(
			r.CharacterNotification,
			r.EveEntity,
			nt,
			nullEveEntity{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
		)
	}
	return ee, nil
}

// CharacterNotificationState is the part of a stored notification which updates compare with ESI.
type CharacterNotificationState struct {
	Body           optional.Optional[string]
	ID             int64
	IsRead         bool
	NotificationID int64
	Title          optional.Optional[string]
	Type           app.EveNotificationType
}

// ListCharacterNotificationStates returns the states of all notifications of a character.
func (st *Storage) ListCharacterNotificationStates(ctx context.Context, characterID int64) ([]CharacterNotificationState, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterNotificationStates: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterNotificationStates(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]CharacterNotificationState, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.TypeName)
		if !found {
			nt = app.UnknownNotification
		}
		oo[i] = CharacterNotificationState{
			Body:           optional.FromNullString(r.Body),
			ID:             r.ID,
			IsRead:         r.IsRead,
			NotificationID: r.NotificationID,
			Title:          optional.FromNullString(r.Title),
			Type:           nt,
		}
	}
	return oo, nil
}

func (st *Storage) ListCharacterNotifications(ctx context.Context, characterID int64) ([]*app.CharacterNotification, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterNotifications: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterNotifications(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(
			r.CharacterNotification,
			r.EveEntity,
			nt,
			nullEveEntity{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
		)
	}
	return ee, nil
}

// ListCharacterNotificationsUnprocessed returns all unprocessed notifications for character characterID.
// Notifications older then earliest are ignored or which have no body or title are ignored.
// Notifications which are duplicates of already processed ones are ignored too.
func (st *Storage) ListCharacterNotificationsUnprocessed(ctx context.Context, characterID int64, earliest time.Time) ([]*app.CharacterNotification, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterNotificationsUnprocessed: %d %v: %w", characterID, earliest, err)
	}
	if characterID == 0 || earliest.IsZero() {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterNotificationsUnprocessed(ctx, queries.ListCharacterNotificationsUnprocessedParams{
		CharacterID: characterID,
		Timestamp:   earliest.UTC(), // make sure we are comparing UTC with UTC
		Timestamp_2: earliest.UTC(),
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(
			r.CharacterNotification,
			r.EveEntity,
			nt,
			nullEveEntity{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
		)
	}
	return ee, nil
}

func characterNotificationFromDBModel(
	cn queries.CharacterNotification,
	sender queries.EveEntity,
	nt app.EveNotificationType,
	recipient nullEveEntity,
) *app.CharacterNotification {
	o2 := &app.CharacterNotification{
		ID:             cn.ID,
		Body:           optional.FromNullString(cn.Body),
		CharacterID:    cn.CharacterID,
		IsProcessed:    cn.IsProcessed,
		IsRead:         cn.IsRead,
		NotificationID: cn.NotificationID,
		Recipient:      eveEntityFromNullableDBModel(recipient),
		Sender:         eveEntityFromDBModel(sender),
		Text:           optional.FromZeroValue(cn.Text),
		Timestamp:      cn.Timestamp,
		Title:          optional.FromNullString(cn.Title),
		Type:           nt,
	}
	return o2
}

type UpdateCharacterNotificationParams struct {
	Body   optional.Optional[string]
	ID     int64
	IsRead optional.Optional[bool] // empty keeps the stored value
	Title  optional.Optional[string]
}

// UpdateCharacterNotifications updates all notifications or none of them.
func (st *Storage) UpdateCharacterNotifications(ctx context.Context, args []UpdateCharacterNotificationParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterNotifications: %w", err)
	}
	wrapArgErr := func(arg UpdateCharacterNotificationParams, err error) error {
		return fmt.Errorf("UpdateCharacterNotifications: ID: %d: %w", arg.ID, err)
	}
	if len(args) == 0 {
		return nil
	}
	// validate before the transaction to avoid blocking other writers for an invalid batch
	for _, arg := range args {
		if arg.ID == 0 {
			return wrapArgErr(arg, app.ErrInvalid)
		}
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for _, arg := range args {
		err := qtx.UpdateCharacterNotification(ctx, queries.UpdateCharacterNotificationParams{
			ID:     arg.ID,
			Body:   optional.ToNullString(arg.Body),
			IsRead: optional.ToNullBool(arg.IsRead),
			Title:  optional.ToNullString(arg.Title),
		})
		if err != nil {
			return wrapArgErr(arg, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

// UpdateCharacterNotificationsSetProcessed marks all notifications with the same notificationID as processed.
func (st *Storage) UpdateCharacterNotificationsSetProcessed(ctx context.Context, characterID, notificationID int64) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterNotificationsSetProcessed: %d %d: %w", characterID, notificationID, err)
	}
	if characterID == 0 || notificationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCharacterNotificationsSetProcessed(ctx, queries.UpdateCharacterNotificationsSetProcessedParams{
		CharacterID:    characterID,
		NotificationID: notificationID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateCharacterNotificationsSetIsRead(ctx context.Context, ids set.Set[int64], isRead bool) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterNotificationsSetIsRead: %s %v: %w", ids, isRead, err)
	}
	if ids.Size() == 0 {
		return nil
	}
	err := st.qRW.UpdateCharacterNotificationsSetIsRead(ctx, queries.UpdateCharacterNotificationsSetIsReadParams{
		IsRead: isRead,
		Ids:    slices.Collect(ids.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
