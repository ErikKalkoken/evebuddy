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

func (s *CharacterService) updateContactsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterContacts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
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
			slices.SortStableFunc(contacts, func(a, b esi.CharactersCharacterIdContactsGetInner) int { // stable: keeps later duplicates last
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
			args := make([]storage.ReplaceCharacterContactParams, 0, len(contacts))
			index := make(map[int64]int, len(contacts))
			var duplicates int
			for _, r := range contacts {
				arg := storage.ReplaceCharacterContactParams{
					ContactID: r.ContactId,
					IsBlocked: optional.FromPtr(r.IsBlocked),
					IsWatched: optional.FromPtr(r.IsWatched),
					LabelIDs:  r.LabelIds,
					Standing:  r.Standing,
				}
				if i, found := index[r.ContactId]; found {
					args[i] = arg // pages can shift while being fetched
					duplicates++
					continue
				}
				index[r.ContactId] = len(args)
				args = append(args, arg)
			}
			if duplicates > 0 {
				slog.Warn("Duplicate contacts received from ESI", "characterID", characterID, "count", duplicates)
			}
			return s.st.ReplaceCharacterContacts(ctx, characterID, args)
		})
}

func (s *CharacterService) updateContactLabelsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterContactLabels {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
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
			labels := maps.Collect(xiter.MapSlice2(incoming, func(x esi.AlliancesAllianceIdContactsLabelsGetInner) (int64, string) {
				return x.LabelId, x.LabelName
			}))
			return s.st.ReplaceCharacterContactLabels(ctx, characterID, labels)
		})
}
