package corporationservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

// statusCacheRecorder records whether SetCorporationSection was ever called
// with a nil status, unlike the real StatusCache which silently no-ops on nil.
type statusCacheRecorder struct {
	calledWithNil bool
}

func (c *statusCacheRecorder) SetCorporationSection(o *app.CorporationSectionStatus) {
	if o == nil {
		c.calledWithNil = true
	}
}

func (c *statusCacheRecorder) UpdateCorporations(ctx context.Context, st statuscache.Storage) error {
	return nil
}

func TestUpdateSectionIfChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(Params{Storage: st, CharacterService: &CharacterServiceFake{
		Token: &app.CharacterToken{AccessToken: "accessToken"},
	}})
	ctx := context.Background()
	t.Run("should report as changed and run update when new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		var hasUpdated bool
		arg := corporationSectionUpdateParams{corporationID: c.ID, section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(_ context.Context, _ corporationSectionUpdateParams) (any, error) {
				return "any", nil
			},
			func(_ context.Context, _ corporationSectionUpdateParams, _ any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
			x, err := st.GetCorporationSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
				assert.False(t, x.HasError())
			}
		}
	})
	t.Run("should report as changed and run update when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		x1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			ErrorMessage:  "error",
			CompletedAt:   time.Now().Add(-5 * time.Second),
		})
		var hasUpdated bool
		arg := corporationSectionUpdateParams{corporationID: c.ID, section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
				return "any", nil
			},
			func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
			x2, err := st.GetCorporationSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should report as unchanged and not run update when data has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		x1 := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			Data:          "old",
			CompletedAt:   time.Now().Add(-5 * time.Second),
		})
		hasUpdated := false
		arg := corporationSectionUpdateParams{corporationID: c.ID, section: section}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(_ context.Context, _ corporationSectionUpdateParams) (any, error) {
				return "old", nil
			},
			func(_ context.Context, _ corporationSectionUpdateParams, _ any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.False(t, hasUpdated)
			x2, err := st.GetCorporationSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.CompletedAt, x1.CompletedAt)
				assert.False(t, x2.HasError())
			}
		}
	})
	t.Run("should mark context for force refresh only when forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		var gotForceRefresh bool
		arg := corporationSectionUpdateParams{corporationID: c.ID, section: section, forceUpdate: true}
		// when
		_, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, _ corporationSectionUpdateParams) (any, error) {
				gotForceRefresh = xgoesi.IsForceRefresh(ctx)
				return "any", nil
			},
			func(_ context.Context, _ corporationSectionUpdateParams, _ any) (bool, error) {
				return true, nil
			})
		// then
		require.NoError(t, err)
		assert.True(t, gotForceRefresh)
	})
	t.Run("should not mark context for force refresh when not forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationMembers
		var gotForceRefresh bool
		arg := corporationSectionUpdateParams{corporationID: c.ID, section: section}
		// when
		_, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, _ corporationSectionUpdateParams) (any, error) {
				gotForceRefresh = xgoesi.IsForceRefresh(ctx)
				return "any", nil
			},
			func(_ context.Context, _ corporationSectionUpdateParams, _ any) (bool, error) {
				return true, nil
			})
		// then
		require.NoError(t, err)
		assert.False(t, gotForceRefresh)
	})
	t.Run("should update when data has not changed and forced", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		section := app.SectionCorporationIndustryJobs
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       section,
			Data:          "old",
			CompletedAt:   time.Now().Add(-5 * time.Second),
		})
		var hasUpdated bool
		arg := corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       section,
			forceUpdate:   true,
		}
		// when
		changed, err := s.updateSectionIfChanged(ctx, arg, false,
			func(ctx context.Context, arg corporationSectionUpdateParams) (any, error) {
				return "old", nil
			},
			func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error) {
				hasUpdated = true
				return true, nil
			})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			assert.True(t, hasUpdated)
		}
	})
}

func TestUpdateSectionIfNeeded_DoesNotCacheStatusWhenErrorPersistFails(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	scs := &statusCacheRecorder{}
	s := NewFake(Params{
		Storage:            st,
		StatusCacheService: scs,
		CharacterService: &CharacterServiceFake{
			Token: &app.CharacterToken{AccessToken: "accessToken"},
		},
	})
	ctx := context.Background()
	testutil.MustTruncateTables(db)
	c := factory.CreateCorporation()
	httpmock.Reset()
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://esi.evetech.net/corporations/%d/wallets", c.ID),
		func(req *http.Request) (*http.Response, error) {
			// close the DB so the section status write that follows this
			// failed ESI call also fails, reproducing the bug
			db.Close()
			return httpmock.NewStringResponse(500, "server error"), nil
		},
	)
	arg := corporationSectionUpdateParams{
		corporationID: c.ID,
		section:       app.SectionCorporationWalletBalances,
		forceUpdate:   true,
	}
	// when
	_, err := s.updateSectionIfNeeded(ctx, arg)
	// then
	assert.Error(t, err)
	assert.False(t, scs.calledWithNil, "SetCorporationSection must not be called with a nil status when persisting the error failed")
}

func TestHasSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("report true when section has changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// when
		got, err := s.hasSectionChanged(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationMembers,
		}, "changed",
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, got)
	})
	t.Run("report true when section does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		// when
		got, err := s.hasSectionChanged(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationMembers,
		}, "changed",
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.True(t, got)
	})
	t.Run("report false when section has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		status := factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: c.ID,
			Section:       app.SectionCorporationMembers,
		})
		// when
		got, err := s.hasSectionChanged(ctx, corporationSectionUpdateParams{
			corporationID: c.ID,
			section:       app.SectionCorporationMembers,
		}, status.ContentHash,
		)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		assert.False(t, got)
	})
}

// TODO: The method will not match against empty role /scopes. Check if that makes sense

func TestCorporationService_HasValidToken(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should report true when matching token was found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		section := app.SectionCorporationMembers
		err := st.UpdateCharacterRoles(ctx, character.ID, section.Roles())
		require.NoError(t, err)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: character.ID,
			Scopes:      section.Scopes(),
		})
		// when
		got, err := s.hasToken(ctx, corporation.ID, section.Roles(), section.Scopes())
		// then
		require.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("should report false when not matching token was found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		section := app.SectionCorporationMembers
		// when
		got, err := s.hasToken(ctx, corporation.ID, section.Roles(), section.Scopes())
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
}
