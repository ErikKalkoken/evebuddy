package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CharacterService) GetWalletJournalEntry(ctx context.Context, characterID int64, refID int64) (*app.CharacterWalletJournalEntry, error) {
	return s.st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
		CharacterID: characterID,
		RefID:       refID,
	})
}

func (s *CharacterService) ListWalletJournalEntries(ctx context.Context, characterID int64) ([]*app.CharacterWalletJournalEntry, error) {
	return s.st.ListCharacterWalletJournalEntries(ctx, characterID)
}

// updateWalletJournalEntryESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletJournalEntryESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterWalletJournal {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWalletJournal")
			cacheKey := fmt.Sprintf("wallet-journal-last-id-%d", characterID)
			lastID, found := s.cache.GetInt64(cacheKey)
			checkLastID := found && !arg.ForceUpdate
			entries, err := xgoesi.FetchPagesWithStop(
				func(page int32) ([]esi.CharactersCharacterIdWalletJournalGetInner, *http.Response, error) {
					return s.esiClient.WalletAPI.GetCharactersCharacterIdWalletJournal(ctx, characterID).Page(page).Execute()
				}, func(x esi.CharactersCharacterIdWalletJournalGetInner) bool {
					return checkLastID && x.Id <= lastID
				})
			if err != nil {
				return false, err
			}
			ids := xslices.Map(entries, func(x esi.CharactersCharacterIdWalletJournalGetInner) int64 {
				return x.Id
			})
			if len(ids) > 0 {
				s.cache.SetInt64(cacheKey, slices.Max(ids), 0)
			}
			slog.Debug("Received wallet journal from ESI", "entries", len(entries), "characterID", characterID)
			return entries, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			entries := data.([]esi.CharactersCharacterIdWalletJournalGetInner)
			existingIDs, err := s.st.ListCharacterWalletJournalEntryIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			var newEntries []esi.CharactersCharacterIdWalletJournalGetInner
			for _, o := range entries {
				if existingIDs.Contains(o.Id) {
					continue
				}
				newEntries = append(newEntries, o)
			}
			slog.Debug("wallet journal", "existing", existingIDs, "entries", entries)
			if len(newEntries) == 0 {
				slog.Info("No new wallet journal entries", "characterID", characterID)
				return true, nil
			}

			var ids set.Set[int64]
			for _, o := range newEntries {
				for _, x := range []*int64{o.FirstPartyId, o.SecondPartyId, o.TaxReceiverId} {
					if x != nil {
						ids.Add(*x)
					}
				}
			}
			_, err = s.eus.AddMissingEntities(ctx, ids)
			if err != nil {
				return false, err
			}
			for _, o := range newEntries {
				arg := storage.CreateCharacterWalletJournalEntryParams{
					Amount:        optional.FromPtr(o.Amount),
					Balance:       optional.FromPtr(o.Balance),
					ContextID:     optional.FromPtr(o.ContextId),
					ContextIDType: optional.FromPtr(o.ContextIdType),
					Date:          o.Date,
					Description:   o.Description,
					FirstPartyID:  optional.FromPtr(o.FirstPartyId),
					RefID:         o.Id,
					CharacterID:   characterID,
					RefType:       o.RefType,
					Reason:        optional.FromPtr(o.Reason),
					SecondPartyID: optional.FromPtr(o.SecondPartyId),
					Tax:           optional.FromPtr(o.Tax),
					TaxReceiverID: optional.FromPtr(o.TaxReceiverId),
				}
				if err := s.st.CreateCharacterWalletJournalEntry(ctx, arg); err != nil {
					return false, err
				}
			}

			slog.Info("Stored new wallet journal entries", "characterID", characterID, "entries", len(newEntries))
			return true, nil
		})
}

const (
	maxTransactionsPerPage = 2_500 // maximum objects returned per page
)

