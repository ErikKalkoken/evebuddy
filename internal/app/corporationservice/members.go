package corporationservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (s *CorporationService) ListMembers(ctx context.Context, corporationID int32) ([]*app.CorporationMember, error) {
	return s.st.ListCorporationMembers(ctx, corporationID)
}

func (s *CorporationService) updateMembersESI(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCorporationMembers {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error) {
			members, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationIdMembers(ctx, arg.CorporationID, nil)
			if err != nil {
				return false, err
			}
			return members, nil
		},
		func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error {
			incoming := set.Of(data.([]int32)...)
			current, err := s.st.ListCorporationMemberIDs(ctx, arg.CorporationID)
			if err != nil {
				return err
			}
			if _, err := s.eus.AddMissingEntities(ctx, incoming); err != nil {
				return err
			}
			added := set.Difference(incoming, current)
			for characterID := range added.All() {
				err := s.st.CreateCorporationMember(ctx, storage.CorporationMemberParams{
					CorporationID: arg.CorporationID,
					CharacterID:   characterID,
				})
				if err != nil {
					return err
				}
			}
			removed := set.Difference(current, incoming)
			if err := s.st.DeleteCorporationMembers(ctx, arg.CorporationID, removed); err != nil {
				return err
			}
			slog.Info(
				"Updated corporation members",
				"corporationID", arg.CorporationID,
				"added", added.Size(),
				"removed", removed.Size(),
			)
			return nil
		})
}
