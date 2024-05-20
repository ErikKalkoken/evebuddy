package storage_test

import (
	"context"
	"database/sql"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestMyCharacter(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get with all dependencies", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		a := factory.CreateEveEntityAlliance()
		f := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityFaction})
		eveC := factory.CreateEveCharacter(storage.CreateEveCharacterParams{AllianceID: a.ID, FactionID: f.ID})
		c1 := factory.CreateMyCharacter(storage.UpdateOrCreateMyCharacterParams{ID: eveC.ID})
		// when
		c2, err := r.GetMyCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.LastLoginAt.Time.UTC(), c2.LastLoginAt.Time.UTC())
			assert.Equal(t, c1.Ship, c2.Ship)
			assert.Equal(t, c1.Location, c2.Location)
			assert.Equal(t, c1.TotalSP, c2.TotalSP)
			assert.Equal(t, c1.WalletBalance, c2.WalletBalance)
			assert.Equal(t, c1.Character.ID, c2.Character.ID)
			assert.Equal(t, c1.Character.Alliance, c2.Character.Alliance)
			assert.Equal(t, c1.Character.Faction, c2.Character.Faction)
		}
	})
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateEveCharacter()
		arg := storage.UpdateOrCreateMyCharacterParams{
			ID: character.ID,
		}
		// when
		err := r.UpdateOrCreateMyCharacter(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetMyCharacter(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, character.ID, r.ID)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		character := factory.CreateEveCharacter()
		home := factory.CreateLocationStructure()
		location := factory.CreateLocationStructure()
		ship := factory.CreateEveType()
		login := time.Now()
		arg := storage.UpdateOrCreateMyCharacterParams{
			ID:            character.ID,
			HomeID:        sql.NullInt64{Int64: home.ID, Valid: true},
			LastLoginAt:   sql.NullTime{Time: login, Valid: true},
			LocationID:    sql.NullInt64{Int64: location.ID, Valid: true},
			ShipID:        sql.NullInt32{Int32: ship.ID, Valid: true},
			TotalSP:       sql.NullInt64{Int64: 123, Valid: true},
			WalletBalance: sql.NullFloat64{Float64: 1.2, Valid: true},
		}
		// when
		err := r.UpdateOrCreateMyCharacter(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetMyCharacter(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, home, r.Home)
				assert.Equal(t, login.UTC(), r.LastLoginAt.Time.UTC())
				assert.Equal(t, location, r.Location)
				assert.Equal(t, ship, r.Ship)
				assert.Equal(t, int64(123), r.TotalSP.Int64)
				assert.Equal(t, 1.2, r.WalletBalance.Float64)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		newLocation := factory.CreateLocationStructure()
		newShip := factory.CreateEveType()
		err := r.UpdateOrCreateMyCharacter(ctx, storage.UpdateOrCreateMyCharacterParams{
			ID:         c1.ID,
			LocationID: sql.NullInt64{Int64: newLocation.ID, Valid: true},
			ShipID:     sql.NullInt32{Int32: newShip.ID, Valid: true},
		})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, newLocation, c2.Location)
				assert.Equal(t, newShip, c2.Ship)
			}
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		// when
		err := r.DeleteMyCharacter(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetMyCharacter(ctx, c.ID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetMyCharacter(ctx, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("should return first character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		factory.CreateMyCharacter()
		// when
		c2, err := r.GetFirstMyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetFirstMyCharacter(ctx)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		c2, err := r.GetMyCharacter(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.ID, c2.ID)
			assert.Equal(t, c1.Location, c2.Location)
		}
	})
}

func TestListMyCharactersShort(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("listed characters have all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		cc, err := r.ListMyCharactersShort(ctx)
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
		factory.CreateMyCharacter()
		factory.CreateMyCharacter()
		// when
		cc, err := r.ListMyCharactersShort(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, cc, 2)
		}
	})

}

func TestListMyCharacters(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("listed characters have all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		cc, err := r.ListMyCharacters(ctx)
		// then
		if assert.NoError(t, err) {
			c2 := cc[0]
			if assert.NotNil(t, c2) {
				assert.Len(t, cc, 1)
				assert.Equal(t, c1.ID, c2.ID)
				assert.Equal(t, c1.LastLoginAt.Time.UTC(), c2.LastLoginAt.Time.UTC())
				assert.Equal(t, c1.Ship, c2.Ship)
				assert.Equal(t, c1.Location, c2.Location)
				assert.Equal(t, c1.TotalSP, c2.TotalSP)
				assert.Equal(t, c1.WalletBalance, c2.WalletBalance)
				assert.Equal(t, c1.Character.ID, c2.Character.ID)
				assert.Equal(t, c1.Character.Alliance, c2.Character.Alliance)
				assert.Equal(t, c1.Character.Faction, c2.Character.Faction)
			}
		}
	})
}

func TestUpdateMyCharacterFields(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update home", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		home := factory.CreateLocationStructure()
		// when
		err := r.UpdateMyCharacterHome(ctx, c1.ID, sql.NullInt64{Int64: home.ID, Valid: true})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, home, c2.Home)
			}
		}

	})
	t.Run("can update last login", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		x := time.Now().Add(1 * time.Hour)
		// when
		err := r.UpdateMyCharacterLastLoginAt(ctx, c1.ID, sql.NullTime{Time: x, Valid: true})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x.UTC(), c2.LastLoginAt.Time.UTC())
			}
		}
	})
	t.Run("can update location", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		location := factory.CreateLocationStructure()
		// when
		err := r.UpdateMyCharacterLocation(ctx, c1.ID, sql.NullInt64{Int64: location.ID, Valid: true})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, location, c2.Location)
			}
		}
	})
	t.Run("can update ship", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		x := factory.CreateEveType()
		// when
		err := r.UpdateMyCharacterShip(ctx, c1.ID, sql.NullInt32{Int32: x.ID, Valid: true})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x, c2.Ship)
			}
		}
	})
	t.Run("can update skill points", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		// when
		totalSP := sql.NullInt64{Int64: int64(rand.IntN(100_000_000)), Valid: true}
		unallocatedSP := sql.NullInt64{Int64: int64(rand.IntN(10_000_000)), Valid: true}
		err := r.UpdateMyCharacterSkillPoints(ctx, c1.ID, totalSP, unallocatedSP)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, totalSP, c2.TotalSP)
				assert.Equal(t, unallocatedSP, c2.UnallocatedSP)
			}
		}
	})
	t.Run("can update wallet balance", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateMyCharacter()
		x := rand.Float64() * 100_000_000
		// when
		err := r.UpdateMyCharacterWalletBalance(ctx, c1.ID, sql.NullFloat64{Float64: x, Valid: true})
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetMyCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, x, c2.WalletBalance.Float64)
			}
		}
	})
}
