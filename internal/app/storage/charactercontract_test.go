package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterContract(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		issuer := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCharacter.Corporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		arg := storage.CreateCharacterContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
		}
		// when
		id, err := st.CreateCharacterContract(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterContract(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, id, o.ID)
				assert.Equal(t, issuer, o.Issuer)
				assert.Equal(t, dateExpired, o.DateExpired)
				assert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
				assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		issuer := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		issuerCorporation := c.EveCharacter.Corporation
		dateExpired := time.Now().Add(12 * time.Hour).UTC()
		dateIssued := time.Now().UTC()
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		arg := storage.CreateCharacterContractParams{
			Availability:        app.ContractAvailabilityPrivate,
			CharacterID:         c.ID,
			ContractID:          42,
			DateExpired:         dateExpired,
			DateIssued:          dateIssued,
			IssuerCorporationID: issuerCorporation.ID,
			IssuerID:            issuer.ID,
			Status:              app.ContractStatusOutstanding,
			Type:                app.ContractTypeCourier,
			EndLocationID:       endLocation.ID,
			StartLocationID:     startLocation.ID,
		}
		// when
		id, err := st.CreateCharacterContract(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterContract(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, id, o.ID)
				assert.Equal(t, issuer, o.Issuer)
				assert.Equal(t, dateExpired, o.DateExpired)
				assert.Equal(t, app.ContractAvailabilityPrivate, o.Availability)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
				assert.Equal(t, endLocation.ToShort(), o.EndLocation)
				assert.Equal(t, startLocation.ToShort(), o.StartLocation)
				assert.Equal(t, endLocation.SolarSystem.ID, o.EndSolarSystem.ID)
				assert.Equal(t, endLocation.SolarSystem.Name, o.EndSolarSystem.Name)
				assert.Equal(t, startLocation.SolarSystem.ID, o.StartSolarSystem.ID)
				assert.Equal(t, startLocation.SolarSystem.Name, o.StartSolarSystem.Name)
				assert.WithinDuration(t, time.Now().UTC(), o.UpdatedAt, 5*time.Second)
			}
		}
	})
	t.Run("can update contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		dateAccepted := time.Now().UTC()
		dateCompleted := time.Now().UTC()
		arg2 := storage.UpdateCharacterContractParams{
			CharacterID:   o1.CharacterID,
			ContractID:    o1.ContractID,
			DateAccepted:  dateAccepted,
			DateCompleted: dateCompleted,
			Status:        app.ContractStatusFinished,
		}
		// when
		err := st.UpdateCharacterContract(ctx, arg2)
		// then
		if assert.NoError(t, err) {
			o2, err := st.GetCharacterContract(ctx, o1.CharacterID, o1.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.ContractStatusFinished, o2.Status)
				assert.Equal(t, optional.New(dateAccepted), o2.DateAccepted)
				assert.Equal(t, optional.New(dateCompleted), o2.DateCompleted)
				assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
			}
		}
	})
	t.Run("can update notified", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			UpdatedAt: time.Now().UTC().Add(-5 * time.Second),
		})
		// when
		err := st.UpdateCharacterContractNotified(ctx, o1.ID, app.ContractStatusInProgress)
		// then
		if assert.NoError(t, err) {
			o2, err := st.GetCharacterContract(ctx, o1.CharacterID, o1.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, app.ContractStatusInProgress, o2.StatusNotified)
				assert.Less(t, o1.UpdatedAt, o2.UpdatedAt)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		e1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		// when
		ids, err := st.ListCharacterContractIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Of(ids...)
			want := set.Of([]int32{e1.ContractID, e2.ContractID, e3.ContractID}...)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list contracts for multiple characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character1 := factory.CreateCharacter()
		c1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character1.ID})
		c2 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character1.ID})
		character2 := factory.CreateCharacter()
		c3 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character2.ID})
		// when
		oo, err := st.ListAllCharacterContracts(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(c1.ID, c2.ID, c3.ID)
			got := set.Of(xslices.Map(oo, func(x *app.CharacterContract) int64 {
				return x.ID
			})...)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list existing contracts for notify", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacter()
		o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: c.ID})
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c.ID,
			UpdatedAt:   now.Add(-12 * time.Hour),
		})
		// when
		oo, err := st.ListCharacterContractsForNotify(ctx, c.ID, now.Add(-10*time.Hour))
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 1)
			assert.Equal(t, o.ID, oo[0].ID)
		}
	})
}

func TestCharacterContractBid(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		bidder := factory.CreateEveEntityCharacter()
		const (
			amount = 123.45
			bidID  = 12345
		)
		dateBid := time.Now().UTC()
		arg := storage.CreateCharacterContractBidParams{
			ContractID: c.ID,
			Amount:     amount,
			BidID:      bidID,
			BidderID:   bidder.ID,
			DateBid:    dateBid,
		}
		// when
		err := st.CreateCharacterContractBid(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterContractBid(ctx, c.ID, bidID)
			if assert.NoError(t, err) {
				assert.InDelta(t, amount, o.Amount, 0.1)
				assert.Equal(t, bidder, o.Bidder)
				assert.Equal(t, dateBid, o.DateBid)
			}
		}
	})
	t.Run("can list existing bids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		b1 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		b2 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		// when
		oo, err := st.ListCharacterContractBids(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterContractBid) int32 {
				return x.BidID
			}))
			want := set.Of(b1.BidID, b2.BidID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list bid IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		b1 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		b2 := factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		// when
		got, err := st.ListCharacterContractBidIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(b1.BidID, b2.BidID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestCharacterContractItem(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		et := factory.CreateEveType()
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  c.ID,
			IsIncluded:  true,
			IsSingleton: true,
			Quantity:    7,
			RawQuantity: -5,
			RecordID:    42,
			TypeID:      et.ID,
		}
		// when
		err := st.CreateCharacterContractItem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterContractItem(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsIncluded)
				assert.True(t, o.IsSingleton)
				assert.Equal(t, 7, o.Quantity)
				assert.Equal(t, -5, o.RawQuantity)
				assert.Equal(t, et, o.Type)
			}
		}
	})
	t.Run("can list existing items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterContract()
		i1 := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		i2 := factory.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{ContractID: c.ID})
		// when
		oo, err := st.ListCharacterContractItems(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterContractItem) int64 {
				return x.RecordID
			}))
			want := set.Of(i1.RecordID, i2.RecordID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
