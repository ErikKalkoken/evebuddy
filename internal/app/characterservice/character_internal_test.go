package characterservice

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func NewFake(st *storage.Storage, args ...Params) *CharacterService {
	scs := statuscacheservice.New(memcache.New(), st)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		StatusCacheService: scs,
		Storage:            st,
	})
	arg := Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	}
	if len(args) > 0 {
		a := args[0]
		if a.SSOService != nil {
			arg.SSOService = a.SSOService
		}
	}
	s := New(arg)
	return s
}

func TestNoScopeDuplicates(t *testing.T) {
	assert.ElementsMatch(t, esiScopes, set.Of(esiScopes...).Slice())
}

func TestUpdateCharacterAssetsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new assets from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/assets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      true,
					"item_id":           1000000016835,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
				{
					"is_blueprint_copy": true,
					"is_singleton":      false,
					"item_id":           1000000016836,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{"1"}}))
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/assets/names/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}))
		// when
		changed, err := s.updateAssetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAssets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 2, ids.Size())
				x, err := st.GetCharacterAsset(ctx, c.ID, 1000000016835)
				if assert.NoError(t, err) {
					assert.Equal(t, eveType.ID, x.Type.ID)
					assert.Equal(t, eveType.Name, x.Type.Name)
					assert.True(t, x.IsBlueprintCopy)
					assert.True(t, x.IsSingleton)
					assert.Equal(t, "Hangar", x.LocationFlag)
					assert.Equal(t, location.ID, x.LocationID)
					assert.Equal(t, "station", x.LocationType)
					assert.Equal(t, "Awesome Name", x.Name)
					assert.Equal(t, int32(1), x.Quantity)
				}
				x, err = st.GetCharacterAsset(ctx, c.ID, 1000000016836)
				if assert.NoError(t, err) {
					assert.Equal(t, "", x.Name)
				}
			}
		}
	})
	t.Run("should remove obsolete items", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID, ItemID: 1000000019999,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/assets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      true,
					"item_id":           1000000016835,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
				{
					"is_blueprint_copy": true,
					"is_singleton":      false,
					"item_id":           1000000016836,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{"1"}}))
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/assets/names/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}))
		// when
		changed, err := s.updateAssetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAssets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int64{1000000016835, 1000000016836}, ids.Slice())
			}
		}
	})
	t.Run("should fetch multiple pages", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		eveType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 3516})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60002959})
		pages := "2"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/assets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      true,
					"item_id":           1000000016835,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/assets/?page=2", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"is_blueprint_copy": true,
					"is_singleton":      false,
					"item_id":           1000000016836,
					"location_flag":     "Hangar",
					"location_id":       60002959,
					"location_type":     "station",
					"quantity":          1,
					"type_id":           3516,
				},
			}).HeaderSet(http.Header{"X-Pages": []string{pages}}))
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/assets/names/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"item_id": 1000000016835,
					"name":    "Awesome Name",
				},
				{
					"item_id": 1000000016836,
					"name":    "None",
				},
			}))
		// when
		changed, err := s.updateAssetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAssets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterAssetIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 2, ids.Size())
				x, err := st.GetCharacterAsset(ctx, c.ID, 1000000016835)
				if assert.NoError(t, err) {
					assert.Equal(t, eveType.ID, x.Type.ID)
					assert.Equal(t, eveType.Name, x.Type.Name)
					assert.True(t, x.IsBlueprintCopy)
					assert.True(t, x.IsSingleton)
					assert.Equal(t, "Hangar", x.LocationFlag)
					assert.Equal(t, location.ID, x.LocationID)
					assert.Equal(t, "station", x.LocationType)
					assert.Equal(t, "Awesome Name", x.Name)
					assert.Equal(t, int32(1), x.Quantity)
				}
				x, err = st.GetCharacterAsset(ctx, c.ID, 1000000016836)
				if assert.NoError(t, err) {
					assert.Equal(t, "", x.Name)
				}
			}
		}
	})
}

func TestUpdateCharacterAttributesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create attributes from ESI response", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		data := map[string]int{
			"charisma":     20,
			"intelligence": 21,
			"memory":       22,
			"perception":   23,
			"willpower":    24,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/attributes/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateAttributesESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAttributes,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterAttributes(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 20, x.Charisma)
				assert.Equal(t, 21, x.Intelligence)
				assert.Equal(t, 22, x.Memory)
				assert.Equal(t, 23, x.Perception)
				assert.Equal(t, 24, x.Willpower)
			}
		}
	})
}

