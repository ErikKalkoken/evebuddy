package model

import (
	"fmt"
	"log/slog"
	"time"
)

const eveUniverseSectionDefaultTimeout = 24 * time.Hour

type EveUniverseSection string

const (
	SectionEveCategories EveUniverseSection = "EveCategories"
	SectionEveCharacters EveUniverseSection = "EveCharacters"
)

type EveUniverseUpdateStatus struct {
	ContentHash   string
	ErrorMessage  string
	LastUpdatedAt time.Time
	Section       EveUniverseSection
}

var EveUniverseSections = []EveUniverseSection{
	SectionEveCategories,
	SectionEveCharacters,
}

var eveUniverseSectionTimeouts = map[EveUniverseSection]time.Duration{
	SectionEveCategories: 24 * time.Hour,
	SectionEveCharacters: 1 * time.Hour,
}

// Timeout returns the time until the data of an update section becomes stale.
func (es EveUniverseSection) Timeout() time.Duration {
	duration, ok := eveUniverseSectionTimeouts[es]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", es)
		return eveUniverseSectionDefaultTimeout
	}
	return duration
}

func (es EveUniverseSection) Key() string {
	return fmt.Sprintf("eveuniverse-section-%s", es)
}
