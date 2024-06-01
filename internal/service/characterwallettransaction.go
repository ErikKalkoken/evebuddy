package service

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
)

const (
	maxTransactionsPerPage = 2_500 // maximum objects returned per page
)

func (s *Service) ListCharacterWalletTransactions(characterID int32) ([]*model.CharacterWalletTransaction, error) {
	ctx := context.Background()
	return s.r.ListCharacterWalletTransactions(ctx, characterID)
}

// updateCharacterWalletTransactionESI updates the wallet journal from ESI and reports wether it has changed.
func (s *Service) updateCharacterWalletTransactionESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionWalletTransactions {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			transactions, err := s.fetchWalletTransactionsESI(ctx, characterID)
			if err != nil {
				return false, err
			}
			slog.Info("Received wallet transactions from ESI", "entries", len(transactions), "characterID", characterID)
			return transactions, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			transactions := data.([]esi.GetCharactersCharacterIdWalletTransactions200Ok)
			ii, err := s.r.ListCharacterWalletTransactionIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.NewFromSlice(ii)
			var newEntries []esi.GetCharactersCharacterIdWalletTransactions200Ok
			for _, e := range transactions {
				if existingIDs.Has(e.TransactionId) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)
			if len(newEntries) == 0 {
				slog.Info("No new wallet transactions", "characterID", characterID)
				return nil
			}
			ids := set.New[int32]()
			for _, e := range newEntries {
				if e.ClientId != 0 {
					ids.Add(e.ClientId)
				}
			}
			_, err = s.AddMissingEveEntities(ctx, ids.ToSlice())
			if err != nil {
				return err
			}

			for _, o := range newEntries {
				_, err = s.GetOrCreateEveTypeESI(ctx, o.TypeId)
				if err != nil {
					return err
				}
				_, err = s.getOrCreateLocationESI(ctx, o.LocationId)
				if err != nil {
					return err
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
					return err
				}
			}
			slog.Info("Stored new wallet transactions", "characterID", characterID, "entries", len(newEntries))
			return nil
		})
}

// fetchWalletTransactionsESI fetches wallet transactions from ESI with paging and returns them.
func (s *Service) fetchWalletTransactionsESI(ctx context.Context, characterID int32) ([]esi.GetCharactersCharacterIdWalletTransactions200Ok, error) {
	var oo2 []esi.GetCharactersCharacterIdWalletTransactions200Ok
	lastID := int64(0)
	maxTransactions, err := s.Dictionary.GetIntWithFallback(model.SettingMaxWalletTransactions, model.SettingMaxWalletTransactionsDefault)
	if err != nil {
		return nil, err
	}
	for {
		var opts *esi.GetCharactersCharacterIdWalletTransactionsOpts
		if lastID > 0 {
			opts = &esi.GetCharactersCharacterIdWalletTransactionsOpts{FromId: optional.NewInt64(lastID)}
		} else {
			opts = nil
		}
		oo, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletTransactions(ctx, characterID, opts)
		if err != nil {
			return nil, err
		}
		oo2 = slices.Concat(oo2, oo)
		isLimitExceeded := (maxTransactions != 0 && len(oo2)+maxTransactionsPerPage > maxTransactions)
		if len(oo) < maxTransactionsPerPage || isLimitExceeded {
			break
		}
		ids := make([]int64, len(oo))
		for i, o := range oo {
			ids[i] = o.TransactionId
		}
		lastID = slices.Min(ids)
	}
	slog.Info("Received wallet transactions", "characterID", characterID, "count", len(oo2))
	return oo2, nil
}
