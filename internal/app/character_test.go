package app_test

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestCharacterAssetIsContainer(t *testing.T) {
	cases := []struct {
		IsSingleton   bool
		EveCategoryID int32
		want          bool
	}{
		{true, app.EveCategoryShip, true},
		{false, app.EveCategoryShip, false},
		{true, app.EveCategoryDrone, false},
	}
	for _, tc := range cases {
		t.Run("should report whether asset is a container", func(t *testing.T) {
			c := &app.EveCategory{ID: tc.EveCategoryID}
			g := &app.EveGroup{Category: c}
			typ := &app.EveType{Group: g}
			ca := app.CharacterAsset{IsSingleton: tc.IsSingleton, Type: typ}
			assert.Equal(t, tc.want, ca.IsContainer())
		})
	}
}

func TestCharacterAssetTypeName(t *testing.T) {
	t.Run("has type", func(t *testing.T) {
		ca := &app.CharacterAsset{
			Type: &app.EveType{
				Name: "Alpha",
			},
		}
		assert.Equal(t, "Alpha", ca.TypeName())
	})
	t.Run("no type", func(t *testing.T) {
		ca := &app.CharacterAsset{}
		assert.Equal(t, "", ca.TypeName())
	})
}

func TestCharacterContractDisplayName(t *testing.T) {
	cases := []struct {
		name     string
		contract *app.CharacterContract
		want     string
	}{
		{
			"courier contract",
			&app.CharacterContract{
				Type:             app.ContractTypeCourier,
				Volume:           10,
				StartSolarSystem: &app.EntityShort[int32]{Name: "Start"},
				EndSolarSystem:   &app.EntityShort[int32]{Name: "End"},
			},
			"Start >> End (10 m3)",
		},
		{
			"courier contract without solar systems",
			&app.CharacterContract{
				Type:   app.ContractTypeCourier,
				Volume: 10,
			},
			"? >> ? (10 m3)",
		},
		{
			"non-courier contract with multiple items",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"first", "second"},
			},
			"[Multiple Items]",
		},
		{
			"non-courier contract with single items",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"first"},
			},
			"first",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.contract.NameDisplay())
		})
	}
}

func TestCharacterNotification(t *testing.T) {
	t.Run("can convert type to title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "AlphaBravoCharlie",
		}
		y := x.TitleFake()
		assert.Equal(t, "Alpha Bravo Charlie", y)
	})
	t.Run("can deal with short name", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "Alpha",
		}
		y := x.TitleFake()
		assert.Equal(t, "Alpha", y)
	})
}

func TestCharacterNotificationBodyPlain(t *testing.T) {
	t.Run("can return body as plain text", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: "Alpha",
			Body: optional.New("**alpha**"),
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.Equal(t, "alpha\n", got.MustValue())
		}
	})
	t.Run("should return empty when body is empty", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: "Alpha",
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}

func TestCharacterPlanetExtractedTypes(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	productType1a := &app.EveType{ID: 1}
	productType1b := &app.EveType{ID: 1}
	productType2 := &app.EveType{ID: 2}
	extractorPin1a := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: productType1a,
	}
	extractorPin1b := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: productType1b,
	}
	extractorPin2 := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: productType2,
	}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	processorPin := &app.PlanetPin{
		Type: processorType,
	}
	t.Run("should return unique extracted types", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			extractorPin1a,
			extractorPin1b,
			extractorPin2,
			processorPin,
		}}
		// when
		x := cp.ExtractedTypes()
		// then
		got := make([]int32, 0)
		for _, o := range x {
			got = append(got, o.ID)
		}
		assert.ElementsMatch(t, []int32{productType1a.ID, productType2.ID}, got)
	})
	t.Run("should return empty when no extractor", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{processorPin}}
		// when
		x := cp.ExtractedTypes()
		// then
		assert.Len(t, x, 0)
	})
	t.Run("should return empty when extractor, but no extraction product", func(t *testing.T) {
		// given
		pin := &app.PlanetPin{
			Type: extractorType,
		}
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{pin}}
		// when
		x := cp.ExtractedTypes()
		// then
		assert.Len(t, x, 0)
	})
}

func TestCharacterPlanetProducedSchematics(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	extractorPin := &app.PlanetPin{
		Type: extractorType,
	}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	schematic1a := &app.EveSchematic{ID: 1}
	processorPin1a := &app.PlanetPin{
		Type:      processorType,
		Schematic: schematic1a,
	}
	schematic1b := &app.EveSchematic{ID: 1}
	processorPin1b := &app.PlanetPin{
		Type:      processorType,
		Schematic: schematic1b,
	}
	schematic2 := &app.EveSchematic{ID: 2}
	processorPin2 := &app.PlanetPin{
		Type:      processorType,
		Schematic: schematic2,
	}
	t.Run("should return produced schematics", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			extractorPin,
			processorPin1a,
			processorPin1b,
			processorPin2,
		}}
		// when
		x := cp.ProducedSchematics()
		// then
		got := make([]int32, 0)
		for _, o := range x {
			got = append(got, o.ID)
		}
		assert.ElementsMatch(t, []int32{schematic1a.ID, schematic2.ID}, got)

	})
	t.Run("should return empty when no processor", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{extractorPin}}
		// when
		x := cp.ProducedSchematics()
		// then
		assert.Len(t, x, 0)
	})
	t.Run("should return empty when producer, but no schematic", func(t *testing.T) {
		// given
		pin := &app.PlanetPin{
			Type: processorType,
		}
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{pin}}
		// when
		x := cp.ExtractedTypes()
		// then
		assert.Len(t, x, 0)
	})
}

