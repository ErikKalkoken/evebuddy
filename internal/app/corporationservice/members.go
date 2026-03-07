package corporationservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CorporationService) ListMembers(ctx context.Context, corporationID int64) ([]*app.CorporationMember, error) {
	return s.st.ListCorporationMembers(ctx, corporationID)
}

func (s *CorporationService) updateMembersESI(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCorporationMembers {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCorporationsCorporationIdMembers")
			members, _, err := s.esiClient.CorporationAPI.GetCorporationsCorporationIdMembers(ctx, arg.corporationID).Execute()
			if err != nil {
				return false, err
			}
			return members, nil
		},
		func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
			incoming := set.Of(data.([]int64)...)
			current, err := s.st.ListCorporationMemberIDs(ctx, arg.corporationID)
			if err != nil {
				return false, err
			}
			if current.Equal(incoming) {
				return false, nil
			}

			if _, err := s.eus.AddMissingEntities(ctx, incoming); err != nil {
				return false, err
			}
			added := set.Difference(incoming, current)
			for characterID := range added.All() {
				err := s.st.CreateCorporationMember(ctx, storage.CorporationMemberParams{
					CorporationID: arg.corporationID,
					CharacterID:   characterID,
				})
				if err != nil {
					return false, err
				}
			}
			removed := set.Difference(current, incoming)
			if err := s.st.DeleteCorporationMembers(ctx, arg.corporationID, removed); err != nil {
				return false, err
			}
			slog.Info(
				"Updated corporation members",
				"corporationID", arg.corporationID,
				"added", added.Size(),
				"removed", removed.Size(),
			)
			return true, nil
		})
}
