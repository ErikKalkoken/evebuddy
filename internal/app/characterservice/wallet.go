package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

func (s *CharacterService) GetWalletJournalEntry(ctx context.Context, characterID int32, refID int64) (*app.CharacterWalletJournalEntry, error) {
	return s.st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
		CharacterID: characterID,
		RefID:       refID,
	})
}

func (s *CharacterService) ListWalletJournalEntries(ctx context.Context, characterID int32) ([]*app.CharacterWalletJournalEntry, error) {
	return s.st.ListCharacterWalletJournalEntries(ctx, characterID)
}

// TODO: Add limit when fetching wallet journal

// updateWalletJournalEntryESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletJournalEntryESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterWalletJournal {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			entries, err := xesi.FetchWithPaging(
				s.concurrencyLimit,
				func(pageNum int) ([]esi.GetCharactersCharacterIdWalletJournal200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdWalletJournalOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletJournal(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received wallet journal from ESI", "entries", len(entries), "characterID", characterID)
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
			var ids set.Set[int32]
			for _, e := range newEntries {
				ids.Add(e.FirstPartyId, e.SecondPartyId, e.TaxReceiverId)
			}
			_, err = s.eus.AddMissingEntities(ctx, ids)
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

const (
	maxTransactionsPerPage = 2_500 // maximum objects returned per page
)

func (s *CharacterService) GetWalletTransactions(ctx context.Context, characterID int32, transactionID int64) (*app.CharacterWalletTransaction, error) {
	return s.st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
		CharacterID:   characterID,
		TransactionID: transactionID,
	})
}

func (s *CharacterService) ListWalletTransactions(ctx context.Context, characterID int32) ([]*app.CharacterWalletTransaction, error) {
	return s.st.ListCharacterWalletTransactions(ctx, characterID)
}

// updateWalletTransactionESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletTransactionESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterWalletTransactions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			transactions, err := s.fetchWalletTransactionsESI(ctx, characterID, arg.MaxWalletTransactions)
			if err != nil {
				return false, err
			}
			return transactions, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			transactions := data.([]esi.GetCharactersCharacterIdWalletTransactions200Ok)
			existingIDs, err := s.st.ListCharacterWalletTransactionIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var newEntries []esi.GetCharactersCharacterIdWalletTransactions200Ok
			for _, e := range transactions {
				if existingIDs.Contains(e.TransactionId) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)
			if len(newEntries) == 0 {
				slog.Info("No new wallet transactions", "characterID", characterID)
				return nil
			}
			var entityIDs, typeIDs set.Set[int32]
			var locationIDs set.Set[int64]
			for _, en := range newEntries {
				entityIDs.Add(en.ClientId)
				locationIDs.Add(en.LocationId)
				typeIDs.Add(en.TypeId)
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, entityIDs)
				return err
			})
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			for _, o := range newEntries {
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
				if err := s.st.CreateCharacterWalletTransaction(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new wallet transactions", "characterID", characterID, "entries", len(newEntries))
			return nil
		})
}

// fetchWalletTransactionsESI fetches wallet transactions from ESI with paging and returns them.
func (s *CharacterService) fetchWalletTransactionsESI(ctx context.Context, characterID int32, maxTransactions int) ([]esi.GetCharactersCharacterIdWalletTransactions200Ok, error) {
	var oo2 []esi.GetCharactersCharacterIdWalletTransactions200Ok
	lastID := int64(0)
	for {
		var opts *esi.GetCharactersCharacterIdWalletTransactionsOpts
		if lastID > 0 {
			opts = &esi.GetCharactersCharacterIdWalletTransactionsOpts{FromId: esioptional.NewInt64(lastID)}
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
	slog.Debug("Received wallet transactions", "characterID", characterID, "count", len(oo2))
	return oo2, nil
}
