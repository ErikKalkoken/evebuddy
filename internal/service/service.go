// Package service contains the business logic
package service

import (
	"net/http"

	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	ihttp "github.com/ErikKalkoken/evebuddy/internal/helper/http"
	"github.com/ErikKalkoken/evebuddy/internal/service/characterstatus"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type Service struct {
	cache           *cache.Cache
	esiClient       *goesi.APIClient
	httpClient      *http.Client
	r               *storage.Storage
	singleGroup     *singleflight.Group
	characterStatus *characterstatus.CharacterStatusCache
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
	s := Service{
		cache:       cache.New(),
		esiClient:   esiClient,
		httpClient:  defaultHttpClient,
		r:           r,
		singleGroup: new(singleflight.Group),
	}
	s.characterStatus = characterstatus.New(s.cache)
	if err := s.characterStatus.InitCache(r); err != nil {
		panic(err)
	}
	return &s
}
