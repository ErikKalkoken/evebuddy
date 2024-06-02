// Package service contains the business logic.
package service

import (
	"net/http"

	"github.com/antihax/goesi"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	ihttp "github.com/ErikKalkoken/evebuddy/internal/helper/http"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
	"github.com/ErikKalkoken/evebuddy/internal/service/characterstatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/service/esistatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveimage"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// Service provides access to all services.
type Service struct {
	// Characters service
	Characters *character.CharacterService
	// Character status service
	CharacterStatus *characterstatus.CharacterStatusCache
	// Dictionary service
	Dictionary *dictionary.Dictionary
	// EveImage service
	EveImage *eveimage.EveImageService
	// ESI status service
	ESIStatus *esistatus.ESIStatus
	// Eve Universe service
	EveUniverse *eveuniverse.EveUniverseService
}

// New creates and returns a new service instance.
// st must point to a valid storage instance
// imageCachePath can be empty. Then a temporary directory will be create and used instead.
func New(st *storage.Storage, imageCacheDir string) *Service {
	httpClient := &http.Client{
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
	dt := dictionary.New(st)
	eu := eveuniverse.New(st, esiClient)
	cache := cache.New()
	cs := characterstatus.New(cache)
	if err := cs.InitCache(st); err != nil {
		panic(err)
	}
	sv := Service{
		Characters:      character.New(st, httpClient, esiClient, cs, dt, eu),
		CharacterStatus: cs,
		Dictionary:      dt,
		EveImage:        eveimage.New(imageCacheDir, httpClient),
		ESIStatus:       esistatus.New(esiClient),
		EveUniverse:     eu,
	}
	return &sv
}
