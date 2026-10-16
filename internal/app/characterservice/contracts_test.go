package characterservice_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNotifyUpdatedContracts(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	const characterID = 7
	earliest := time.Now().UTC().Add(-6 * time.Hour)
	now := time.Now().UTC()
	cases := []struct {
		name           string
		acceptorID     int64
		status         app.ContractStatus
		statusNotified app.ContractStatus
		typ            app.ContractType
		updatedAt      time.Time
		shouldNotify   bool
	}{
		{"notify new courier 1", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 2", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 3", 42, app.ContractStatusFailed, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"don't notify courier", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"notify new item exchange", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, true},
		{"don't notify again", 42, app.ContractStatusInProgress, app.ContractStatusInProgress, app.ContractTypeCourier, now, false},
		{"don't notify when acceptor is character", characterID, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"don't notify when contract is too old", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now.Add(-12 * time.Hour), false},
		{"don't notify item exchange", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			if tc.acceptorID != 0 {
				f.CreateEveEntityCharacter(app.EveEntity{ID: tc.acceptorID})
			}
			ec := f.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
			c := f.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
			o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
				AcceptorID:     tc.acceptorID,
				CharacterID:    c.ID,
				Status:         tc.status,
				StatusNotified: tc.statusNotified,
				Type:           tc.typ,
				UpdatedAt:      tc.updatedAt,
			})
			var sendCount int
			// when
			err := s.NotifyUpdatedContracts(t.Context(), o.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			require.NoError(t, err)
			xassert.Equal(t, tc.shouldNotify, sendCount == 1)
		})
	}
}

func TestListAllCharacterContractSlotsPersonal(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	cases := []struct {
		name              string
		contractingLevel  int64
		advancedLevel     int64
		personalContracts int
		wantUsed          int
		wantFree          int
		wantTotal         int
	}{
		{
			name:              "contracing and open contracts",
			contractingLevel:  3,
			advancedLevel:     0,
			personalContracts: 2,
			wantUsed:          2,
			wantFree:          11,
			wantTotal:         13,
		},
		{
			name:              "contracing and no contracts",
			contractingLevel:  3,
			advancedLevel:     0,
			personalContracts: 0,
			wantUsed:          0,
			wantFree:          13,
			wantTotal:         13,
		},
		{
			name:              "no skills ",
			contractingLevel:  0,
			advancedLevel:     0,
			personalContracts: 1,
			wantUsed:          1,
			wantFree:          0,
			wantTotal:         1,
		},
		{
			name:              "no skills and no contracts",
			contractingLevel:  0,
			advancedLevel:     0,
			personalContracts: 0,
			wantUsed:          0,
			wantFree:          1,
			wantTotal:         1,
		},
		{
			name:              "advanced contracing and open contracts",
			contractingLevel:  5,
			advancedLevel:     2,
			personalContracts: 2,
			wantUsed:          2,
			wantFree:          27,
			wantTotal:         29,
		},
		{
			name:              "advanced contracing and no contracts",
			contractingLevel:  5,
			advancedLevel:     2,
			personalContracts: 0,
			wantUsed:          0,
			wantFree:          29,
			wantTotal:         29,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := f.CreateCharacter()
			if tc.contractingLevel > 0 {
				skill := f.CreateEveType(storage.CreateEveTypeParams{
					ID: app.EveTypeContracting,
				})
				f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:      c.ID,
					TypeID:           skill.ID,
					ActiveSkillLevel: tc.contractingLevel,
				})
			}
			if tc.advancedLevel > 0 {
				skill := f.CreateEveType(storage.CreateEveTypeParams{
					ID: app.EveTypeAdvancedContracting,
				})
				f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:      c.ID,
					TypeID:           skill.ID,
					ActiveSkillLevel: tc.advancedLevel,
				})
			}
			for range tc.personalContracts {
				f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
					CharacterID:    c.ID,
					ForCorporation: false,
				})
			}

			// given - contracts to ignore
			issuer := f.CreateEveEntityCharacter()
			issuerCorporation := f.CreateEveEntityCorporation()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:         c.ID,
				IssuerID:            issuer.ID,
				IssuerCorporationID: issuerCorporation.ID,
				ForCorporation:      false,
			})
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				ForCorporation: true,
			})
			acceptor := f.CreateEveEntityCharacter()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID: c.ID,
				Status:      app.ContractStatusInProgress,
				AcceptorID:  acceptor.ID,
			})
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				ForCorporation: false,
				AssigneeID:     c.EveCharacter.Corporation.ID,
			})

			// when
			oo, err := s.ListAllCharacterContractSlotsPersonal(t.Context())

			// then
			require.NoError(t, err)
			require.Len(t, oo, 1)
			got := oo[0]
			want := app.CharacterContractSlots{
				CharacterID:     c.ID,
				CharacterName:   c.EveCharacter.Name,
				CorporationID:   c.EveCharacter.Corporation.ID,
				CorporationName: c.EveCharacter.Corporation.Name,
				Free:            tc.wantFree,
				Used:            tc.wantUsed,
				Total:           tc.wantTotal,
				IsCorporation:   false,
			}
			xassert.Equal(t, want, got)
		})
	}
}

