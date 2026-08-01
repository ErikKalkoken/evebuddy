package app

import (
	"cmp"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	npcCorporationIDBegin = 1_000_000
	npcCorporationIDEnd   = 2_000_000
	npcCharacterIDBegin   = 3_000_000
	npcCharacterIDEnd     = 4_000_000
)

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int64
	Name     string
}

// EveEntity returns itself.
// Needed to comply with the same interface as many other eve universe models.
func (ee *EveEntity) EveEntity() *EveEntity {
	return ee
}

func (ee *EveEntity) IDOrZero() int64 {
	if ee == nil {
		return 0
	}
	return ee.ID
}

func (ee *EveEntity) NameOrZero() string {
	if ee == nil {
		return ""
	}
	return ee.Name
}

func (ee *EveEntity) CategoryDisplay() string {
	if ee == nil {
		return "?"
	}
	return xstrings.Title(ee.Category.String())
}

// IsValid reports whether an entity is valid.
func (ee *EveEntity) IsValid() bool {
	if ee == nil {
		return false
	}
	return ee.Category.IsKnown()
}

// IsCharacter reports whether an entity is a character.
func (ee *EveEntity) IsCharacter() bool {
	if ee == nil {
		return false
	}
	return ee.Category == EveEntityCharacter
}

// IsNPC reports whether an entity is a NPC.
//
// This function only works for characters and corporations and returns an empty value for anything else..
func (ee *EveEntity) IsNPC() optional.Optional[bool] {
	if ee == nil {
		return optional.Optional[bool]{}
	}
	switch ee.Category {
	case EveEntityCharacter:
		return optional.New(IsNPCCharacter(ee.ID))
	case EveEntityCorporation:
		return optional.New(IsNPCCorporation(ee.ID))
	}
	return optional.Optional[bool]{}
}

func (ee *EveEntity) Compare(other *EveEntity) int {
	if ee == nil {
		return 0
	}
	return cmp.Compare(ee.Name, other.Name)
}

// InfoLink returns a link to show information in Eve format.
func (ee *EveEntity) InfoLink() (string, error) {
	if ee == nil {
		return "", fmt.Errorf("no value: %w", ErrInvalid)
	}

	makeLink := func(typeID int64, itemID int64) string {
		x := fmt.Sprintf("showinfo:%d", typeID)
		if itemID > 0 {
			x += fmt.Sprintf("//%d", itemID)
		}
		return x
	}

	switch ee.Category {
	case EveEntityAlliance:
		return makeLink(EveTypeAlliance, ee.ID), nil
	case EveEntityCharacter:
		return makeLink(EveTypeCharacter, ee.ID), nil
	case EveEntityConstellation:
		return makeLink(EveTypeConstellation, ee.ID), nil
	case EveEntityCorporation:
		return makeLink(EveTypeCorporation, ee.ID), nil
	case EveEntityFaction:
		return makeLink(EveTypeFaction, ee.ID), nil
	case EveEntityInventoryType:
		return makeLink(ee.ID, 0), nil
	case EveEntityRegion:
		return makeLink(EveTypeRegion, ee.ID), nil
	case EveEntitySolarSystem:
		return makeLink(EveTypeSolarSystem, ee.ID), nil
	case EveEntityStation:
		return makeLink(EveTypeCaldariLogisticsStation, ee.ID), nil
	}
	return "", fmt.Errorf("not defined for category %s: %w", ee.Category, ErrInvalid)
}

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
	EveEntityUnknown
)

// IsKnown reports whether a category is known.
func (eec EveEntityCategory) IsKnown() bool {
	return eec != EveEntityUndefined && eec != EveEntityUnknown
}

func (eec EveEntityCategory) String() string {
	switch eec {
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
	case EveEntityUnknown:
		return "unknown"
	default:
		return "?"
	}
}

// IsNPCCharacter reports whether an entity ID represents a NPC character.
func IsNPCCharacter(id int64) bool {
	if id >= npcCharacterIDBegin && id < npcCharacterIDEnd {
		return true
	}
	return false
}

// IsNPCCorporation reports whether an entity ID represents a NPC corporation.
func IsNPCCorporation(id int64) bool {
	if id >= npcCorporationIDBegin && id < npcCorporationIDEnd {
		return true
	}
	return false
}
