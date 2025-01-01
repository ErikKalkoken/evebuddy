package character

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
)

func (s *CharacterService) ListCharacterContracts(ctx context.Context, characterID int32) ([]*app.CharacterContract, error) {
	return s.st.ListCharacterContracts(ctx, characterID)
}

// updateCharacterContractsESI updates the wallet journal from ESI and reports wether it has changed.
func (s *CharacterService) updateCharacterContractsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionContracts {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			oo, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdContracts200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdContractsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContracts(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Info("Received contracts from ESI", "count", len(oo), "characterID", characterID)
			return oo, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			oo := data.([]esi.GetCharactersCharacterIdContracts200Ok)
			ii, err := s.st.ListCharacterContractIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ii)
			var newEntries []esi.GetCharactersCharacterIdContracts200Ok
			for _, o := range oo {
				if existingIDs.Contains(o.ContractId) {
					continue
				}
				newEntries = append(newEntries, o)
			}
			slog.Debug("contracts", "existing", existingIDs, "entries", oo)
			if len(newEntries) == 0 {
				slog.Info("No new contracts", "characterID", characterID)
				return nil
			}
			ids := set.New[int32]()
			for _, o := range newEntries {
				ids.Add(o.IssuerId)
				ids.Add(o.IssuerCorporationId)
				if o.AcceptorId != 0 {
					ids.Add(o.AcceptorId)
				}
				if o.AssigneeId != 0 {
					ids.Add(o.AssigneeId)
				}
			}
			_, err = s.EveUniverseService.AddMissingEveEntities(ctx, ids.ToSlice())
			if err != nil {
				return err
			}
			for _, o := range newEntries {
				arg := storage.CreateCharacterContractParams{
					AcceptorID:          o.AcceptorId,
					AssigneeID:          o.AssigneeId,
					Availability:        o.Availability,
					Buyout:              o.Buyout,
					CharacterID:         characterID,
					Collateral:          o.Collateral,
					ContractID:          o.ContractId,
					DateAccepted:        o.DateAccepted,
					DateCompleted:       o.DateCompleted,
					DateExpired:         o.DateExpired,
					DateIssued:          o.DateIssued,
					DaysToComplete:      o.DaysToComplete,
					EndLocationID:       o.EndLocationId,
					ForCorporation:      o.ForCorporation,
					IssuerCorporationID: o.IssuerCorporationId,
					IssuerID:            o.IssuerId,
					Price:               o.Price,
					Reward:              o.Reward,
					StartLocationID:     o.StartLocationId,
					Status:              o.Status,
					Title:               o.Title,
					Type:                o.Type_,
					Volume:              o.Volume,
				}
				if _, err := s.st.CreateCharacterContract(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new contracts", "characterID", characterID, "count", len(newEntries))
			return nil
		})
}
