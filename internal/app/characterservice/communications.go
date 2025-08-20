package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
)

func (s *CharacterService) CountNotifications(ctx context.Context, characterID int32) (map[app.EveNotificationGroup][]int, error) {
	types, err := s.st.CountCharacterNotifications(ctx, characterID)
	if err != nil {
		return nil, err
	}
	values := make(map[app.EveNotificationGroup][]int)
	for name, v := range types {
		g := app.EveNotificationType(name).Group()
		if _, ok := values[g]; !ok {
			values[g] = make([]int, 2)
		}
		values[g][0] += v[0]
		values[g][1] += v[1]
	}
	return values, nil
}

func (s *CharacterService) NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[app.EveNotificationType], notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyCommunications-%d", characterID), func() (any, error) {
		nn, err := s.st.ListCharacterNotificationsUnprocessed(ctx, characterID, earliest)
		if err != nil {
			return nil, err
		}
		if len(nn) == 0 {
			return nil, nil
		}
		for _, n := range nn {
			if !typesEnabled.Contains(n.Type) {
				continue
			}
			title, content, err := s.RenderNotificationSummary(ctx, n)
			if err != nil {
				return nil, err
			}
			notify(title, content)
			if err := s.st.UpdateCharacterNotificationSetProcessed(ctx, n.ID); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("NotifyCommunications for character %d: %w", characterID, err)
	}
	return nil
}

// NotificationRecipient returns the recipient for a notification.
func (s *CharacterService) NotificationRecipient(ctx context.Context, n *app.CharacterNotification) (*app.EveEntity, error) {
	character, err := s.st.GetCharacter(ctx, n.CharacterID)
	if err != nil {
		return nil, err
	}
	switch app.EveNotificationType(n.Type).Category() {
	case app.EveEntityCorporation:
		return character.EveCharacter.Corporation, nil
	case app.EveEntityAlliance:
		if !character.EveCharacter.HasAlliance() {
			return character.EveCharacter.Corporation, nil
		}
		return character.EveCharacter.Alliance, nil
	default:
		return character.EveCharacter.ToEveEntity(), nil
	}
}

// RenderNotificationSummary renders a summary from a character notification.
func (s *CharacterService) RenderNotificationSummary(ctx context.Context, n *app.CharacterNotification) (string, string, error) {
	recipient, err := s.NotificationRecipient(ctx, n)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("%s: New Communication from %s", recipient.Name, n.Sender.Name)
	content := n.Title.ValueOrZero()
	return title, content, nil
}

func (s *CharacterService) ListNotificationsForGroup(ctx context.Context, characterID int32, ng app.EveNotificationGroup) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsForTypes(ctx, characterID, app.NotificationGroupTypes(ng))
}

func (s *CharacterService) ListNotificationsAll(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsAll(ctx, characterID)
}

func (s *CharacterService) ListNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsUnread(ctx, characterID)
}

func (s *CharacterService) updateNotificationsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
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
				title, body, err := s.ens.RenderESI(ctx, o.Type, o.Text, o.Timestamp)
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
			g.SetLimit(s.concurrencyLimit)
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
					nt, found := s.st.EveNotificationTypeFromESIString(n.Type_)
					if !found {
						nt = app.UnknownNotification
					}
					title, body, err := s.ens.RenderESI(ctx, nt, n.Text, n.Timestamp)
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
		nt, found := s.st.EveNotificationTypeFromESIString(n.Type_)
		if !found {
			continue
		}
		ids2, err := s.ens.EntityIDs(nt, n.Text)
		if errors.Is(err, app.ErrNotFound) {
			continue
		}
		if err != nil {
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
