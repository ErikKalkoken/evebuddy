package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/contracttest"
	"github.com/ErikKalkoken/evebuddy/internal/optional"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateContractsEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	c := factory.CreateCharacter()
	factory.CreateCharacterContract(storage.CreateCharacterContractParams{
		CharacterID: c.ID,
		AcceptorID:  c.ID,
		Collateral:  optional.New(100.0),
		Type:        app.ContractTypeCourier,
		Status:      app.ContractStatusInProgress,
	})
	factory.CreateCharacterContract(storage.CreateCharacterContractParams{
		CharacterID: c.ID,
		AcceptorID:  c.ID,
		Collateral:  optional.New(200.0),
		Type:        app.ContractTypeCourier,
		Status:      app.ContractStatusInProgress,
	})
	o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
		CharacterID: c.ID,
		Type:        app.ContractTypeAuction,
		Status:      app.ContractStatusInProgress,
	})
	factory.CreateCharacterContractBid(storage.CreateContractBidParams{
		Amount:     12.3,
		BidderID:   c.ID,
		ContractID: o.ID,
	})

	// when
	err := s.updateContractsEscrow(t.Context(), c.ID)

	// then
	require.NoError(t, err)
	c2, err := st.GetCharacter(t.Context(), c.ID)
	require.NoError(t, err)
	got := c2.ContractsEscrow
	assert.Equal(t, optional.New(312.3), got)
}

func TestUpdateContractsESIShared(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	contracttest.RunUpdateContractsESITests(t, db, factory, contracttest.Owner{
		ESIPath: "characters",
		Setup: func(t *testing.T) contracttest.Fixture {
			c := factory.CreateCharacter()
			factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
			factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
			return contracttest.Fixture{
				OwnerID:             c.ID,
				IssuerID:            c.ID,
				IssuerCorporationID: c.EveCharacter.Corporation.ID,
			}
		},
		Sync: func(ctx context.Context, characterID int64) error {
			_, err := s.updateContractsESI(ctx, characterSectionUpdateParams{
				characterID: characterID,
				forceUpdate: true,
				section:     app.SectionCharacterContracts,
			})
			return err
		},
		CreateContract: func(t *testing.T, characterID int64, arg contracttest.CreateParams) *app.Contract {
			o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				CharacterID: characterID,
				Status:      arg.Status,
				Type:        arg.Type,
			})
			return &o.Contract
		},
		CreateBid: func(t *testing.T, contractPK, bidID int64, amount float64) {
			factory.CreateCharacterContractBid(storage.CreateContractBidParams{
				Amount:     amount,
				BidID:      bidID,
				ContractID: contractPK,
			})
		},
		GetContract: func(ctx context.Context, characterID, contractID int64) (*app.Contract, error) {
			o, err := st.GetCharacterContract(ctx, characterID, contractID)
			if err != nil {
				return nil, err
			}
			return &o.Contract, nil
		},
		ListContractIDs: st.ListCharacterContractIDs,
		ListItems:       st.ListCharacterContractItems,
		ListBids:        st.ListCharacterContractBids,
	})
}

func TestUpdateContractsESICalculatedValues(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	setup := func(t *testing.T) *app.Character {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"acceptor_id":           c.ID,
				"assignee_id":           0,
				"availability":          "public",
				"collateral":            100.0,
				"contract_id":           42,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"status":                "in_progress",
				"type":                  "courier",
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/contracts/42/items", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		return c
	}
	update := func(characterID int64) error {
		_, err := s.updateContractsESI(t.Context(), characterSectionUpdateParams{
			characterID: characterID,
			section:     app.SectionCharacterContracts,
		})
		return err
	}
	t.Run("should update escrow and items value when contracts changed", func(t *testing.T) {
		// given
		c := setup(t)
		// when
		err := update(c.ID)
		// then
		require.NoError(t, err)
		c2, err := st.GetCharacter(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, 100.0, c2.ContractsEscrow)
		xassert.NotEmpty(t, c2.ContractItemsValue)
	})
	t.Run("should update missing values when contracts are unchanged", func(t *testing.T) {
		// given
		c := setup(t)
		require.NoError(t, update(c.ID))
		require.NoError(t, st.UpdateCharacterContractsEscrow(t.Context(), c.ID, optional.Optional[float64]{}))
		require.NoError(t, st.UpdateCharacterContractItemsValue(t.Context(), c.ID, optional.Optional[float64]{}))
		// when
		err := update(c.ID)
		// then
		require.NoError(t, err)
		c2, err := st.GetCharacter(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, 100.0, c2.ContractsEscrow)
		xassert.NotEmpty(t, c2.ContractItemsValue)
	})
}
