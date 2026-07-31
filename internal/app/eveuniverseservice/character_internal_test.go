package eveuniverseservice

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestOptionalID(t *testing.T) {
	tests := []struct {
		name     string
		input    optional.Optional[*app.EveEntity]
		expected optional.Optional[int64]
	}{
		{
			name: "returns optional with ID when EveEntity is present",
			input: optional.New(&app.EveEntity{
				ID:   42,
				Name: "Jita",
			}),
			expected: optional.New(int64(42)),
		},
		{
			name:     "returns empty optional when input is empty",
			input:    optional.Optional[*app.EveEntity]{},
			expected: optional.Optional[int64]{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optionalID(tt.input)
			xassert.Equal(t, tt.expected, got)
		})
	}
}

func TestHasCharacterChanged(t *testing.T) {
	now := time.Now()

	// Base valid character parameters
	baseParams := storage.CreateEveCharacterParams{
		ID:               1001,
		Name:             "Pilot One",
		Birthday:         now,
		Gender:           "male",
		CorporationID:    2001,
		RaceID:           1,
		BloodlineID:      10,
		CorporationTitle: optional.New("CEO"),
		Description:      optional.New("A brave space pilot"),
		SecurityStatus:   optional.New(5.0),
		AllianceID:       optional.New[int64](3001),
		FactionID:        optional.New[int64](4001),
	}

	// Helper function to build a matching app.EveCharacter from baseParams
	newMatchingCharacter := func() *app.EveCharacter {
		return &app.EveCharacter{
			ID:               1001,
			Name:             "Pilot One",
			Birthday:         now,
			Gender:           "male",
			Corporation:      &app.EveEntity{ID: 2001},
			Race:             &app.EveRace{ID: 1},
			Bloodline:        optional.New(&app.EntityShort{ID: 10}),
			CorporationTitle: optional.New("CEO"),
			Description:      optional.New("A brave space pilot"),
			SecurityStatus:   optional.New(5.0),
			Alliance:         optional.New(&app.EveEntity{ID: 3001}),
			Faction:          optional.New(&app.EveEntity{ID: 4001}),
		}
	}

	tests := []struct {
		name     string
		char     func() *app.EveCharacter
		params   storage.CreateEveCharacterParams
		expected bool
	}{
		{
			name:     "Nil character should return true",
			char:     func() *app.EveCharacter { return nil },
			params:   baseParams,
			expected: true,
		},
		{
			name:     "Identical character and params should return false",
			char:     newMatchingCharacter,
			params:   baseParams,
			expected: false,
		},
		{
			name: "Changed ID should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.ID = 9999
				return c
			},
			params:   baseParams,
			expected: true,
		},
		{
			name: "Changed Name should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Name = "Different Name"
				return c
			},
			params:   baseParams,
			expected: true,
		},
		{
			name: "Changed Corporation ID should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Corporation = &app.EveEntity{ID: 9999}
				return c
			},
			params:   baseParams,
			expected: true,
		},
		{
			name: "Changed SecurityStatus should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.SecurityStatus = optional.New(1.0)
				return c
			},
			params:   baseParams,
			expected: true,
		},
		{
			name: "Changed Alliance ID should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Alliance = optional.New(&app.EveEntity{ID: 8888})
				return c
			},
			params:   baseParams,
			expected: true,
		},
		{
			name: "Missing Alliance in character but present in params should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Alliance = optional.Optional[*app.EveEntity]{}
				return c
			},
			params:   baseParams,
			expected: true,
		},
		{
			name: "Empty Alliance on both character and params should return false",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Alliance = optional.Optional[*app.EveEntity]{}
				return c
			},
			params: func() storage.CreateEveCharacterParams {
				p := baseParams
				p.AllianceID = optional.Optional[int64]{}
				return p
			}(),
			expected: false,
		},
		{
			name: "Missing Bloodline in character (zero ID fallback) matches zero BloodlineID in params",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Bloodline = optional.Optional[*app.EntityShort]{}
				return c
			},
			params: func() storage.CreateEveCharacterParams {
				p := baseParams
				p.BloodlineID = 0
				return p
			}(),
			expected: false,
		},
		{
			name: "Missing Bloodline in character when params expects a non-zero ID should return true",
			char: func() *app.EveCharacter {
				c := newMatchingCharacter()
				c.Bloodline = optional.Optional[*app.EntityShort]{}
				return c
			},
			params:   baseParams, // has BloodlineID = 10
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCharacterChanged(tt.char(), tt.params)
			xassert.Equal(t, tt.expected, got)
		})
	}
}
