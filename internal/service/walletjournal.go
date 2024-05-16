package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) ListWalletJournalEntries(characterID int32) ([]*model.WalletJournalEntry, error) {
	ctx := context.Background()
	return s.r.ListWalletJournalEntries(ctx, characterID)
}

// UpdateWalletJournalEntryESI updates the wallet journal from ESI and returns the count of new entries.
func (s *Service) UpdateWalletJournalEntryESI(characterID int32) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateWalletJournalEntryESI-%d", characterID)
	x, err, _ := s.singleGroup.Do(key, func() (any, error) {
		return s.updateWalletJournalEntryESI(ctx, characterID)
	})
	hasChanged := x.(bool)
	return hasChanged, err
}

func (s *Service) updateWalletJournalEntryESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	entries, r, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletJournal(ctx, token.CharacterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received wallet journal from ESI", "entries", len(entries), "characterID", token.CharacterID)
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionWalletJournal, r)
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
	slog.Info("wallet journal", "existing", existingIDs, "entries", entries)
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

	for _, e := range newEntries {
		arg := storage.CreateWalletJournalEntryParams{
			Amount:        e.Amount,
			Balance:       e.Balance,
			ContextID:     e.ContextId,
			ContextIDType: e.ContextIdType,
			Date:          e.Date,
			Description:   e.Description,
			FirstPartyID:  e.FirstPartyId,
			ID:            e.Id,
			MyCharacterID: characterID,
			RefType:       e.RefType,
			Reason:        e.Reason,
			SecondPartyID: e.SecondPartyId,
			Tax:           e.Tax,
			TaxReceiverID: e.TaxReceiverId,
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
