package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestUpdateCharacterJumpClonesESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
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
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 22118})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60003463})
		factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1021348135816})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/clones/", c.ID),
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
		c := factory.CreateCharacter()
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
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/clones/", c.ID),
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
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return time of next available jump with skill", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(now.Add(-6 * time.Hour)),
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
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(now.Add(-6 * time.Hour)),
		})
		x, err := cs.calcNextCloneJump(ctx, c)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, now.Add(18*time.Hour), x.MustValue(), 10*time.Second)
		}
	})
	t.Run("should return time of next available jump without skill and never jumped before", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(time.Time{}),
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
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(now.Add(-20 * time.Hour)),
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
		c := factory.CreateCharacter()
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
