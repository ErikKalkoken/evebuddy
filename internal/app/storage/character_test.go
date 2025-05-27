package storage_test

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCharacter(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get with all dependencies", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := factory.CreateEveEntityAlliance()
		f := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		eveC := factory.CreateEveCharacter(storage.CreateEveCharacterParams{AllianceID: a.ID, FactionID: f.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: eveC.ID})
		// when
		c2, err := r.GetCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.LastLoginAt.ValueOrZero().UTC(), c2.LastLoginAt.ValueOrZero().UTC())
			assert.Equal(t, c1.Ship, c2.Ship)
			assert.Equal(t, c1.Location, c2.Location)
			assert.Equal(t, c1.TotalSP, c2.TotalSP)
			assert.Equal(t, c1.WalletBalance, c2.WalletBalance)
			assert.Equal(t, c1.EveCharacter.ID, c2.EveCharacter.ID)
			assert.Equal(t, c1.EveCharacter.Alliance, c2.EveCharacter.Alliance)
			assert.Equal(t, c1.EveCharacter.Faction, c2.EveCharacter.Faction)
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		err := r.DeleteCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		// when
		c, err := r.GetAnyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Contains(t, []int32{c1.ID, c2.ID}, c.ID)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetAnyCharacter(ctx)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		c2, err := r.GetCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Location, c2.Location)
		}
	})
}

func TestCharacterCreate(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateEveCharacter()
		arg := storage.CreateCharacterParams{
			ID: character.ID,
		}
		// when
		err := r.CreateCharacter(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetCharacter(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, character.ID, r.ID)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateEveCharacter()
		home := factory.CreateEveLocationStructure()
		location := factory.CreateEveLocationStructure()
		ship := factory.CreateEveType()
		login := time.Now()
		cloneJump := time.Now()
		arg := storage.CreateCharacterParams{
			ID:              character.ID,
			AssetValue:      optional.From(3.4),
			HomeID:          optional.From(home.ID),
			LastCloneJumpAt: optional.From(cloneJump),
			LastLoginAt:     optional.From(login),
			LocationID:      optional.From(location.ID),
			ShipID:          optional.From(ship.ID),
			TotalSP:         optional.From(123),
			WalletBalance:   optional.From(1.2),
		}
		// when
		err := r.CreateCharacter(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetCharacter(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, home, r.Home)
				assert.Equal(t, cloneJump.UTC(), r.LastCloneJumpAt.ValueOrZero().UTC())
				assert.Equal(t, login.UTC(), r.LastLoginAt.ValueOrZero().UTC())
				assert.Equal(t, location, r.Location)
				assert.Equal(t, ship, r.Ship)
				assert.Equal(t, 123, r.TotalSP.ValueOrZero())
				assert.Equal(t, 1.2, r.WalletBalance.ValueOrZero())
				assert.Equal(t, 3.4, r.AssetValue.ValueOrZero())
			}
		}
	})
	t.Run("report error when character already exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		arg := storage.CreateCharacterParams{ID: c1.ID}
		// when
		err := r.CreateCharacter(ctx, arg)
		// then
		assert.ErrorIs(t, err, app.ErrAlreadyExists)
	})
}

func TestListCharactersShort(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("listed characters have all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		cc, err := r.ListCharactersShort(ctx)
		// then
		if assert.NoError(t, err) {
			c2 := cc[0]
			assert.Len(t, cc, 1)
			assert.Equal(t, c1.ID, c2.ID)
		}
	})
	t.Run("can list characters", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateCharacter()
		factory.CreateCharacter()
		// when
		cc, err := r.ListCharactersShort(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, cc, 2)
		}
	})

}

func TestListCharacters(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("listed characters have all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		cc, err := r.ListCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			c2 := cc[0]
			if assert.NotNil(t, c2) {
				assert.Len(t, cc, 1)
				assert.Equal(t, c1.ID, c2.ID)
				assert.Equal(t, c1.LastLoginAt.ValueOrZero().UTC(), c2.LastLoginAt.ValueOrZero().UTC())
				assert.Equal(t, c1.Ship, c2.Ship)
				assert.Equal(t, c1.Location, c2.Location)
				assert.Equal(t, c1.TotalSP, c2.TotalSP)
				assert.Equal(t, c1.WalletBalance, c2.WalletBalance)
				assert.Equal(t, c1.EveCharacter.ID, c2.EveCharacter.ID)
				assert.Equal(t, c1.EveCharacter.Alliance, c2.EveCharacter.Alliance)
				assert.Equal(t, c1.EveCharacter.Faction, c2.EveCharacter.Faction)
			}
		}
	})
}

