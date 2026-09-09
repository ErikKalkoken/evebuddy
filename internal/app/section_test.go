package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterSectionScopes(t *testing.T) {
	t.Run("known section", func(t *testing.T) {
		got := app.SectionCharacterAssets.Scopes()
		assert.True(t, got.Size() > 0)
	})
	t.Run("unknown section", func(t *testing.T) {
		got := app.CharacterSection("bogus").Scopes()
		xassert.Equal(t, 0, got.Size())
	})
}

func TestCharacterSectionString(t *testing.T) {
	xassert.Equal(t, "assets", app.SectionCharacterAssets.String())
}

func TestCharacterSectionTimeout(t *testing.T) {
	t.Run("known section", func(t *testing.T) {
		xassert.EqualDuration(t, 3600*time.Second, app.SectionCharacterAssets.Timeout(), 0)
	})
	t.Run("unknown section", func(t *testing.T) {
		got := app.CharacterSection("bogus").Timeout()
		xassert.EqualDuration(t, 3600*time.Second, got, 0)
	})
}

func TestCorporationSectionWalletJournal(t *testing.T) {
	xassert.Equal(t, app.SectionCorporationWalletJournal3, app.CorporationSectionWalletJournal(app.Division3))
}

func TestCorporationSectionWalletTransactions(t *testing.T) {
	xassert.Equal(t, app.SectionCorporationWalletTransactions3, app.CorporationSectionWalletTransactions(app.Division3))
}

func TestCorporationSectionDivision(t *testing.T) {
	t.Run("wallet section", func(t *testing.T) {
		xassert.Equal(t, app.Division3, app.SectionCorporationWalletJournal3.Division())
	})
	t.Run("non-wallet section", func(t *testing.T) {
		xassert.Equal(t, app.DivisionZero, app.SectionCorporationAssets.Division())
	})
}

func TestCorporationSectionString(t *testing.T) {
	xassert.Equal(t, "assets", app.SectionCorporationAssets.String())
}

func TestCorporationSectionTimeout(t *testing.T) {
	t.Run("known section", func(t *testing.T) {
		xassert.EqualDuration(t, 3600*time.Second, app.SectionCorporationAssets.Timeout(), 0)
	})
	t.Run("unknown section", func(t *testing.T) {
		got := app.CorporationSection("bogus").Timeout()
		xassert.EqualDuration(t, 3600*time.Second, got, 0)
	})
}

func TestCorporationSectionRoles(t *testing.T) {
	t.Run("known section", func(t *testing.T) {
		got := app.SectionCorporationAssets.Roles()
		assert.True(t, got.Contains(app.RoleDirector))
	})
	t.Run("section with no required roles", func(t *testing.T) {
		got := app.SectionCorporationContracts.Roles()
		xassert.Equal(t, 0, got.Size())
	})
	t.Run("unknown section", func(t *testing.T) {
		got := app.CorporationSection("bogus").Roles()
		assert.True(t, got.Contains(app.RoleDirector))
	})
}

func TestCorporationSectionScopes(t *testing.T) {
	t.Run("known section", func(t *testing.T) {
		got := app.SectionCorporationAssets.Scopes()
		assert.True(t, got.Size() > 0)
	})
	t.Run("unknown section", func(t *testing.T) {
		got := app.CorporationSection("bogus").Scopes()
		xassert.Equal(t, 0, got.Size())
	})
}

func TestEveUniverseSectionScopes(t *testing.T) {
	got := app.SectionEveCharacters.Scopes()
	xassert.Equal(t, 0, got.Size())
}

func TestEveUniverseSectionString(t *testing.T) {
	xassert.Equal(t, "characters", app.SectionEveCharacters.String())
}

func TestEveUniverseSectionTimeout(t *testing.T) {
	t.Run("known section", func(t *testing.T) {
		xassert.EqualDuration(t, time.Hour, app.SectionEveCharacters.Timeout(), 0)
	})
	t.Run("unknown section", func(t *testing.T) {
		got := app.EveUniverseSection("bogus").Timeout()
		xassert.EqualDuration(t, 24*time.Hour, got, 0)
	})
}

func TestScopes(t *testing.T) {
	got := app.Scopes()
	assert.True(t, got.Size() > 0)
}
