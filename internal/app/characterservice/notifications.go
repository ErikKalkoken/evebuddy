package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) GetNotification(ctx context.Context, characterID, notificationID int64) (*app.CharacterNotification, error) {
	return s.st.GetCharacterNotification(ctx, characterID, notificationID)
}

func (s *CharacterService) NotifyNotifications(ctx context.Context, characterID int64, earliest time.Time, typesEnabled set.Set[app.EveNotificationType]) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyCommunications-%d", characterID), func() (any, error) {
		nn, err := s.st.ListCharacterNotificationsUnprocessed(ctx, characterID, earliest)
		if err != nil {
			return nil, fmt.Errorf("notify communications: %w", err)
		}
		if len(nn) == 0 {
			return nil, nil
		}
		for _, n := range nn {
			if !typesEnabled.Contains(n.Type) {
				continue
			}
			if err := s.SendDesktopNotification(ctx, n); err != nil {
				return nil, fmt.Errorf("notify communications: %w", err)
			}
			if err := s.st.UpdateCharacterNotificationsSetProcessed(ctx, n.CharacterID, n.NotificationID); err != nil {
				return nil, fmt.Errorf("notify communications: %w", err)
			}
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("NotifyCommunications for character %d: %w", characterID, err)
	}
	return nil
}

func (s *CharacterService) SetNotificationsAsRead(ctx context.Context, ids set.Set[int64]) error {
	return s.st.UpdateCharacterNotificationsSetIsRead(ctx, ids, true)
}

func (s *CharacterService) SendDesktopNotification(ctx context.Context, n *app.CharacterNotification) error {
	var recipient string
	v, ok := n.Recipient.Value()
	if ok {
		recipient = v.Name
	} else {
		n, err := s.getCharacterName(ctx, n.CharacterID)
		if err != nil {
			return fmt.Errorf("SendDesktopNotification: %w", err)
		}
		recipient = n
	}
	title := fmt.Sprintf("%s: New Communication from %s", recipient, n.Sender.Name)
	content := n.Title.ValueOrZero()
	s.sendDesktopNotification(title, content)
	return nil
}

func (s *CharacterService) ListAllNotifications(ctx context.Context) ([]*app.CharacterNotification, error) {
	return s.st.ListAllCharacterNotifications(ctx)
}

func (s *CharacterService) ListNotifications(ctx context.Context, characterID int64) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotifications(ctx, characterID)
}

