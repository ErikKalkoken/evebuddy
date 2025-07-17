package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
)

// TODO: Add tests for NotifyCommunications

func (s *CharacterService) NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[string], notify func(title, content string)) error {
	nn, err := s.st.ListCharacterNotificationsUnprocessed(ctx, characterID, earliest)
	if err != nil {
		return err
	}
	if len(nn) == 0 {
		return nil
	}
	characterName, err := s.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
	for _, n := range nn {
		if !typesEnabled.Contains(n.Type) {
			continue
		}
		title := fmt.Sprintf("%s: New Communication from %s", characterName, n.Sender.Name)
		content := n.Title.ValueOrZero()
		notify(title, content)
		if err := s.st.UpdateCharacterNotificationSetProcessed(ctx, n.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) ListNotificationsTypes(ctx context.Context, characterID int32, ng app.NotificationGroup) ([]*app.CharacterNotification, error) {
	types := evenotification.GroupTypes[ng]
	t2 := make([]string, len(types))
	for i, v := range types {
		t2[i] = string(v)
	}
	return s.st.ListCharacterNotificationsTypes(ctx, characterID, t2)
}

func (s *CharacterService) ListNotificationsAll(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsAll(ctx, characterID)
}

func (s *CharacterService) ListNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsUnread(ctx, characterID)
}

func (s *CharacterService) updateNotificationsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterNotifications {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			notifications, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdNotifications(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received notifications from ESI", "characterID", characterID, "count", len(notifications))
			return notifications, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			notifications := data.([]esi.GetCharactersCharacterIdNotifications200Ok)
			existingIDs, err := s.st.ListCharacterNotificationIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var newNotifs []esi.GetCharactersCharacterIdNotifications200Ok
			var existingNotifs []esi.GetCharactersCharacterIdNotifications200Ok
			for _, n := range notifications {
				if existingIDs.Contains(n.NotificationId) {
					existingNotifs = append(existingNotifs, n)
				} else {
					newNotifs = append(newNotifs, n)
				}
			}
			if err := s.loadEntitiesForNotifications(ctx, existingNotifs); err != nil {
				return err
			}
			var updatedCount int
			for _, n := range existingNotifs {
				o, err := s.st.GetCharacterNotification(ctx, characterID, n.NotificationId)
				if err != nil {
					slog.Error("Failed to get existing character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					continue
				}
				arg1 := storage.UpdateCharacterNotificationParams{
					ID:     o.ID,
					IsRead: o.IsRead,
					Title:  o.Title,
					Body:   o.Body,
				}
				arg2 := storage.UpdateCharacterNotificationParams{
					ID:     o.ID,
					IsRead: n.IsRead,
				}
				title, body, err := s.ens.RenderESI(ctx, n.Type_, n.Text, n.Timestamp)
				if errors.Is(err, app.ErrNotFound) {
					// do nothing
				} else if err != nil {
					slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
				} else {
					arg2.Title.Set(title)
					arg2.Body.Set(body)
				}
				if arg2 != arg1 {
					if err := s.st.UpdateCharacterNotification(ctx, arg2); err != nil {
						return err
					}
					updatedCount++
				}
			}
			if updatedCount > 0 {
				slog.Info("Updated notifications", "characterID", characterID, "count", updatedCount)
			}
			if len(newNotifs) == 0 {
				slog.Info("No new notifications", "characterID", characterID)
				return nil
			}
			if err := s.loadEntitiesForNotifications(ctx, newNotifs); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterNotificationParams, len(newNotifs))
			g := new(errgroup.Group)
			for i, n := range newNotifs {
				g.Go(func() error {
					arg := storage.CreateCharacterNotificationParams{
						CharacterID:    characterID,
						IsRead:         n.IsRead,
						NotificationID: n.NotificationId,
						SenderID:       n.SenderId,
						Text:           n.Text,
						Timestamp:      n.Timestamp,
						Type:           n.Type_,
					}
					title, body, err := s.ens.RenderESI(ctx, n.Type_, n.Text, n.Timestamp)
					if errors.Is(err, app.ErrNotFound) {
						// do nothing
					} else if err != nil {
						slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					} else {
						arg.Title.Set(title)
						arg.Body.Set(body)
					}
					args[i] = arg
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return err
			}
			for _, arg := range args {
				if err := s.st.CreateCharacterNotification(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new notifications", "characterID", characterID, "entries", len(newNotifs))
			return nil
		})
}

func (s *CharacterService) loadEntitiesForNotifications(ctx context.Context, notifications []esi.GetCharactersCharacterIdNotifications200Ok) error {
	if len(notifications) == 0 {
		return nil
	}
	var ids set.Set[int32]
	for _, n := range notifications {
		if n.SenderId != 0 {
			ids.Add(n.SenderId)
		}
		ids2, err := s.ens.EntityIDs(n.Type_, n.Text)
		if errors.Is(err, app.ErrNotFound) {
			continue
		} else if err != nil {
			return err
		}
		ids.AddSeq(ids2.All())
	}
	if ids.Size() > 0 {
		_, err := s.eus.AddMissingEntities(ctx, ids)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) NotifyExpiredExtractions(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	planets, err := s.ListPlanets(ctx, characterID)
	if err != nil {
		return err
	}
	characterName, err := s.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
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
		notify(title, content)
		arg := storage.UpdateCharacterPlanetLastNotifiedParams{
			CharacterID:  characterID,
			EvePlanetID:  p.EvePlanet.ID,
			LastNotified: expiration,
		}
		if err := s.st.UpdateCharacterPlanetLastNotified(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}