func TestUpdateCharacterFields(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update home", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		home := factory.CreateEveLocationStructure()
		// when
		err := r.UpdateCharacterHome(ctx, c1.ID, optional.From(home.ID))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, home, c2.Home)
			}
		}

	})
	t.Run("can update last clone jump with a time", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		x := time.Now().Add(1 * time.Hour)
		// when
		err := r.UpdateCharacterLastCloneJump(ctx, c1.ID, optional.From(x))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x.UTC(), c2.LastCloneJumpAt.ValueOrZero().UTC())
			}
		}
	})
	t.Run("can update last clone jump with zero time", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		x := time.Time{}
		// when
		err := r.UpdateCharacterLastCloneJump(ctx, c1.ID, optional.From(x))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x, c2.LastCloneJumpAt.MustValue())
			}
		}
	})
	t.Run("should return empty when last clone jump not updated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		c2, err := r.GetCharacter(ctx, c1.ID)
		if assert.NoError(t, err) {
			assert.True(t, c2.LastCloneJumpAt.IsEmpty())
		}
	})
	t.Run("can update last login", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		x := time.Now().Add(1 * time.Hour)
		// when
		err := r.UpdateCharacterLastLoginAt(ctx, c1.ID, optional.From(x))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x.UTC(), c2.LastLoginAt.ValueOrZero().UTC())
			}
		}
	})
	t.Run("can update location", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		location := factory.CreateEveLocationStructure()
		// when
		err := r.UpdateCharacterLocation(ctx, c1.ID, optional.From(location.ID))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, location, c2.Location)
			}
		}
	})
	t.Run("can update ship", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		x := factory.CreateEveType()
		// when
		err := r.UpdateCharacterShip(ctx, c1.ID, optional.From(x.ID))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x, c2.Ship)
			}
		}
	})
	t.Run("can update is training watched", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		err := r.UpdateCharacterIsTrainingWatched(ctx, c1.ID, true)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.True(t, c2.IsTrainingWatched)
			}
		}
	})
	t.Run("can update is training watched 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		c2, err := r.GetCharacter(ctx, c1.ID)
		if assert.NoError(t, err) {
			assert.True(t, c2.IsTrainingWatched)
		}
		// when
		err = r.UpdateCharacterIsTrainingWatched(ctx, c1.ID, false)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.False(t, c2.IsTrainingWatched)
			}
		}
	})
	t.Run("can update skill points", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		totalSP := optional.From(rand.IntN(100_000_000))
		unallocatedSP := optional.From(rand.IntN(10_000_000))
		err := r.UpdateCharacterSkillPoints(ctx, c1.ID, totalSP, unallocatedSP)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, totalSP, c2.TotalSP)
				assert.Equal(t, unallocatedSP, c2.UnallocatedSP)
			}
		}
	})
	t.Run("can update wallet balance", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		x := rand.Float64() * 100_000_000
		// when
		err := r.UpdateCharacterWalletBalance(ctx, c1.ID, optional.From(x))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x, c2.WalletBalance.ValueOrZero())
			}
		}
	})
	t.Run("can disable all training watchers", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{
			IsTrainingWatched: true,
		})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{
			IsTrainingWatched: true,
		})
		// when
		err := r.DisableAllTrainingWatchers(ctx)
		// then
		if assert.NoError(t, err) {
			c1, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.False(t, c1.IsTrainingWatched)
			}
			c2, err := r.GetCharacter(ctx, c2.ID)
			if assert.NoError(t, err) {
				assert.False(t, c2.IsTrainingWatched)
			}
		}
	})
}

func TestCharacterAssetValue(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{
			AssetValue: optional.From(1.23),
		})
		v := 1234.6
		// when
		err := r.UpdateCharacterAssetValue(ctx, c1.ID, optional.From(v))
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, v, c2.AssetValue.ValueOrZero())
			}
		}
	})
	t.Run("can reset", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{
			AssetValue: optional.From(1.23),
		})
		// when
		err := r.UpdateCharacterAssetValue(ctx, c1.ID, optional.Optional[float64]{})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.True(t, c2.AssetValue.IsEmpty())
			}
		}
	})
	t.Run("can get set value", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		v := 1234.6
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{
			AssetValue: optional.From(v),
		})
		// when
		got, err := r.GetCharacterAssetValue(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, v, got.ValueOrZero())
		}
	})
	t.Run("can get empty value", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		if err := r.UpdateCharacterAssetValue(ctx, c1.ID, optional.Optional[float64]{}); err != nil {
			t.Fatal(err)
		}
		// when
		got, err := r.GetCharacterAssetValue(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}
