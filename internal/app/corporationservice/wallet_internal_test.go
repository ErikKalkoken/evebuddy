package corporationservice

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateWalletBalancesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new entries from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		data := []map[string]any{
			{
				"balance":  123.45,
				"division": 1,
			},
			{
				"balance":  223.45,
				"division": 2,
			},
			{
				"balance":  323.45,
				"division": 3,
			},
			{
				"balance":  423.45,
				"division": 4,
			},
			{
				"balance":  523.45,
				"division": 5,
			},
			{
				"balance":  623.45,
				"division": 6,
			},
			{
				"balance":  723.45,
				"division": 7,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/`,
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		changed, err := s.updateWalletBalancesESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCorporationWalletBalances(ctx, c.ID)
			if assert.NoError(t, err) {
				got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationWalletBalance) (int32, float64) {
					return x.DivisionID, x.Balance
				}))
				want := map[int32]float64{
					1: 123.45,
					2: 223.45,
					3: 323.45,
					4: 423.45,
					5: 523.45,
					6: 623.45,
					7: 723.45,
				}
				assert.Equal(t, want, got)
			}
		}
	})
	t.Run("should update existing balances", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		for id := range 7 {
			err := st.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
				CorporationID: c.ID,
				DivisionID:    int32(id + 1),
			})
			assert.NoError(t, err)
		}
		data := []map[string]any{
			{
				"balance":  123.45,
				"division": 1,
			},
			{
				"balance":  223.45,
				"division": 2,
			},
			{
				"balance":  323.45,
				"division": 3,
			},
			{
				"balance":  423.45,
				"division": 4,
			},
			{
				"balance":  523.45,
				"division": 5,
			},
			{
				"balance":  623.45,
				"division": 6,
			},
			{
				"balance":  723.45,
				"division": 7,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/`,
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		changed, err := s.updateWalletBalancesESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletBalances,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCorporationWalletBalances(ctx, c.ID)
			if assert.NoError(t, err) {
				got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationWalletBalance) (int32, float64) {
					return x.DivisionID, x.Balance
				}))
				want := map[int32]float64{
					1: 123.45,
					2: 223.45,
					3: 323.45,
					4: 423.45,
					5: 523.45,
					6: 623.45,
					7: 723.45,
				}
				assert.Equal(t, want, got)
			}
		}
	})
}

func TestUpdateWalletJournalEntryESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new entry from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		firstParty := factory.CreateEveEntityCorporation(app.EveEntity{ID: 2112625428})
		secondParty := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		data := []map[string]any{
			{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "Contract Deposit",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/\d+/journal/`,
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateWalletJournalESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
				CorporationID: c.ID,
				DivisionID:    1,
				RefID:         89,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, -100000.0, e.Amount)
				assert.Equal(t, 500000.4316, e.Balance)
				assert.Equal(t, int64(4), e.ContextID)
				assert.Equal(t, "contract_id", e.ContextIDType)
				assert.Equal(t, time.Date(2018, 02, 23, 14, 31, 32, 0, time.UTC), e.Date)
				assert.Equal(t, "Contract Deposit", e.Description)
				assert.Equal(t, firstParty.ID, e.FirstParty.ID)
				assert.Equal(t, "contract_deposit", e.RefType)
				assert.Equal(t, secondParty.ID, e.SecondParty.ID)
			}
			ids, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
	t.Run("should add new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{CorporationID: c.ID})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		data := []map[string]any{
			{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "Contract Deposit",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/\d+/journal/`,
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateWalletJournalESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e2, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
				CorporationID: c.ID,
				DivisionID:    1,
				RefID:         89,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, "Contract Deposit", e2.Description)
			}
			ids, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 2, ids.Size())
			}
		}
	})
	t.Run("should ignore existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
			RefID:         89,
			Description:   "existing",
		})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		data := []map[string]any{
			{
				"amount":          -100000,
				"balance":         500000.4316,
				"context_id":      4,
				"context_id_type": "contract_id",
				"date":            "2018-02-23T14:31:32Z",
				"description":     "Contract Deposit",
				"first_party_id":  2112625428,
				"id":              89,
				"ref_type":        "contract_deposit",
				"second_party_id": 1000132,
			}}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/\d+/journal/`,
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		_, err := s.updateWalletJournalESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		// then
		if assert.NoError(t, err) {
			e2, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
				CorporationID: c.ID,
				DivisionID:    1,
				RefID:         89,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, "existing", e2.Description)
			}
			ids, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000132})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/corporations/%d/wallets/%d/journal/", c.ID, 1),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"amount":          -100000,
					"balance":         500000.4316,
					"context_id":      4,
					"context_id_type": "contract_id",
					"date":            "2018-02-23T14:31:32Z",
					"description":     "First",
					"first_party_id":  2112625428,
					"id":              89,
					"ref_type":        "contract_deposit",
					"second_party_id": 1000132,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/corporations/%d/wallets/%d/journal/?page=2", c.ID, 1),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"amount":          -110000,
					"balance":         500000.4316,
					"context_id":      4,
					"context_id_type": "contract_id",
					"date":            "2018-02-23T15:31:32Z",
					"description":     "Second",
					"first_party_id":  2112625428,
					"id":              90,
					"ref_type":        "contract_deposit",
					"second_party_id": 1000132,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}),
		)
		// when
		changed, err := s.updateWalletJournalESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletJournal1,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				if assert.Equal(t, 2, ids.Size()) {
					x1, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
						CorporationID: c.ID,
						DivisionID:    1,
						RefID:         89,
					})
					if assert.NoError(t, err) {
						assert.Equal(t, "First", x1.Description)
					}
					x2, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
						CorporationID: c.ID,
						DivisionID:    1,
						RefID:         90,
					})
					if assert.NoError(t, err) {
						assert.Equal(t, "Second", x2.Description)
					}
				}
			}
		}
	})
}

func TestListWalletJournalEntries(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		e2 := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletJournalEntry()
		// when
		oo, err := s.ListWalletJournalEntries(ctx, c.ID, 1)
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CorporationWalletJournalEntry) int64 {
				return x.RefID
			})...)
			want := set.Of(e1.RefID, e2.RefID)
			xassert.EqualSet(t, want, got)
		}
	})
}

func TestUpdateWalletTransactionESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new transaction from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		client := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := []map[string]any{
			{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1234567890,
				"type_id":        587,
				"unit_price":     1.23,
			}}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/\d+/transactions/`,
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateWalletTransactionESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e, err := st.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
				CorporationID: c.ID,
				DivisionID:    1,
				TransactionID: 1234567890,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, client, e.Client)
				assert.Equal(t, time.Date(2016, 10, 24, 9, 0, 0, 0, time.UTC), e.Date)
				assert.True(t, e.IsBuy)
				assert.Equal(t, location.ID, e.Location.ID)
				assert.Equal(t, int64(67890), e.JournalRefID)
				assert.Equal(t, int32(1), e.Quantity)
				assert.Equal(t, eveType.ID, e.Type.ID)
				assert.Equal(t, 1.23, e.UnitPrice)
			}
			ids, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
	t.Run("should add new transaction", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{CorporationID: c.ID})
		client := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})
		data := []map[string]any{
			{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1234567890,
				"type_id":        587,
				"unit_price":     1.23,
			}}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/\d+/transactions/`,
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateWalletTransactionESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e, err := st.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
				CorporationID: c.ID,
				DivisionID:    1,
				TransactionID: 1234567890,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, client, e.Client)
				assert.Equal(t, time.Date(2016, 10, 24, 9, 0, 0, 0, time.UTC), e.Date)
				assert.True(t, e.IsBuy)
				assert.Equal(t, location.ID, e.Location.ID)
				assert.Equal(t, int64(67890), e.JournalRefID)
				assert.Equal(t, int32(1), e.Quantity)
				assert.Equal(t, eveType.ID, e.Type.ID)
				assert.Equal(t, 1.23, e.UnitPrice)
			}
			ids, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 2, ids.Size())
			}
		}
	})
	t.Run("should ignore when transaction already exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
			TransactionID: 1234567890,
		})
		data := []map[string]any{
			{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1234567890,
				"type_id":        587,
				"unit_price":     1.23,
			}}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/wallets/\d+/transactions/`,
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		_, err := s.updateWalletTransactionESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		// then
		if assert.NoError(t, err) {
			got, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				want := set.Of[int64](1234567890)
				xassert.EqualSet(t, want, got)
			}
		}
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60014719})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 587})

		data := make([]map[string]any, 0)
		for i := range 2500 {
			data = append(data, map[string]any{
				"client_id":      54321,
				"date":           "2016-10-24T09:00:00Z",
				"is_buy":         true,
				"is_personal":    true,
				"journal_ref_id": 67890,
				"location_id":    60014719,
				"quantity":       1,
				"transaction_id": 1000002500 - i,
				"type_id":        587,
				"unit_price":     1.23,
			})
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/corporations/%d/wallets/%d/transactions/", c.ID, 1),
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/corporations/%d/wallets/%d/transactions/?from_id=1000000001", c.ID, 1),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"client_id":      54321,
					"date":           "2016-10-24T08:00:00Z",
					"is_buy":         true,
					"is_personal":    true,
					"journal_ref_id": 67891,
					"location_id":    60014719,
					"quantity":       1,
					"transaction_id": 1000000000,
					"type_id":        587,
					"unit_price":     9.23,
				},
			}))
		// when
		_, err := s.updateWalletTransactionESI(ctx, app.CorporationSectionUpdateParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		// then
		if assert.NoError(t, err) {
			ids, err := st.ListCorporationWalletTransactionIDs(ctx, storage.CorporationDivision{
				CorporationID: c.ID,
				DivisionID:    1,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 2501, ids.Size())
			}
		}
	})
}

func TestListWalletTransactions(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		t1 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		t2 := factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletTransaction()
		// when
		tt, err := s.ListWalletTransactions(ctx, c.ID, 1)
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(tt, func(x *app.CorporationWalletTransaction) int64 {
				return x.TransactionID
			})...)
			want := set.Of(t1.TransactionID, t2.TransactionID)
			xassert.EqualSet(t, want, got)
		}
	})
}
