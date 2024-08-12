package character

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) CalcCharacterNotificationUnreadCounts(ctx context.Context, characterID int32) (map[app.NotificationCategory]int, error) {
	types, err := s.st.CalcCharacterNotificationUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, err
	}
	categories := make(map[app.NotificationCategory]int)
	for name, count := range types {
		c := app.Notification2category[name]
		categories[c] += count
	}
	return categories, nil
}

func (s *CharacterService) ListCharacterNotifications(ctx context.Context, characterID int32, types []string) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotifications(ctx, characterID, types)
}

func (s *CharacterService) updateCharacterNotificationsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
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
			return notifications, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			notifications := data.([]esi.GetCharactersCharacterIdNotifications200Ok)
			ii, err := s.st.ListCharacterNotificationIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ii)
			var newNotifs []esi.GetCharactersCharacterIdNotifications200Ok
			var existingNotifs []esi.GetCharactersCharacterIdNotifications200Ok
			for _, n := range notifications {
				if existingIDs.Has(n.NotificationId) {
					existingNotifs = append(existingNotifs, n)
				} else {
					newNotifs = append(newNotifs, n)
				}
			}
			for _, n := range existingNotifs {
				o, err := s.st.GetCharacterNotification(ctx, characterID, n.NotificationId)
				if err != nil {
					return err
				}
				if o.IsRead != n.IsRead {
					if err := s.st.UpdateCharacterNotificationIsRead(ctx, characterID, o.ID, n.IsRead); err != nil {
						return err
					}
				}
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
			_, err = s.EveUniverseService.AddMissingEveEntities(ctx, senderIDs.ToSlice())
			if err != nil {
				return err
			}
			for _, n := range newNotifs {
				arg := storage.CreateCharacterNotificationParams{
					CharacterID:    characterID,
					IsRead:         n.IsRead,
					NotificationID: n.NotificationId,
					SenderID:       n.SenderId,
					Text:           n.Text,
					Timestamp:      n.Timestamp,
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
