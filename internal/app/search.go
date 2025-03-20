package app

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SearchCategory string

const (
	SearchAgent         SearchCategory = "agent"
	SearchAlliance      SearchCategory = "alliance"
	SearchCharacter     SearchCategory = "character"
	SearchConstellation SearchCategory = "constellation"
	SearchCorporation   SearchCategory = "corporation"
	SearchFaction       SearchCategory = "faction"
	SearchRegion        SearchCategory = "region"
	SearchSolarSystem   SearchCategory = "solar_system"
	SearchStation       SearchCategory = "station"
	SearchType          SearchCategory = "inventory_type"
)

func (x SearchCategory) String() string {
	titler := cases.Title(language.English)
	return titler.String(strings.ReplaceAll(string(x), "_", " "))
}

// SearchCategories returns all available search categories
func SearchCategories() []SearchCategory {
	return []SearchCategory{
		SearchAgent,
		SearchAlliance,
		SearchCharacter,
		SearchConstellation,
		SearchCorporation,
		SearchFaction,
		SearchRegion,
		SearchSolarSystem,
		SearchStation,
		SearchType,
	}
}
