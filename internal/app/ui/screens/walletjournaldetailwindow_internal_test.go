package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestShowCharacterWalletJournalEntryWindowAsync(t *testing.T) {
	t.Run("can show entry", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			FirstPartyID:  optional.New(factory.CreateEveEntityCharacter().ID),
			SecondPartyID: optional.New(factory.CreateEveEntityCharacter().ID),
			Tax:           optional.New(10.0),
			TaxReceiverID: optional.New(factory.CreateEveEntityCorporation().ID),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show entry with related entity", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			ContextID:     optional.New(factory.CreateEveEntityCharacter().ID),
			ContextIDType: optional.New("character_id"),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show entry with related contract", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID:   c.ID,
			ContextID:     optional.New(o.ContractID),
			ContextIDType: optional.New("contract_id"),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("can show entry with related market transaction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		e := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			ContextID:     optional.New[int64](42),
			ContextIDType: optional.New("market_transaction_id"),
		})
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, e.CharacterID, "Bruce", e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", e.CharacterID, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when entry can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, c.ID, "Bruce", 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", c.ID, 42))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d", 7, 42))
		// when
		ShowCharacterWalletJournalEntryWindowAsync(u, 7, "Bruce", 42)
		// then
		assert.Nil(t, w.Content())
	})
}

func TestShowCorporationWalletJournalEntryWindowAsync(t *testing.T) {
	t.Run("can show entry", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		e := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			Division:      app.Division1,
			FirstPartyID:  optional.New(factory.CreateEveEntityCharacter().ID),
			SecondPartyID: optional.New(factory.CreateEveEntityCharacter().ID),
			Tax:           optional.New(10.0),
			TaxReceiverID: optional.New(factory.CreateEveEntityCorporation().ID),
		})
		// when
		ShowCorporationWalletJournalEntryWindowAsync(u, e.CorporationID, "Wayne Enterprises", app.Division1, e.RefID)
		// then
		w, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d-%d", e.CorporationID, app.Division1, e.RefID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when entry can not be loaded", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.SectionCorporationWalletJournal1)
		// when
		ShowCorporationWalletJournalEntryWindowAsync(u, c.ID, "Wayne Enterprises", app.Division1, 42)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d-%d", c.ID, app.Division1, 42))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("walletjournalentry-%d-%d-%d", 7, app.Division1, 42))
		// when
		ShowCorporationWalletJournalEntryWindowAsync(u, 7, "Wayne Enterprises", app.Division1, 42)
		// then
		assert.Nil(t, w.Content())
	})
}
