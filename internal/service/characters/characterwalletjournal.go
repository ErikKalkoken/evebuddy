package characters

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/helper/goesi"
	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
)

func (s *Characters) ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*model.CharacterWalletJournalEntry, error) {
	return s.r.ListCharacterWalletJournalEntries(ctx, characterID)
}

// updateCharacterWalletJournalEntryESI updates the wallet journal from ESI and reports wether it has changed.
func (s *Characters) updateCharacterWalletJournalEntryESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionWalletJournal {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			entries, err := goesi.FetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdWalletJournal200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdWalletJournalOpts{
						Page: optional.NewInt32(int32(pageNum)),
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
			ii, err := s.r.ListCharacterWalletJournalEntryIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ii)
			var newEntries []esi.GetCharactersCharacterIdWalletJournal200Ok
			for _, e := range entries {
				if existingIDs.Has(e.Id) {
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
			_, err = s.EveUniverse.AddMissingEveEntities(ctx, ids.ToSlice())
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
				if err := s.r.CreateCharacterWalletJournalEntry(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new wallet journal entries", "characterID", characterID, "entries", len(newEntries))
			return nil
		})
}
