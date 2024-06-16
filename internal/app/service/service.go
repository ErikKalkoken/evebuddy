// Package service contains all services.
package service

import (
	"net/http"

	"github.com/antihax/goesi"

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatus"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"github.com/ErikKalkoken/evebuddy/internal/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/httptransport"
)

// Service is the main service and provides access to all other services.
// It ensures all services are properly initialized
// and allows re-use of shared resources like the http client.
// Services provide access to the business logic for the UI.
type Service struct {
	// Cache service
	Cache *cache.Cache
	// Character service
	Character *character.CharacterService
	// Dictionary service
	Dictionary *dictionary.DictionaryService
	// EveImage service
	EveImage *eveimage.EveImageService
	// ESI status service
	ESIStatus *esistatus.ESIStatusService
	// Eve Universe service
	EveUniverse *eveuniverse.EveUniverseService
	// Status cache service
	StatusCache *statuscache.StatusCacheService
}

// New creates and returns a new instance of the main service.
// st must point to a valid storage instance
// imageCachePath can be empty. Then a temporary directory will be create and used instead.
func New(st *storage.Storage, imageCacheDir string) *Service {
	httpClient := &http.Client{
		Transport: httptransport.LoggedTransport{},
	}
	esiHttpClient := &http.Client{
		Transport: httptransport.LoggedTransportWithRetries{
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
	dt := dictionary.New(st)
	cache := cache.New()
	sc := statuscache.New(cache)
	if err := sc.InitCache(st); err != nil {
		panic(err)
	}
	eu := eveuniverse.New(st, esiClient, dt, sc)
	sv := Service{
		Cache:       cache,
		Character:   character.New(st, httpClient, esiClient, sc, dt, eu),
		Dictionary:  dt,
		EveImage:    eveimage.New(imageCacheDir, httpClient),
		ESIStatus:   esistatus.New(esiClient),
		EveUniverse: eu,
		StatusCache: sc,
	}
	return &sv
}
