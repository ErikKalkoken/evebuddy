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

func (s *CharacterService) ListCharacterContractItems(ctx context.Context, contractID int64) ([]*app.CharacterContractItem, error) {
	return s.st.ListCharacterContractItems(ctx, contractID)
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
			// fetch any missing eve entity
			ids := set.New[int32]()
			for _, o := range oo {
				ids.Add(o.IssuerId)
				ids.Add(o.IssuerCorporationId)
				if o.AcceptorId != 0 {
					ids.Add(o.AcceptorId)
				}
				if o.AssigneeId != 0 {
					ids.Add(o.AssigneeId)
				}
			}
			_, err := s.EveUniverseService.AddMissingEveEntities(ctx, ids.ToSlice())
			if err != nil {
				return err
			}
			// identify new contracts
			ii, err := s.st.ListCharacterContractIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ii)
			var existingObjs, newObjs []esi.GetCharactersCharacterIdContracts200Ok
			for _, o := range oo {
				if existingIDs.Contains(o.ContractId) {
					existingObjs = append(existingObjs, o)
				} else {
					newObjs = append(newObjs, o)
				}
			}
			slog.Debug("contracts", "existing", existingIDs, "entries", oo)
			// create new entries
			if len(newObjs) > 0 {
				for _, o := range newObjs {
					err := s.createNewContract(ctx, characterID, o)
					if err != nil {
						slog.Error("create contract", "contract", o, "error", err)
						continue
					}
				}
				slog.Info("Stored new contracts", "characterID", characterID, "count", len(newObjs))
			}
			if len(existingObjs) > 0 {
				for _, o := range existingObjs {
					arg := storage.UpdateCharacterContractParams{
						AcceptorID:    o.AcceptorId,
						AssigneeID:    o.AssigneeId,
						DateAccepted:  o.DateAccepted,
						DateCompleted: o.DateCompleted,
						CharacterID:   characterID,
						ContractID:    o.ContractId,
						Status:        o.Status,
					}
					if err := s.st.UpdateCharacterContract(ctx, arg); err != nil {
						return err
					}
				}
				slog.Info("Updated contracts", "characterID", characterID, "count", len(existingObjs))
			}
			return nil
		})
}

func (s *CharacterService) createNewContract(ctx context.Context, characterID int32, o esi.GetCharactersCharacterIdContracts200Ok) error {
	if o.StartLocationId != 0 {
		_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, o.StartLocationId)
		if err != nil {
			return err
		}
	}
	if o.EndLocationId != 0 {
		_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, o.EndLocationId)
		if err != nil {
			return err
		}
	}
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
	id, err := s.st.CreateCharacterContract(ctx, arg)
	if err != nil {
		return err
	}
	items, _, err := s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContractsContractIdItems(ctx, characterID, o.ContractId, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		et, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, it.TypeId)
		if err != nil {
			return err
		}
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  id,
			IsIncluded:  it.IsIncluded,
			IsSingleton: it.IsSingleton,
			Quantity:    it.Quantity,
			RawQuantity: it.RawQuantity,
			RecordID:    it.RecordId,
			TypeID:      et.ID,
		}
		if err := s.st.CreateCharacterContractItem(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}