func TestGetAttributes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := NewFake(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetAttributes(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacterAttributes()
		// when
		x2, err := cs.GetAttributes(ctx, x1.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestUpdateContractESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new courier contract from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		contractID := int32(42)
		startLocation := factory.CreateEveLocationStructure()
		endLocation := factory.CreateEveLocationStructure()
		volume := 0.01
		contractData := []map[string]any{
			{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"contract_id":           contractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      0,
				"end_location_id":       endLocation.ID,
				"for_corporation":       true,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 1000000.01,
				"reward":                0.01,
				"start_location_id":     startLocation.ID,
				"status":                "finished",
				"type":                  "courier",
				"volume":                volume,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, contractData),
		)
		recordID := int64(123456)
		quantity := 7
		et := factory.CreateEveType()
		itemData := []map[string]any{
			{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/%d/items/", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, itemData),
		)
		// when
		changed, err := s.updateContractsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionContracts,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterContract(ctx, c.ID, contractID)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o.DateAccepted.MustValue())
				assert.Equal(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o.DateCompleted.MustValue())
				assert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
				assert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
				assert.Equal(t, startLocation.ID, o.StartLocation.ID)
				assert.Equal(t, endLocation.ID, o.EndLocation.ID)
				assert.Equal(t, app.ContractStatusFinished, o.Status)
				assert.Equal(t, app.ContractTypeCourier, o.Type)
				assert.Equal(t, volume, o.Volume)
			}
			ii, err := st.ListCharacterContractItems(ctx, o.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 1)
				assert.Equal(t, quantity, ii[0].Quantity)
				assert.Equal(t, recordID, ii[0].RecordID)
				assert.Equal(t, et, ii[0].Type)
			}
		}
	})
	t.Run("should create new auction contract from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		contractID := int32(42)
		startLocation := factory.CreateEveLocationStructure()
		buyout := 10000000000.01
		volume := 0.01
		contractData := []map[string]any{
			{
				"acceptor_id":           0,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                buyout,
				"contract_id":           contractID,
				"date_expired":          "2017-06-13T13:12:32Z",
				"date_issued":           "2017-06-05T13:12:32Z",
				"days_to_complete":      0,
				"for_corporation":       false,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"start_location_id":     startLocation.ID,
				"status":                "outstanding",
				"type":                  "auction",
				"volume":                volume,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, contractData),
		)
		recordID := int64(123456)
		quantity := 7
		et := factory.CreateEveType()
		itemData := []map[string]any{
			{
				"is_included":  true,
				"is_singleton": false,
				"quantity":     quantity,
				"record_id":    recordID,
				"type_id":      et.ID,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/%d/items/", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, itemData),
		)
		const (
			bidID  = 123456
			amount = 123.45
		)
		bidder := factory.CreateEveEntityCharacter()
		bidData := []map[string]any{
			{
				"amount":    amount,
				"bid_id":    bidID,
				"bidder_id": bidder.ID,
				"date_bid":  "2017-01-01T10:10:10Z",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/%d/bids/", c.ID, contractID),
			httpmock.NewJsonResponderOrPanic(200, bidData),
		)
		// when
		changed, err := s.updateContractsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionContracts,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterContract(ctx, c.ID, contractID)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2017, 6, 13, 13, 12, 32, 0, time.UTC), o.DateExpired)
				assert.Equal(t, time.Date(2017, 6, 5, 13, 12, 32, 0, time.UTC), o.DateIssued)
				assert.Equal(t, startLocation.ID, o.StartLocation.ID)
				assert.Equal(t, app.ContractStatusOutstanding, o.Status)
				assert.Equal(t, app.ContractTypeAuction, o.Type)
				assert.Equal(t, buyout, o.Buyout)
			}
			ii, err := st.ListCharacterContractItems(ctx, o.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 1)
				assert.Equal(t, quantity, ii[0].Quantity)
				assert.Equal(t, recordID, ii[0].RecordID)
				assert.Equal(t, et, ii[0].Type)
			}
			bb, err := st.ListCharacterContractBids(ctx, o.ID)
			if assert.NoError(t, err) {
				assert.Len(t, bb, 1)
				assert.InDelta(t, amount, bb[0].Amount, 0.1)
				assert.Equal(t, bidder, bb[0].Bidder)
				assert.Equal(t, time.Date(2017, 1, 1, 10, 10, 10, 0, time.UTC), bb[0].DateBid)
			}
		}
	})
	t.Run("should update existing contract", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusOutstanding,
			Type:         app.ContractTypeCourier,
		})
		acceptor := factory.CreateEveEntityCharacter()
		data := []map[string]any{
			{
				"acceptor_id":           acceptor.ID,
				"assignee_id":           0,
				"availability":          "public",
				"buyout":                o1.Buyout,
				"contract_id":           o1.ContractID,
				"date_accepted":         "2017-06-06T13:12:32Z",
				"date_completed":        "2017-06-07T13:12:32Z",
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete,
				"for_corporation":       true,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price,
				"reward":                o1.Reward,
				"status":                "finished",
				"type":                  "courier",
				"volume":                o1.Volume,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateContractsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionContracts,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o2, err := st.GetCharacterContract(ctx, c.ID, o1.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, acceptor, o2.Acceptor)
				assert.Equal(t, app.ContractStatusFinished, o2.Status)
				assert.Equal(t, time.Date(2017, 6, 6, 13, 12, 32, 0, time.UTC), o2.DateAccepted.MustValue())
				assert.Equal(t, time.Date(2017, 6, 7, 13, 12, 32, 0, time.UTC), o2.DateCompleted.MustValue())
				assert.Equal(t, o1.DateIssued, o2.DateIssued)
				assert.Equal(t, o1.DateExpired, o2.DateExpired)
			}
		}
	})
	t.Run("should not update unchanged contract", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		o1 := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:  c.ID,
			Availability: app.ContractAvailabilityPublic,
			Status:       app.ContractStatusOutstanding,
			Type:         app.ContractTypeCourier,
		})
		data := []map[string]any{
			{
				"availability":          "public",
				"buyout":                o1.Buyout,
				"contract_id":           o1.ContractID,
				"date_expired":          o1.DateExpired.Format(time.RFC3339),
				"date_issued":           o1.DateIssued.Format(time.RFC3339),
				"days_to_complete":      o1.DaysToComplete,
				"for_corporation":       true,
				"issuer_corporation_id": c.EveCharacter.Corporation.ID,
				"issuer_id":             c.ID,
				"price":                 o1.Price,
				"reward":                o1.Reward,
				"status":                "outstanding",
				"type":                  "courier",
				"volume":                o1.Volume,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/contracts/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateContractsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionContracts,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o2, err := st.GetCharacterContract(ctx, c.ID, o1.ContractID)
			if assert.NoError(t, err) {
				assert.Equal(t, o1.UpdatedAt, o2.UpdatedAt)
			}
		}
	})
}

func TestUpdateCharacterImplantsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new implants from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		t1 := factory.CreateEveType()
		t2 := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{t1.ID, t2.ID}))

		// when
		changed, err := s.updateImplantsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionImplants,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				got := set.Of[int32]()
				for _, o := range oo {
					got.Add(o.EveType.ID)
				}
				want := set.Of(t1.ID, t2.ID)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}

func TestUpdateCharacterIndustryJobsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new job from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.Manufacturing, x.Activity)
				assert.EqualValues(t, 1015116533326, x.BlueprintID)
				assert.EqualValues(t, 60006382, x.BlueprintLocation.ID)
				assert.EqualValues(t, 118.01, x.Cost.MustValue())
				assert.EqualValues(t, 548, x.Duration)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 56, 14, 0, time.UTC), x.EndDate)
				assert.EqualValues(t, 60006382, x.Facility.ID)
				assert.EqualValues(t, 498338451, x.Installer.ID)
				assert.EqualValues(t, 229136101, x.JobID)
				assert.EqualValues(t, 200, x.LicensedRuns.MustValue())
				assert.EqualValues(t, 60006382, x.OutputLocation.ID)
				assert.EqualValues(t, 1, x.Runs)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
				assert.EqualValues(t, 60006382, x.Station.ID)
				assert.Equal(t, app.JobReady, x.Status)
			}
		}
	})
	t.Run("should update existing job", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		blueprintType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		installer := factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID:         c.ID,
			BlueprintTypeID:     blueprintType.ID,
			InstallerID:         installer.ID,
			BlueprintLocationID: location.ID,
			OutputLocationID:    location.ID,
			FacilityID:          location.ID,
			StationID:           location.ID,
			Status:              app.JobActive,
			EndDate:             time.Now(),
		})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "delivered",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.Manufacturing, x.Activity)
				assert.EqualValues(t, 1015116533326, x.BlueprintID)
				assert.EqualValues(t, 60006382, x.BlueprintLocation.ID)
				assert.EqualValues(t, 118.01, x.Cost.MustValue())
				assert.EqualValues(t, 548, x.Duration)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 56, 14, 0, time.UTC), x.EndDate)
				assert.EqualValues(t, 60006382, x.Facility.ID)
				assert.EqualValues(t, 498338451, x.Installer.ID)
				assert.EqualValues(t, 229136101, x.JobID)
				assert.EqualValues(t, 200, x.LicensedRuns.MustValue())
				assert.EqualValues(t, 60006382, x.OutputLocation.ID)
				assert.EqualValues(t, 1, x.Runs)
				assert.Equal(t, time.Date(2014, 7, 19, 15, 47, 6, 0, time.UTC), x.StartDate)
				assert.EqualValues(t, 60006382, x.Station.ID)
				assert.Equal(t, app.JobDelivered, x.Status)
			}
		}
	})
	t.Run("should fix incorrect status", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              "2014-07-19T15:56:14Z",
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            "2014-07-19T15:47:06Z",
					"station_id":            60006382,
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {

				assert.Equal(t, app.JobReady, x.Status)
			}
		}
	})
	t.Run("should not fix status when correct", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now().Add(+3 * time.Hour)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"activity_id":           1,
					"blueprint_id":          1015116533326,
					"blueprint_location_id": 60006382,
					"blueprint_type_id":     2047,
					"cost":                  118.01,
					"duration":              548,
					"end_date":              endDate.Format("2006-01-02T15:04:05Z"),
					"facility_id":           60006382,
					"installer_id":          498338451,
					"job_id":                229136101,
					"licensed_runs":         200,
					"output_location_id":    60006382,
					"runs":                  1,
					"start_date":            startDate.Format("2006-01-02T15:04:05Z"),
					"station_id":            60006382,
					"status":                "active",
				},
			}))

		// when
		changed, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterIndustryJob(ctx, c.ID, 229136101)
			if assert.NoError(t, err) {
				assert.Equal(t, app.JobActive, x.Status)
			}
		}
	})
	t.Run("should support all activity IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2047})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 498338451})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60006382})
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now().Add(+3 * time.Hour)

		makeObj := func(jobID, activityID int32) map[string]any {
			template := map[string]any{
				"activity_id":           activityID,
				"blueprint_id":          1015116533326,
				"blueprint_location_id": 60006382,
				"blueprint_type_id":     2047,
				"cost":                  118.01,
				"duration":              548,
				"end_date":              endDate.Format("2006-01-02T15:04:05Z"),
				"facility_id":           60006382,
				"installer_id":          498338451,
				"job_id":                jobID,
				"licensed_runs":         200,
				"output_location_id":    60006382,
				"runs":                  1,
				"start_date":            startDate.Format("2006-01-02T15:04:05Z"),
				"station_id":            60006382,
				"status":                "active",
			}
			return maps.Clone(template)
		}
		objs := make([]map[string]any, 0)
		activities := []int32{
			int32(app.Manufacturing),
			int32(app.Copying),
			int32(app.Invention),
			int32(app.MaterialEfficiencyResearch),
			int32(app.TimeEfficiencyResearch),
			int32(app.Reactions1),
			int32(app.Reactions2),
		}
		for jobID, activityID := range activities {
			objs = append(objs, makeObj(int32(jobID), activityID))
		}

		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/industry/jobs/\?include_completed=true`,
			httpmock.NewJsonResponderOrPanic(200, objs))

		// when
		_, err := s.updateIndustryJobsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionIndustryJobs,
		})
		// then
		if assert.NoError(t, err) {
			for jobID, activityID := range activities {
				j, err := st.GetCharacterIndustryJob(ctx, c.ID, int32(jobID))
				if assert.NoError(t, err) {
					assert.Equal(t, activityID, int32(j.Activity))
				}
			}
		}
	})
}

func TestUpdateCharacterJumpClonesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	data := map[string]any{
		"home_location": map[string]any{
			"location_id":   1021348135816,
			"location_type": "structure",
		},
		"jump_clones": []map[string]any{
			{
				"implants":      []int{22118},
				"jump_clone_id": 12345,
				"location_id":   60003463,
				"location_type": "station",
				"name":          "Alpha",
			},
		},
	}
	t.Run("should create new clones from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 22118})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60003463})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1021348135816})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/clones/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateJumpClonesESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionJumpClones,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterJumpClone(ctx, c.ID, 12345)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(12345), o.CloneID)
				assert.Equal(t, "Alpha", o.Name)
				assert.Equal(t, int64(60003463), o.Location.ID)
				if assert.Len(t, o.Implants, 1) {
					x := o.Implants[0]
					assert.Equal(t, int32(22118), x.EveType.ID)
				}
			}
		}
	})
	t.Run("should update existing clone", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		implant1 := factory.CreateEveType(storage.CreateEveTypeParams{ID: 22118})
		implant2 := factory.CreateEveType()
		station := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60003463})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1021348135816})
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: c.ID,
			JumpCloneID: 12345,
			Implants:    []int32{implant2.ID},
			LocationID:  station.ID,
			Name:        "Bravo",
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/clones/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateJumpClonesESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionJumpClones,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterJumpClone(ctx, c.ID, 12345)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(12345), o.CloneID)
				assert.Equal(t, "Alpha", o.Name)
				assert.Equal(t, station.ID, o.Location.ID)
				if assert.Len(t, o.Implants, 1) {
					x := o.Implants[0]
					assert.Equal(t, int32(implant1.ID), x.EveType.ID)
				}
			}
		}
	})
}

func TestCharacterNextAvailableCloneJump(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := NewFake(st)
	ctx := context.Background()
	t.Run("should return time of next available jump with skill", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			LastCloneJumpAt: optional.From(now.Add(-6 * time.Hour)),
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			EveTypeID:        app.EveTypeInfomorphSynchronizing,
			ActiveSkillLevel: 3,
		})
		x, err := cs.calcNextCloneJump(ctx, c)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, now.Add(15*time.Hour), x.MustValue(), 10*time.Second)
		}
	})
	t.Run("should return time of next available jump without skill", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			LastCloneJumpAt: optional.From(now.Add(-6 * time.Hour)),
		})
		x, err := cs.calcNextCloneJump(ctx, c)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, now.Add(18*time.Hour), x.MustValue(), 10*time.Second)
		}
	})
	t.Run("should return time of next available jump without skill and never jumped before", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			LastCloneJumpAt: optional.From(time.Time{}),
		})
		x, err := cs.calcNextCloneJump(ctx, c)
		if assert.NoError(t, err) {
			assert.Equal(t, time.Time{}, x.MustValue())
		}
	})
	t.Run("should return zero time when next jump available now", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			LastCloneJumpAt: optional.From(now.Add(-20 * time.Hour)),
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			EveTypeID:        app.EveTypeInfomorphSynchronizing,
			ActiveSkillLevel: 5,
		})
		x, err := cs.calcNextCloneJump(ctx, c)
		if assert.NoError(t, err) {
			assert.Equal(t, time.Time{}, x.MustValue())
		}
	})
	t.Run("should return empty time when last jump not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterMinimal()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			EveTypeID:        app.EveTypeInfomorphSynchronizing,
			ActiveSkillLevel: 5,
		})
		x, err := cs.calcNextCloneJump(ctx, c)
		if assert.NoError(t, err) {
			assert.True(t, x.IsEmpty())
		}
	})
}

func TestCanFetchMailHeadersWithPaging(t *testing.T) {
	// given
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	var objs []esi.GetCharactersCharacterIdMail200Ok
	var mailIDs []int32
	for i := range 55 {
		id := int32(1000 - i)
		mailIDs = append(mailIDs, id)
		o := esi.GetCharactersCharacterIdMail200Ok{
			From:       90000001,
			IsRead:     true,
			Labels:     []int32{3},
			MailId:     id,
			Recipients: []esi.GetCharactersCharacterIdMailRecipient{{RecipientId: 90000002, RecipientType: "character"}},
			Subject:    fmt.Sprintf("Test Mail %d", id),
			Timestamp:  time.Now(),
		}
		objs = append(objs, o)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/v1/characters/1/mail/",
		httpmock.NewJsonResponderOrPanic(200, objs[:50]),
	)
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/v1/characters/1/mail/?last_mail_id=951",
		httpmock.NewJsonResponderOrPanic(200, objs[50:]),
	)
	// when
	mails, err := s.fetchMailHeadersESI(ctx, 1, 1000)

	// then
	if assert.NoError(t, err) {
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
		assert.Len(t, mails, 55)

		newIDs := make([]int32, 0, 55)
		for _, m := range mails {
			newIDs = append(newIDs, m.MailId)
		}
		assert.Equal(t, mailIDs, newIDs)
	}
}

func TestUpdateMailLabel(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewFake(st)
	t.Run("should create new mail labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"labels": []map[string]any{
					{
						"color":        "#660066",
						"label_id":     16,
						"name":         "PINK",
						"unread_count": 4,
					},
					{
						"color":        "#FFFFFF",
						"label_id":     32,
						"name":         "WHITE",
						"unread_count": 0,
					},
				},
				"total_unread_count": 4,
			}),
		)
		// when
		_, err := s.updateMailLabelsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionMailLabels,
		})
		// then
		if assert.NoError(t, err) {
			labels, err := st.ListCharacterMailLabelsOrdered(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, labels, 2)
			}
		}
	})
	t.Run("should update existing mail labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		l1 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     16,
			Name:        "BLACK",
			Color:       "#000000",
			UnreadCount: 99,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"labels": []map[string]any{
					{
						"color":        "#660066",
						"label_id":     16,
						"name":         "PINK",
						"unread_count": 4,
					},
					{
						"color":        "#FFFFFF",
						"label_id":     32,
						"name":         "WHITE",
						"unread_count": 0,
					},
				},
				"total_unread_count": 4,
			}),
		)
		// when
		_, err := s.updateMailLabelsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionMailLabels,
		})
		// then
		if assert.NoError(t, err) {
			l2, err := st.GetCharacterMailLabel(ctx, c.ID, l1.LabelID)
			if assert.NoError(t, err) {
				assert.Equal(t, "PINK", l2.Name)
				assert.Equal(t, "#660066", l2.Color)
				assert.Equal(t, 4, l2.UnreadCount)
			}
		}
	})
}

func TestUpdateCharacterNotificationsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new notification from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg"}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, "InsurancePayoutMsg", o.Type)
				assert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text)
				assert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
	t.Run("should add new notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg"}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
				assert.Equal(t, int64(42), o.NotificationID)
				assert.Equal(t, sender, o.Sender)
				assert.Equal(t, "InsurancePayoutMsg", o.Type)
				assert.Equal(t, "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n", o.Text)
				assert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 2, ids.Size())
			}
		}
	})
	t.Run("should update isRead for existing notification", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID:    c.ID,
			NotificationID: 42,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		sender := factory.CreateEveEntityCorporation(app.EveEntity{ID: 54321})
		data := []map[string]any{{
			"is_read":         true,
			"notification_id": 42,
			"sender_id":       sender.ID,
			"sender_type":     "corporation",
			"text":            "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n",
			"timestamp":       "2017-08-16T10:08:00Z",
			"type":            "InsurancePayoutMsg"}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/notifications/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.updateNotificationsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionNotifications,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterNotification(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.True(t, o.IsRead)
			}
			ids, err := st.ListCharacterNotificationIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 1, ids.Size())
			}
		}
	})
}

func TestListCharacterNotifications(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: string(evenotification.BillOutOfMoneyMsg)})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: string(evenotification.BillPaidCorpAllMsg)})
		factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID, Type: "alpha"})
		// when
		tt, err := s.ListNotificationsTypes(ctx, c.ID, app.GroupBills)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, tt, 2)
		}
	})
}

func TestUpdateCharacterPlanetsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should update planets from scratch (minimal)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2254})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2256})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"last_update":     "2016-11-28T16:42:51Z",
					"num_pins":        77,
					"owner_id":        c.ID,
					"planet_id":       40023691,
					"planet_type":     "plasma",
					"solar_system_id": 30000379,
					"upgrade_level":   3,
				},
			}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/planets/40023691/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{

				"links": []map[string]any{
					{
						"destination_pin_id": 1000000017022,
						"link_level":         0,
						"source_pin_id":      1000000017021,
					},
				},
				"pins": []map[string]any{
					{
						"latitude":  1.55087844973,
						"longitude": 0.717145933308,
						"pin_id":    1000000017021,
						"type_id":   2254,
					},
					{
						"latitude":  1.53360639935,
						"longitude": 0.709775584394,
						"pin_id":    1000000017022,
						"type_id":   2256,
					},
				},
				"routes": []map[string]any{
					{
						"content_type_id":    2393,
						"destination_pin_id": 1000000017030,
						"quantity":           20,
						"route_id":           4,
						"source_pin_id":      1000000017029,
					},
				},
			}))
		// when
		changed, err := s.updatePlanetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			p, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), p.LastUpdate)
				assert.Equal(t, 3, p.UpgradeLevel)
				pins, err := st.ListPlanetPins(ctx, p.ID)
				if assert.NoError(t, err) {
					got := make([]int64, 0)
					for _, x := range pins {
						got = append(got, x.ID)
					}
					assert.ElementsMatch(t, []int64{1000000017021, 1000000017022}, got)
				}
			}
		}
	})
	t.Run("should update planets from scratch (all field)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		contentType := factory.CreateEveType()
		productType := factory.CreateEveType()
		pinType := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"last_update":     "2016-11-28T16:42:51Z",
					"num_pins":        77,
					"owner_id":        c.ID,
					"planet_id":       40023691,
					"planet_type":     "plasma",
					"solar_system_id": 30000379,
					"upgrade_level":   3,
				},
			}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/planets/40023691/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"links": []map[string]any{
					{
						"destination_pin_id": 1000000017022,
						"link_level":         0,
						"source_pin_id":      1000000017021,
					},
				},
				"pins": []map[string]any{
					{
						"contents": []map[string]any{
							{
								"amount":  42,
								"type_id": contentType.ID,
							},
						},
						"expiry_time": "2024-12-04T09:39:08Z",
						"extractor_details": map[string]any{
							"cycle_time":  1800,
							"head_radius": 0.013043995015323162,
							"heads": []map[string]any{
								{
									"head_id":   0,
									"latitude":  1.7599653005599976,
									"longitude": 4.165635108947754,
								},
							},
							"product_type_id": productType.ID,
							"qty_per_cycle":   1081,
						},
						"install_time":     "2024-12-03T07:39:08Z",
						"last_cycle_start": "2024-12-03T07:39:12Z",
						"latitude":         1.7196671962738037,
						"longitude":        4.1244120597839355,
						"pin_id":           1000000017021,
						"type_id":          pinType.ID,
					},
				},
				"routes": []map[string]any{
					{
						"content_type_id":    2393,
						"destination_pin_id": 1000000017030,
						"quantity":           20,
						"route_id":           4,
						"source_pin_id":      1000000017029,
					},
				},
			}))
		// when
		changed, err := s.updatePlanetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			p, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), p.LastUpdate)
				assert.Equal(t, 3, p.UpgradeLevel)
				pins, err := st.ListPlanetPins(ctx, p.ID)
				if assert.NoError(t, err) {
					assert.Len(t, pins, 1)
					pin, err := st.GetPlanetPin(ctx, p.ID, 1000000017021)
					if assert.NoError(t, err) {
						assert.Equal(t, time.Date(2024, 12, 4, 9, 39, 8, 0, time.UTC), pin.ExpiryTime.ValueOrZero())
						assert.Equal(t, time.Date(2024, 12, 3, 7, 39, 8, 0, time.UTC), pin.InstallTime.ValueOrZero())
						assert.Equal(t, time.Date(2024, 12, 3, 7, 39, 12, 0, time.UTC), pin.LastCycleStart.ValueOrZero())
						assert.Equal(t, productType, pin.ExtractorProductType)
						assert.Equal(t, pinType, pin.Type)
					}
				}
			}
		}
	})
	t.Run("should update planets and remove obsoletes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID: c.ID,
			EvePlanetID: 40023691,
		})
		factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID: c.ID,
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2254})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2256})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"last_update":     "2016-11-28T16:42:51Z",
					"num_pins":        77,
					"owner_id":        c.ID,
					"planet_id":       40023691,
					"planet_type":     "plasma",
					"solar_system_id": 30000379,
					"upgrade_level":   3,
				},
			}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/planets/40023691/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"links": []map[string]any{
						{
							"destination_pin_id": 1000000017022,
							"link_level":         0,
							"source_pin_id":      1000000017021,
						},
					},
					"pins": []map[string]any{
						{
							"latitude":  1.55087844973,
							"longitude": 0.717145933308,
							"pin_id":    1000000017021,
							"type_id":   2254,
						},
						{
							"latitude":  1.53360639935,
							"longitude": 0.709775584394,
							"pin_id":    1000000017022,
							"type_id":   2256,
						},
					},
					"routes": []map[string]any{
						{
							"content_type_id":    2393,
							"destination_pin_id": 1000000017030,
							"quantity":           20,
							"route_id":           4,
							"source_pin_id":      1000000017029,
						},
					},
				},
			}))
		// when
		changed, err := s.updatePlanetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterPlanets(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, oo, 1)
				o, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
				if assert.NoError(t, err) {
					assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), o.LastUpdate)
					assert.Equal(t, 3, o.UpgradeLevel)
				}
			}
		}
	})
}

func TestUpdateCharacterRolesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should update roles", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/roles/`,
			httpmock.NewJsonResponderOrPanic(200, map[string][]string{
				"roles": {
					"Director",
					"Station_Manager",
				},
			}),
		)
		// when
		changed, err := s.updateRolesESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionRoles,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			got, err := st.ListCharacterRoles(ctx, c.ID)
			if assert.NoError(t, err) {
				want := set.Of(app.RoleDirector, app.RoleStationManager)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}

