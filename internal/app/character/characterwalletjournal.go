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

func (s *CharacterService) ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*app.CharacterWalletJournalEntry, error) {
	return s.st.ListCharacterWalletJournalEntries(ctx, characterID)
}

// updateCharacterWalletJournalEntryESI updates the wallet journal from ESI and reports wether it has changed.
func (s *CharacterService) updateCharacterWalletJournalEntryESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionWalletJournal {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			entries, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdWalletJournal200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdWalletJournalOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletJournal(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Info("Received wallet journal from ESI", "entries", len(entries), "characterID", characterID)
			return entries, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			entries := data.([]esi.GetCharactersCharacterIdWalletJournal200Ok)
			existingIDs, err := s.st.ListCharacterWalletJournalEntryIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var newEntries []esi.GetCharactersCharacterIdWalletJournal200Ok
			for _, e := range entries {
				if existingIDs.Contains(e.Id) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet journal", "existing", existingIDs, "entries", entries)
			if len(newEntries) == 0 {
				slog.Info("No new wallet journal entries", "characterID", characterID)
				return nil
			}
			ids := set.New[int32]()
			for _, e := range newEntries {
				if e.FirstPartyId != 0 {
					ids.Add(e.FirstPartyId)
				}
				if e.SecondPartyId != 0 {
					ids.Add(e.SecondPartyId)
				}
				if e.TaxReceiverId != 0 {
					ids.Add(e.TaxReceiverId)
				}
			}
			_, err = s.EveUniverseService.AddMissingEveEntities(ctx, ids.ToSlice())
			if err != nil {
				return err
			}
			for _, o := range newEntries {
				arg := storage.CreateCharacterWalletJournalEntryParams{
					Amount:        o.Amount,
					Balance:       o.Balance,
					ContextID:     o.ContextId,
					ContextIDType: o.ContextIdType,
					Date:          o.Date,
					Description:   o.Description,
					FirstPartyID:  o.FirstPartyId,
					RefID:         o.Id,
					CharacterID:   characterID,
					RefType:       o.RefType,
					Reason:        o.Reason,
					SecondPartyID: o.SecondPartyId,
					Tax:           o.Tax,
					TaxReceiverID: o.TaxReceiverId,
				}
				if err := s.st.CreateCharacterWalletJournalEntry(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new wallet journal entries", "characterID", characterID, "entries", len(newEntries))
			return nil
		})
}
