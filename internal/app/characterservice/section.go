package characterservice

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

func (s *CharacterService) StartUpdateTickerCharacters(d time.Duration) {
	go func() {
		for {
			ctx := context.Background()
			go func() {
				if err := s.notifyCharactersIfNeeded(ctx); err != nil {
					slog.Error("Failed to notify characters", "error", err)
				}
			}()

			go func() {
				if err := s.UpdateCharactersIfNeeded(ctx, false); err != nil {
					slog.Error("Failed to update characters", "error", err)
				}
			}()
			<-time.Tick(d)
		}
	}()
}

func (s *CharacterService) UpdateCharactersIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && xgoesi.IsDailyDowntime() {
		slog.Info("Skipping regular update of characters during daily downtime")
		return nil
	}
	characters, err := s.ListCharacterIDs(ctx)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for c := range characters.All() {
		wg.Go(func() {
			s.UpdateCharacterAndRefreshIfNeeded(ctx, c, forceUpdate)
		})
	}
	slog.Debug("Started updating characters", "characters", characters, "forceUpdate", forceUpdate)
	wg.Wait()
	slog.Debug("Finished updating characters", "characters", characters, "forceUpdate", forceUpdate)
	return nil
}

func (s *CharacterService) notifyCharactersIfNeeded(ctx context.Context) error {
	characters, err := s.ListCharacters(ctx)
	if err != nil {
		return err
	}
	if len(characters) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, c := range characters {
		if c.IsTrainingWatched && s.settings.NotifyTrainingEnabled() {
			wg.Go(func() {
				err := s.NotifyExpiredTrainingForWatched(ctx, c.ID, s.sendDesktopNotification)
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

func (s *CharacterService) notifyNewCommunications(ctx context.Context, characterID int64) {
	earliest := s.settings.NotifyCommunicationsEarliest()
	xx := s.settings.NotificationTypesEnabled()
	var typesEnabled set.Set[app.EveNotificationType]
	for x := range xx.All() {
		nt, found := app.EveNotificationTypeFromString(x)
		if !found {
			continue
		}
		typesEnabled.Add(nt)
	}
	err := s.NotifyCommunications(
		ctx,
		characterID,
		earliest,
		typesEnabled,
		s.sendDesktopNotification,
	)
	if err != nil {
		slog.Error("Notify communications", "characterID", characterID, "error", err)
	}
}

// UpdateCharacterAndRefreshIfNeeded runs update for all sections of a character if needed
// and refreshes the UI accordingly.
func (s *CharacterService) UpdateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int64, forceUpdate bool) {
	sections := set.Of(app.CharacterSections...)
	id := "characters-" + s.signals.PseudoUniqueID()
	s.signals.UpdateStarted.Emit(ctx, id)
	defer s.signals.UpdateStopped.Emit(ctx, id)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	key := fmt.Sprintf("updateCharacterAndRefreshIfNeeded-cancel-%d", characterID)
	s.signals.CharacterRemoved.AddListener(func(_ context.Context, c *app.EntityShort) {
		if c != nil && c.ID == characterID {
			cancel() // abort updates when the character is removed
		}
	}, key)
	defer func() {
		s.signals.CharacterRemoved.RemoveListener(key)
	}()
	slog.Debug("Starting to check character sections for update", "sections", sections)
	// _, err := u.TokenSource(ctx, characterID)
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
				for _, g := range group {
					if !mySections.Contains(g) {
						continue
					}
					s.UpdateCharacterSectionAndRefreshIfNeeded(ctx, characterID, g, forceUpdate)
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
	for x := range sections.All() {
		wg.Go(func() {
			s.UpdateCharacterSectionAndRefreshIfNeeded(ctx, characterID, x, forceUpdate)
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
func (s *CharacterService) UpdateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int64, section app.CharacterSection, forceUpdate bool) {
	logErr := func(err error) {
		slog.Error("Failed to process update for character section",
			"characterID", characterID,
			"section", section,
			"forcedUpdate", forceUpdate,
			"error", err,
		)
	}
	hasChanged, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
		characterID: characterID,
		forceUpdate: forceUpdate,
		section:     section,
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
			s.signals.CharacterRemoved.AddListener(func(_ context.Context, c *app.EntityShort) {
				if c != nil && c.ID == characterID {
					cancel() // abort updates when the character is removed
				}
			}, key)
			defer func() {
				s.signals.CharacterRemoved.RemoveListener(key)
			}()
			_, err := s.DownloadMissingMailBodies(ctx, characterID)
			if err != nil {
				slog.Warn("DownloadMissingMailBodies", "characterID", characterID, "error", err)
			}
		}()
		if s.settings.NotifyMailsEnabled() {
			earliest := s.settings.NotifyMailsEarliest()
			if err := s.NotifyMails(ctx, characterID, earliest, s.sendDesktopNotification); err != nil {
				logErr(err)
			}
		}
	case app.SectionCharacterContracts:
		if s.settings.NotifyContractsEnabled() {
			earliest := s.settings.NotifyContractsEarliest()
			if err := s.NotifyUpdatedContracts(ctx, characterID, earliest, s.sendDesktopNotification); err != nil {
				logErr(err)
			}
		}
	case app.SectionCharacterNotifications:
		if s.settings.NotifyCommunicationsEnabled() {
			s.notifyNewCommunications(ctx, characterID)
		}
	case app.SectionCharacterPlanets:
		if s.settings.NotifyPIEnabled() {
			earliest := s.settings.NotifyPIEarliest()
			if err := s.NotifyExpiredExtractions(ctx, characterID, earliest, s.sendDesktopNotification); err != nil {
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
			s.signals.CharacterSectionChanged.Emit(ctx, arg)
		})
	}
	wg.Go(func() {
		s.signals.CharacterSectionUpdated.Emit(ctx, arg)
	})
	wg.Wait()
}

// HasSection reports whether a section exists at all for a character.
func (s *CharacterService) HasSection(ctx context.Context, characterID int64, section app.CharacterSection) (bool, error) {
	x, err := s.st.GetCharacterSectionStatus(ctx, characterID, section)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !x.IsMissing(), nil
}

type characterSectionUpdateParams struct {
	characterID int64
	forceUpdate bool
	section     app.CharacterSection
}

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CharacterService) UpdateSectionIfNeeded(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.characterID == 0 || arg.section == "" {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	if !arg.forceUpdate {
		status, err := s.st.GetCharacterSectionStatus(ctx, arg.characterID, arg.section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return false, err
			}
		} else {
			if !status.HasError() && !status.IsExpired() {
				return false, nil
			}
			if status.HasError() && !status.WasUpdatedWithinErrorTimedOut() {
				return false, nil
			}
		}
	}
	var f func(context.Context, characterSectionUpdateParams) (bool, error)
	switch arg.section {
	case app.SectionCharacterAssets:
		f = s.updateAssetsESI
	case app.SectionCharacterAttributes:
		f = s.updateAttributesESI
	case app.SectionCharacterContacts:
		f = s.updateContactsESI
	case app.SectionCharacterContactLabels:
		f = s.updateContactLabelsESI
	case app.SectionCharacterContracts:
		f = s.updateContractsESI
	case app.SectionCharacterImplants:
		f = s.updateImplantsESI
	case app.SectionCharacterIndustryJobs:
		f = s.updateIndustryJobsESI
	case app.SectionCharacterJumpClones:
		f = s.updateJumpClonesESI
	case app.SectionCharacterLocation:
		f = s.updateLocationESI
	case app.SectionCharacterLoyaltyPoints:
		f = s.updateLoyaltyPointEntriesESI
	case app.SectionCharacterMailHeaders:
		f = s.updateMailHeadersESI
	case app.SectionCharacterMailLabels:
		f = s.updateMailLabelsESI
	case app.SectionCharacterMailLists:
		f = s.updateMailListsESI
	case app.SectionCharacterMarketOrders:
		f = s.updateMarketOrdersESI
	case app.SectionCharacterNotifications:
		f = s.updateNotificationsESI
	case app.SectionCharacterOnline:
		f = s.updateOnlineESI
	case app.SectionCharacterRoles:
		f = s.updateRolesESI
	case app.SectionCharacterPlanets:
		f = s.updatePlanetsESI
	case app.SectionCharacterShip:
		f = s.updateShipESI
	case app.SectionCharacterSkillqueue:
		f = s.updateSkillqueueESI
	case app.SectionCharacterSkills:
		f = s.updateSkillsESI
	case app.SectionCharacterWalletBalance:
		f = s.updateWalletBalanceESI
	case app.SectionCharacterWalletJournal:
		f = s.updateWalletJournalEntryESI
	case app.SectionCharacterWalletTransactions:
		f = s.updateWalletTransactionESI
	default:
		return false, fmt.Errorf("update section: unknown section: %s", arg.section)
	}
	key := fmt.Sprintf("update-character-section-%s-%d", arg.section, arg.characterID)
	hasChanged, err, _ := xsingleflight.Do(&s.sfg, key, func() (bool, error) {
		return f(ctx, arg)
	})
	if err != nil {
		s.recordUpdateFailed(ctx, arg, err)
		return false, fmt.Errorf("update character section from ESI for %+v: %w", arg, err)
	}
	slog.Info(
		"Character section update completed",
		"characterID", arg.characterID,
		"section", arg.section,
		"forced", arg.forceUpdate,
		"hasChanged", hasChanged,
	)
	return hasChanged, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
func (s *CharacterService) updateSectionIfChanged(
	ctx context.Context,
	arg characterSectionUpdateParams,
	skipChangeDetection bool,
	fetch func(ctx context.Context, characterID int64) (any, error), // returns data from ESI
	update func(ctx context.Context, characterID int64, data any) (bool, error), // reports whether it has changed
) (bool, error) {
	err := s.recordUpdateStarted(ctx, arg)
	if err != nil {
		return false, err
	}
	ts, err := s.TokenSource(ctx, arg.characterID, arg.section.Scopes())
	if err != nil {
		return false, err
	}
	ctx = xgoesi.NewContextWithAuth(ctx, arg.characterID, ts)
	data, err := fetch(ctx, arg.characterID)
	if err != nil {
		return false, err
	}
	hash, err := calcContentHash(data)
	if err != nil {
		return false, err
	}

	// identify whether update is needed
	var needsUpdate bool
	if arg.forceUpdate || skipChangeDetection {
		needsUpdate = true
	} else {
		b, err := s.hasSectionChanged(ctx, arg, hash)
		if err != nil {
			return false, err
		}
		needsUpdate = b
	}

	var hasChanged bool
	if needsUpdate {
		b, err := update(ctx, arg.characterID, data)
		if err != nil {
			return false, err
		}
		hasChanged = b
	}
	if err := s.recordUpdateSuccessful(ctx, arg, hash); err != nil {
		return false, err
	}
	slog.Debug(
		"Has section changed",
		slog.Any("characterID", arg.characterID),
		slog.Any("section", arg.section),
		slog.Any("hasChanged", hasChanged || arg.forceUpdate),
	)
	return hasChanged, nil
}

func (s *CharacterService) recordUpdateStarted(ctx context.Context, arg characterSectionUpdateParams) error {
	startedAt := optional.New(time.Now())
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.characterID,
		Section:     arg.section,
		StartedAt:   &startedAt,
	})
	if err != nil {
		return err
	}
	s.scs.SetCharacterSection(o)
	return nil
}

func (s *CharacterService) recordUpdateSuccessful(ctx context.Context, arg characterSectionUpdateParams, hash string) error {
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.characterID,
		CompletedAt:  &completedAt,
		ContentHash:  &hash,
		ErrorMessage: &errorMessage,
		Section:      arg.section,
		StartedAt:    &startedAt2,
	})
	if err != nil {
		return err
	}
	s.scs.SetCharacterSection(o)
	return nil
}

func (s *CharacterService) recordUpdateFailed(ctx context.Context, arg characterSectionUpdateParams, err error) {
	slog.Error("Character section update failed", "characterID", arg.characterID, "section", arg.section, "error", err)
	errorMessage := err.Error()
	startedAt := optional.Optional[time.Time]{}
	o, err2 := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.characterID,
		Section:      arg.section,
		ErrorMessage: &errorMessage,
		StartedAt:    &startedAt,
	})
	if err2 != nil {
		slog.Error("record error for failed section update", "characterID", arg.characterID, "error", err2)
	} else {
		s.scs.SetCharacterSection(o)
	}
}

func (s *CharacterService) hasSectionChanged(ctx context.Context, arg characterSectionUpdateParams, hash string) (bool, error) {
	status, err := s.st.GetCharacterSectionStatus(ctx, arg.characterID, arg.section)
	if errors.Is(err, app.ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	hasChanged := status.ContentHash != hash
	return hasChanged, nil
}

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
