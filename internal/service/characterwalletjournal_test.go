package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestUpdateWalletJournalEntryESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should create new entry from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		firstParty := factory.CreateEveEntityCharacter(model.EveEntity{ID: 2112625428})
		secondParty := factory.CreateEveEntityCorporation(model.EveEntity{ID: 1000132})
		data := `[
			{
			  "amount": -100000,
			  "balance": 500000.4316,
			  "context_id": 4,
			  "context_id_type": "contract_id",
			  "date": "2018-02-23T14:31:32Z",
			  "description": "Contract Deposit",
			  "first_party_id": 2112625428,
			  "id": 89,
			  "ref_type": "contract_deposit",
			  "second_party_id": 1000132
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v6/characters/%d/wallet/journal/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterWalletJournalEntryESI(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e, err := r.GetCharacterWalletJournalEntry(ctx, c.ID, 89)
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
			ids, err := r.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 1)
			}
		}
	})
	t.Run("should add new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(model.EveEntity{ID: 1000132})
		data := `[
			{
			  "amount": -100000,
			  "balance": 500000.4316,
			  "context_id": 4,
			  "context_id_type": "contract_id",
			  "date": "2018-02-23T14:31:32Z",
			  "description": "Contract Deposit",
			  "first_party_id": 2112625428,
			  "id": 89,
			  "ref_type": "contract_deposit",
			  "second_party_id": 1000132
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v6/characters/%d/wallet/journal/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterWalletJournalEntryESI(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			e2, err := r.GetCharacterWalletJournalEntry(ctx, c.ID, 89)
			if assert.NoError(t, err) {
				assert.Equal(t, "Contract Deposit", e2.Description)
			}
			ids, err := r.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 2)
			}
		}
	})
	t.Run("should ignore existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID: c.ID,
			RefID:       89,
			Description: "existing",
		})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 2112625428})
		factory.CreateEveEntityCorporation(model.EveEntity{ID: 1000132})
		data := `[
			{
			  "amount": -100000,
			  "balance": 500000.4316,
			  "context_id": 4,
			  "context_id_type": "contract_id",
			  "date": "2018-02-23T14:31:32Z",
			  "description": "Contract Deposit",
			  "first_party_id": 2112625428,
			  "id": 89,
			  "ref_type": "contract_deposit",
			  "second_party_id": 1000132
			}
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v6/characters/%d/wallet/journal/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		_, err := s.updateCharacterWalletJournalEntryESI(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			e2, err := r.GetCharacterWalletJournalEntry(ctx, c.ID, 89)
			if assert.NoError(t, err) {
				assert.Equal(t, "existing", e2.Description)
			}
			ids, err := r.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids, 1)
			}
		}
	})

}

func TestListWalletJournalEntries(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	s := NewService(r)
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		// when
		ee, err := s.ListCharacterWalletJournalEntries(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ee, 3)
		}
	})
}
