package service

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) ListWalletTransactions(characterID int32) ([]*model.CharacterWalletTransaction, error) {
	ctx := context.Background()
	return s.r.ListCharacterWalletTransactions(ctx, characterID)
}

// TODO: Add ability to fetch more then one page from ESI

// updateWalletTransactionESI updates the wallet journal from ESI and reports wether it has changed.
func (s *Service) updateWalletTransactionESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	entries, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletTransactions(ctx, token.CharacterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received wallet transactions from ESI", "entries", len(entries), "characterID", token.CharacterID)
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionWalletTransactions, entries)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	ii, err := s.r.ListCharacterWalletTransactionIDs(ctx, characterID)
	if err != nil {
		return false, err
	}
	existingIDs := set.NewFromSlice(ii)
	var newEntries []esi.GetCharactersCharacterIdWalletTransactions200Ok
	for _, e := range entries {
		if existingIDs.Has(e.TransactionId) {
			continue
		}
		newEntries = append(newEntries, e)
	}
	slog.Debug("wallet transaction", "existing", existingIDs, "entries", entries)
	if len(newEntries) == 0 {
		slog.Info("No new wallet transactions", "characterID", token.CharacterID)
		return false, nil
	}
	ids := set.New[int32]()
	for _, e := range newEntries {
		if e.ClientId != 0 {
			ids.Add(e.ClientId)
		}
	}
	_, err = s.AddMissingEveEntities(ctx, ids.ToSlice())
	if err != nil {
		return false, err
	}

	for _, o := range newEntries {
		_, err = s.getOrCreateEveTypeESI(ctx, o.TypeId)
		if err != nil {
			return false, err
		}
		_, err = s.getOrCreateLocationESI(ctx, o.LocationId)
		if err != nil {
			return false, err
		}
		arg := storage.CreateCharacterWalletTransactionParams{
			ClientID:      o.ClientId,
			Date:          o.Date,
			EveTypeID:     o.TypeId,
			IsBuy:         o.IsBuy,
			IsPersonal:    o.IsPersonal,
			JournalRefID:  o.JournalRefId,
			LocationID:    o.LocationId,
			CharacterID:   characterID,
			Quantity:      o.Quantity,
			TransactionID: o.TransactionId,
			UnitPrice:     o.UnitPrice,
		}
		if err := s.r.CreateCharacterWalletTransaction(ctx, arg); err != nil {
			return false, err
		}
	}
	slog.Info("Stored new wallet transactions", "characterID", token.CharacterID, "entries", len(newEntries))
	return true, nil
}
