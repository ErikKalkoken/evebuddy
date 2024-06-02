// Package service contains the business logic.
package service

import (
	"net/http"

	"github.com/antihax/goesi"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	ihttp "github.com/ErikKalkoken/evebuddy/internal/helper/http"
	"github.com/ErikKalkoken/evebuddy/internal/service/characters"
	"github.com/ErikKalkoken/evebuddy/internal/service/characterstatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/service/esistatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type Service struct {
	// Characters service
	Characters *characters.Characters
	// Character status service
	CharacterStatus *characterstatus.CharacterStatusCache
	// Dictionary service
	Dictionary *dictionary.Dictionary
	// ESI status service
	ESIStatus *esistatus.ESIStatus
	// Eve Universe service
	EveUniverse *eveuniverse.EveUniverse
}

func NewService(r *storage.Storage) *Service {
	defaultHttpClient := &http.Client{
		Transport: ihttp.LoggedTransport{},
	}
	esiHttpClient := &http.Client{
		Transport: ihttp.LoggedTransportWithRetries{
			MaxRetries: 3,
			StatusCodesToRetry: []int{
				http.StatusBadGateway,
				http.StatusGatewayTimeout,
				http.StatusServiceUnavailable,
			},
		},
	}
	userAgent := "EveBuddy kalkoken87@gmail.com"
	esiClient := goesi.NewAPIClient(esiHttpClient, userAgent)
	dt := dictionary.New(r)
	eu := eveuniverse.New(r, esiClient)
	cache := cache.New()
	cs := characterstatus.New(cache)
	if err := cs.InitCache(r); err != nil {
		panic(err)
	}
	s := Service{
		Characters:      characters.New(r, defaultHttpClient, esiClient, cs, dt, eu),
		CharacterStatus: cs,
		Dictionary:      dt,
		ESIStatus:       esistatus.New(esiClient),
		EveUniverse:     eu,
	}
	return &s
}
