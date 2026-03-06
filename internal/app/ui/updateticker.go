package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// ticker
const (
	characterSectionsUpdateTicker = 60 * time.Second
)

// update general sections

// update character sections

func (u *baseUI) startUpdateTickerCharacters() {
	go func() {
		for {
			ctx := context.Background()
			go func() {
				if err := u.notifyCharactersIfNeeded(ctx); err != nil {
					slog.Error("Failed to notify characters", "error", err)
				}
			}()

			go func() {
				if err := u.UpdateCharactersIfNeeded(ctx, false); err != nil {
					slog.Error("Failed to update characters", "error", err)
				}
			}()
			<-time.Tick(characterSectionsUpdateTicker)
		}
	}()
}

func (u *baseUI) UpdateCharactersIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && u.ess.IsDailyDowntime() {
		slog.Info("Skipping regular update of characters during daily downtime")
		return nil
	}
	characters, err := u.cs.ListCharacterIDs(ctx)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for c := range characters.All() {
		wg.Go(func() {
			u.UpdateCharacterAndRefreshIfNeeded(ctx, c, forceUpdate)
		})
	}
	slog.Debug("Started updating characters", "characters", characters, "forceUpdate", forceUpdate)
	wg.Wait()
	slog.Debug("Finished updating characters", "characters", characters, "forceUpdate", forceUpdate)
	return nil
}

func (u *baseUI) notifyCharactersIfNeeded(ctx context.Context) error {
	characters, err := u.cs.ListCharacters(ctx)
	if err != nil {
		return err
	}
	if len(characters) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, c := range characters {
		if c.IsTrainingWatched && u.settings.NotifyTrainingEnabled() {
			wg.Go(func() {
				err := u.cs.NotifyExpiredTrainingForWatched(ctx, c.ID, u.sendDesktopNotification)
				if err != nil {
					slog.Error("Notify expired training", "characterID", c.ID, "error", err)
				}
			})
		}
	}
	slog.Debug("Started notifying characters", "characters", characters)
	wg.Wait()
	slog.Debug("Finished notifying characters", "characters", characters)
	return nil
}

func (u *baseUI) notifyNewCommunications(ctx context.Context, characterID int64) {
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
		slog.Error("Notify communications", "characterID", characterID, "error", err)
	}
}

// UpdateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (u *baseUI) UpdateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int64, forceUpdate bool) {
	if u.isOffline {
		return
	}
	var sections set.Set[app.CharacterSection] // sections registered for update
	if u.isMobile && !u.isForeground.Load() {
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

	id := "characters-" + uniqueID()
	u.signals.UpdateStarted.Emit(ctx, id)
	defer u.signals.UpdateStopped.Emit(ctx, id)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	key := fmt.Sprintf("updateCharacterAndRefreshIfNeeded-cancel-%d", characterID)
	u.signals.CharacterRemoved.AddListener(func(_ context.Context, c *app.EntityShort) {
		if c != nil && c.ID == characterID {
			cancel() // abort updates when the character is removed
		}
	}, key)
	defer func() {
		u.signals.CharacterRemoved.RemoveListener(key)
	}()
	slog.Debug("Starting to check character sections for update", "sections", sections)
	// _, err := u.cs.TokenSource(ctx, characterID)
	// if err != nil {
	// 	slog.Error("Failed to refresh token for update", "characterID", characterID, "error", err)
	// }
	var wg sync.WaitGroup
	// updateGroup starts a sequential update of group and removes them from sections.
	// It skips all updates for group if one of the group's sections has not been registered for update.
	updateGroup := func(group []app.CharacterSection) {
		mySections := set.Intersection(sections, set.Collect(slices.Values(group)))
		if mySections.Size() > 0 {
			wg.Go(func() {
				for _, s := range group {
					if !mySections.Contains(s) {
						continue
					}
					u.UpdateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
				}
			})
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
	})

	updateGroup([]app.CharacterSection{
		app.SectionCharacterContactLabels,
		app.SectionCharacterContacts,
	})

	updateGroup([]app.CharacterSection{
		app.SectionCharacterSkills,
		app.SectionCharacterSkillqueue,
	})

	updateGroup([]app.CharacterSection{
		app.SectionCharacterWalletBalance,
		app.SectionCharacterWalletJournal,
		app.SectionCharacterWalletTransactions,
	})

	// Other sections
	for s := range sections.All() {
		wg.Go(func() {
			u.UpdateCharacterSectionAndRefreshIfNeeded(ctx, characterID, s, forceUpdate)
		})
	}
	slog.Debug("Started updating character", "characterID", characterID, "sections", sections, "forceUpdate", forceUpdate)
	wg.Wait()
	slog.Debug("Finished updating character", "characterID", characterID, "sections", sections, "forceUpdate", forceUpdate)
}

// UpdateCharacterSectionAndRefreshIfNeeded runs update for a character section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on character sections needs to be included
// to make sure they are refreshed when data changes.
func (u *baseUI) UpdateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int64, section app.CharacterSection, forceUpdate bool) {
	logErr := func(err error) {
		slog.Error("Failed to process update for character section",
			"characterID", characterID,
			"section", section,
			"forcedUpdate", forceUpdate,
			"error", err,
		)
	}
	hasChanged, err := u.cs.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
		CharacterID: characterID,
		ForceUpdate: forceUpdate,
		Section:     section,
	})
	if err != nil {
		logErr(err)
		return
	}

	switch section {
	case app.SectionCharacterMailHeaders:
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			key := fmt.Sprintf("cancel-DownloadMissingMailBodies-%d", characterID)
			u.signals.CharacterRemoved.AddListener(func(_ context.Context, c *app.EntityShort) {
				if c != nil && c.ID == characterID {
					cancel() // abort updates when the character is removed
				}
			}, key)
			defer func() {
				u.signals.CharacterRemoved.RemoveListener(key)
			}()
			_, err := u.cs.DownloadMissingMailBodies(ctx, characterID)
			if err != nil {
				slog.Warn("DownloadMissingMailBodies", "characterID", characterID, "error", err)
			}
		}()
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
	}

	needsRefresh := hasChanged || forceUpdate
	arg := app.CharacterSectionUpdated{
		CharacterID:  characterID,
		Section:      section,
		NeedsRefresh: needsRefresh,
	}
	var wg sync.WaitGroup
	if needsRefresh {
		wg.Go(func() {
			u.signals.CharacterSectionChanged.Emit(ctx, arg)
		})
	}
	wg.Go(func() {
		u.signals.CharacterSectionUpdated.Emit(ctx, arg)
	})
	wg.Wait()
}

// update corporation sections

// uniqueID returns a pseudo unique ID.
func uniqueID() string {
	currentTime := time.Now().UnixNano()
	randomNumber := rand.Uint64()
	return fmt.Sprintf("%d-%d", currentTime, randomNumber)
}
