package service

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) ListWalletJournalEntries(characterID int32) ([]*model.CharacterWalletJournalEntry, error) {
	ctx := context.Background()
	return s.r.ListWalletJournalEntries(ctx, characterID)
}

// TODO: Add ability to fetch more then one page from ESI

// updateWalletJournalEntryESI updates the wallet journal from ESI and reports wether it has changed.
func (s *Service) updateWalletJournalEntryESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	entries, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletJournal(ctx, token.CharacterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received wallet journal from ESI", "entries", len(entries), "characterID", token.CharacterID)
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionWalletJournal, entries)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	ii, err := s.r.ListWalletJournalEntryIDs(ctx, characterID)
	if err != nil {
		return false, err
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
		slog.Info("No new wallet journal entries", "characterID", token.CharacterID)
		return false, nil
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
	_, err = s.AddMissingEveEntities(ctx, ids.ToSlice())

	for _, o := range newEntries {
		arg := storage.CreateWalletJournalEntryParams{
			Amount:        o.Amount,
			Balance:       o.Balance,
			ContextID:     o.ContextId,
			ContextIDType: o.ContextIdType,
			Date:          o.Date,
			Description:   o.Description,
			FirstPartyID:  o.FirstPartyId,
			ID:            o.Id,
			MyCharacterID: characterID,
			RefType:       o.RefType,
			Reason:        o.Reason,
			SecondPartyID: o.SecondPartyId,
			Tax:           o.Tax,
			TaxReceiverID: o.TaxReceiverId,
		}
		if err != nil {
			return false, err
		}
		if err := s.r.CreateWalletJournalEntry(ctx, arg); err != nil {
			return false, err
		}
	}
	slog.Info("Stored new wallet journal entries", "characterID", token.CharacterID, "entries", len(newEntries))
	return true, nil
}
