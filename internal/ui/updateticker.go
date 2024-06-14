package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
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
	for _, s := range model.GeneralSections {
		go func(s model.GeneralSection) {
			u.updateGeneralSectionAndRefreshIfNeeded(context.TODO(), s, forceUpdate)
		}(s)
	}
}

func (u *ui) updateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section model.GeneralSection, forceUpdate bool) {
	hasChanged, err := u.sv.EveUniverse.UpdateSection(ctx, section, forceUpdate)
	if err != nil {
		slog.Error("Failed to update general section", "section", section, "err", err)
		return
	}
	switch section {
	case model.SectionEveCategories:
		if hasChanged {
			u.shipsArea.refresh()
			u.skillCatalogueArea.refresh()
		}
	case model.SectionEveCharacters, model.SectionEveMarketPrices:
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
				cc, err := u.sv.Character.ListCharactersShort(context.TODO())
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
	for _, s := range model.CharacterSections {
		go func(s model.CharacterSection) {
			u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
		}(s)
	}
}

// updateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *ui) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s model.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.sv.Character.UpdateSectionIfNeeded(
		ctx, character.UpdateSectionParams{
			CharacterID: characterID,
			Section:     s,
			ForceUpdate: forceUpdate,
		})
	if err != nil {
		slog.Error("Failed to update character section", "characterID", characterID, "section", s, "err", err)
		return
	}
	isCurrent := characterID == u.characterID()
	switch s {
	case model.SectionAssets:
		if isCurrent && hasChanged {
			u.assetsArea.redraw()
		}
		if hasChanged {
			u.assetSearchArea.refresh()
			u.wealthArea.refresh()
		}
	case model.SectionAttributes:
		if isCurrent && hasChanged {
			u.attributesArea.refresh()
		}
	case model.SectionImplants:
		if isCurrent && hasChanged {
			u.implantsArea.refresh()
		}
	case model.SectionJumpClones:
		if isCurrent && hasChanged {
			u.jumpClonesArea.redraw()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case model.SectionLocation,
		model.SectionOnline,
		model.SectionShip,
		model.SectionWalletBalance:
		if hasChanged {
			u.overviewArea.refresh()
			u.wealthArea.refresh()
		}
	case model.SectionMailLabels,
		model.SectionMailLists,
		model.SectionMails:
		if isCurrent && hasChanged {
			u.mailArea.refresh()
		}
		if hasChanged {
			u.overviewArea.refresh()
		}
	case model.SectionSkills:
		if isCurrent && hasChanged {
			u.skillCatalogueArea.refresh()
			u.shipsArea.refresh()
			u.overviewArea.refresh()
		}
	case model.SectionSkillqueue:
		if isCurrent {
			u.skillqueueArea.refresh()
		}
	case model.SectionWalletJournal:
		if isCurrent && hasChanged {
			u.walletJournalArea.refresh()
		}
	case model.SectionWalletTransactions:
		if isCurrent && hasChanged {
			u.walletTransactionArea.refresh()
		}
	default:
		slog.Warn(fmt.Sprintf("section not part of the update ticker: %s", s))
	}
}