func (s *CharacterService) updateNotificationsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterNotifications {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdNotifications")
			notifications, _, err := s.esiClient.CharacterAPI.GetCharactersCharacterIdNotifications(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received notifications from ESI",
				slog.Any("characterID", characterID),
				slog.Any("count", len(notifications)),
			)
			return notifications, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			notifications := data.([]esi.CharactersCharacterIdNotificationsGetInner)
			existingIDs, err := s.st.ListCharacterNotificationIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			var incomingIDs set.Set[int64]
			var newNotifs []esi.CharactersCharacterIdNotificationsGetInner
			var existingNotifs []esi.CharactersCharacterIdNotificationsGetInner
			for _, n := range notifications {
				incomingIDs.Add(n.NotificationId)
				if existingIDs.Contains(n.NotificationId) {
					existingNotifs = append(existingNotifs, n)
				} else {
					newNotifs = append(newNotifs, n)
				}
			}
			if err := s.loadEntitiesForNotifications(ctx, characterID, existingNotifs); err != nil {
				return false, err
			}

			// Remove deleted notifications
			if ids := set.Difference(existingIDs, incomingIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterNotifications(ctx, characterID, ids); err != nil {
					return false, err
				}
				slog.Info("Removed deleted notifications", "characterID", characterID, "count", ids.Size())
			}

			var updatedCount int
			for _, n := range existingNotifs {
				current, err := s.st.GetCharacterNotification(ctx, characterID, n.NotificationId)
				if err != nil {
					slog.Error("Failed to get existing character notification",
						slog.Any("characterID", characterID),
						slog.Any("NotificationID", n.NotificationId),
						slog.Any("error", err))
					continue
				}
				var title2, body2 optional.Optional[string]
				t, b, err := s.ens.RenderESI(ctx, current.Type, optional.FromPtr(n.Text), n.Timestamp)
				if errors.Is(err, app.ErrNotFound) {
					// do nothing
				} else if err != nil {
					slog.Error("Failed to render character notification",
						slog.Any("characterID", characterID),
						slog.Any("NotificationID", n.NotificationId),
						slog.Any("error", err),
					)
					continue
				} else {
					title2.Set(t)
					body2.Set(b)
				}
				isRead2 := optional.FromPtr(n.IsRead).ValueOrZero()
				if !arg.forceUpdate {
					isRead2 = isRead2 || current.IsRead
				}
				var needsUpdate bool
				if arg.forceUpdate ||
					body2 != current.Body ||
					title2 != current.Title ||
					(isRead2 && !current.IsRead) {
					needsUpdate = true
				}
				if needsUpdate {
					err := s.st.UpdateCharacterNotification(ctx, storage.UpdateCharacterNotificationParams{
						Body:   body2,
						ID:     current.ID,
						IsRead: isRead2,
						Title:  title2,
					})
					if err != nil {
						return false, err
					}
					updatedCount++
				}
			}
			var changed bool
			if updatedCount > 0 {
				slog.Info("Updated notifications",
					slog.Any("characterID", characterID),
					slog.Any("count", updatedCount),
				)
				changed = true
			}
			if len(newNotifs) == 0 {
				slog.Info("No new notifications", "characterID", characterID)
				return changed, nil
			}

			if err := s.loadEntitiesForNotifications(ctx, characterID, newNotifs); err != nil {
				return false, err
			}
			character, err := s.st.GetCharacter(ctx, characterID)
			if err != nil {
				return false, err
			}
			if _, err := s.eus.AddMissingEntities(ctx, character.EveCharacter.EntityIDs()); err != nil {
				return false, err
			}
			args := make([]storage.CreateCharacterNotificationParams, len(newNotifs))
			g := new(errgroup.Group)
			g.SetLimit(s.concurrencyLimit)
			for i, n := range newNotifs {
				g.Go(func() error {
					arg := storage.CreateCharacterNotificationParams{
						CharacterID:    characterID,
						IsRead:         optional.FromPtr(n.IsRead).ValueOrZero(),
						NotificationID: n.NotificationId,
						SenderID:       n.SenderId,
						Text:           optional.FromPtr(n.Text),
						Timestamp:      n.Timestamp,
						Type:           n.Type,
					}

					nt, found := storage.EveNotificationTypeFromESIString(n.Type)
					if !found {
						nt = app.UnknownNotification
					}
					var recipientID int64
					switch nt.Category() {
					case app.EveEntityCorporation:
						recipientID = character.EveCharacter.Corporation.ID
					case app.EveEntityAlliance:
						v, ok := character.EveCharacter.Alliance.Value()
						if ok {
							recipientID = v.ID
						} else {
							recipientID = character.EveCharacter.Corporation.ID
						}
					default:
						recipientID = character.ID
					}
					arg.RecipientID = optional.New(recipientID)

					title, body, err := s.ens.RenderESI(ctx, nt, optional.FromPtr(n.Text), n.Timestamp)
					if errors.Is(err, app.ErrNotFound) {
						// do nothing
					} else if err != nil {
						slog.Error("Failed to render character notification",
							slog.Any("characterID", characterID),
							slog.Any("NotificationID", n.NotificationId),
							slog.Any("error", err),
						)
					} else {
						arg.Title.Set(title)
						arg.Body.Set(body)
					}

					args[i] = arg
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return false, err
			}
			for _, arg := range args {
				if err := s.st.CreateCharacterNotification(ctx, arg); err != nil {
					return false, err
				}
			}
			slog.Info("Stored new notifications", "characterID", characterID, "entries", len(newNotifs))

			return true, nil
		})
}

func (s *CharacterService) loadEntitiesForNotifications(ctx context.Context, characterID int64, notifications []esi.CharactersCharacterIdNotificationsGetInner) error {
	if len(notifications) == 0 {
		return nil
	}
	// resolve senders (mandatory)
	var ids set.Set[int64]
	for _, n := range notifications {
		if n.SenderId != 0 {
			ids.Add(n.SenderId)
		}
	}
	_, err := s.eus.AddMissingEntities(ctx, ids)
	if err != nil {
		return err
	}

	// resolve IDs in text field (optional)
	ids.Clear()
	for _, n := range notifications {
		nt, found := storage.EveNotificationTypeFromESIString(n.Type)
		if !found {
			continue
		}
		ids2, err := s.ens.EntityIDs(nt, optional.FromPtr(n.Text))
		if errors.Is(err, app.ErrNotFound) {
			continue
		}
		if err != nil {
			slog.Warn("Failed to extract entity IDs from notifications", "characterID", characterID, "error", err)
			continue
		}
		ids.AddSeq(ids2.All())
	}
	if ids.Size() > 0 {
		_, err := s.eus.AddMissingEntities(ctx, ids)
		if err != nil {
			slog.Warn("Failed to resolve entity IDs from notifications", "characterID", characterID, "error", err)
		}
	}
	return nil
}
