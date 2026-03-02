package characterservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
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
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdContacts")
			rows, err := xgoesi.FetchPages(
				func(page int32) ([]esi.CharactersCharacterIdContactsGetInner, *http.Response, error) {
					return s.esiClient.ContactsAPI.GetCharactersCharacterIdContacts(ctx, characterID).Page(page).Execute()
				},
			)
			if err != nil {
				return nil, err
			}
			slices.SortFunc(rows, func(a, b esi.CharactersCharacterIdContactsGetInner) int {
				return cmp.Compare(a.ContactId, b.ContactId)
			})

			slog.Debug("Received contacts from ESI", "count", len(rows), "characterID", characterID)
			return rows, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			rows := data.([]esi.CharactersCharacterIdContactsGetInner)
			incomingIDs := set.Collect(xiter.MapSlice(rows, func(x esi.CharactersCharacterIdContactsGetInner) int64 {
				return x.ContactId
			}))

			_, err := s.eus.AddMissingEntities(ctx, incomingIDs)
			if err != nil {
				return false, err
			}

			for _, r := range rows {
				err = s.st.UpdateOrCreateCharacterContact(ctx, storage.UpdateOrCreateCharacterContactParams{
					CharacterID: characterID,
					ContactID:   r.ContactId,
					IsBlocked:   optional.FromPtr(r.IsBlocked),
					IsWatched:   optional.FromPtr(r.IsWatched),
					Standing:    r.Standing,
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info("Updated loyalty points entries", "characterID", characterID, "count", incomingIDs.Size())

			// Delete obsolete entries
			currentIDs, err := s.st.ListCharacterContactIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			obsoleteIDs := set.Difference(incomingIDs, currentIDs)
			if obsoleteIDs.Size() > 0 {
				err := s.st.DeleteCharacterContacts(ctx, characterID, obsoleteIDs)
				if err != nil {
					return false, err
				}
				slog.Info("Deleted obsolete loyalty points entries", "characterID", characterID, "count", obsoleteIDs.Size())
			}
			return true, nil
		})
}
