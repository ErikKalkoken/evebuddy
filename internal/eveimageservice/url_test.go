package eveimageservice_test

import (
	"fmt"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
)

var testCases = []struct {
	id    int32
	size  int
	valid bool
}{
	{42, -1, false},
	{42, 0, false},
	{42, 16, false},
	{42, 32, true},
	{42, 64, true},
	{42, 128, true},
	{42, 256, true},
	{42, 512, true},
	{42, 1024, true},
	{42, 2048, false},
}

func TestCharacterPortraitURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("character ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.CharacterPortraitURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestCorporationLogoURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("corporation ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.CorporationLogoURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/corporations/%d/logo?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestAllianceLogoURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("alliance ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.AllianceLogoURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/alliances/%d/logo?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestFactionLogoURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("faction ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.FactionLogoURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/corporations/%d/logo?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestInventoryTypeRenderURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("inventory type render ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.InventoryTypeRenderURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/types/%d/render?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestInventoryTypeIconURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("inventory type icon ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.InventoryTypeIconURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/types/%d/icon?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestInventoryTypeXURL_ReplaceInvalids(t *testing.T) {
	cases := []struct {
		name string
		id   int32
		want string
	}{
		{"corporation", 2, "https://images.evetech.net/corporations/1/logo?size=64"},
		{"alliance", 16159, "https://images.evetech.net/alliances/1/logo?size=64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := eveimageservice.InventoryTypeIconURL(tc.id, 64)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)

			got, err = eveimageservice.InventoryTypeRenderURL(tc.id, 64)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)

			got, err = eveimageservice.InventoryTypeBPCURL(tc.id, 64)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)

			got, err = eveimageservice.InventoryTypeBPOURL(tc.id, 64)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestInventoryTypeXURL_InvalidIDs(t *testing.T) {
	for _, id := range []int32{0, 1, 3, 4, 5} {
		t.Run(fmt.Sprintf("invalid inventory id %d", id), func(t *testing.T) {
			_, err := eveimageservice.InventoryTypeIconURL(id, 64)
			assert.ErrorIs(t, err, eveimageservice.ErrInvalid)

			_, err = eveimageservice.InventoryTypeRenderURL(id, 64)
			assert.ErrorIs(t, err, eveimageservice.ErrInvalid)

			_, err = eveimageservice.InventoryTypeBPCURL(id, 64)
			assert.ErrorIs(t, err, eveimageservice.ErrInvalid)

			_, err = eveimageservice.InventoryTypeBPOURL(id, 64)
			assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
		})
	}
}

func TestInventoryTypeBPOURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("inventory type BPO ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.InventoryTypeBPOURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/types/%d/bp?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}

func TestInventoryTypeBPCURL(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("inventory type BPC ID:%d size:%d", tc.id, tc.size), func(t *testing.T) {
			got, err := eveimageservice.InventoryTypeBPCURL(tc.id, tc.size)
			if tc.valid && assert.NoError(t, err) {
				want := fmt.Sprintf("https://images.evetech.net/types/%d/bpc?size=%d", tc.id, tc.size)
				assert.Equal(t, want, got)
			} else {
				assert.ErrorIs(t, err, eveimageservice.ErrInvalid)
			}
		})
	}
}