func TestListAllCharacterContractSlotsCorporation(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	cases := []struct {
		name             string
		contractingLevel int64
		activeContracts  int
		wantUsed         int
		wantFree         int
		wantTotal        int
	}{
		{
			name:             "contracing and open contracts",
			contractingLevel: 3,
			activeContracts:  2,
			wantUsed:         2,
			wantFree:         38,
			wantTotal:        40,
		},
		{
			name:             "contracing and no contracts",
			contractingLevel: 3,
			activeContracts:  0,
			wantUsed:         0,
			wantFree:         40,
			wantTotal:        40,
		},
		{
			name:             "no skills",
			contractingLevel: 0,
			activeContracts:  1,
			wantUsed:         1,
			wantFree:         9,
			wantTotal:        10,
		},
		{
			name:             "no skills and no contracts",
			contractingLevel: 0,
			activeContracts:  0,
			wantUsed:         0,
			wantFree:         10,
			wantTotal:        10,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := f.CreateCharacter()
			if tc.contractingLevel > 0 {
				skill := f.CreateEveType(storage.CreateEveTypeParams{
					ID: app.EveTypeCorporationContracting,
				})
				f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:      c.ID,
					TypeID:           skill.ID,
					ActiveSkillLevel: tc.contractingLevel,
				})
			}
			for range tc.activeContracts {
				f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
					CharacterID:    c.ID,
					ForCorporation: true,
				})
			}
			// given - contracts to ignore
			issuer := f.CreateEveEntityCharacter()
			issuerCorporation := f.CreateEveEntityCorporation()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:         c.ID,
				IssuerID:            issuer.ID,
				IssuerCorporationID: issuerCorporation.ID,
				ForCorporation:      true,
			})
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				ForCorporation: false,
			})
			acceptor := f.CreateEveEntityCharacter()
			f.CreateCharacterContractCourier(storage.CreateCharacterContractParams{
				CharacterID:    c.ID,
				Status:         app.ContractStatusInProgress,
				AcceptorID:     acceptor.ID,
				ForCorporation: true,
			})

			// when
			oo, err := s.ListAllCharacterContractSlotsCorporation(t.Context())

			// then
			require.NoError(t, err)
			require.Len(t, oo, 1)
			got := oo[0]
			want := app.CharacterContractSlots{
				CharacterID:     c.ID,
				CharacterName:   c.EveCharacter.Name,
				CorporationID:   c.EveCharacter.Corporation.ID,
				CorporationName: c.EveCharacter.Corporation.Name,
				Free:            tc.wantFree,
				Used:            tc.wantUsed,
				Total:           tc.wantTotal,
				IsCorporation:   true,
			}
			xassert.Equal(t, want, got)
		})
	}
}

func TestGetContract(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can return a contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract()
		// when
		got, err := s.GetContract(t.Context(), o.CharacterID, o.ContractID)
		// then
		require.NoError(t, err)
		assert.Equal(t, o.ID, got.ID)
	})
	t.Run("should return own error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := s.GetContract(t.Context(), 1, 1)
		// then
		assert.Error(t, err)
	})
}

func TestCountContractBids(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can count bids for a contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract()
		factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID})
		factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID})
		// when
		got, err := s.CountContractBids(t.Context(), o.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 2, got)
	})
}

