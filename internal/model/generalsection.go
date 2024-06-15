package model

import (
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const generalSectionDefaultTimeout = 24 * time.Hour

// A general section represents a topic that can be updated, e.g. market prices
type GeneralSection string

const (
	SectionEveCategories   GeneralSection = "Eve_Categories"
	SectionEveCharacters   GeneralSection = "Eve_Characters"
	SectionEveMarketPrices GeneralSection = "Eve_MarketPrices"
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

func (gs GeneralSection) DisplayName() string {
	t := strings.ReplaceAll(string(gs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
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
