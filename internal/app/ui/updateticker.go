package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

const (
	characterSectionsUpdateTicker = 60 * time.Second
	generalSectionsUpdateTicker   = 300 * time.Second
	notifyEarliestFallback        = 24 * time.Hour
)

func (u *BaseUI) sendDesktopNotification(title, content string) {
	u.FyneApp.SendNotification(fyne.NewNotification(title, content))
	slog.Info("desktop notification sent", "title", title, "content", content)
}

func (u *BaseUI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.UpdateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *BaseUI) UpdateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
	if !forceUpdate && u.IsMobile() && !u.isForeground.Load() {
		slog.Debug("Skipping general sections update while in background")
		return
	}
	ctx := context.Background()
	for _, s := range app.GeneralSections {
		go func(s app.GeneralSection) {
			u.UpdateGeneralSectionAndRefreshIfNeeded(ctx, s, forceUpdate)
		}(s)
	}
}

func (u *BaseUI) UpdateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.EveUniverseService.UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	switch section {
	case app.SectionEveCategories:
		if hasChanged {
			u.ShipsArea.Refresh()
			u.SkillCatalogueArea.Refresh()
		}
	case app.SectionEveCharacters, app.SectionEveMarketPrices:
		// nothing to refresh
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

func (u *BaseUI) startUpdateTickerCharacters() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	ctx := context.Background()
	go func() {
		for {
			if err := u.updateCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to update characters", "error", err)
			}
			if err := u.notifyCharactersIfNeeded(ctx); err != nil {
				slog.Error("Failed to notify characters", "error", err)
			}
			<-ticker.C
		}
	}()
}

func (u *BaseUI) updateCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.CharacterService.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		go u.UpdateCharacterAndRefreshIfNeeded(ctx, c.ID, false)
	}
	return nil
}

func (u *BaseUI) notifyCharactersIfNeeded(ctx context.Context) error {
	cc, err := u.CharacterService.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	for _, c := range cc {
		u.notifyExpiredExtractionsIfNeeded(ctx, c.ID)
		u.notifyExpiredTrainingIfneeded(ctx, c.ID)
	}
	return nil
}

// UpdateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *BaseUI) UpdateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.IsOffline {
		return
	}
	var sections []app.CharacterSection
	if u.IsMobile() && !u.isForeground.Load() {
		// only update what is needed for notifications on mobile when running in background to save battery
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			sections = append(sections, app.SectionNotifications)
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyContractsEnabled, settingNotifyContractsEnabledDefault) {
			sections = append(sections, app.SectionContracts)
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault) {
			sections = append(sections, app.SectionContracts)
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault) {
			sections = append(sections, app.SectionPlanets)
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
			sections = append(sections, app.SectionSkillqueue)
		}
	} else {
		sections = app.CharacterSections
	}
	if len(sections) == 0 {
		return
	}
	slog.Debug("Starting to check character sections for update", "sections", sections)
	for _, s := range sections {
		s := s
		go u.UpdateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// UpdateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *BaseUI) UpdateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.CharacterService.UpdateSectionIfNeeded(
		ctx, character.UpdateSectionParams{
			CharacterID:           characterID,
			Section:               s,
			ForceUpdate:           forceUpdate,
			MaxMails:              u.FyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault),
			MaxWalletTransactions: u.FyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault),
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isShown := characterID == u.CharacterID()
	needsRefresh := hasChanged || forceUpdate
	switch s {
	case app.SectionAssets:
		if needsRefresh {
			v, err := u.CharacterService.UpdateCharacterAssetTotalValue(ctx, characterID)
			if err != nil {
				slog.Error("update asset total value", "characterID", characterID, "err", err)
			}
			if isShown {
				u.character.AssetValue.Set(v)
			}
			u.AssetSearchArea.Refresh()
			u.WealthArea.Refresh()
		}
		if isShown && needsRefresh {
			u.AssetsArea.Redraw()
		}
	case app.SectionAttributes:
		if isShown && needsRefresh {
			u.AttributesArea.Refresh()
		}
	case app.SectionContracts:
		if isShown && needsRefresh {
			u.ContractsArea.Refresh()
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyContractsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.FyneApp.Preferences(), settingNotifyContractsEarliest)
				if err := u.CharacterService.NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify contract update", "error", err)
				}
			}()
		}
	case app.SectionImplants:
		if isShown && needsRefresh {
			u.ImplantsArea.Refresh()
		}
	case app.SectionJumpClones:
		if isShown && needsRefresh {
			u.JumpClonesArea.Redraw()
		}
		if needsRefresh {
			u.OverviewArea.Refresh()
		}
	case app.SectionLocation,
		app.SectionOnline,
		app.SectionShip:
		if needsRefresh {
			u.LocationsArea.Refresh()
		}
	case app.SectionPlanets:
		if isShown && needsRefresh {
			u.PlanetArea.Refresh()
		}
		if needsRefresh {
			u.ColoniesArea.Refresh()
			u.notifyExpiredExtractionsIfNeeded(ctx, characterID)
		}
	case app.SectionMailLabels,
		app.SectionMailLists:
		if isShown && needsRefresh {
			u.MailArea.Refresh()
		}
		if needsRefresh {
			u.OverviewArea.Refresh()
		}
	case app.SectionMails:
		if isShown && needsRefresh {
			u.MailArea.Refresh()
		}
		if needsRefresh {
			go u.OverviewArea.Refresh()
			go u.UpdateMailIndicator()
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.FyneApp.Preferences(), settingNotifyMailsEarliest)
				if err := u.CharacterService.NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify mails", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionNotifications:
		if isShown && needsRefresh {
			u.NotificationsArea.Refresh()
		}
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.FyneApp.Preferences(), settingNotifyCommunicationsEarliest)
				typesEnabled := set.NewFromSlice(u.FyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
				if err := u.CharacterService.NotifyCommunications(ctx, characterID, earliest, typesEnabled, u.sendDesktopNotification); err != nil {
					slog.Error("notify communications", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionSkills:
		if isShown && needsRefresh {
			u.SkillCatalogueArea.Refresh()
			u.ShipsArea.Refresh()
			u.PlanetArea.Refresh()
		}
		if needsRefresh {
			u.TrainingArea.Refresh()
		}
	case app.SectionSkillqueue:
		if u.FyneApp.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
			err := u.CharacterService.EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
		if isShown {
			u.SkillqueueArea.Refresh()
		}
		if needsRefresh {
			u.TrainingArea.Refresh()
			u.notifyExpiredTrainingIfneeded(ctx, characterID)
		}
	case app.SectionWalletBalance:
		if needsRefresh {
			u.OverviewArea.Refresh()
			u.WealthArea.Refresh()
		}
	case app.SectionWalletJournal:
		if isShown && needsRefresh {
			u.WalletJournalArea.Refresh()
		}
	case app.SectionWalletTransactions:
		if isShown && needsRefresh {
			u.WalletTransactionArea.Refresh()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker: %s", s))
	}
}

// calcNotifyEarliest returns the earliest time for a class of notifications.
// Might return a zero time in some circumstances.
func calcNotifyEarliest(pref fyne.Preferences, settingEarliest string) time.Time {
	earliest, err := time.Parse(time.RFC3339, pref.String(settingEarliest))
	if err != nil {
		// Recording the earliest when enabling a switch was added later for mails and communications
		// This workaround avoids a potential notification spam from older items.
		earliest = time.Now().UTC().Add(-notifyEarliestFallback)
		pref.SetString(settingEarliest, earliest.Format(time.RFC3339))
	}
	timeoutDays := pref.IntWithFallback(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
	var timeout time.Time
	if timeoutDays > 0 {
		timeout = time.Now().UTC().Add(-time.Duration(timeoutDays) * time.Hour)
	}
	if earliest.After(timeout) {
		return earliest
	}
	return timeout
}

func (u *BaseUI) notifyExpiredTrainingIfneeded(ctx context.Context, characerID int32) {
	if u.FyneApp.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
		go func() {
			// earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
			if err := u.CharacterService.NotifyExpiredTraining(ctx, characerID, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired training", "error", err)
			}
		}()
	}
}

func (u *BaseUI) notifyExpiredExtractionsIfNeeded(ctx context.Context, characterID int32) {
	if u.FyneApp.Preferences().BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault) {
		go func() {
			earliest := calcNotifyEarliest(u.FyneApp.Preferences(), settingNotifyPIEarliest)
			if err := u.CharacterService.NotifyExpiredExtractions(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
				slog.Error("notify expired extractions", "characterID", characterID, "error", err)
			}
		}()
	}
}
