package ui

import (
	"context"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// update general sections

func (u *baseUI) startUpdateTickerGeneralSections() {
	go func() {
		for range time.Tick(generalSectionsUpdateTicker) {
			ctx := context.Background()
			u.updateGeneralSectionsIfNeeded(ctx, false)
		}
	}()
}

func (u *baseUI) updateGeneralSectionsIfNeeded(ctx context.Context, forceUpdate bool) {
	if !forceUpdate && !u.isDesktop && !u.isForeground.Load() {
		slog.Debug("Skipping general sections update while in background")
		return
	}
	if !forceUpdate && u.ess.IsDailyDowntime() {
		slog.Info("Skipping regular update of general sections during daily downtime")
		return
	}
	for _, s := range app.GeneralSections {
		u.updateGeneralSectionAndRefreshIfNeeded(ctx, s, forceUpdate)
	}
}

func (u *baseUI) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	logErr := func(err error) {
		slog.Error("Failed to update general section", "section", section, "err", err)
	}
	if u.onSectionUpdateStarted != nil && u.onSectionUpdateCompleted != nil {
		u.onSectionUpdateStarted()
		defer u.onSectionUpdateCompleted()
	}
	changed, err := u.eus.UpdateSection(ctx, app.GeneralSectionUpdateParams{
		Section:           section,
		ForceUpdate:       forceUpdate,
		OnUpdateStarted:   u.onSectionUpdateStarted,
		OnUpdateCompleted: u.onSectionUpdateCompleted,
	})
	if err != nil {
		logErr(err)
		return
	}
	if changed.Size() == 0 && !forceUpdate {
		return
	}
	switch section {
	case app.SectionEveMarketPrices:
		types, err := u.eus.ListEveTypeIDs(ctx)
		if err != nil {
			logErr(err)
			return
		}
		if !changed.ContainsAny(types.All()) {
			return
		}
	}
	u.generalSectionChanged.Emit(ctx, generalSectionUpdated{
		section:      section,
		forcedUpdate: forceUpdate,
		changed:      changed,
	})
}

// update character sections

func (u *baseUI) startUpdateTickerCharacters() {
	go func() {
		for range time.Tick(characterSectionsUpdateTicker) {
			ctx := context.Background()
			if err := u.updateCharactersIfNeeded(ctx, false); err != nil {
				slog.Error("Failed to update characters", "error", err)
			}
			if err := u.notifyCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to notify characters", "error", err)
			}
		}
	}()
}

func (u *baseUI) updateCharactersIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && u.ess.IsDailyDowntime() {
		slog.Info("Skipping regular update of characters during daily downtime")
		return nil
	}
	cc, err := u.cs.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.updateCharacterAndRefreshIfNeeded(ctx, c.ID, forceUpdate)
	}
	slog.Debug("started update status characters")
	return nil
}

func (u *baseUI) notifyCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.cs.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.notifyExpiredTrainingIfNeeded(ctx, c.ID)
	}
	slog.Debug("started notify characters")
	return nil
}

func (u *baseUI) notifyExpiredTrainingIfNeeded(ctx context.Context, characterID int32) {
	if u.settings.NotifyTrainingEnabled() {
		go func() {
			// TODO: earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
			err := u.cs.NotifyExpiredTraining(ctx, characterID, u.sendDesktopNotification)
			if err != nil {
				slog.Error("notify expired training", "error", err)
			}
		}()
	}
}