func TestCharacterPlanetExtractionsExpire(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	productType := &app.EveType{ID: 42}
	processorPin := &app.PlanetPin{Type: processorType}
	t.Run("should return final expiration date", func(t *testing.T) {
		// given
		et1 := time.Now().Add(5 * time.Hour).UTC()
		et2 := time.Now().Add(10 * time.Hour).UTC()
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type:                 extractorType,
				ExpiryTime:           optional.New(et2),
				ExtractorProductType: productType,
			},
			{
				Type:                 extractorType,
				ExpiryTime:           optional.New(et1),
				ExtractorProductType: productType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpiryTime()
		// then
		assert.Equal(t, et2, x)
	})
	t.Run("should return expiration date in the past", func(t *testing.T) {
		// given
		et1 := time.Now().Add(-5 * time.Hour).UTC()
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type:                 extractorType,
				ExpiryTime:           optional.New(et1),
				ExtractorProductType: productType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpiryTime()
		// then
		assert.Equal(t, et1, x)
	})
	t.Run("should return zero time when no expiration date", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type: extractorType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpiryTime()
		// then
		assert.True(t, x.IsZero())
	})
}

type MyCS struct {
	items []*app.CharacterSkillqueueItem
	err   error
}

func (cs MyCS) ListSkillqueueItems(context.Context, int32) ([]*app.CharacterSkillqueueItem, error) {
	if cs.err != nil {
		return nil, cs.err
	}
	return cs.items, nil
}

func makeSkillQueueItem(characterID int32, args ...app.CharacterSkillqueueItem) *app.CharacterSkillqueueItem {
	var arg app.CharacterSkillqueueItem
	if len(args) > 0 {
		arg = args[0]
	}
	now := time.Now()
	arg.CharacterID = characterID
	if arg.FinishedLevel == 0 {
		arg.FinishedLevel = rand.IntN(5) + 1
	}
	if arg.LevelEndSP == 0 {
		arg.LevelEndSP = rand.IntN(1_000_000)
	}
	if arg.StartDate.IsZero() {
		hours := rand.IntN(10)*24 + 3
		arg.StartDate = now.Add(time.Duration(-hours) * time.Hour)
	}
	if arg.FinishDate.IsZero() {
		hours := rand.IntN(10)*24 + 3
		arg.FinishDate = now.Add(time.Duration(hours) * time.Hour)
	}
	if arg.GroupName == "" {
		arg.GroupName = "Group"
	}
	if arg.SkillName == "" {
		arg.SkillName = "Skill"
	}
	if arg.SkillDescription == "" {
		arg.SkillDescription = "Description"
	}
	return &arg
}

func TestCharacterSkillqueue(t *testing.T) {
	characterID := int32(42)
	t.Run("can return information about an active skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		item1 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:  time.Now().Add(-3 * time.Hour),
			FinishDate: time.Now().Add(3 * time.Hour),
		})
		item2 := makeSkillQueueItem(characterID, app.CharacterSkillqueueItem{
			StartDate:  time.Now().Add(3 * time.Hour),
			FinishDate: time.Now().Add(7 * time.Hour),
		})
		cs := MyCS{items: []*app.CharacterSkillqueueItem{item1, item2}}
		err := sq.Update(cs, characterID)
		if assert.NoError(t, err) {
			assert.Equal(t, characterID, sq.CharacterID())
			assert.Equal(t, 2, sq.Size())
			assert.Equal(t, item1, sq.Current())
			assert.Equal(t, item2, sq.Item(1))
			assert.InDelta(t, 0.5, sq.Completion().ValueOrZero(), 0.01)
			assert.WithinDuration(t, toTime(7*time.Hour), toTime(sq.Remaining().ValueOrZero()), 10*time.Second)
			assert.True(t, sq.IsActive())
		}
	})
	t.Run("can return information about an empty skill queue", func(t *testing.T) {
		sq := app.NewCharacterSkillqueue()
		assert.Equal(t, int32(0), sq.CharacterID())
		assert.Equal(t, 0, sq.Size())
		assert.Nil(t, sq.Current())
		assert.Nil(t, sq.Item(1))
		assert.True(t, sq.Completion().IsEmpty())
		assert.False(t, sq.IsActive())
	})
}

func toTime(d time.Duration) time.Time {
	return time.Now().Add(d)
}

