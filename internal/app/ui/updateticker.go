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
					u.updateCharacterAndRefreshIfNeeded(context.TODO(), c.ID, false)
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
		go func(s app.CharacterSection) {
			u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
		}(s)
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
	switch s {
	case app.SectionAssets:
		if isShown && hasChanged {
			u.assetsArea.redraw()
		}
		if hasChanged {
			u.assetSearchArea.refresh()
			u.wealthArea.refresh()
		}
	case app.SectionAttributes:
		if isShown && hasChanged {
			u.attributesArea.refresh()
		}
	case app.SectionImplants:
		if isShown && hasChanged {
			u.implantsArea.refresh()
		}
	case app.SectionJumpClones:
		if isShown && hasChanged {
			u.jumpClonesArea.redraw()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case app.SectionLocation,
		app.SectionOnline,
		app.SectionShip,
		app.SectionWalletBalance:
		if hasChanged {
			u.overviewArea.refresh()
			u.wealthArea.refresh()
		}
	case app.SectionMailLabels,
		app.SectionMailLists:
		if isShown && hasChanged {
			u.mailArea.refresh()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case app.SectionMails:
		if isShown && hasChanged {
			u.mailArea.refresh()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault) {
			go u.processMails(ctx, characterID)
		}
	case app.SectionNotifications:
		if isShown && hasChanged {
			u.notificationsArea.refresh()
		}
		if u.fyneApp.Preferences().BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault) {
			go u.processNotifications(ctx, characterID)
		}
	case app.SectionSkills:
		if isShown && hasChanged {
			u.skillCatalogueArea.refresh()
			u.shipsArea.refresh()
			u.overviewArea.refresh()
		}
	case app.SectionSkillqueue:
		if isShown {
			u.skillqueueArea.refresh()
		}
	case app.SectionWalletJournal:
		if isShown && hasChanged {
			u.walletJournalArea.refresh()
		}
	case app.SectionWalletTransactions:
		if isShown && hasChanged {
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
	var characterName string
	character, err := u.CharacterService.GetCharacter(ctx, characterID)
	if err != nil {
		slog.Error("Failed to fetch character", "characterID", characterID, "error", err)
		characterName = "?"
	} else {
		characterName = character.EveCharacter.Name
	}
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
	var characterName string
	character, err := u.CharacterService.GetCharacter(ctx, characterID)
	if err != nil {
		slog.Error("Failed to fetch character", "characterID", characterID, "error", err)
		characterName = "?"
	} else {
		characterName = character.EveCharacter.Name
	}
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
