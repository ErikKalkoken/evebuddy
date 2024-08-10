package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
)

const (
	characterSectionsUpdateTicker = 10 * time.Second
	generalSectionsUpdateTicker   = 60 * time.Second
)

func (u *ui) startUpdateTickerGeneralSections() {
	ticker := time.NewTicker(generalSectionsUpdateTicker)
	go func() {
		for {
			u.updateGeneralSectionsAndRefreshIfNeeded(false)
			<-ticker.C
		}
	}()
}

func (u *ui) updateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool) {
	for _, s := range app.GeneralSections {
		go func(s app.GeneralSection) {
			u.updateGeneralSectionAndRefreshIfNeeded(context.TODO(), s, forceUpdate)
		}(s)
	}
}

func (u *ui) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section app.GeneralSection, forceUpdate bool) {
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

func (u *ui) startUpdateTickerCharacters() {
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
func (u *ui) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
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
func (u *ui) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s app.CharacterSection, forceUpdate bool) {
	maxMails, err := u.DictionaryService.IntWithFallback(settingMaxMails, settingMaxMailsDefault)
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	maxWalletTransactions, err := u.DictionaryService.IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	hasChanged, err := u.CharacterService.UpdateSectionIfNeeded(
		ctx, character.UpdateSectionParams{
			CharacterID:           characterID,
			Section:               s,
			ForceUpdate:           forceUpdate,
			MaxMails:              maxMails,
			MaxWalletTransactions: maxWalletTransactions,
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isCurrent := characterID == u.characterID()
	switch s {
	case app.SectionAssets:
		if isCurrent && hasChanged {
			u.assetsArea.redraw()
		}
		if hasChanged {
			u.assetSearchArea.refresh()
			u.wealthArea.refresh()
		}
	case app.SectionAttributes:
		if isCurrent && hasChanged {
			u.attributesArea.refresh()
		}
	case app.SectionImplants:
		if isCurrent && hasChanged {
			u.implantsArea.refresh()
		}
	case app.SectionJumpClones:
		if isCurrent && hasChanged {
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
		app.SectionMailLists,
		app.SectionMails:
		if isCurrent && hasChanged {
			u.mailArea.refresh()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case app.SectionSkills:
		if isCurrent && hasChanged {
			u.skillCatalogueArea.refresh()
			u.shipsArea.refresh()
			u.overviewArea.refresh()
		}
	case app.SectionSkillqueue:
		if isCurrent {
			u.skillqueueArea.refresh()
		}
	case app.SectionWalletJournal:
		if isCurrent && hasChanged {
			u.walletJournalArea.refresh()
		}
	case app.SectionWalletTransactions:
		if isCurrent && hasChanged {
			u.walletTransactionArea.refresh()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker: %s", s))
	}
}
