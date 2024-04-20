package model

import (
	"example/evebuddy/internal/api/images"
	"fmt"

	"fyne.io/fyne/v2"
)

type EveEntityCategory int

// Supported categories of EveEntity
const (
	EveEntityUndefined EveEntityCategory = iota
	EveEntityAlliance
	EveEntityCharacter
	EveEntityConstellation
	EveEntityCorporation
	EveEntityFaction
	EveEntityInventoryType
	EveEntityMailList
	EveEntityRegion
	EveEntitySolarSystem
	EveEntityStation
)

func (e EveEntityCategory) String() string {
	switch e {
	case EveEntityUndefined:
		return "undefined"
	case EveEntityAlliance:
		return "alliance"
	case EveEntityCharacter:
		return "character"
	case EveEntityConstellation:
		return "constellation"
	case EveEntityCorporation:
		return "corporation"
	case EveEntityFaction:
		return "faction"
	case EveEntityInventoryType:
		return "inventory type"
	case EveEntityMailList:
		return "mailing list"
	case EveEntityRegion:
		return "region"
	case EveEntitySolarSystem:
		return "solar system"
	case EveEntityStation:
		return "station"
	default:
		return "unknown"
	}
}

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

// IconURL returns the URL for an icon image of an entity.
func (e *EveEntity) IconURL(size int) (fyne.URI, error) {
	switch e.Category {
	case EveEntityAlliance:
		return images.AllianceLogoURL(e.ID, size)
	case EveEntityCharacter:
		return images.CharacterPortraitURL(e.ID, size)
	case EveEntityCorporation:
		return images.CorporationLogoURL(e.ID, size)
	case EveEntityFaction:
		return images.FactionLogoURL(e.ID, size)
	case EveEntityInventoryType:
		return images.InventoryTypeRenderURL(e.ID, size)
	}
	return nil, fmt.Errorf("can not match category: %v", e.Category)
}
