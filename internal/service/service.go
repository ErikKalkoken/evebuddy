// Package service contains the business logic
package service

import (
	"net/http"
	"time"

	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	myHttp "example/evebuddy/internal/helper/http"
	"example/evebuddy/internal/storage"
)

type Service struct {
	httpClient  *http.Client
	esiClient   *goesi.APIClient
	r           *storage.Storage
	singleGroup *singleflight.Group
}

func NewService(r *storage.Storage) *Service {
	httpClient := &http.Client{
		Timeout:   time.Second * 30, // Timeout after 30 seconds
		Transport: myHttp.CustomTransport{},
	}
	userAgent := "EveBuddy kalkoken87@gmail.com"
	esiClient := goesi.NewAPIClient(httpClient, userAgent)
	s := Service{
		httpClient:  httpClient,
		esiClient:   esiClient,
		r:           r,
		singleGroup: new(singleflight.Group),
	}
	return &s
}