func (u *baseUI) notifyNewCommunications(ctx context.Context, characterID int32) {
	earliest := u.settings.NotifyCommunicationsEarliest()
	xx := u.settings.NotificationTypesEnabled()
	var typesEnabled set.Set[app.EveNotificationType]
	for s := range xx.All() {
		nt, found := app.EveNotificationTypeFromString(s)
		if !found {
			continue
		}
		typesEnabled.Add(nt)
	}
	err := u.cs.NotifyCommunications(
		ctx,
		characterID,
		earliest,
		typesEnabled,
		u.sendDesktopNotification,
	)
	if err != nil {
		slog.Error("notify communications", "characterID", characterID, "error", err)
	}
}

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *baseUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	slog.Info("updateCharacterAndRefreshIfNeeded: started", "characterID", characterID, "forceUpdate", forceUpdate)
	if u.isOffline {
		return
	}
	var sections set.Set[app.CharacterSection] // sections registered for update
	if !u.isDesktop && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.settings.NotifyCommunicationsEnabled() {
			sections.Add(app.SectionCharacterNotifications)
		}
		if u.settings.NotifyContractsEnabled() {
			sections.Add(app.SectionCharacterContracts)
		}
		if u.settings.NotifyMailsEnabled() {
			sections.Add(app.SectionCharacterMailLabels)
			sections.Add(app.SectionCharacterMailLists)
			sections.Add(app.SectionCharacterMailHeaders)
		}
		if u.settings.NotifyPIEnabled() {
			sections.Add(app.SectionCharacterPlanets)
		}
		if u.settings.NotifyTrainingEnabled() {
			sections.Add(app.SectionCharacterSkillqueue)
			sections.Add(app.SectionCharacterSkills)
		}
	} else {
		sections = set.Of(app.CharacterSections...)
	}
	if sections.Size() == 0 {
		return
	}

	slog.Debug("Starting to check character sections for update", "sections", sections)
	_, err := u.cs.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh token for update", "characterID", characterID, "error", err)
	}

	// updateGroup starts a sequential update of group and removes them from sections.
	// It skips all updates for group if one of the group's sections has not been registered for update.
	updateGroup := func(group []app.CharacterSection) {
		mySections := set.Intersection(sections, set.Of(group...))
		if mySections.Size() > 0 {
			go func() {
				for _, s := range group {
					if !mySections.Contains(s) {
						continue
					}
					u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
				}
			}()
		}
		sections.Delete(group...)
	}

	// Some sections are fetched sequentially.
	// This is done in part for to prioritize some sections that would
	// and in part to ensure sections that others logically depend on are fetched first.

	updateGroup([]app.CharacterSection{
		app.SectionCharacterMailLabels,
		app.SectionCharacterMailLists,
		app.SectionCharacterMailHeaders,
		app.SectionCharacterMailBodies,
	})

	updateGroup([]app.CharacterSection{
		app.SectionCharacterSkills,
		app.SectionCharacterSkillqueue,
	})

	// Other sections
	for s := range sections.All() {
		go u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// updateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, section app.CharacterSection, forceUpdate bool) {
	logErr := func(err error) {
		slog.Error("Failed to process update for character section",
			"characterID", characterID,
			"section", section,
			"forcedUpdate", forceUpdate,
			"error", err,
		)
	}
	hasChanged, err := u.cs.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
		CharacterID:           characterID,
		ForceUpdate:           forceUpdate,
		MarketOrderRetention:  time.Duration(u.settings.MarketOrderRetentionDays()) * time.Hour * 24,
		MaxMails:              u.settings.MaxMails(),
		MaxWalletTransactions: u.settings.MaxWalletTransactions(),
		OnUpdateCompleted:     u.onSectionUpdateCompleted,
		OnUpdateStarted:       u.onSectionUpdateStarted,
		Section:               section,
	})
	if err != nil {
		logErr(err)
		return
	}

	needsRefresh := hasChanged || forceUpdate
	if needsRefresh {
		u.characterSectionChanged.Emit(ctx, characterSectionUpdated{
			characterID:  characterID,
			forcedUpdate: forceUpdate,
			section:      section,
		})
	}
	switch section {
	case app.SectionCharacterMailHeaders:
		if u.settings.NotifyMailsEnabled() {
			earliest := u.settings.NotifyMailsEarliest()
			if err := u.cs.NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
				logErr(err)
			}
		}
	case app.SectionCharacterContracts:
		if u.settings.NotifyContractsEnabled() {
			earliest := u.settings.NotifyContractsEarliest()
			if err := u.cs.NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
				logErr(err)
			}
		}
	case app.SectionCharacterNotifications:
		if u.settings.NotifyCommunicationsEnabled() {
			u.notifyNewCommunications(ctx, characterID)
		}
	case app.SectionCharacterPlanets:
		if u.settings.NotifyPIEnabled() {
			earliest := u.settings.NotifyPIEarliest()
			if err := u.cs.NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
				logErr(err)
			}
		}
	case app.SectionCharacterSkillqueue:
		if u.settings.NotifyTrainingEnabled() {
			if needsRefresh {
				u.notifyExpiredTrainingIfNeeded(ctx, characterID)
			}
			err := u.cs.EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				logErr(err)
			}
		}
	}
}

// update corporation sections

func (u *baseUI) startUpdateTickerCorporations() {
	go func() {
		for range time.Tick(characterSectionsUpdateTicker) {
			if err := u.updateCorporationsIfNeeded(context.Background(), false); err != nil {
				slog.Error("Failed to update corporations", "error", err)
			}
		}
	}()
}

func (u *baseUI) updateCorporationsIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && u.ess.IsDailyDowntime() {
		slog.Info("Skipping regular update of corporations during daily downtime")
		return nil
	}
	changed, err := u.rs.UpdateCorporations(ctx)
	if err != nil {
		return err
	}
	if changed {
		u.updateStatus()
	}
	all, err := u.rs.ListCorporationIDs(ctx)
	if err != nil {
		return err
	}
	for id := range all.All() {
		go u.updateCorporationAndRefreshIfNeeded(ctx, id, forceUpdate)
	}
	slog.Debug("started update status corporations")
	return nil
}

// updateCorporationAndRefreshIfNeeded runs update for all sections of a corporation if needed
// and refreshes the UI accordingly.
func (u *baseUI) updateCorporationAndRefreshIfNeeded(ctx context.Context, corporationID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	if !u.isDesktop && !u.isForeground.Load() && !forceUpdate {
		// nothing to update
		return
	}
	sections := set.Of(app.CorporationSections...)
	slog.Debug("Starting to check corporation sections for update", "corporationID", corporationID, "sections", sections)

	for s := range sections.All() {
		go u.updateCorporationSectionAndRefreshIfNeeded(ctx, corporationID, s, forceUpdate)
	}
}

// updateCorporationSectionAndRefreshIfNeeded runs update for a corporation section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on corporation sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) updateCorporationSectionAndRefreshIfNeeded(ctx context.Context, corporationID int32, section app.CorporationSection, forceUpdate bool) {
	hasChanged, err := u.rs.UpdateSectionIfNeeded(
		ctx, app.CorporationSectionUpdateParams{
			CorporationID:         corporationID,
			ForceUpdate:           forceUpdate,
			MaxWalletTransactions: u.settings.MaxWalletTransactions(),
			Section:               section,
			OnUpdateStarted:       u.onSectionUpdateStarted,
			OnUpdateCompleted:     u.onSectionUpdateCompleted,
		},
	)
	if err != nil {
		slog.Error("Failed to update corporation section", "corporationID", corporationID, "section", section, "err", err)
		return
	}
	if !hasChanged && !forceUpdate {
		return
	}
	u.corporationSectionChanged.Emit(ctx, corporationSectionUpdated{
		corporationID: corporationID,
		forcedUpdate:  forceUpdate,
		section:       section,
	})
}
