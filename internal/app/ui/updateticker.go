package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

// update general sections

func (u *baseUI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			ctx := context.Background()
			u.updateGeneralSectionsIfNeeded(ctx, false)
			<-ticker.C
		}
	}()
}

func (u *baseUI) updateGeneralSectionsIfNeeded(ctx context.Context, forceUpdate bool) {
	if !forceUpdate && !u.isDesktop && !u.isForeground.Load() {
		slog.Debug("Skipping general sections update while in background")
		return
	}
	if !forceUpdate && isNowDailyDowntime() {
		slog.Info("Skipping regular update of general sections during daily downtime")
		return
	}
	for _, s := range app.GeneralSections {
		go func() {
			u.updateGeneralSectionAndRefreshIfNeeded(ctx, s, forceUpdate)
		}()
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
	changed, err := u.eus.UpdateSection(ctx, app.GeneralUpdateSectionParams{
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
	case app.SectionEveEntities:
		u.updateHome()
		u.updateCharacter()
	case app.SectionEveTypes:
		u.characterShips.update()
		u.characterSkillCatalogue.update()
	case app.SectionEveCharacters:
		if changed.Contains(u.currentCharacterID()) {
			u.reloadCurrentCharacter()
		}
		characters, err := u.cs.ListCharacterIDs(ctx)
		if err != nil {
			logErr(err)
			return
		}
		if characters.ContainsAny(changed.All()) {
			u.characterOverview.update()
			u.updateStatus()
		}
	case app.SectionEveCorporations:
		if changed.Contains(u.currentCorporationID()) {
			u.corporationSheet.update()
		}
		c := u.currentCharacter()
		if c == nil {
			break
		}
		if changed.Contains(c.EveCharacter.Corporation.ID) {
			u.characterCorporation.update()
		}
		cc, err := u.cs.ListCharacterCorporations(ctx)
		if err != nil {
			logErr(err)
			return
		}
		if changed.ContainsAny(xiter.MapSlice(cc, func(x *app.EntityShort[int32]) int32 {
			return x.ID
		})) {
			u.updateStatus()
		}
	case app.SectionEveMarketPrices:
		types, err := u.eus.ListEveTypeIDs(ctx)
		if err != nil {
			logErr(err)
			return
		}
		if !changed.ContainsAny(types.All()) {
			break
		}
		u.characterAsset.update()
		u.characterOverview.update()
		u.assets.update()
		u.reloadCurrentCharacter()
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

// update character sections

func (u *baseUI) startUpdateTickerCharacters() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	go func() {
		for {
			ctx := context.Background()
			if err := u.updateCharactersIfNeeded(ctx, false); err != nil {
				slog.Error("Failed to update characters", "error", err)
			}
			if err := u.notifyCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to notify characters", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *baseUI) updateCharactersIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && isNowDailyDowntime() {
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
		go u.notifyExpiredExtractionsIfNeeded(ctx, c.ID)
		go u.notifyExpiredTrainingIfNeeded(ctx, c.ID)
	}
	slog.Debug("started notify characters")
	return nil
}

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *baseUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections []app.CharacterSection
	if !u.isDesktop && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.settings.NotifyCommunicationsEnabled() {
			sections = append(sections, app.SectionCharacterNotifications)
		}
		if u.settings.NotifyContractsEnabled() {
			sections = append(sections, app.SectionCharacterContracts)
		}
		if u.settings.NotifyMailsEnabled() {
			sections = append(sections, app.SectionCharacterMailLabels)
			sections = append(sections, app.SectionCharacterMailLists)
			sections = append(sections, app.SectionCharacterMails)
		}
		if u.settings.NotifyPIEnabled() {
			sections = append(sections, app.SectionCharacterPlanets)
		}
		if u.settings.NotifyTrainingEnabled() {
			sections = append(sections, app.SectionCharacterSkillqueue)
			sections = append(sections, app.SectionCharacterSkills)
		}
	} else {
		sections = app.CharacterSections
	}
	if len(sections) == 0 {
		return
	}
	slog.Debug("Starting to check character sections for update", "sections", sections)
	_, err := u.cs.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		slog.Error("Failed to refresh token for update", "characterID", characterID, "error", err)
	}
	for _, s := range sections {
		go u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// updateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	updateArg := app.CharacterUpdateSectionParams{
		CharacterID:           characterID,
		Section:               s,
		ForceUpdate:           forceUpdate,
		MaxMails:              u.settings.MaxMails(),
		MaxWalletTransactions: u.settings.MaxWalletTransactions(),
		OnUpdateStarted:       u.onSectionUpdateStarted,
		OnUpdateCompleted:     u.onSectionUpdateCompleted,
	}
	hasChanged, err := u.cs.UpdateSectionIfNeeded(ctx, updateArg)
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isShown := characterID == u.currentCharacterID()

	var corporationID int32
	character := u.currentCharacter()
	if character != nil {
		corporationID = character.EveCharacter.Corporation.ID
		ok, err := u.rs.HasCorporation(ctx, corporationID)
		if err != nil {
			slog.Error("Failed to check corporation exists", "error", err)
		}
		if !ok {
			corporationID = 0
		}
	}

	needsRefresh := hasChanged || forceUpdate
	if needsRefresh {
		u.characterSectionUpdated.Emit(ctx, updateArg)
	}

	switch s {
	case app.SectionCharacterAssets:
		if needsRefresh {
			u.assets.update()
			u.wealth.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterAsset.update()
				u.characterSheet.update()
			}
		}
	case app.SectionCharacterAttributes:
		if isShown && needsRefresh {
			u.characterAttributes.update()
		}
	case app.SectionCharacterContracts:
		if needsRefresh {
			u.contracts.update()
		}
		if u.settings.NotifyContractsEnabled() {
			go func() {
				earliest := u.settings.NotifyContractsEarliest()
				if err := u.cs.NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify contract update", "error", err)
				}
			}()
		}
	case app.SectionCharacterImplants:
		if needsRefresh {
			u.augmentations.update()
			if isShown {
				u.characterAugmentations.update()
			}
		}
	case app.SectionCharacterJumpClones:
		if needsRefresh {
			u.characterOverview.update()
			u.clones.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterJumpClones.update()
			}
		}
	case app.SectionCharacterIndustryJobs:
		if needsRefresh {
			u.industryJobs.update()
			u.slotsManufacturing.update()
			u.slotsReactions.update()
			u.slotsResearch.update()
		}
	case app.SectionCharacterLocation, app.SectionCharacterOnline, app.SectionCharacterShip:
		if needsRefresh {
			u.characterLocations.update()
			if isShown {
				u.reloadCurrentCharacter()
			}
		}
	case app.SectionCharacterMailLabels, app.SectionCharacterMailLists:
		if needsRefresh {
			u.characterOverview.update()
			if isShown {
				u.characterMails.update()
			}
		}
	case app.SectionCharacterMails:
		if needsRefresh {
			go u.characterOverview.update()
			go u.updateMailIndicator()
			if isShown {
				u.characterMails.update()
			}
		}
		if u.settings.NotifyMailsEnabled() {
			go func() {
				earliest := u.settings.NotifyMailsEarliest()
				if err := u.cs.NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify mails", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionCharacterNotifications:
		if isShown && needsRefresh {
			u.characterCommunications.update()
		}
		if u.settings.NotifyCommunicationsEnabled() {
			go func() {
				earliest := u.settings.NotifyCommunicationsEarliest()
				typesEnabled := u.settings.NotificationTypesEnabled()
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
			}()
		}
	case app.SectionCharacterPlanets:
		if needsRefresh {
			u.colonies.update()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionCharacterRoles:
		if needsRefresh {
			u.updateStatus()
			if isShown {
				u.characterSheet.update()
			}
			if corporationID == 0 {
				return
			}
			if err := u.rs.RemoveSectionDataWhenPermissionLost(ctx, corporationID); err != nil {
				slog.Error("Failed to remove corp data after character role change", "characterID", characterID, "error", err)
			}
			u.updateCorporationAndRefreshIfNeeded(ctx, corporationID, true)
		}
	case app.SectionCharacterSkills:
		if needsRefresh {
			u.training.update()
			u.slotsManufacturing.update()
			u.slotsReactions.update()
			u.slotsResearch.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterSkillCatalogue.update()
				u.characterShips.update()
			}
		}

	case app.SectionCharacterSkillqueue:
		if needsRefresh {
			u.training.update()
			u.notifyExpiredTrainingIfNeeded(ctx, characterID)
			// if isShown {
			// 	u.characterSkillQueue.update()
			// }
		}
		if u.settings.NotifyTrainingEnabled() {
			err := u.cs.EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
	case app.SectionCharacterWalletBalance:
		if needsRefresh {
			u.characterOverview.update()
			u.wealth.update()
			if isShown {
				u.reloadCurrentCharacter()
				u.characterWallet.update()
			}
		}
	case app.SectionCharacterWalletJournal:
		if isShown && needsRefresh {
			u.characterWallet.journal.update()
		}
	case app.SectionCharacterWalletTransactions:
		if isShown && needsRefresh {
			u.characterWallet.transactions.update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the refresh ticker: %s", s))
	}
}

// update corporation sections

func (u *baseUI) startUpdateTickerCorporations() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	ctx := context.Background()
	go func() {
		for {
			if err := u.updateCorporationsIfNeeded(ctx, false); err != nil {
				slog.Error("Failed to update corporations", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *baseUI) updateCorporationsIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && isNowDailyDowntime() {
		slog.Info("Skipping regular update of corporations during daily downtime")
		return nil
	}
	removed, err := u.rs.RemoveStaleCorporations(ctx)
	if err != nil {
		return err
	}
	if removed {
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
	sections := app.CorporationSections
	slog.Debug("Starting to check corporation sections for update", "sections", sections)
	for _, s := range sections {
		go u.updateCorporationSectionAndRefreshIfNeeded(ctx, corporationID, s, forceUpdate)
	}
}

// updateCorporationSectionAndRefreshIfNeeded runs update for a corporation section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on corporation sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) updateCorporationSectionAndRefreshIfNeeded(ctx context.Context, corporationID int32, s app.CorporationSection, forceUpdate bool) {
	hasChanged, err := u.rs.UpdateSectionIfNeeded(
		ctx, app.CorporationUpdateSectionParams{
			CorporationID:         corporationID,
			ForceUpdate:           forceUpdate,
			MaxWalletTransactions: u.settings.MaxWalletTransactions(),
			Section:               s,
			OnUpdateStarted:       u.onSectionUpdateStarted,
			OnUpdateCompleted:     u.onSectionUpdateCompleted,
		},
	)
	if err != nil {
		slog.Error("Failed to update corporation section", "corporationID", corporationID, "section", s, "err", err)
		return
	}
	if !hasChanged && !forceUpdate {
		return
	}
	isShown := corporationID == u.currentCorporationID()
	switch s {
	case app.SectionCorporationIndustryJobs:
		u.industryJobs.update()
		u.slotsManufacturing.update()
		u.slotsReactions.update()
		u.slotsResearch.update()
	case app.SectionCorporationMembers:
		if isShown {
			u.corporationMember.update()
		}
	case app.SectionCorporationDivisions:
		if isShown {
			for _, d := range app.Divisions {
				u.corporationWallets[d].updateName()
			}
		}
	case app.SectionCorporationWalletBalances:
		if isShown {
			for _, d := range app.Divisions {
				u.corporationWallets[d].updateBalance()
			}
			u.updateCorporationWalletTotal()
		}
	case app.SectionCorporationWalletJournal1:
		if isShown {
			u.corporationWallets[app.Division1].journal.update()
		}
	case app.SectionCorporationWalletJournal2:
		if isShown {
			u.corporationWallets[app.Division2].journal.update()
		}
	case app.SectionCorporationWalletJournal3:
		if isShown {
			u.corporationWallets[app.Division3].journal.update()
		}
	case app.SectionCorporationWalletJournal4:
		if isShown {
			u.corporationWallets[app.Division4].journal.update()
		}
	case app.SectionCorporationWalletJournal5:
		if isShown {
			u.corporationWallets[app.Division5].journal.update()
		}
	case app.SectionCorporationWalletJournal6:
		if isShown {
			u.corporationWallets[app.Division6].journal.update()
		}
	case app.SectionCorporationWalletJournal7:
		if isShown {
			u.corporationWallets[app.Division7].journal.update()
		}
	case app.SectionCorporationWalletTransactions1:
		if isShown {
			u.corporationWallets[app.Division1].transactions.update()
		}
	case app.SectionCorporationWalletTransactions2:
		if isShown {
			u.corporationWallets[app.Division2].transactions.update()
		}
	case app.SectionCorporationWalletTransactions3:
		if isShown {
			u.corporationWallets[app.Division3].transactions.update()
		}
	case app.SectionCorporationWalletTransactions4:
		if isShown {
			u.corporationWallets[app.Division4].transactions.update()
		}
	case app.SectionCorporationWalletTransactions5:
		if isShown {
			u.corporationWallets[app.Division5].transactions.update()
		}
	case app.SectionCorporationWalletTransactions6:
		if isShown {
			u.corporationWallets[app.Division6].transactions.update()
		}
	case app.SectionCorporationWalletTransactions7:
		if isShown {
			u.corporationWallets[app.Division7].transactions.update()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the refresh ticker: %s", s))
	}
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

func (u *baseUI) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
	if u.settings.NotifyPIEnabled() {
		go func() {
			earliest := u.settings.NotifyPIEarliest()
			err := u.cs.NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification)
			if err != nil {
				slog.Error("notify expired extractions", "characterID", characterID, "error", err)
			}
		}()
	}
}

// isNowDailyDowntime reports whether the daily downtime is expected to happen currently.
func isNowDailyDowntime() bool {
	return isTimeWithinRange(downtimeStart, downtimeDuration, time.Now())
}

func isTimeWithinRange(start string, duration time.Duration, t time.Time) bool {
	t2, err := time.Parse("15:04", t.UTC().Format("15:04"))
	if err != nil {
		panic(err)
	}
	start2, err := time.Parse("15:04", start)
	if err != nil {
		panic(err)
	}
	end := start2.Add(duration)
	if t2.Before(start2) {
		return false
	}
	if t2.After(end) {
		return false
	}
	return true
}
