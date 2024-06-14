package model

import (
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
func (gs GeneralSection) Timeout() time.Duration {
	duration, ok := generalSectionTimeouts[gs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", gs)
		return generalSectionDefaultTimeout
	}
	return duration
}