// TODO: Add tests for UpdateSectionIfNeeded()

func TestUpdateCharacterSectionIfChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		token := factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		section := app.SectionImplants
		hasUpdated := false
		accessToken := ""
		arg := app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				accessToken = ctx.Value(goesi.ContextAccessToken).(string)
				return "any", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.Equal(t, accessToken, token.AccessToken)
			assert.True(t, hasUpdated)
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
				assert.False(t, x.HasError())
			}
		}
	})
	t.Run("should report as changed and run update when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		section := app.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      section,
			ErrorMessage: "error",
			CompletedAt:  time.Now().Add(-5 * time.Second),
		})
		hasUpdated := false
		arg := app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				return "any", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
			x2, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should report as unchanged and not run update when data has not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		section := app.SectionImplants
		x1 := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
			CompletedAt: time.Now().Add(-5 * time.Second),
		})
		hasUpdated := false
		arg := app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg,
			func(ctx context.Context, characterID int32) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, characterID int32, data any) error {
				hasUpdated = true
				return nil
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.False(t, hasUpdated)
			x2, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
}

func TestUpdateCharacterSkillsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should update skills from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 41})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 42})
		data := map[string]any{
			"skills": []map[string]any{
				{
					"active_skill_level":   3,
					"skill_id":             41,
					"skillpoints_in_skill": 10000,
					"trained_skill_level":  4,
				},
				{
					"active_skill_level":   1,
					"skill_id":             42,
					"skillpoints_in_skill": 20000,
					"trained_skill_level":  2,
				},
			},
			"total_sp": 90000,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/skills/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateSkillsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			c2, err := st.GetCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 90000, c2.TotalSP.ValueOrZero())
			}
			o1, err := st.GetCharacterSkill(ctx, c.ID, 41)
			if assert.NoError(t, err) {
				assert.Equal(t, 3, o1.ActiveSkillLevel)
				assert.Equal(t, 10000, o1.SkillPointsInSkill)
				assert.Equal(t, 4, o1.TrainedSkillLevel)
			}
			o2, err := st.GetCharacterSkill(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, 1, o2.ActiveSkillLevel)
				assert.Equal(t, 20000, o2.SkillPointsInSkill)
				assert.Equal(t, 2, o2.TrainedSkillLevel)
			}
		}
	})
	t.Run("should delete skills not returned from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 41})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 42})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			EveTypeID:   41,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			EveTypeID:   42,
		})
		data := map[string]any{
			"skills": []map[string]any{
				{
					"active_skill_level":   3,
					"skill_id":             41,
					"skillpoints_in_skill": 10000,
					"trained_skill_level":  4,
				},
			},
			"total_sp": 90000,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/skills/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateSkillsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterSkillIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int32{41}, ids.Slice())
			}
		}
	})

}

