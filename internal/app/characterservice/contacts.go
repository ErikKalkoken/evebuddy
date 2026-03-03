package characterservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func (s *CharacterService) ListContacts(ctx context.Context, characterID int64) ([]*app.CharacterContact, error) {
	return s.st.ListCharacterContacts(ctx, characterID)
}

func (s *CharacterService) updateContactsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterContacts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdContacts")
			contacts, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CharactersCharacterIdContactsGetInner, *http.Response, error) {
					return s.esiClient.ContactsAPI.GetCharactersCharacterIdContacts(ctx, characterID).Page(page).Execute()
				},
			)
			if err != nil {
				return nil, err
			}
			slices.SortFunc(contacts, func(a, b esi.CharactersCharacterIdContactsGetInner) int {
				return cmp.Compare(a.ContactId, b.ContactId)
			})
			slog.Debug("Received contacts from ESI", "count", len(contacts), "characterID", characterID)
			return contacts, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			contacts := data.([]esi.CharactersCharacterIdContactsGetInner)

			incomingIDs := set.Collect(xiter.MapSlice(contacts, func(x esi.CharactersCharacterIdContactsGetInner) int64 {
				return x.ContactId
			}))
			_, err := s.eus.AddMissingEntities(ctx, incomingIDs)
			if err != nil {
				return false, err
			}
			for _, r := range contacts {
				err = s.st.UpdateOrCreateCharacterContact(ctx, storage.UpdateOrCreateCharacterContactParams{
					CharacterID: characterID,
					ContactID:   r.ContactId,
					IsBlocked:   optional.FromPtr(r.IsBlocked),
					IsWatched:   optional.FromPtr(r.IsWatched),
					Standing:    r.Standing,
					LabelIDs:    r.LabelIds,
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info("Updated contacts", "characterID", characterID, "count", incomingIDs.Size())

			// Delete obsolete entries
			currentIDs, err := s.st.ListCharacterContactIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			obsoleteIDs := set.Difference(currentIDs, incomingIDs)
			if obsoleteIDs.Size() > 0 {
				err := s.st.DeleteCharacterContacts(ctx, characterID, obsoleteIDs)
				if err != nil {
					return false, err
				}
				slog.Info("Deleted obsolete contacts", "characterID", characterID, "count", obsoleteIDs.Size())
			}
			return true, nil
		})
}

func (s *CharacterService) updateContactLabelsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterContactLabels {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdContactsLabels")
			labels, _, err := s.esiClient.ContactsAPI.GetCharactersCharacterIdContactsLabels(ctx, characterID).Execute()
			if err != nil {
				return nil, err
			}
			slog.Debug("Received labels from ESI", "count", len(labels), "characterID", characterID)
			return labels, nil
		},
		func(ctx context.Context, characterID int64, x any) (bool, error) {
			incoming := x.([]esi.AlliancesAllianceIdContactsLabelsGetInner)
			incoming2 := maps.Collect(xiter.MapSlice2(incoming, func(x esi.AlliancesAllianceIdContactsLabelsGetInner) (int64, string) {
				return x.LabelId, x.LabelName
			}))

			current, err := s.st.ListCharacterContactLabels(ctx, characterID)
			if err != nil {
				return false, err
			}
			current2 := maps.Collect(xiter.MapSlice2(current, func(x *app.CharacterContactLabel) (int64, string) {
				return x.LabelID, x.Name
			}))

			var changed int
			for id2, name2 := range incoming2 {
				if name1, ok := current2[id2]; ok && name1 == name2 {
					continue
				}
				err := s.st.UpdateOrCreateCharacterContactLabel(ctx, storage.UpdateOrCreateCharacterContactLabelParams{
					CharacterID: characterID,
					LabelID:     id2,
					Name:        name2,
				})
				if err != nil {
					return false, err
				}
				changed++
			}
			slog.Info("Updated contact labels", "characterID", characterID, "count", changed)

			// Delete obsolete labels
			currentIDs := set.Collect(maps.Keys(current2))
			incomingIDs := set.Collect(maps.Keys(incoming2))
			obsoleteIDs := set.Difference(currentIDs, incomingIDs)
			if obsoleteIDs.Size() > 0 {
				err := s.st.DeleteCharacterContactLabels(ctx, characterID, obsoleteIDs)
				if err != nil {
					return false, err
				}
				slog.Info("Deleted obsolete contact labels", "characterID", characterID, "count", obsoleteIDs.Size())
				changed += obsoleteIDs.Size()
			}
			return changed > 0, nil
		})
}
