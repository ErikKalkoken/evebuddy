package character

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character/notificationtype"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/notification"
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
			"A bill of **{{.amount}}** ISK, due **{{.dueDate}}** owed by {{.debtor}} to {{.creditor}} " +
				"was issued on {{.currentDate}}. This bill is for {{.billType}}.",
		))

		if err := t.Execute(&out, map[string]string{
			"amount":      humanize.Commaf(data.Amount),
			"dueDate":     FromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
			"debtor":      makeEveEntityProfileURL(entities[data.DebtorID]),
			"creditor":    makeEveEntityProfileURL(entities[data.CreditorID]),
			"currentDate": FromLDAPTime(data.CurrentDate).Format(app.TimeDefaultFormat),
			"billType":    billTypeName(data.BillTypeID),
		}); err != nil {
			return title, body, err
		}
		body.Set(out.String())
	case "StructureUnderAttack":
		title.Set("Structure under attack")
		var data notification.StructureUnderAttack
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		attackChar, err := s.EveUniverseService.GetOrCreateEveCharacterESI(ctx, data.CharID)
		if err != nil {
			return title, body, err
		}
		structureType, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, data.StructureTypeID)
		if err != nil {
			return title, body, err
		}
		solarSystem, err := s.EveUniverseService.GetOrCreateEveSolarSystemESI(ctx, data.SolarsystemID)
		if err != nil {
			return title, body, err
		}
		structure, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, data.StructureID)
		if err != nil {
			return title, body, err
		}
		structureName := structure.DisplayName2()
		if structureName == "" {
			structureName = "?"
		}
		var out strings.Builder
		t := template.Must(template.New(type_).Parse(
			"The {{.structureType}} **{{.structureName}}**{{.location}} in {{.solarSystem}} is under attack.\n\n" +
				"Attacking Character: {{.character}}\n\n" +
				"Attacking Corporation: {{.corporation}}\n\n" +
				"Attacking Alliance: {{.alliance}}",
		))
		if err := t.Execute(&out, map[string]string{
			"structureType": structureType.Name,
			"structureName": structureName,
			"location":      "",
			"solarSystem":   makeLocationLink(solarSystem),
			"character":     attackChar.Name,
			"corporation":   makeCorporationLink(data.CorpName),
			"alliance":      makeAllianceLink(data.AllianceName),
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

func makeLocationLink(ess *app.EveSolarSystem) string {
	x := fmt.Sprintf(
		"%s (%s)",
		makeMarkDownLink(ess.Name, makeDotLanProfileURL(ess.Name, dotlanSolarSystem)),
		ess.Constellation.Region.Name,
	)
	return x
}

func makeCorporationLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanCorporation))
}

func makeAllianceLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanAlliance))
}

func makeEveEntityProfileURL(e *app.EveEntity) string {
	var url string
	switch e.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(e.Name, dotlanAlliance)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(e.Name, dotlanCorporation)
	}
	return makeMarkDownLink(e.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}