// func TestListWalletJournalEntries(t *testing.T) {
// 	db, r, factory := testutil.NewDBOnDisk(t.TempDir())
// 	defer db.Close()
// 	s := newCharacterService(st)
// 	t.Run("can list existing entries", func(t *testing.T) {
// 		// given
// 		testutil.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{CharacterID: c.ID})
// 		factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{CharacterID: c.ID})
// 		factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{CharacterID: c.ID})
// 		// when
// 		ee, err := s.ListWalletJournalEntries(c.ID)
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Len(t, ee, 3)
// 		}
// 	})
// }

func TestUpdateSkillqueueESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		t1 := factory.CreateEveType()
		t2 := factory.CreateEveType()
		data := []map[string]any{
			{
				"finish_date":    "2016-06-29T10:47:00Z",
				"finished_level": 3,
				"queue_position": 0,
				"skill_id":       t1.ID,
				"start_date":     "2016-06-29T10:46:00Z",
			},
			{
				"finish_date":    "2016-07-15T10:47:00Z",
				"finished_level": 4,
				"queue_position": 1,
				"skill_id":       t1.ID,
				"start_date":     "2016-06-29T10:47:00Z",
			},
			{
				"finish_date":    "2016-08-30T10:47:00Z",
				"finished_level": 2,
				"queue_position": 2,
				"skill_id":       t2.ID,
				"start_date":     "2016-07-15T10:47:00Z",
			}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/skillqueue/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.UpdateSkillqueueESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkillqueue,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 3)
			}
		}
	})
}

func TestHasTokenWithScopes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should return true when token has same scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID, Scopes: esiScopes})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
	t.Run("should return false when token is missing scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		esiScopes2 := []string{"esi-assets.read_assets.v1"}
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID, Scopes: esiScopes2})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.False(t, x)
		}
	})
	t.Run("should return true when token has at least requested scopes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID, Scopes: slices.Concat(esiScopes, []string{"extra"})})
		// when
		x, err := s.HasTokenWithScopes(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
}
