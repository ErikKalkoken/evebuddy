// Package service contains the business logic
package service

import (
	"net/http"
	"time"

	myHttp "example/evebuddy/internal/helper/http"
	"example/evebuddy/internal/storage"

	"github.com/antihax/goesi"
)

type Service struct {
	httpClient *http.Client
	esiClient  *goesi.APIClient
	r          *storage.Storage
}

func NewService(r *storage.Storage) *Service {
	httpClient := &http.Client{
		Timeout:   time.Second * 30, // Timeout after 30 seconds
		Transport: myHttp.CustomTransport{},
	}
	userAgent := "EveBuddy kalkoken87@gmail.com"
	esiClient := goesi.NewAPIClient(httpClient, userAgent)
	s := Service{
		httpClient: httpClient,
		esiClient:  esiClient,
		r:          r,
	}
	return &s
}
