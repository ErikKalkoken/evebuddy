package corporationservice

import (
	"context"
	"fmt"
	"maps"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestUpdateDivisionsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	t.Run("should create new entries from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{
			Token: &app.CharacterToken{AccessToken: "accessToken"},
		}})
		c := factory.CreateCorporation()

		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/divisions", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
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
			}),
		)
		// when
		changed, err := s.updateDivisionsESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationDivisions,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		// wallets
		wallets, err := st.ListCorporationWalletNames(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(wallets, func(x *app.CorporationWalletName) (app.Division, string) {
			return x.Division, x.Name
		}))
		want := map[app.Division]string{
			app.Division1: "Rich Wallet 1",
		}
		xassert.Equal(t, want, got)
		// hangar
		hangars, err := st.ListCorporationHangarNames(ctx, c.ID)
		require.NoError(t, err)
		got2 := maps.Collect(xiter.MapSlice2(hangars, func(x *app.CorporationHangarName) (app.Division, string) {
			return x.Division, x.Name
		}))
		want2 := map[app.Division]string{
			app.Division1: "Awesome Hangar 1",
		}
		xassert.Equal(t, want2, got2)
	})
	t.Run("should update existing balances", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{AccessToken: "accessToken"}}})
		c := factory.CreateCorporation()
		for _, d := range app.Divisions {
			err := st.UpdateOrCreateCorporationWalletBalance(ctx, storage.UpdateOrCreateCorporationWalletBalanceParams{
				CorporationID: c.ID,
				Division:      d,
			})
			assert.NoError(t, err)
		}

		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/corporations/%d/wallets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"balance":  123.45,
				"division": 1,
			}, {
				"balance":  223.45,
				"division": 2,
			}, {
				"balance":  323.45,
				"division": 3,
			}, {
				"balance":  423.45,
				"division": 4,
			}, {
				"balance":  523.45,
				"division": 5,
			}, {
				"balance":  623.45,
				"division": 6,
			}, {
				"balance":  723.45,
				"division": 7,
			}}),
		)
		// when
		changed, err := s.updateWalletBalancesESI(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationWalletBalances,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		oo, err := st.ListCorporationWalletBalances(ctx, c.ID)
		require.NoError(t, err)
		got := maps.Collect(xiter.MapSlice2(oo, func(x *app.CorporationWalletBalance) (app.Division, float64) {
			return x.Division, x.Balance
		}))
		want := map[app.Division]float64{
			app.Division1: 123.45,
			app.Division2: 223.45,
			app.Division3: 323.45,
			app.Division4: 423.45,
			app.Division5: 523.45,
			app.Division6: 623.45,
			app.Division7: 723.45,
		}
		xassert.Equal(t, want, got)
	})
}

func TestListHangarNames(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("returns stored names merged with defaults", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationHangarName(storage.UpdateOrCreateCorporationHangarNameParams{
			CorporationID: c.ID,
			Division:      app.Division1,
			Name:          "Awesome Hangar",
		})
		// when
		got, err := s.ListHangarNames(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, "Awesome Hangar", got[app.Division1])
		assert.Equal(t, "2nd Division", got[app.Division2])
	})
	t.Run("returns defaults when corporation unknown", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		got, err := s.ListHangarNames(ctx, 42)
		// then
		require.NoError(t, err)
		assert.Equal(t, "1st Division", got[app.Division1])
	})
	t.Run("returns error when fetching names fails", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		s := NewFake(Params{Storage: st})
		c := factory.CreateCorporation()
		if _, err := db.Exec("DROP TABLE corporation_hangar_names"); err != nil {
			t.Fatal(err)
		}
		// when
		_, err := s.ListHangarNames(ctx, c.ID)
		// then
		assert.Error(t, err)
	})
}