func TestGetContractTopBid(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can return top bid for a contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract()
		factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 100})
		top := factory.CreateCharacterContractBid(storage.CreateContractBidParams{ContractID: o.ID, Amount: 500})
		// when
		got, err := s.GetContractTopBid(t.Context(), o.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, top.BidID, got.BidID)
	})
	t.Run("should return own error when contract has no bids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract()
		// when
		_, err := s.GetContractTopBid(t.Context(), o.ID)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestListAllContracts(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	dateIssued := time.Now().UTC().Add(-time.Hour)
	// createCopy stores a copy of the same contract for a character.
	createCopy := func(characterID, issuerID int64, forCorporation bool) *app.CharacterContract {
		return factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:    characterID,
			ContractID:     42,
			DateIssued:     dateIssued,
			ForCorporation: forCorporation,
			IssuerID:       issuerID,
			Type:           app.ContractTypeItemExchange,
		})
	}
	t.Run("can list contracts across all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterContract()
		factory.CreateCharacterContract()
		// when
		got, err := s.ListAllContracts(t.Context())
		// then
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
	for _, issuerFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("should keep copy of issuer when stored first: %v", issuerFirst), func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			issuer := factory.CreateCharacterFull()
			other := factory.CreateCharacterFull()
			factory.CreateEveEntityCharacter(app.EveEntity{ID: issuer.ID})
			if issuerFirst {
				createCopy(issuer.ID, issuer.ID, false)
				createCopy(other.ID, issuer.ID, false)
			} else {
				createCopy(other.ID, issuer.ID, false)
				createCopy(issuer.ID, issuer.ID, false)
			}
			// when
			got, err := s.ListAllContracts(t.Context())
			// then
			require.NoError(t, err)
			require.Len(t, got, 1)
			xassert.Equal(t, issuer.ID, got[0].CharacterID)
		})
	}
	t.Run("should keep copy of issuer for contracts issued for a corporation", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		other := factory.CreateCharacterFull()
		issuer := factory.CreateCharacterFull()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: issuer.ID})
		createCopy(other.ID, issuer.ID, true)
		createCopy(issuer.ID, issuer.ID, true)
		// when
		got, err := s.ListAllContracts(t.Context())
		// then
		require.NoError(t, err)
		require.Len(t, got, 1)
		xassert.Equal(t, issuer.ID, got[0].CharacterID)
	})
	t.Run("should keep copy of character with lowest ID when issuer is not one of them", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		stranger := factory.CreateEveEntityCharacter()
		low, high := c1, c2
		if high.ID < low.ID {
			low, high = high, low
		}
		createCopy(high.ID, stranger.ID, false)
		createCopy(low.ID, stranger.ID, false)
		// when
		got, err := s.ListAllContracts(t.Context())
		// then
		require.NoError(t, err)
		require.Len(t, got, 1)
		xassert.Equal(t, low.ID, got[0].CharacterID)
	})
	t.Run("should keep contracts which differ", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		stranger := factory.CreateEveEntityCharacter()
		createCopy(c1.ID, stranger.ID, false)
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: c2.ID,
			ContractID:  42,
			DateIssued:  dateIssued,
			IssuerID:    stranger.ID,
			Type:        app.ContractTypeCourier, // same ID, different type
		})
		// when
		got, err := s.ListAllContracts(t.Context())
		// then
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

func TestListContractItems(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can list items for a contract", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterContract()
		item := factory.CreateCharacterContractItem(storage.CreateContractItemParams{ContractID: o.ID})
		// when
		got, err := s.ListContractItems(t.Context(), o.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, item.RecordID, got[0].RecordID)
		}
	})
}

func TestNotifyUpdatedContractsMessage(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	earliest := time.Now().UTC().Add(-6 * time.Hour)
	cases := []struct {
		name   string
		status app.ContractStatus
		typ    app.ContractType
		want   string // content with %s for the contract name
	}{
		{"courier accepted", app.ContractStatusInProgress, app.ContractTypeCourier, "Contract '%s' has been accepted by Bruce"},
		{"courier delivered", app.ContractStatusFinished, app.ContractTypeCourier, "Contract '%s' has been delivered"},
		{"courier failed", app.ContractStatusFailed, app.ContractTypeCourier, "Contract '%s' has been failed by Bruce"},
		{"item exchange accepted", app.ContractStatusFinished, app.ContractTypeItemExchange, "Contract '%s' has been accepted by Bruce"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			acceptor := f.CreateEveEntityCharacter(app.EveEntity{ID: 42, Name: "Bruce"})
			c := f.CreateCharacterFull()
			o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
				AcceptorID:  acceptor.ID,
				CharacterID: c.ID,
				Status:      tc.status,
				Type:        tc.typ,
				UpdatedAt:   time.Now().UTC(),
			})
			var titles, contents []string
			notify := func(title, content string) {
				titles = append(titles, title)
				contents = append(contents, content)
			}
			// when
			err := s.NotifyUpdatedContracts(t.Context(), c.ID, earliest, notify)
			// then
			require.NoError(t, err)
			xassert.Equal(t, []string{c.EveCharacter.Name + ": Contract updated"}, titles)
			xassert.Equal(t, []string{fmt.Sprintf(tc.want, o.NameDisplay())}, contents)
			o2, err := st.GetCharacterContract(t.Context(), c.ID, o.ContractID)
			require.NoError(t, err)
			xassert.Equal(t, tc.status, o2.StatusNotified)
			// a second run must not notify again
			err = s.NotifyUpdatedContracts(t.Context(), c.ID, earliest, notify)
			require.NoError(t, err)
			assert.Len(t, contents, 1)
		})
	}
}
