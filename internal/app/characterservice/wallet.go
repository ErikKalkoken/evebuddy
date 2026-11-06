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
	"github.com/ErikKalkoken/evebuddy/internal/app/walletsync"
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
func (s *CharacterService) updateWalletJournalEntryESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterWalletJournal {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWalletJournal")
			cacheKey := fmt.Sprintf("wallet-journal-last-id-%d", characterID)
			lastID, found := s.cache.GetInt64(cacheKey)
			checkLastID := found && !arg.forceUpdate
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
			slog.Debug("wallet journal", "existing", existingIDs, "entries", entries)
			return walletsync.SyncJournal(
				ctx,
				xslices.Map(entries, func(x esi.CharactersCharacterIdWalletJournalGetInner) walletsync.ESIJournalEntry {
					return walletsync.ESIJournalEntry(x)
				}),
				existingIDs,
				func(ctx context.Context, ids set.Set[int64]) error {
					_, err := s.eus.AddMissingEntities(ctx, ids)
					return err
				},
				func(ctx context.Context, o walletsync.ESIJournalEntry) error {
					return s.st.CreateCharacterWalletJournalEntry(ctx, storage.CreateCharacterWalletJournalEntryParams{
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
					})
				},
				slog.Int64("characterID", characterID),
			)
		})
}

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
func (s *CharacterService) updateWalletTransactionESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterWalletTransactions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			cacheKey := fmt.Sprintf("wallet-transactions-last-id-%d", characterID)
			lastID, found := s.cache.GetInt64(cacheKey)
			checkLastID := found && !arg.forceUpdate
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdWalletTransactions")
			transactions, err := walletsync.FetchTransactions(
				func(fromID int64) ([]walletsync.ESITransaction, error) {
					req := s.esiClient.WalletAPI.GetCharactersCharacterIdWalletTransactions(ctx, characterID)
					if fromID > 0 {
						req = req.FromId(fromID)
					}
					oo, _, err := req.Execute()
					if err != nil {
						return nil, err
					}
					return xslices.Map(oo, func(x esi.CharactersCharacterIdWalletTransactionsGetInner) walletsync.ESITransaction {
						return walletsync.ESITransaction(x)
					}), nil
				},
				s.settings.MaxWalletTransactions(),
				lastID,
				checkLastID,
			)
			if err != nil {
				return false, err
			}
			slog.Debug("Received wallet transactions", "characterID", characterID, "count", len(transactions))
			ids := xslices.Map(transactions, func(x walletsync.ESITransaction) int64 {
				return x.TransactionId
			})
			if len(ids) > 0 {
				s.cache.SetInt64(cacheKey, slices.Max(ids), 0)
			}
			return transactions, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			transactions := data.([]walletsync.ESITransaction)
			existingIDs, err := s.st.ListCharacterWalletTransactionIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)
			return walletsync.SyncTransactions(
				ctx,
				transactions,
				existingIDs,
				func(ctx context.Context, entityIDs, locationIDs, typeIDs set.Set[int64]) error {
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
					return g.Wait()
				},
				func(ctx context.Context, o walletsync.ESITransaction) error {
					return s.st.CreateCharacterWalletTransaction(ctx, storage.CreateCharacterWalletTransactionParams{
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
					})
				},
				slog.Int64("characterID", characterID),
			)
		})
}

func (s *CharacterService) updateWalletBalanceESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterWalletBalance {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, false,
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
