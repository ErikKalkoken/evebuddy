package corporationservice

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
			if _, err := s.eus.AddMissingEntities(ctx, incoming); err != nil {
				return false, err
			}
			return s.st.ReplaceCorporationMembers(ctx, arg.corporationID, incoming)
		})
}
