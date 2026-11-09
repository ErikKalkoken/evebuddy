package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sync"
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
			stored, err := s.st.ListCharacterNotificationStates(ctx, characterID)
			if err != nil {
				return false, err
			}
			storedByID := make(map[int64]storage.CharacterNotificationState, len(stored))
			for _, o := range stored {
				storedByID[o.NotificationID] = o
			}
			var incomingIDs set.Set[int64]
			var newNotifs []esi.CharactersCharacterIdNotificationsGetInner
			var existingNotifs []esi.CharactersCharacterIdNotificationsGetInner
			for _, n := range notifications {
				if incomingIDs.Contains(n.NotificationId) {
					continue // ESI might return duplicates
				}
				incomingIDs.Add(n.NotificationId)
				if _, found := storedByID[n.NotificationId]; found {
					existingNotifs = append(existingNotifs, n)
				} else {
					newNotifs = append(newNotifs, n)
				}
			}
			if err := s.loadEntitiesForNotifications(ctx, characterID, existingNotifs); err != nil {
				return false, err
			}

			// Remove deleted notifications
			if ids := set.Difference(set.Collect(maps.Keys(storedByID)), incomingIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterNotifications(ctx, characterID, ids); err != nil {
					return false, err
				}
				slog.Info("Removed deleted notifications", "characterID", characterID, "count", ids.Size())
			}

			type renderResult struct {
				title, body optional.Optional[string]
				ok          bool
			}
			results := make([]renderResult, len(existingNotifs))
			g := new(errgroup.Group)
			g.SetLimit(s.concurrencyLimit)
			for i, n := range existingNotifs {
				g.Go(func() error {
					title, body, ok, err := s.renderNotification(ctx, characterID, storedByID[n.NotificationId].Type, n)
					if err != nil {
						return err
					}
					results[i] = renderResult{title: title, body: body, ok: ok}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return false, err
			}
			var updateArgs []storage.UpdateCharacterNotificationParams
			for i, n := range existingNotifs {
				r := results[i]
				if !r.ok {
					continue
				}
				current := storedByID[n.NotificationId]
				// only set when changed, so a local read made during this update is kept
				var isRead2 optional.Optional[bool]
				isReadESI := optional.FromPtr(n.IsRead).ValueOrZero()
				if arg.forceUpdate || (isReadESI && !current.IsRead) {
					isRead2.Set(isReadESI)
				}
				var needsUpdate bool
				if arg.forceUpdate ||
					r.body != current.Body ||
					r.title != current.Title ||
					!isRead2.IsEmpty() {
					needsUpdate = true
				}
				if needsUpdate {
					updateArgs = append(updateArgs, storage.UpdateCharacterNotificationParams{
						Body:   r.body,
						ID:     current.ID,
						IsRead: isRead2,
						Title:  r.title,
					})
				}
			}
			if err := s.st.UpdateCharacterNotifications(ctx, updateArgs); err != nil {
				return false, err
			}
			var changed bool
			if len(updateArgs) > 0 {
				slog.Info("Updated notifications",
					slog.Any("characterID", characterID),
					slog.Any("count", len(updateArgs)),
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
			g = new(errgroup.Group)
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

					// stored without title and body when rendering failed
					title, body, _, err := s.renderNotification(ctx, characterID, nt, n)
					if err != nil {
						return err
					}
					arg.Title = title
					arg.Body = body

					args[i] = arg
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return false, err
			}
			if err := s.st.CreateCharacterNotifications(ctx, args); err != nil {
				return false, err
			}
			slog.Info("Stored new notifications", "characterID", characterID, "entries", len(newNotifs))

			return true, nil
		})
}

// renderNotification returns the rendered title and body of a notification.
// ok is false when rendering failed. err is only set when ctx was canceled.
func (s *CharacterService) renderNotification(ctx context.Context, characterID int64, nt app.EveNotificationType, n esi.CharactersCharacterIdNotificationsGetInner) (title, body optional.Optional[string], ok bool, err error) {
	t, b, err := s.ens.RenderESI(ctx, nt, optional.FromPtr(n.Text), n.Timestamp)
	if errors.Is(err, app.ErrNotFound) {
		return title, body, true, nil // no renderer for this type
	}
	if err != nil {
		if ctx.Err() != nil {
			return title, body, false, err
		}
		slog.Error("Failed to render character notification",
			slog.Any("characterID", characterID),
			slog.Any("NotificationID", n.NotificationId),
			slog.Any("error", err),
		)
		return title, body, false, nil
	}
	return optional.New(t), optional.New(b), true, nil
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
			if ctx.Err() != nil {
				return err
			}
			slog.Warn("Failed to resolve entity IDs from notifications", "characterID", characterID, "error", err)
		}
	}
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
				if err != nil && ctx.Err() == nil {
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
	err := s.NotifyNotifications(ctx, characterID, earliest, typesEnabled)
	if err != nil && ctx.Err() == nil {
		slog.Error("Notify communications", "characterID", characterID, "error", err)
	}
}
