package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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
func (*Storage) EveNotificationTypeToESIString(nt app.EveNotificationType) (string, bool) {
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

func (st *Storage) CountCharacterNotifications(ctx context.Context, characterID int32) (map[app.EveNotificationType][]int, error) {
	rows, err := st.qRO.CountCharacterNotifications(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("count notifications for character %d: %w", characterID, err)
	}
	m := make(map[app.EveNotificationType][]int)
	for _, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.Name)
		if !found {
			nt = app.UnknownNotification
		}
		m[nt] = []int{int(r.TotalCount), int(r.UnreadCount.Float64)}
	}
	return m, nil
}

type CreateCharacterNotificationParams struct {
	Body           optional.Optional[string]
	CharacterID    int32
	IsProcessed    bool
	IsRead         bool
	NotificationID int64
	RecipientID    optional.Optional[int32]
	SenderID       int32
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string]
	Type           string
}

func (arg CreateCharacterNotificationParams) isValid() bool {
	return arg.CharacterID != 0 && arg.NotificationID != 0 && arg.SenderID != 0
}

func (st *Storage) CreateCharacterNotification(ctx context.Context, arg CreateCharacterNotificationParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterNotification: %+v: %w", arg, err)
	}
	if !arg.isValid() {
		return wrapErr(app.ErrInvalid)
	}
	typeID, err := st.GetOrCreateNotificationType(ctx, arg.Type)
	if err != nil {
		return err
	}
	if err := st.qRW.CreateCharacterNotification(ctx, queries.CreateCharacterNotificationParams{
		Body:           optional.ToNullString(arg.Body),
		CharacterID:    int64(arg.CharacterID),
		IsRead:         arg.IsRead,
		IsProcessed:    arg.IsProcessed,
		NotificationID: arg.NotificationID,
		RecipientID:    optional.ToNullInt64(arg.RecipientID),
		SenderID:       int64(arg.SenderID),
		Text:           arg.Text,
		Timestamp:      arg.Timestamp,
		Title:          optional.ToNullString(arg.Title),
		TypeID:         typeID,
	}); err != nil {
		arg.Body.Clear()
		arg.Text = ""
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCharacterNotification(ctx context.Context, characterID int32, notificationID int64) (*app.CharacterNotification, error) {
	arg := queries.GetCharacterNotificationParams{
		CharacterID:    int64(characterID),
		NotificationID: notificationID,
	}
	r, err := st.qRO.GetCharacterNotification(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get character notification %+v: %w", arg, convertGetError(err))
	}
	nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
	if !found {
		nt = app.UnknownNotification
	}
	o := characterNotificationFromDBModel(characterNotificationFromDBModelParams{
		cn: r.CharacterNotification,
		nt: nt,
		recipient: nullEveEntry{
			category: r.RecipientCategory,
			id:       r.CharacterNotification.RecipientID,
			name:     r.RecipientName,
		},
		sender: r.EveEntity,
	})
	return o, err
}

func (st *Storage) GetOrCreateNotificationType(ctx context.Context, name string) (int64, error) {
	id, err := func() (int64, error) {
		tx, err := st.dbRW.Begin()
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		id, err := qtx.GetNotificationTypeID(ctx, name)
		if errors.Is(err, sql.ErrNoRows) {
			id, err = qtx.CreateNotificationType(ctx, name)
		}
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return id, nil
	}()
	if err != nil {
		return 0, fmt.Errorf("get or create notification type %s: %w", name, err)
	}
	return id, nil
}

func (st *Storage) ListCharacterNotificationIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterNotificationIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list character notification ids for character %d: %w", characterID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterNotificationsForTypes(ctx context.Context, characterID int32, types set.Set[app.EveNotificationType]) ([]*app.CharacterNotification, error) {
	names := make([]string, 0)
	for t := range types.All() {
		s, ok := st.EveNotificationTypeToESIString(t)
		if !ok {
			continue
		}
		names = append(names, s)
	}
	if len(names) == 0 {
		return []*app.CharacterNotification{}, nil
	}
	arg := queries.ListCharacterNotificationsTypesParams{
		CharacterID: int64(characterID),
		Names:       names,
	}
	rows, err := st.qRO.ListCharacterNotificationsTypes(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list notification types %+v: %w", arg, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(characterNotificationFromDBModelParams{
			cn: r.CharacterNotification,
			nt: nt,
			recipient: nullEveEntry{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
			sender: r.EveEntity,
		})

	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsAll(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	rows, err := st.qRO.ListCharacterNotificationsAll(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list all notifications for character %d: %w", characterID, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(characterNotificationFromDBModelParams{
			cn: r.CharacterNotification,
			nt: nt,
			recipient: nullEveEntry{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
			sender: r.EveEntity,
		})
	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	rows, err := st.qRO.ListCharacterNotificationsUnread(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list unread notification for character %d: %w", characterID, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(characterNotificationFromDBModelParams{
			cn: r.CharacterNotification,
			nt: nt,
			recipient: nullEveEntry{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
			sender: r.EveEntity,
		})
	}
	return ee, nil
}

