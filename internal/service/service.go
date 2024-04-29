// Package service contains the business logic
package service

import (
	"net/http"
	"time"

	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	ihttp "example/evebuddy/internal/helper/http"
	"example/evebuddy/internal/storage"
)

type Service struct {
	httpClient  *http.Client
	esiClient   *goesi.APIClient
	r           *storage.Storage
	singleGroup *singleflight.Group
}

func NewService(r *storage.Storage) *Service {
	defaultHttpClient := &http.Client{
		Timeout:   time.Second * 30,
		Transport: ihttp.LoggedTransport{},
	}
	esiHttpClient := &http.Client{
		Timeout:   time.Second * 30,
		Transport: ihttp.ESITransport{},
	}
	userAgent := "EveBuddy kalkoken87@gmail.com"
	esiClient := goesi.NewAPIClient(esiHttpClient, userAgent)
	s := Service{
		httpClient:  defaultHttpClient,
		esiClient:   esiClient,
		r:           r,
		singleGroup: new(singleflight.Group),
	}
	return &s
}
