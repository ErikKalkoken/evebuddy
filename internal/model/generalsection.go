package model

import (
	"fmt"
	"log/slog"
	"time"
)

const generalSectionDefaultTimeout = 24 * time.Hour

type GeneralSection string

const (
	SectionEveCategories   GeneralSection = "EveCategories"
	SectionEveCharacters   GeneralSection = "EveCharacters"
	SectionEveMarketPrices GeneralSection = "EveMarketPrices"
)

// Updates status of a general section
type GeneralSectionStatus struct {
	ID           int64
	ContentHash  string
	ErrorMessage string
	CompletedAt  time.Time
	Section      GeneralSection
	StartedAt    time.Time
	UpdatedAt    time.Time
}

var GeneralSections = []GeneralSection{
	SectionEveCategories,
	SectionEveCharacters,
	SectionEveMarketPrices,
}

var generalSectionTimeouts = map[GeneralSection]time.Duration{
	SectionEveCategories:   24 * time.Hour,
	SectionEveCharacters:   1 * time.Hour,
	SectionEveMarketPrices: 6 * time.Hour,
}

// Timeout returns the time until the data of an update section becomes stale.
func (es GeneralSection) Timeout() time.Duration {
	duration, ok := generalSectionTimeouts[es]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", es)
		return generalSectionDefaultTimeout
	}
	return duration
}

func (es GeneralSection) KeyCompletedAt() string {
	return fmt.Sprintf("general-section-%s-completed-at", es)
}

func (es GeneralSection) KeyError() string {
	return fmt.Sprintf("general-section-%s-error", es)
}

func (es GeneralSection) KeyStartedAt() string {
	return fmt.Sprintf("eveuniverse-section-%s-started-at", es)
}
