package service

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) ListCharacterWalletJournalEntries(characterID int32) ([]*model.CharacterWalletJournalEntry, error) {
	ctx := context.Background()
	return s.r.ListCharacterWalletJournalEntries(ctx, characterID)
}

// TODO: Add ability to fetch more then one page from ESI

// updateCharacterWalletJournalEntryESI updates the wallet journal from ESI and reports wether it has changed.
func (s *Service) updateCharacterWalletJournalEntryESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	entries, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletJournal(ctx, token.CharacterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received wallet journal from ESI", "entries", len(entries), "characterID", token.CharacterID)
	changed, err := s.hasCharacterSectionChanged(ctx, characterID, model.CharacterSectionWalletJournal, entries)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	ii, err := s.r.ListCharacterWalletJournalEntryIDs(ctx, characterID)
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
		if err != nil {
			return false, err
		}
		if err := s.r.CreateCharacterWalletJournalEntry(ctx, arg); err != nil {
			return false, err
		}
	}
	slog.Info("Stored new wallet journal entries", "characterID", token.CharacterID, "entries", len(newEntries))
	return true, nil
}
