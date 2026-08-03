package storage_test

import (
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveCorporation(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			corporationID = 42
			memberCount   = 888
			name          = "name"
			taxRate       = 0.12
			ticker        = "ABC"
			url           = "https://www.example.com"
			warEligible   = false
			description   = "description"
		)
		station := f.CreateEveEntity()

		// when
		err := st.UpdateOrCreateEveCorporation(t.Context(), storage.UpdateOrCreateEveCorporationParams{
			Description:   description,
			HomeStationID: station.ID,
			ID:            corporationID,
			MemberCount:   memberCount,
			Name:          name,
			TaxRate:       taxRate,
			Ticker:        ticker,
			URL:           optional.New(url),
			WarEligible:   false,
		})

		// then
		require.NoError(t, err)
		got, err := st.GetEveCorporation(t.Context(), corporationID)
		require.NoError(t, err)
		assert.Empty(t, got.Alliance)
		assert.Empty(t, got.Ceo)
		assert.Empty(t, got.Creator)
		assert.Empty(t, got.Faction)
		xassert.Equal(t, corporationID, got.ID)
		xassert.Equal(t, description, got.Description)
		xassert.Equal(t, memberCount, got.MemberCount)
		xassert.Equal(t, name, got.Name)
		xassert.Equal(t, taxRate, got.TaxRate)
		xassert.Equal(t, ticker, got.Ticker)
		xassert.EqualOptional(t, station, got.HomeStation)
		xassert.EqualOptional(t, url, got.URL)
		xassert.EqualOptional(t, warEligible, got.WarEligible)
	})

	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			corporationID = 42
			memberCount   = 888
			name          = "name"
			taxRate       = 0.12
			ticker        = "ABC"
			url           = "https://www.example.com"
			warEligible   = false
			description   = "description"
		)
		station := f.CreateEveEntity()
		alliance := f.CreateEveEntityAlliance()
		faction := f.CreateEveEntity()
		ceo := f.CreateEveEntityCharacter()
		creator := f.CreateEveEntityCharacter()
		dateFounded := f.RandomTime()

		// when
		err := st.UpdateOrCreateEveCorporation(t.Context(), storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			DateFounded:   optional.New(dateFounded),
			Description:   description,
			FactionID:     optional.New(faction.ID),
			HomeStationID: station.ID,
			ID:            corporationID,
			MemberCount:   memberCount,
			Name:          name,
			TaxRate:       taxRate,
			Ticker:        ticker,
			URL:           optional.New(url),
			WarEligible:   false,
		})

		// then
		require.NoError(t, err)
		got, err := st.GetEveCorporation(t.Context(), corporationID)
		require.NoError(t, err)
		xassert.Equal(t, corporationID, got.ID)
		xassert.Equal(t, description, got.Description)
		xassert.Equal(t, memberCount, got.MemberCount)
		xassert.Equal(t, name, got.Name)
		xassert.Equal(t, taxRate, got.TaxRate)
		xassert.Equal(t, ticker, got.Ticker)
		xassert.EqualOptional(t, alliance, got.Alliance)
		xassert.EqualOptional(t, ceo, got.Ceo)
		xassert.EqualOptional(t, creator, got.Creator)
		xassert.EqualOptional(t, dateFounded, got.DateFounded)
		xassert.EqualOptional(t, faction, got.Faction)
		xassert.EqualOptional(t, station, got.HomeStation)
		xassert.EqualOptional(t, url, got.URL)
		xassert.EqualOptional(t, warEligible, got.WarEligible)
	})

	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := f.CreateEveCorporation()
		const (
			memberCount = 888
			name        = "name"
			taxRate     = 0.12
			ticker      = "ABC"
			url         = "https://www.example.com"
			warEligible = false
			description = "description"
		)
		station := f.CreateEveEntity()
		alliance := f.CreateEveEntityAlliance()
		faction := f.CreateEveEntity()
		ceo := f.CreateEveEntityCharacter()
		creator := f.CreateEveEntityCharacter()
		dateFounded := f.RandomTime()

		// when
		err := st.UpdateOrCreateEveCorporation(t.Context(), storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			DateFounded:   optional.New(dateFounded),
			Description:   description,
			FactionID:     optional.New(faction.ID),
			HomeStationID: station.ID,
			ID:            x1.ID,
			MemberCount:   memberCount,
			Name:          name,
			TaxRate:       taxRate,
			Ticker:        ticker,
			URL:           optional.New(url),
			WarEligible:   false,
		})

		// then
		require.NoError(t, err)
		got, err := st.GetEveCorporation(t.Context(), x1.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1.ID, got.ID)
		xassert.Equal(t, description, got.Description)
		xassert.Equal(t, memberCount, got.MemberCount)
		xassert.Equal(t, name, got.Name)
		xassert.Equal(t, taxRate, got.TaxRate)
		xassert.Equal(t, ticker, got.Ticker)
		xassert.EqualOptional(t, alliance, got.Alliance)
		xassert.EqualOptional(t, ceo, got.Ceo)
		xassert.EqualOptional(t, creator, got.Creator)
		xassert.EqualOptional(t, dateFounded, got.DateFounded)
		xassert.EqualOptional(t, faction, got.Faction)
		xassert.EqualOptional(t, station, got.HomeStation)
		xassert.EqualOptional(t, url, got.URL)
		xassert.EqualOptional(t, warEligible, got.WarEligible)
	})

	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCorporation()
		// when
		c2, err := st.GetEveCorporation(t.Context(), c1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, c1.Name, c2.Name)
	})

	t.Run("can fetch corporation by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCorporation()
		// when
		c2, err := st.GetEveCorporation(t.Context(), c1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, c1, c2)

	})

	t.Run("list corporation IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCorporation()
		c2 := f.CreateEveCorporation()
		// when
		got, err := st.ListEveCorporationIDs(t.Context())
		// then
		require.NoError(t, err)
		want := set.Of(c1.ID, c2.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCorporation()
		// when
		err := st.UpdateEveCorporationName(t.Context(), c1.ID, "Alpha")
		// then
		require.NoError(t, err)
		// assert.False(t, created)
		r, err := st.GetEveCorporation(t.Context(), c1.ID)
		require.NoError(t, err)
		xassert.Equal(t, "Alpha", r.Name)
	})
}
