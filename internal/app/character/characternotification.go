package character

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character/notificationtype"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
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

func (s *CharacterService) ListCharacterNotificationsTypes(ctx context.Context, characterID int32, types []string) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsTypes(ctx, characterID, types)
}

func (s *CharacterService) ListCharacterNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsUnread(ctx, characterID)
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
			var updatedCount int
			for _, n := range existingNotifs {
				o, err := s.st.GetCharacterNotification(ctx, characterID, n.NotificationId)
				if err != nil {
					slog.Error("Failed to get existing character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					continue
				}
				arg1 := storage.UpdateCharacterNotificationParams{
					ID:          o.ID,
					IsRead:      o.IsRead,
					CharacterID: characterID,
					Title:       o.Title,
					Body:        o.Body,
				}
				title, body, err := s.renderCharacterNotification(ctx, characterID, n.Type_, n.Text)
				if err != nil {
					slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					continue
				}
				arg2 := storage.UpdateCharacterNotificationParams{
					ID:          o.ID,
					IsRead:      n.IsRead,
					CharacterID: characterID,
					Title:       title,
					Body:        body,
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
			_, err = s.EveUniverseService.AddMissingEveEntities(ctx, senderIDs.ToSlice())
			if err != nil {
				return err
			}
			for _, n := range newNotifs {
				title, body, err := s.renderCharacterNotification(ctx, characterID, n.Type_, n.Text)
				if err != nil {
					slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					continue
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

func (s *CharacterService) renderCharacterNotification(ctx context.Context, characterID int32, type_, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case "CorpAllBillMsg":
		title.Set("Bill issued")
		var data notificationtype.CorpAllBillMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CreditorID, data.DebtorID})
		if err != nil {
			return title, body, err
		}
		var out strings.Builder
		t := template.Must(template.New(type_).Parse(
			"A bill of **{{.Amount}}** ISK, due **{{.DueDate}}** owed by {{.Debtor}} to {{.Creditor}} " +
				"was issued on {{.CurrentDate}}. This bill is for {{.BillType}}.",
		))

		if err = t.Execute(&out, map[string]string{
			"Amount":      humanize.Commaf(data.Amount),
			"DueDate":     FromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
			"Debtor":      makeEveEntityProfileURL(entities[data.DebtorID]),
			"Creditor":    makeEveEntityProfileURL(entities[data.CreditorID]),
			"CurrentDate": FromLDAPTime(data.CurrentDate).Format(app.TimeDefaultFormat),
			"BillType":    billTypeName(data.BillTypeID),
		}); err != nil {
			return title, body, err
		}
		body.Set(out.String())
	}
	return title, body, nil
}

func billTypeName(id int32) string {
	switch id {
	case 7:
		return "Infrastructure Hub"
	}
	return "?"
}

func makeEveEntityProfileURL(e *app.EveEntity) string {
	const baseURL = "https://evemaps.dotlan.net"
	var path string
	switch e.Category {
	case app.EveEntityAlliance:
		path = "alliance"
	case app.EveEntityCorporation:
		path = "corp"
	default:
		return e.Name
	}
	name := strings.ReplaceAll(e.Name, " ", "_")
	url := fmt.Sprintf("%s/%s/%s", baseURL, path, name)
	return fmt.Sprintf("[%s](%s)", e.Name, url)
}
