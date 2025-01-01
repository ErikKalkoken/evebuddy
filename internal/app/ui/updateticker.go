package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

const (
	characterSectionsUpdateTicker = 10 * time.Second
	generalSectionsUpdateTicker   = 60 * time.Second
)

func (u *UI) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.updateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *UI) updateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
	for _, s := range app.GeneralSections {
		go func(s app.GeneralSection) {
			u.updateGeneralSectionAndRefreshIfNeeded(context.TODO(), s, forceUpdate)
		}(s)
	}
}

func (u *UI) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
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

func (u *UI) startUpdateTickerCharacters() {
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
						go u.notifyExpiredExtractions(context.TODO(), c.ID)
					}
					if u.fyneApp.Preferences().BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault) {
						go func() {
							err := u.notifyExpiredTraining(context.TODO(), c.ID)
							if err != nil {
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
func (u *UI) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
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
func (u *UI) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
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
			go u.processMails(ctx, characterID)
		}
	case app.SectionNotifications:
		if isShown && needsRefresh {
			u.notificationsArea.refresh()
		}
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go u.processNotifications(ctx, characterID)
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

func (u *UI) processNotifications(ctx context.Context, characterID int32) {
	maxAge := u.fyneApp.Preferences().IntWithFallback(settingMaxAge, settingMaxAgeDefault)
	nn, err := u.CharacterService.ListCharacterNotificationsUnprocessed(ctx, characterID)
	if err != nil {
		slog.Error("Failed to fetch notifications for processing", "characterID", characterID, "error", err)
		return
	}
	characterName := u.StatusCacheService.CharacterName(characterID)
	typesEnabled := set.NewFromSlice(u.fyneApp.Preferences().StringList(settingNotificationsTypesEnabled))
	oldest := time.Now().UTC().Add(time.Second * time.Duration(maxAge) * -1)
	for _, n := range nn {
		if !typesEnabled.Contains(n.Type) || n.Timestamp.Before(oldest) {
			continue
		}
		title := fmt.Sprintf("%s: New Communication from %s", characterName, n.Sender.Name)
		x := fyne.NewNotification(title, n.Title.ValueOrZero())
		u.fyneApp.SendNotification(x)
		if err := u.CharacterService.UpdateCharacterNotificationSetProcessed(ctx, n); err != nil {
			slog.Error("Failed to set notification as processed", "characterID", characterID, "id", n.ID, "error", err)
			return
		}
	}
}

func (u *UI) processMails(ctx context.Context, characterID int32) {
	maxAge := u.fyneApp.Preferences().IntWithFallback(settingMaxAge, settingMaxAgeDefault)
	mm, err := u.CharacterService.ListCharacterMailHeadersForUnprocessed(ctx, characterID)
	if err != nil {
		slog.Error("Failed to fetch mails for processing", "characterID", characterID, "error", err)
		return
	}
	characterName := u.StatusCacheService.CharacterName(characterID)
	oldest := time.Now().UTC().Add(time.Second * time.Duration(maxAge) * -1)
	for _, m := range mm {
		if m.Timestamp.Before(oldest) {
			continue
		}
		title := fmt.Sprintf("%s: New Mail from %s", characterName, m.From)
		body := m.Subject
		x := fyne.NewNotification(title, body)
		u.fyneApp.SendNotification(x)
		if err := u.CharacterService.UpdateCharacterMailSetProcessed(ctx, m.ID); err != nil {
			slog.Error("Failed to set mail as processed", "characterID", characterID, "id", m.MailID, "error", err)
			return
		}
	}
}

func (u *UI) notifyExpiredExtractions(ctx context.Context, characterID int32) {
	planets, err := u.CharacterService.ListCharacterPlanets(ctx, characterID)
	if err != nil {
		slog.Error("failed to fetch character planets for notifications", "error", err)
		return
	}
	characterName := u.StatusCacheService.CharacterName(characterID)
	x := u.fyneApp.Preferences().String(settingNotifyPIEarliest)
	earliest, _ := time.Parse(time.RFC3339, x) // time when setting was enabled
	for _, p := range planets {
		expiration := p.ExtractionsExpiryTime()
		if expiration.IsZero() || expiration.After(time.Now()) || expiration.Before(earliest) {
			continue
		}
		if p.LastNotified.ValueOrZero().Equal(expiration) {
			continue
		}
		title := fmt.Sprintf("%s: PI extraction expired", characterName)
		extracted := strings.Join(p.ExtractedTypeNames(), ",")
		content := fmt.Sprintf("Extraction expired at %s for %s", p.EvePlanet.Name, extracted)
		u.fyneApp.SendNotification(fyne.NewNotification(title, content))
		if err := u.CharacterService.UpdateCharacterPlanetLastNotified(ctx, characterID, p.EvePlanet.ID, expiration); err != nil {
			slog.Error("failed to update last notified", "error", err)
		}
	}
}

func (u *UI) notifyExpiredTraining(ctx context.Context, characterID int32) error {
	c, err := u.CharacterService.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if !c.IsTrainingWatched {
		return nil
	}
	t, err := u.CharacterService.GetCharacterTotalTrainingTime(ctx, characterID)
	if err != nil {
		return err
	}
	if !t.IsEmpty() {
		return nil
	}
	title := fmt.Sprintf("%s: No skill in training", c.EveCharacter.Name)
	content := "There is currently no skill being trained for this character."
	u.fyneApp.SendNotification(fyne.NewNotification(title, content))
	return u.CharacterService.UpdateCharacterIsTrainingWatched(ctx, characterID, false)
}
