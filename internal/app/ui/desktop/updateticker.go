package desktop

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
	characterSectionsUpdateTicker = 10 * time.Second
	generalSectionsUpdateTicker   = 60 * time.Second
	notifyEarliestFallback        = 24 * time.Hour
)

func (u *DesktopUI) sendDesktopNotification(title, content string) {
	u.fyneApp.SendNotification(fyne.NewNotification(title, content))
	slog.Info("desktop notification sent", "title", title, "content", content)
}

func (u *DesktopUI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.updateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *DesktopUI) updateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
	for _, s := range app.GeneralSections {
		go func(s app.GeneralSection) {
			u.updateGeneralSectionAndRefreshIfNeeded(context.TODO(), s, forceUpdate)
		}(s)
	}
}

func (u *DesktopUI) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.EveUniverseService.UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	switch section {
	case app.SectionEveCategories:
		if hasChanged {
			u.shipsArea.refresh()
			u.skillCatalogueArea.refresh()
		}
	case app.SectionEveCharacters, app.SectionEveMarketPrices:
		// nothing to refresh
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker refresh: %s", section))
	}
}

func (u *DesktopUI) startUpdateTickerCharacters() {
	ticker := time.NewTicker(characterSectionsUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := u.CharacterService.ListCharactersShort(context.TODO())
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					go u.updateCharacterAndRefreshIfNeeded(context.TODO(), c.ID, false)
					if u.fyneApp.Preferences().BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault) {
						go func() {
							earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyPIEarliest)
							if err := u.CharacterService.NotifyExpiredExtractions(context.TODO(), c.ID, earliest, u.sendDesktopNotification); err != nil {
								slog.Error("notify expired extractions", "characterID", c.ID, "error", err)
							}
						}()
					}
					if u.fyneApp.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
						go func() {
							// earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyTrainingEarliest)
							if err := u.CharacterService.NotifyExpiredTraining(context.TODO(), c.ID, u.sendDesktopNotification); err != nil {
								slog.Error("notify expired training", "error", err)
							}
						}()
					}
				}
			}()
			<-ticker.C
		}
	}()
}

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *DesktopUI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	if u.IsOffline {
		return
	}
	for _, s := range app.CharacterSections {
		s := s
		go u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
	}
}

// updateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *DesktopUI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.CharacterService.UpdateSectionIfNeeded(
		ctx, character.UpdateSectionParams{
			CharacterID:           characterID,
			Section:               s,
			ForceUpdate:           forceUpdate,
			MaxMails:              u.fyneApp.Preferences().IntWithFallback(settingMaxMails, settingMaxMailsDefault),
			MaxWalletTransactions: u.fyneApp.Preferences().IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault),
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isShown := characterID == u.characterID()
	needsRefresh := hasChanged || forceUpdate
	switch s {
	case app.SectionAssets:
		if isShown && needsRefresh {
			u.assetsArea.redraw()
		}
		if needsRefresh {
			u.assetSearchArea.refresh()
			u.wealthArea.refresh()
		}
	case app.SectionAttributes:
		if isShown && needsRefresh {
			u.attributesArea.refresh()
		}
	case app.SectionContracts:
		if isShown && needsRefresh {
			u.contractsArea.refresh()
		}
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyContractsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyContractsEarliest)
				if err := u.CharacterService.NotifyUpdatedContracts(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify contract update", "error", err)
				}
			}()
		}
	case app.SectionImplants:
		if isShown && needsRefresh {
			u.implantsArea.refresh()
		}
	case app.SectionJumpClones:
		if isShown && needsRefresh {
			u.jumpClonesArea.redraw()
		}
		if needsRefresh {
			u.overviewArea.refresh()
		}
	case app.SectionLocation,
		app.SectionOnline,
		app.SectionShip:
		if needsRefresh {
			u.locationsArea.refresh()
		}
	case app.SectionPlanets:
		if isShown && needsRefresh {
			u.planetArea.refresh()
		}
		if needsRefresh {
			u.coloniesArea.refresh()
		}
	case app.SectionMailLabels,
		app.SectionMailLists:
		if isShown && needsRefresh {
			u.mailArea.refresh()
		}
		if needsRefresh {
			u.overviewArea.refresh()
		}
	case app.SectionMails:
		if isShown && needsRefresh {
			u.mailArea.refresh()
		}
		if needsRefresh {
			u.overviewArea.refresh()
		}
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyMailsEarliest)
				if err := u.CharacterService.NotifyMails(ctx, characterID, earliest, u.sendDesktopNotification); err != nil {
					slog.Error("notify mails", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionNotifications:
		if isShown && needsRefresh {
			u.notificationsArea.refresh()
		}
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go func() {
				earliest := calcNotifyEarliest(u.fyneApp.Preferences(), settingNotifyCommunicationsEarliest)
				typesEnabled := set.NewFromSlice(u.fyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
				if err := u.CharacterService.NotifyCommunications(ctx, characterID, earliest, typesEnabled, u.sendDesktopNotification); err != nil {
					slog.Error("notify communications", "characterID", characterID, "error", err)
				}
			}()
		}
	case app.SectionSkills:
		if isShown && needsRefresh {
			u.skillCatalogueArea.refresh()
			u.shipsArea.refresh()
			u.planetArea.refresh()
		}
		if needsRefresh {
			u.trainingArea.refresh()
		}
	case app.SectionSkillqueue:
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
			err := u.CharacterService.EnableTrainingWatcher(ctx, characterID)
			if err != nil {
				slog.Error("Failed to enable training watcher", "characterID", characterID, "error", err)
			}
		}
		if isShown {
			u.skillqueueArea.refresh()
		}
		if needsRefresh {
			u.trainingArea.refresh()
		}
	case app.SectionWalletBalance:
		if needsRefresh {
			u.overviewArea.refresh()
			u.wealthArea.refresh()
		}
	case app.SectionWalletJournal:
		if isShown && needsRefresh {
			u.walletJournalArea.refresh()
		}
	case app.SectionWalletTransactions:
		if isShown && needsRefresh {
			u.walletTransactionArea.refresh()
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
