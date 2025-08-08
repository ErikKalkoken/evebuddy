package corporationservice

import (
	"context"
	"maps"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type CharacterServiceFake struct {
	Token          *app.CharacterToken
	CorporationIDs set.Set[int32]
	Error          error
}

func (s *CharacterServiceFake) ValidCharacterTokenForCorporation(ctx context.Context, corporationID int32, roles set.Set[app.Role], scopes set.Set[string]) (*app.CharacterToken, error) {
	return s.Token, s.Error
}

func (s *CharacterServiceFake) ListCharacterCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	return s.CorporationIDs, s.Error
}

func NewFake(st *storage.Storage, args ...Params) *CorporationService {
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
		arg.CharacterService = args[0].CharacterService
	}
	s := New(arg)
	return s
}

func TestUpdateDivisionsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new entries from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{
			Token: &app.CharacterToken{AccessToken: "accessToken"},
		}})
		c := factory.CreateCorporation()
		data := map[string]any{
			"hangar": []map[string]any{
				{
					"division": 1,
					"name":     "Awesome Hangar 1",
				},
			},
			"wallet": []map[string]any{
				{
					"division": 1,
					"name":     "Rich Wallet 1",
				},
			},
		}
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/divisions/`,
			httpmock.NewJsonResponderOrPanic(200, data),
		)
		// when
		changed, err := s.updateDivisionsESI(ctx, app.CorporationUpdateSectionParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationDivisions,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			// wallets
			wallets, err := st.ListCorporationWalletNames(ctx, c.ID)
			if assert.NoError(t, err) {
				got := maps.Collect(xiter.MapSlice2(wallets, func(x *app.CorporationWalletName) (int32, string) {
					return x.DivisionID, x.Name
				}))
				want := map[int32]string{
					1: "Rich Wallet 1",
				}
				assert.Equal(t, want, got)
			}
			// hangar
			hangars, err := st.ListCorporationHangarNames(ctx, c.ID)
			if assert.NoError(t, err) {
				got := maps.Collect(xiter.MapSlice2(hangars, func(x *app.CorporationHangarName) (int32, string) {
					return x.DivisionID, x.Name
				}))
				want := map[int32]string{
					1: "Awesome Hangar 1",
				}
				assert.Equal(t, want, got)
			}
		}
	})
	t.Run("should update existing balances", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		s := NewFake(st, Params{CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
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
		changed, err := s.updateWalletBalancesESI(ctx, app.CorporationUpdateSectionParams{
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
