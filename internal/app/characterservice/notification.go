package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) CountNotifications(ctx context.Context, characterID int32) (map[app.NotificationGroup][]int, error) {
	types, err := s.st.CountCharacterNotifications(ctx, characterID)
	if err != nil {
		return nil, err
	}
	values := make(map[app.NotificationGroup][]int)
	for name, v := range types {
		c := evenotification.Type2group[evenotification.Type(name)]
		if _, ok := values[c]; !ok {
			values[c] = make([]int, 2)
		}
		values[c][0] += v[0]
		values[c][1] += v[1]
	}
	return values, nil
}

// TODO: Add tests for NotifyCommunications

func (cs *CharacterService) NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[string], notify func(title, content string)) error {
	nn, err := cs.st.ListCharacterNotificationsUnprocessed(ctx, characterID, earliest)
	if err != nil {
		return err
	}
	if len(nn) == 0 {
		return nil
	}
	characterName, err := cs.getCharacterName(ctx, characterID)
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
		if err := cs.st.UpdateCharacterNotificationSetProcessed(ctx, n.ID); err != nil {
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
	if arg.Section != app.SectionNotifications {
		panic("called with wrong section")
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
				title, body, err := s.ens.RenderESI(ctx, n.Type_, n.Text, n.Timestamp)
				if err != nil {
					slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
				}
				arg2 := storage.UpdateCharacterNotificationParams{
					ID:     o.ID,
					IsRead: n.IsRead,
					Title:  title,
					Body:   body,
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
			senderIDs := set.New[int32]()
			for _, n := range newNotifs {
				if n.SenderId != 0 {
					senderIDs.Add(n.SenderId)
				}
			}
			_, err = s.eus.AddMissingEntities(ctx, senderIDs.ToSlice())
			if err != nil {
				return err
			}
			for _, n := range newNotifs {
				title, body, err := s.ens.RenderESI(ctx, n.Type_, n.Text, n.Timestamp)
				if err != nil {
					slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
				}
				arg := storage.CreateCharacterNotificationParams{
					Body:           body,
					CharacterID:    characterID,
					IsRead:         n.IsRead,
					NotificationID: n.NotificationId,
					SenderID:       n.SenderId,
					Text:           n.Text,
					Timestamp:      n.Timestamp,
					Title:          title,
					Type:           n.Type_,
				}
				if err := s.st.CreateCharacterNotification(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new notifications", "characterID", characterID, "entries", len(newNotifs))
			return nil
		})
}
