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
	charactersUpdateTicker      = 10 * time.Second
	eveDataUpdateTicker         = 60 * time.Second
	eveCharacterUpdateTimeout   = 3600 * time.Second
	eveCategoriesUpdateTimeout  = 24 * time.Hour
	eveCategoriesKeyLastUpdated = "eve-categories-last-updated"
	eveCharactersKeyLastUpdated = "eve-characters-last-updated"
)

type EveUniverseSection string

const (
	SectionCategories EveUniverseSection = "categories"
	SectionCharacters EveUniverseSection = "characters"
)

type EveUniverseUpdateStatus struct {
	ContentHash   string
	ErrorMessage  string
	LastUpdatedAt time.Time
	Section       EveUniverseSection
}

func (u *ui) startUpdateTickerEveCharacters() {
	ticker := time.NewTicker(eveDataUpdateTicker)
	go func() {
		ctx := context.Background()
		for {
			err := func() error {
				lastUpdated, ok, err := u.sv.Dictionary.GetTime(eveCharactersKeyLastUpdated)
				if err != nil {
					return err
				}
				if ok && time.Now().Before(lastUpdated.Add(eveCharacterUpdateTimeout)) {
					return nil
				}
				slog.Info("Started updating eve characters")
				if err := u.sv.EveUniverse.UpdateAllEveCharactersESI(ctx); err != nil {
					return err
				}
				slog.Info("Finished updating eve characters")
				if err := u.sv.Dictionary.SetTime(eveCharactersKeyLastUpdated, time.Now()); err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				slog.Error("Failed to update eve characters: %s", err)
			}
			<-ticker.C
		}
	}()
}

func (u *ui) startUpdateTickerEveCategorySkill() {
	ticker := time.NewTicker(eveDataUpdateTicker)
	go func() {
		ctx := context.TODO()
		for {
			err := func() error {
				lastUpdated, ok, err := u.sv.Dictionary.GetTime(eveCategoriesKeyLastUpdated)
				if err != nil {
					return err
				}
				if ok && time.Now().Before(lastUpdated.Add(eveCategoriesUpdateTimeout)) {
					return nil
				}
				slog.Info("Started updating categories")
				if err := u.sv.EveUniverse.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategorySkill); err != nil {
					return err
				}
				if err := u.sv.EveUniverse.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategoryShip); err != nil {
					return err
				}
				if err := u.sv.EveUniverse.UpdateEveShipSkills(ctx); err != nil {
					return err
				}
				slog.Info("Finished updating categories")
				if err := u.sv.Dictionary.SetTime(eveCategoriesKeyLastUpdated, time.Now()); err != nil {
					return err
				}
				u.shipsArea.refresh()
				u.skillCatalogueArea.redraw()
				return nil
			}()
			if err != nil {
				slog.Error("Failed to update skill category: %s", err)
			}
			u.skillCatalogueArea.refresh()
			<-ticker.C
		}
	}()
}

func (u *ui) startUpdateTickerCharacters() {
	ticker := time.NewTicker(charactersUpdateTicker)
	go func() {
		ctx := context.Background()
		for {
			func() {
				cc, err := u.sv.Characters.ListCharactersShort(ctx)
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					u.updateCharacterAndRefreshIfNeeded(ctx, c.ID, false)
				}
			}()
			<-ticker.C
		}
	}()
}

// updateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *ui) updateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool) {
	for _, s := range model.CharacterSections {
		go func(s model.CharacterSection) {
			u.updateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
		}(s)
	}
}

func (u *ui) updateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s model.CharacterSection, forceUpdate bool) {
	hasChanged, err := u.sv.Characters.UpdateCharacterSection(
		ctx, character.UpdateCharacterSectionParams{
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
			u.assetSearchArea.refresh()
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
