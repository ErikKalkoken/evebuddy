package eveimageservice_test

import (
	"fmt"

	"testing"

	"github.com/stretchr/testify/assert"

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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
			}
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
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
				assert.ErrorIs(t, err, eveimageservice.ErrInvalidSize)
			}
		})
	}
}