func (st *Storage) ListCharacterNotificationsUnprocessed(ctx context.Context, characterID int32, earliest time.Time) ([]*app.CharacterNotification, error) {
	arg := queries.ListCharacterNotificationsUnprocessedParams{
		CharacterID: int64(characterID),
		Timestamp:   earliest,
	}
	rows, err := st.qRO.ListCharacterNotificationsUnprocessed(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list unprocessed notifications %+v: %w", arg, err)
	}
	ee := make([]*app.CharacterNotification, len(rows))
	for i, r := range rows {
		nt, found := EveNotificationTypeFromESIString(r.NotificationType.Name)
		if !found {
			nt = app.UnknownNotification
		}
		ee[i] = characterNotificationFromDBModel(characterNotificationFromDBModelParams{
			cn: r.CharacterNotification,
			nt: nt,
			recipient: nullEveEntry{
				category: r.RecipientCategory,
				id:       r.CharacterNotification.RecipientID,
				name:     r.RecipientName,
			},
			sender: r.EveEntity,
		})
	}
	return ee, nil
}

type characterNotificationFromDBModelParams struct {
	cn        queries.CharacterNotification
	sender    queries.EveEntity
	nt        app.EveNotificationType
	recipient nullEveEntry
}

func characterNotificationFromDBModel(arg characterNotificationFromDBModelParams) *app.CharacterNotification {
	o2 := &app.CharacterNotification{
		ID:             arg.cn.ID,
		Body:           optional.FromNullString(arg.cn.Body),
		CharacterID:    int32(arg.cn.CharacterID),
		IsProcessed:    arg.cn.IsProcessed,
		IsRead:         arg.cn.IsRead,
		NotificationID: arg.cn.NotificationID,
		Recipient:      eveEntityFromNullableDBModel(arg.recipient),
		Sender:         eveEntityFromDBModel(arg.sender),
		Text:           arg.cn.Text,
		Timestamp:      arg.cn.Timestamp,
		Title:          optional.FromNullString(arg.cn.Title),
		Type:           arg.nt,
	}
	return o2
}

type UpdateCharacterNotificationParams struct {
	ID     int64
	Body   optional.Optional[string]
	IsRead bool
	Title  optional.Optional[string]
}

func (st *Storage) UpdateCharacterNotification(ctx context.Context, arg UpdateCharacterNotificationParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("UpdateCharacterNotification: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.UpdateCharacterNotificationParams{
		ID:     arg.ID,
		Body:   optional.ToNullString(arg.Body),
		IsRead: arg.IsRead,
		Title:  optional.ToNullString(arg.Title),
	}
	if err := st.qRW.UpdateCharacterNotification(ctx, arg2); err != nil {
		return fmt.Errorf("update character notification %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) UpdateCharacterNotificationSetProcessed(ctx context.Context, id int64) error {
	if err := st.qRW.UpdateCharacterNotificationSetProcessed(ctx, id); err != nil {
		return fmt.Errorf("update notification set processed for id %d: %w", id, err)
	}
	return nil
}