func TestSkillqueueItemCompletion(t *testing.T) {
	t.Run("should calculate when started at 0", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:       now.Add(time.Hour * -1),
			FinishDate:      now.Add(time.Hour * +3),
			LevelStartSP:    0,
			LevelEndSP:      100,
			TrainingStartSP: 0,
		}
		assert.InDelta(t, 0.25, q.CompletionP(), 0.01)
	})
	t.Run("should calculate when with sp offset 1", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:       now.Add(time.Hour * -1),
			FinishDate:      now.Add(time.Hour * +1),
			LevelStartSP:    0,
			LevelEndSP:      100,
			TrainingStartSP: 50,
		}
		assert.InDelta(t, 0.75, q.CompletionP(), 0.01)
	})
	t.Run("should calculate when with sp offset 2", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:       now.Add(time.Hour * -2),
			FinishDate:      now.Add(time.Hour * +1),
			LevelStartSP:    0,
			LevelEndSP:      100,
			TrainingStartSP: 25,
		}
		assert.InDelta(t, 0.75, q.CompletionP(), 0.01)
	})
	t.Run("should calculate when with sp offset 3", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:       now.Add(time.Hour * -2),
			FinishDate:      now.Add(time.Hour * +1),
			LevelStartSP:    100,
			LevelEndSP:      200,
			TrainingStartSP: 125,
		}
		assert.InDelta(t, 0.75, q.CompletionP(), 0.01)
	})
	t.Run("should return 0 when starting in the future", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:  now.Add(time.Hour * +1),
			FinishDate: now.Add(time.Hour * +3),
		}
		assert.Equal(t, 0.0, q.CompletionP())
	})
	t.Run("should return 1 when finished in the past", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:  now.Add(time.Hour * -3),
			FinishDate: now.Add(time.Hour * -1),
		}
		assert.Equal(t, 1.0, q.CompletionP())
	})
}

func TestSkillqueueItemDuration(t *testing.T) {
	t.Run("should return duration when possible", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate:  now.Add(time.Hour * +1),
			FinishDate: now.Add(time.Hour * +3),
		}
		d := q.Duration()
		assert.Equal(t, 2*time.Hour, d.MustValue())
	})
	t.Run("should return null when duration can not be calculated 1", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			StartDate: now.Add(time.Hour * +1),
		}
		d := q.Duration()
		assert.True(t, d.IsEmpty())
	})
	t.Run("should return null when duration can not be calculated 2", func(t *testing.T) {
		now := time.Now()
		q := app.CharacterSkillqueueItem{
			FinishDate: now.Add(time.Hour * +1),
		}
		d := q.Duration()
		assert.True(t, d.IsEmpty())
	})
	t.Run("should return null when duration can not be calculated 3", func(t *testing.T) {
		q := app.CharacterSkillqueueItem{}
		d := q.Duration()
		assert.True(t, d.IsEmpty())
	})
}

func makeItem(startDate, finishDate time.Time) app.CharacterSkillqueueItem {
	return app.CharacterSkillqueueItem{
		StartDate:       startDate,
		FinishDate:      finishDate,
		LevelStartSP:    0,
		LevelEndSP:      1000,
		TrainingStartSP: 0,
	}
}

func TestSkillqueueItemRemaining(t *testing.T) {
	t.Run("should return correct value when finish in the future", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now, now.Add(time.Hour*+3))
		d := q.Remaining()
		assert.InDelta(t, 3*time.Hour, d.MustValue(), 10000)
	})
	t.Run("should return correct value when start and finish in the future", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now.Add(time.Hour*+1), now.Add(time.Hour*+3))
		d := q.Remaining()
		assert.InDelta(t, 2*time.Hour, d.MustValue(), 10000)
	})
	t.Run("should return 0 remaining when completed", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now.Add(time.Hour*-3), now.Add(time.Hour*-2))
		d := q.Remaining()
		assert.Equal(t, time.Duration(0), d.MustValue())
	})
	t.Run("should return null when no finish date", func(t *testing.T) {
		now := time.Now()
		q := makeItem(now, time.Time{})
		d := q.Remaining()
		assert.True(t, d.IsEmpty())
	})
	t.Run("should return null when no start date", func(t *testing.T) {
		now := time.Now()
		q := makeItem(time.Time{}, now.Add(time.Hour*+2))
		d := q.Remaining()
		assert.True(t, d.IsEmpty())
	})
}

func TestTokenRemainsValid(t *testing.T) {
	t.Run("return true, when token remains valid within duration", func(t *testing.T) {
		x := app.CharacterToken{ExpiresAt: time.Now().Add(60 * time.Second)}
		assert.True(t, x.RemainsValid(55*time.Second))
	})
	t.Run("return false, when token expired within duration", func(t *testing.T) {
		x := app.CharacterToken{ExpiresAt: time.Now().Add(60 * time.Second)}
		assert.False(t, x.RemainsValid(65*time.Second))
	})
}