func (s *CharacterService) GetWalletTransactions(ctx context.Context, characterID int64, transactionID int64) (*app.CharacterWalletTransaction, error) {
	return s.st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
		CharacterID:   characterID,
		TransactionID: transactionID,
	})
}

func (s *CharacterService) ListWalletTransactions(ctx context.Context, characterID int64) ([]*app.CharacterWalletTransaction, error) {
	return s.st.ListCharacterWalletTransactions(ctx, characterID)
}

// updateWalletTransactionESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletTransactionESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterWalletTransactions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			cacheKey := fmt.Sprintf("wallet-transactions-last-id-%d", characterID)
			lastID, found := s.cache.GetInt64(cacheKey)
			checkLastID := found && !arg.ForceUpdate
			transactions, err := s.fetchWalletTransactionsESI(ctx, characterID, arg.MaxWalletTransactions, lastID, checkLastID)
			if err != nil {
				return false, err
			}
			ids := xslices.Map(transactions, func(x esi.CharactersCharacterIdWalletTransactionsGetInner) int64 {
				return x.TransactionId
			})
			if len(ids) > 0 {
				s.cache.SetInt64(cacheKey, slices.Max(ids), 0)
			}
			return transactions, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			transactions := data.([]esi.CharactersCharacterIdWalletTransactionsGetInner)
			existingIDs, err := s.st.ListCharacterWalletTransactionIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			var newEntries []esi.CharactersCharacterIdWalletTransactionsGetInner
			for _, e := range transactions {
				if existingIDs.Contains(e.TransactionId) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)
			if len(newEntries) == 0 {
				slog.Info("No new wallet transactions", "characterID", characterID)
				return true, nil
			}

			var entityIDs, typeIDs set.Set[int64]
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
				return false, err
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
					return false, err
				}
			}

			slog.Info("Stored new wallet transactions", "characterID", characterID, "entries", len(newEntries))
			return true, nil
		})
}

// fetchWalletTransactionsESI fetches wallet transactions from ESI with paging and returns them.
func (s *CharacterService) fetchWalletTransactionsESI(ctx context.Context, characterID int64, maxTransactions int, lastID int64, checkLastID bool) ([]esi.CharactersCharacterIdWalletTransactionsGetInner, error) {
	var transactions []esi.CharactersCharacterIdWalletTransactionsGetInner
	var fromID int64
	ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWalletTransactions")
	for {
		var oo []esi.CharactersCharacterIdWalletTransactionsGetInner
		var err error
		if fromID > 0 {
			oo, _, err = s.esiClient.WalletAPI.GetCharactersCharacterIdWalletTransactions(ctx, characterID).FromId(fromID).Execute()
		} else {
			oo, _, err = s.esiClient.WalletAPI.GetCharactersCharacterIdWalletTransactions(ctx, characterID).Execute()
		}
		if err != nil {
			return nil, err
		}
		transactions = slices.Concat(transactions, oo)
		isLimitExceeded := (maxTransactions != 0 && len(transactions)+maxTransactionsPerPage > maxTransactions)
		if len(oo) < maxTransactionsPerPage || isLimitExceeded {
			break
		}
		ids := xslices.Map(oo, func(x esi.CharactersCharacterIdWalletTransactionsGetInner) int64 {
			return x.TransactionId
		})
		if checkLastID && slices.Contains(ids, lastID) {
			break // stop reading once a known ID is found
		}
		fromID = slices.Min(ids)
	}
	slog.Debug("Received wallet transactions", "characterID", characterID, "count", len(transactions))
	return transactions, nil
}

func (s *CharacterService) updateWalletBalanceESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterWalletBalance {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWallet")
			balance, _, err := s.esiClient.WalletAPI.GetCharactersCharacterIdWallet(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			return balance, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			balance := data.(float64)
			if err := s.st.UpdateCharacterWalletBalance(ctx, characterID, optional.New(balance)); err != nil {
				return false, err
			}
			return true, nil
		})
}
